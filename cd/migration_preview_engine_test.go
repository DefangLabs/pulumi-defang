package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/common/util/rpcutil"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumirpc "github.com/pulumi/pulumi/sdk/v3/proto/go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const migrationTestProviderName = "migrationtest"

// TestMain doubles this test binary as a real Pulumi resource-provider plugin.
// The engine executes it through a pulumi-resource-migrationtest symlink.
func TestMain(m *testing.M) {
	if filepath.Base(os.Args[0]) == "pulumi-resource-"+migrationTestProviderName {
		os.Exit(runMigrationTestProvider())
	}
	os.Exit(m.Run())
}

func runMigrationTestProvider() int {
	cancel := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		<-signals
		close(cancel)
	}()

	handle, err := rpcutil.ServeWithOptions(rpcutil.ServeOptions{
		Cancel: cancel,
		Init: func(server *grpc.Server) error {
			pulumirpc.RegisterResourceProviderServer(server, migrationTestProvider{})
			return nil
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := fmt.Fprintln(os.Stdout, handle.Port); err != nil {
		return 1
	}
	if err := <-handle.Done; err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

type migrationTestProvider struct {
	pulumirpc.UnimplementedResourceProviderServer
}

func (migrationTestProvider) GetSchema(
	context.Context,
	*pulumirpc.GetSchemaRequest,
) (*pulumirpc.GetSchemaResponse, error) {
	return &pulumirpc.GetSchemaResponse{Schema: `{
  "name": "migrationtest",
  "version": "1.0.0",
  "resources": {
    "migrationtest:index:Database": {
      "inputProperties": {"immutable": {"type": "string"}},
      "requiredInputs": ["immutable"],
      "properties": {"immutable": {"type": "string"}},
      "required": ["immutable"]
    }
  }
}`}, nil
}

func (migrationTestProvider) GetPluginInfo(
	context.Context,
	*emptypb.Empty,
) (*pulumirpc.PluginInfo, error) {
	return &pulumirpc.PluginInfo{Version: "1.0.0"}, nil
}

func (migrationTestProvider) CheckConfig(
	_ context.Context,
	req *pulumirpc.CheckRequest,
) (*pulumirpc.CheckResponse, error) {
	return &pulumirpc.CheckResponse{Inputs: req.GetNews()}, nil
}

func (migrationTestProvider) DiffConfig(
	context.Context,
	*pulumirpc.DiffRequest,
) (*pulumirpc.DiffResponse, error) {
	return &pulumirpc.DiffResponse{Changes: pulumirpc.DiffResponse_DIFF_NONE}, nil
}

func (migrationTestProvider) Configure(
	context.Context,
	*pulumirpc.ConfigureRequest,
) (*pulumirpc.ConfigureResponse, error) {
	return &pulumirpc.ConfigureResponse{
		AcceptSecrets:   true,
		AcceptResources: true,
		AcceptOutputs:   true,
	}, nil
}

func (migrationTestProvider) Check(
	_ context.Context,
	req *pulumirpc.CheckRequest,
) (*pulumirpc.CheckResponse, error) {
	return &pulumirpc.CheckResponse{Inputs: req.GetNews()}, nil
}

func (migrationTestProvider) Diff(
	_ context.Context,
	req *pulumirpc.DiffRequest,
) (*pulumirpc.DiffResponse, error) {
	oldValue := req.GetOldInputs().GetFields()["immutable"].GetStringValue()
	newValue := req.GetNews().GetFields()["immutable"].GetStringValue()
	if oldValue == newValue {
		return &pulumirpc.DiffResponse{Changes: pulumirpc.DiffResponse_DIFF_NONE}, nil
	}
	if err := recordMigrationProviderCall("diff-replace"); err != nil {
		return nil, err
	}
	return &pulumirpc.DiffResponse{
		Changes:  pulumirpc.DiffResponse_DIFF_SOME,
		Replaces: []string{"immutable"},
		Diffs:    []string{"immutable"},
		DetailedDiff: map[string]*pulumirpc.PropertyDiff{
			"immutable": {
				Kind:      pulumirpc.PropertyDiff_UPDATE_REPLACE,
				InputDiff: true,
			},
		},
		HasDetailedDiff: true,
	}, nil
}

func (migrationTestProvider) Create(
	_ context.Context,
	req *pulumirpc.CreateRequest,
) (*pulumirpc.CreateResponse, error) {
	if err := recordMigrationProviderCall("create"); err != nil {
		return nil, err
	}
	return &pulumirpc.CreateResponse{Id: "database-id", Properties: req.GetProperties()}, nil
}

func (migrationTestProvider) Update(
	_ context.Context,
	req *pulumirpc.UpdateRequest,
) (*pulumirpc.UpdateResponse, error) {
	if !req.GetPreview() {
		if err := recordMigrationProviderCall("update"); err != nil {
			return nil, err
		}
	}
	return &pulumirpc.UpdateResponse{Properties: req.GetNews()}, nil
}

func (migrationTestProvider) Delete(
	context.Context,
	*pulumirpc.DeleteRequest,
) (*emptypb.Empty, error) {
	if err := recordMigrationProviderCall("delete"); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func recordMigrationProviderCall(call string) error {
	path := os.Getenv("DEFANG_MIGRATION_TEST_PROVIDER_LOG")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintln(file, call)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

type migrationTestDatabase struct {
	pulumi.CustomResourceState
}

type migrationTestDatabaseArgs struct {
	Immutable pulumi.StringInput `pulumi:"immutable"`
}

type migrationTestDatabaseArgsInternal struct {
	Immutable string `pulumi:"immutable"`
}

func (migrationTestDatabaseArgs) ElementType() reflect.Type {
	return reflect.TypeOf((*migrationTestDatabaseArgsInternal)(nil)).Elem()
}

func migrationTestProgram(name, immutable string, alias resource.URN) pulumi.RunFunc {
	return func(ctx *pulumi.Context) error {
		var database migrationTestDatabase
		opts := []pulumi.ResourceOption{}
		if alias != "" {
			opts = append(opts, pulumi.Aliases([]pulumi.Alias{{URN: pulumi.URN(alias)}}))
		}
		return ctx.RegisterResource(
			migrationTestProviderName+":index:Database",
			name,
			&migrationTestDatabaseArgs{Immutable: pulumi.String(immutable)},
			&database,
			opts...,
		)
	}
}

func TestEngineMigrationPreviewBlocksForcedReplacementBeforeMutation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test provider launcher uses a Unix executable symlink")
	}
	_, err := exec.LookPath("pulumi")
	require.NoError(t, err, "Pulumi CLI is required for the engine-level migration safety regression")

	providerDir := t.TempDir()
	testBinary, err := os.Executable()
	require.NoError(t, err)
	require.NoError(t, os.Symlink(
		testBinary,
		filepath.Join(providerDir, "pulumi-resource-"+migrationTestProviderName),
	))

	const (
		projectName  = "migration-engine-test"
		stackName    = "beta"
		resourceType = migrationTestProviderName + ":index:Database"
	)
	oldURN := resource.NewURN(
		tokens.QName(stackName), tokens.PackageName(projectName), "", resourceType, "legacy-database",
	)
	newURN := resource.NewURN(
		tokens.QName(stackName), tokens.PackageName(projectName), "", resourceType, "current-database",
	)

	providerLog := filepath.Join(t.TempDir(), "provider-calls.log")
	workDir := t.TempDir()
	backendDir := t.TempDir()
	pulumiHome := t.TempDir()
	ws, err := auto.NewLocalWorkspace(t.Context(),
		auto.WorkDir(workDir),
		auto.PulumiHome(pulumiHome),
		auto.Program(migrationTestProgram("legacy-database", "old-shape", "")),
		auto.EnvVars(map[string]string{
			"DEFANG_MIGRATION_TEST_PROVIDER_LOG": providerLog,
			"PATH":                               providerDir + string(os.PathListSeparator) + os.Getenv("PATH"),
			"PULUMI_BACKEND_URL":                 "file://" + backendDir,
			"PULUMI_CONFIG_PASSPHRASE":           "test-passphrase",
			"PULUMI_SKIP_UPDATE_CHECK":           "true",
		}),
		auto.Project(workspace.Project{
			Name:    tokens.PackageName(projectName),
			Runtime: workspace.NewProjectRuntimeInfo("go", nil),
		}),
	)
	require.NoError(t, err)

	stack, err := auto.UpsertStack(t.Context(), stackName, ws)
	require.NoError(t, err)
	_, err = stack.Up(t.Context(), optup.SuppressProgress())
	require.NoError(t, err)
	initialCalls, err := os.ReadFile(providerLog) //nolint:gosec
	require.NoError(t, err)
	require.Contains(t, strings.Fields(string(initialCalls)), "create")

	// From this point on, any actual Create/Delete proves the pre-Up gate let a
	// destructive provider operation through.
	require.NoError(t, os.WriteFile(providerLog, nil, 0o600))
	stack.Workspace().SetProgram(migrationTestProgram("current-database", "new-shape", oldURN))

	previewErr := verifyMigrationPreview(
		t.Context(),
		&stack,
		legacyStatePreparation{resources: []migrationPreviewResource{{oldURN: oldURN, newURN: newURN}}},
		"migration-engine-test",
		"never",
		nil,
	)
	var destructiveErr *destructiveMigrationPreviewError
	require.ErrorAs(t, previewErr, &destructiveErr)

	calls, err := os.ReadFile(providerLog) //nolint:gosec
	require.NoError(t, err)
	callList := strings.Fields(string(calls))
	require.Contains(t, callList, "diff-replace", "mock provider must explicitly request replacement")
	require.NotContains(t, callList, "create", "guard must abort before an actual provider Create")
	require.NotContains(t, callList, "delete", "guard must abort before provider Delete")
	require.NotContains(t, callList, "update", "guard must abort before provider Update")

	deployment, err := stack.Export(t.Context())
	require.NoError(t, err)
	identities, err := resourceIdentitiesIn(deployment)
	require.NoError(t, err)
	urns := make([]resource.URN, 0, len(identities))
	for _, identity := range identities {
		urns = append(urns, identity.urn)
	}
	require.Contains(t, urns, oldURN)
	require.NotContains(t, urns, newURN)
}
