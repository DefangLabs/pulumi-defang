package program

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
)

const (
	testRG       = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/proj-stack/providers/Microsoft.App"
	testAppID    = testRG + "/containerApps/app"
	testWorkerID = testRG + "/containerApps/worker"
	testCertID   = testRG + "/managedEnvironments/proj/managedCertificates/mc-proj-app-1234"
	testUploadID = testRG + "/managedEnvironments/proj/certificates/byo"
)

const (
	testURNPrefix = "urn:pulumi:s::p::"
	projectURN    = testURNPrefix + "defang-azure:index:Project::p"
	serviceT      = "defang-azure:index:Project$defang-azure:index:Service"
	appURN        = testURNPrefix + serviceT + "::app"
	workerURN     = testURNPrefix + serviceT + "::worker"
	acaT          = serviceT + "$azure-native:app:ContainerApp"
	envURN        = testURNPrefix + "defang-azure:index:Project$azure-native:app:ManagedEnvironment::p"
	dbURN         = testURNPrefix + "defang-azure:index:Project$azure-native:dbforpostgresql:Server::db"
)

func testState(t *testing.T) []byte {
	t.Helper()
	res := func(urn, typ, parent, id string, deps ...string) map[string]any {
		r := map[string]any{"urn": urn, "type": typ, "parent": parent, "dependencies": deps}
		if id != "" {
			r["outputs"] = map[string]any{"id": id}
		}
		return r
	}
	b, err := json.Marshal(map[string]any{"resources": []any{
		res(projectURN, "defang-azure:index:Project", "", ""),
		// The environment is listed after the app on purpose: selection must
		// not depend on checkpoint order.
		res(appURN, "defang-azure:index:Service", projectURN, ""),
		res(testURNPrefix+acaT+"::app", containerAppType, appURN, testAppID, envURN),
		res(workerURN, "defang-azure:index:Service", projectURN, ""),
		res(testURNPrefix+acaT+"::worker", containerAppType, workerURN, testWorkerID),
		res(envURN, "azure-native:app:ManagedEnvironment", projectURN, "env"),
		res(dbURN, "azure-native:dbforpostgresql:Server", projectURN, "db"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var errTestBoom = errors.New("boom")

func TestAzureContainerAppsToRelease(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		want    []string
	}{
		{"no targets", nil, []string{testAppID, testWorkerID}},
		{"service component", []string{appURN}, []string{testAppID}},
		{"container app", []string{testURNPrefix + acaT + "::worker"}, []string{testWorkerID}},
		{"project", []string{projectURN}, []string{testAppID, testWorkerID}},
		{"dependency", []string{envURN}, []string{testAppID}},
		{"unrelated", []string{dbURN}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AzureContainerAppsToRelease(testState(t), tt.targets)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAzureContainerAppsToRelease_NonAzure(t *testing.T) {
	state := `{"resources":[{"urn":"u","type":"aws:ecs/service:Service","outputs":{"id":"x"}}]}`
	got, err := AzureContainerAppsToRelease([]byte(state), nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
	got, err = AzureContainerAppsToRelease(nil, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty state: got %v, %v", got, err)
	}
}

type fakeCustomDomainAPI struct {
	domains map[string][]*armappcontainers.CustomDomain
	getErr  error
	inUse   int // number of delete attempts that report CertificateInUse
	calls   []string
}

func (f *fakeCustomDomainAPI) customDomains(
	_ context.Context, app *arm.ResourceID,
) ([]*armappcontainers.CustomDomain, error) {
	f.calls = append(f.calls, "get "+app.Name)
	return f.domains[app.Name], f.getErr
}

func (f *fakeCustomDomainAPI) clearCustomDomains(_ context.Context, app *arm.ResourceID) error {
	f.calls = append(f.calls, "clear "+app.Name)
	f.domains[app.Name] = nil
	return nil
}

func (f *fakeCustomDomainAPI) deleteManagedCert(_ context.Context, cert *arm.ResourceID) error {
	f.calls = append(f.calls, "delete "+cert.Parent.Name+"/"+cert.Name)
	if f.inUse > 0 {
		f.inUse--
		return &azcore.ResponseError{StatusCode: http.StatusConflict, ErrorCode: "CertificateInUse"}
	}
	return nil
}

func TestReleaseCustomDomains_UnbindsBeforeDeletingCert(t *testing.T) {
	shortRetryDelay(t)
	api := &fakeCustomDomainAPI{
		domains: map[string][]*armappcontainers.CustomDomain{
			"app": {
				{Name: ptr("app.example.com"), CertificateID: ptr(testCertID)},
				{Name: ptr("www.example.com"), CertificateID: ptr(testUploadID)},
				{Name: ptr("pending.example.com")},
			},
		},
		inUse: 2,
	}
	releaseCustomDomains(t.Context(), api, []string{testAppID, testWorkerID}, t.Logf)
	const deleteCert = "delete proj/mc-proj-app-1234"
	want := []string{
		"get app", "clear app",
		deleteCert, deleteCert, deleteCert, // two CertificateInUse retries
		"get worker",
	}
	if !slices.Equal(api.calls, want) {
		t.Errorf("calls = %v, want %v", api.calls, want)
	}
}

func TestReleaseCustomDomains_AppGone(t *testing.T) {
	api := &fakeCustomDomainAPI{getErr: &azcore.ResponseError{StatusCode: http.StatusNotFound}}
	var logged []string
	releaseCustomDomains(t.Context(), api, []string{testAppID}, func(f string, _ ...any) { logged = append(logged, f) })
	if !slices.Equal(api.calls, []string{"get app"}) || len(logged) != 0 {
		t.Errorf("calls = %v, logged = %v", api.calls, logged)
	}
}

func TestReleaseCustomDomains_GetErrorIsLogged(t *testing.T) {
	api := &fakeCustomDomainAPI{getErr: errTestBoom}
	var logged []string
	logf := func(f string, _ ...any) { logged = append(logged, f) }
	releaseCustomDomains(t.Context(), api, []string{testAppID, "not-an-id"}, logf)
	if len(logged) != 2 {
		t.Errorf("logged = %v, want 2 entries", logged)
	}
}

func TestDeleteCertWhenUnused_StopsAtDeadline(t *testing.T) {
	shortRetryDelay(t)
	api := &fakeCustomDomainAPI{inUse: 1 << 30}
	cert, err := arm.ParseResourceID(testCertID)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := deleteCertWhenUnused(ctx, api, cert); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want deadline exceeded", err)
	}
}

func shortRetryDelay(t *testing.T) {
	t.Helper()
	old := certInUseRetryDelay
	certInUseRetryDelay = time.Millisecond
	t.Cleanup(func() { certInUseRetryDelay = old })
}
