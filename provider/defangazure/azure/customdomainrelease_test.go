package azure

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	armappcontainers "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v3"
)

const (
	testRG       = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/proj-stack/providers/Microsoft.App"
	testAppID    = testRG + "/containerApps/app"
	testCertID   = testRG + "/managedEnvironments/proj/managedCertificates/mc-proj-app-1234"
	testUploadID = testRG + "/managedEnvironments/proj/certificates/byo"
)

var errTestBoom = errors.New("boom")

type fakeCustomDomainAPI struct {
	domains []*armappcontainers.CustomDomain
	getErr  error
	inUse   int // number of delete attempts that report CertificateInUse
	calls   []string
}

func (f *fakeCustomDomainAPI) customDomains(
	_ context.Context, app *arm.ResourceID,
) ([]*armappcontainers.CustomDomain, error) {
	f.calls = append(f.calls, "get "+app.Name)
	return f.domains, f.getErr
}

func (f *fakeCustomDomainAPI) clearCustomDomains(_ context.Context, app *arm.ResourceID) error {
	f.calls = append(f.calls, "clear "+app.Name)
	f.domains = nil
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

func shortRetryDelay(t *testing.T) {
	t.Helper()
	old := certInUseRetryDelay
	certInUseRetryDelay = time.Millisecond
	t.Cleanup(func() { certInUseRetryDelay = old })
}

func mustParseID(t *testing.T, id string) *arm.ResourceID {
	t.Helper()
	r, err := arm.ParseResourceID(id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestReleaseApp_UnbindsBeforeDeletingCert(t *testing.T) {
	shortRetryDelay(t)
	api := &fakeCustomDomainAPI{
		domains: []*armappcontainers.CustomDomain{
			{Name: to.Ptr("app.example.com"), CertificateID: to.Ptr(testCertID)},
			{Name: to.Ptr("www.example.com"), CertificateID: to.Ptr(testUploadID)},
			{Name: to.Ptr("pending.example.com")},
		},
		inUse: 2,
	}
	if err := releaseApp(t.Context(), api, mustParseID(t, testAppID), t.Logf); err != nil {
		t.Fatal(err)
	}
	const deleteCert = "delete proj/mc-proj-app-1234"
	want := []string{"get app", "clear app", deleteCert, deleteCert, deleteCert} // two CertificateInUse retries
	if !slices.Equal(api.calls, want) {
		t.Errorf("calls = %v, want %v", api.calls, want)
	}
}

func TestReleaseApp_NoCustomDomains(t *testing.T) {
	api := &fakeCustomDomainAPI{}
	if err := releaseApp(t.Context(), api, mustParseID(t, testAppID), t.Logf); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(api.calls, []string{"get app"}) {
		t.Errorf("calls = %v", api.calls)
	}
}

func TestReleaseApp_AppGone(t *testing.T) {
	api := &fakeCustomDomainAPI{getErr: &azcore.ResponseError{StatusCode: http.StatusNotFound}}
	if err := releaseApp(t.Context(), api, mustParseID(t, testAppID), t.Logf); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestReleaseApp_GetError(t *testing.T) {
	api := &fakeCustomDomainAPI{getErr: errTestBoom}
	if err := releaseApp(t.Context(), api, mustParseID(t, testAppID), t.Logf); !errors.Is(err, errTestBoom) {
		t.Errorf("err = %v, want %v", err, errTestBoom)
	}
}

func TestReleaseCustomDomains_BadID(t *testing.T) {
	if err := ReleaseCustomDomains(t.Context(), "not-an-id", t.Logf); err == nil {
		t.Error("want an error for an invalid ID")
	}
}

func TestDeleteCertWhenUnused_StopsAtDeadline(t *testing.T) {
	shortRetryDelay(t)
	api := &fakeCustomDomainAPI{inUse: 1 << 30}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := deleteCertWhenUnused(ctx, api, mustParseID(t, testCertID)); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want deadline exceeded", err)
	}
}
