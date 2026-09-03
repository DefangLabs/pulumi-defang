package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/DefangLabs/pulumi-defang/cd/program"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/stretchr/testify/require"
)

// These states retain the URN/type/parent shapes from real deployments while
// intentionally omitting inputs and outputs, which may contain customer
// secrets. The legacy AWS fixture came from defang-mvp's TypeScript CD; the
// GCP fixture came from its Go CD; the new fixtures came from this CD's
// recorded previews. mixed-state-gcp is a failed partial takeover.
func fixtureDeployment(t *testing.T, name string) apitype.UntypedDeployment {
	t.Helper()
	// Callers pass only fixture constants declared in this test file.
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec
	require.NoError(t, err)
	var doc apitype.UntypedDeployment
	require.NoError(t, json.Unmarshal(data, &doc))
	return doc
}

func fixtureIdentities(t *testing.T, name string) []resourceIdentity {
	t.Helper()
	resources, err := resourceIdentitiesIn(fixtureDeployment(t, name))
	require.NoError(t, err)
	return resources
}

func fixtureWithoutTypes(t *testing.T, name string, types ...string) apitype.UntypedDeployment {
	t.Helper()
	deployment := fixtureDeployment(t, name)
	var snapshot apitype.DeploymentV3
	require.NoError(t, json.Unmarshal(deployment.Deployment, &snapshot))
	snapshot.Resources = slices.DeleteFunc(snapshot.Resources, func(res apitype.ResourceV3) bool {
		return slices.Contains(types, string(res.Type))
	})
	data, err := json.Marshal(snapshot)
	require.NoError(t, err)
	deployment.Deployment = data
	return deployment
}

func configFor(cloud string) configMap {
	return configMap{"defang:provider": {Value: cloud}}
}

const (
	testProjectName = "my-app"
	testStackName   = "beta"
	testTarget      = testProjectName + "/" + testStackName

	gcpManagedCompose = `services:
  app:
    image: nginx
  postgres-service:
    image: postgres:16
    x-defang-postgres: true
    environment:
      POSTGRES_USER: djangouser
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: djangodatabase
  redis-service:
    image: redis:7
    x-defang-redis: true
`
	awsManagedCompose = `services:
  postgres:
    image: postgres:16
    x-defang-postgres: true
  redis:
    image: redis:7
    x-defang-redis: true
`
)

var errSecretExport = errors.New("export failed; decrypted password=TOP-SECRET")

type fakeStack struct {
	deployment apitype.UntypedDeployment
	err        error
}

func (f fakeStack) Export(context.Context) (apitype.UntypedDeployment, error) {
	return f.deployment, f.err
}

func prepareFixture(
	t *testing.T, fixture, cloud, composeYAML, recipe string, enforce bool,
) (program.ServiceAliases, error) {
	t.Helper()
	aliases := program.ServiceAliases{}
	err := prepareLegacyState(t.Context(), fakeStack{deployment: fixtureDeployment(t, fixture)}, recipe,
		testProjectName, testStackName, []byte(composeYAML), cloud, configFor(cloud), aliases, enforce)
	return aliases, err
}

func TestCurrentAndEmptyStatesPassForTheirCloud(t *testing.T) {
	tests := []struct {
		fixture string
		cloud   string
	}{
		{"empty-state.json", cloudGCP},
		{"new-state-aws.json", cloudAWS},
		{"new-state-gcp.json", cloudGCP},
		{"new-state-azure.json", cloudAzure},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			aliases, err := prepareFixture(t, tt.fixture, tt.cloud, "services: {}\n", "", true)
			require.NoError(t, err)
			require.Empty(t, aliases)
			require.Empty(t, foreignResources(fixtureIdentities(t, tt.fixture), tt.cloud))
		})
	}
}

func TestRealLegacyStatesPrepareAliasesBeforeReportingRuntimeBlockers(t *testing.T) {
	tests := []struct {
		name          string
		fixture       string
		cloud         string
		composeYAML   string
		wantAlias     string
		wantBlockType string
	}{
		{
			name:        "GCP private build plugin",
			fixture:     "legacy-state-gcp.json",
			cloud:       cloudGCP,
			composeYAML: gcpManagedCompose,
			wantAlias: "postgres-service/instance=gcp:sql/databaseInstance:DatabaseInstance::" +
				"my-app-postgres-service-postgres",
			wantBlockType: "cloudbuild:index:CloudBuild",
		},
		{
			name:          "AWS Node dynamic provider",
			fixture:       "legacy-state-aws.json",
			cloud:         cloudAWS,
			composeYAML:   awsManagedCompose,
			wantAlias:     "postgres/instance=aws:rds/instance:Instance::e-cd-s***-postgres",
			wantBlockType: "pulumi-nodejs:dynamic:Resource",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases, err := prepareFixture(t, tt.fixture, tt.cloud, tt.composeYAML, "", true)
			var migrationErr *legacyStateError
			require.ErrorAs(t, err, &migrationErr)
			require.Contains(t, stableAliasSummary(aliases), tt.wantAlias)
			require.Contains(t, err.Error(), tt.wantBlockType)
		})
	}
}

func TestRealLegacyStatesWithoutUnavailableBuildResourcesAreAdoptable(t *testing.T) {
	tests := []struct {
		name        string
		deployment  apitype.UntypedDeployment
		cloud       string
		composeYAML string
		wantAliases []string
	}{
		{
			name:        "GCP",
			deployment:  fixtureWithoutTypes(t, "legacy-state-gcp.json", "cloudbuild:index:CloudBuild"),
			cloud:       cloudGCP,
			composeYAML: gcpManagedCompose,
			wantAliases: []string{
				"postgres-service/database=gcp:sql/database:Database::my-app-postgres-service-postgres-db",
				"postgres-service/instance=gcp:sql/databaseInstance:DatabaseInstance::my-app-postgres-service-postgres",
				"postgres-service/user=gcp:sql/user:User::my-app-postgres-service-postgres-user",
				"redis-service/instance=gcp:redis/instance:Instance::my-app-redis-service-redis",
			},
		},
		{
			name:        "AWS",
			deployment:  fixtureWithoutTypes(t, "legacy-state-aws.json", "pulumi-nodejs:dynamic:Resource"),
			cloud:       cloudAWS,
			composeYAML: awsManagedCompose,
			wantAliases: []string{
				"postgres/instance=aws:rds/instance:Instance::e-cd-s***-postgres",
				"postgres/security-group=aws:ec2/securityGroup:SecurityGroup::E-cd-s***-postgres",
				"postgres/subnet-group=aws:rds/subnetGroup:SubnetGroup::postgres",
				"redis/cluster=aws:elasticache/replicationGroup:ReplicationGroup::redis",
				"redis/security-group=aws:ec2/securityGroup:SecurityGroup::E-cd-s***-redis",
				"redis/subnet-group=aws:elasticache/subnetGroup:SubnetGroup::E-cd-s***-redis",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliases := program.ServiceAliases{}
			err := prepareLegacyState(t.Context(), fakeStack{deployment: tt.deployment}, "", testProjectName,
				testStackName, []byte(tt.composeYAML), tt.cloud, configFor(tt.cloud), aliases, true)
			require.NoError(t, err)
			require.Equal(t, tt.wantAliases, stableAliasSummary(aliases))
		})
	}
}

func TestEveryReleasedLegacyResourceHasAReviewedDisposition(t *testing.T) {
	for _, tt := range []struct {
		fixture string
		cloud   string
	}{
		{"legacy-state-aws.json", cloudAWS},
		{"legacy-state-gcp.json", cloudGCP},
	} {
		t.Run(tt.fixture, func(t *testing.T) {
			for _, res := range foreignResources(fixtureIdentities(t, tt.fixture), tt.cloud) {
				reason := unsupportedLegacyTypes[res.typ]
				reviewed := reason != "" || len(specsFor(tt.cloud, res.typ)) != 0 || isReplaceableLegacyResource(res)
				require.Truef(t, reviewed, "unclassified legacy resource %s", res.display())
			}
		})
	}
}

func TestUnknownSameCloudResourcesFailClosed(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		cloud       string
		composeYAML string
		typ         string
		resource    string
	}{
		{"AWS arbitrary S3 bucket", "legacy-state-aws.json", cloudAWS, awsManagedCompose,
			"aws:s3/bucket:Bucket", "customer-data"},
		{"AWS EFS", "legacy-state-aws.json", cloudAWS, awsManagedCompose,
			"aws:efs/fileSystem:FileSystem", "customer-files"},
		{"GCP arbitrary bucket", "legacy-state-gcp.json", cloudGCP, gcpManagedCompose,
			"gcp:storage/bucket:Bucket", "customer-data"},
		{"GCP secret", "legacy-state-gcp.json", cloudGCP, gcpManagedCompose,
			"gcp:secretmanager/secret:Secret", "customer-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blockedType := tt.typ
			deployment := fixtureWithoutTypes(t, tt.fixture,
				"cloudbuild:index:CloudBuild", "pulumi-nodejs:dynamic:Resource")
			resources, err := resourceIdentitiesIn(deployment)
			require.NoError(t, err)
			resources = append(resources, mkIdentity(
				"urn:pulumi:beta::my-app::"+tt.typ+"::"+tt.resource,
				"urn:pulumi:beta::my-app::pulumi:pulumi:Stack::my-app-beta",
			))

			plan := analyzeLegacyState(resources, mustDesiredServices(t, tt.composeYAML), tt.cloud, configFor(tt.cloud))
			require.Condition(t, func() bool {
				return slices.ContainsFunc(plan.blockers, func(problem migrationProblem) bool {
					return problem.resource.typ == blockedType &&
						strings.Contains(problem.reason, "no reviewed replacement or adoption rule")
				})
			})
		})
	}
}

func TestMissingOrAmbiguousDatabaseAliasesBlockUp(t *testing.T) {
	deployment := fixtureWithoutTypes(t, "legacy-state-gcp.json", "cloudbuild:index:CloudBuild")
	tests := []struct {
		name        string
		composeYAML string
	}{
		{"database service removed", "services:\n  app:\n    image: nginx\n"},
		{"database service renamed", strings.ReplaceAll(gcpManagedCompose, "postgres-service:", "renamed-postgres:")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName,
				testStackName, []byte(tt.composeYAML), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, true)
			require.Error(t, err)
			require.Contains(t, err.Error(), "no unique matching managed service")
		})
	}
}

func TestDuplicateExplicitAliasesAreAmbiguous(t *testing.T) {
	const legacyURN = "urn:pulumi:beta::my-app::aws:rds/instance:Instance::legacy-postgres"
	services := map[string]desiredService{
		"first":  {kind: servicePostgres, aliases: map[string]string{compose.AliasInstance: legacyURN}},
		"second": {kind: servicePostgres, aliases: map[string]string{compose.AliasInstance: legacyURN}},
	}
	resources := []resourceIdentity{
		mkIdentity(legacyURN, ""),
		mkIdentity("urn:pulumi:beta::my-app::defang-mvp:shared/ecs/defang:Defang::defang", ""),
	}
	plan := analyzeLegacyState(resources, services, cloudAWS, configFor(cloudAWS))
	require.Len(t, plan.blockers, 1)
	require.Contains(t, plan.blockers[0].reason, "no unique matching managed service")
}

func TestMixedStateAdoptsSurvivingLegacyDatabasesButRejectsDuplicateTarget(t *testing.T) {
	deployment := fixtureWithoutTypes(t, "mixed-state-gcp.json", "cloudbuild:index:CloudBuild")
	aliases := program.ServiceAliases{}
	require.NoError(t, prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName,
		testStackName, []byte(gcpManagedCompose), cloudGCP, configFor(cloudGCP), aliases, true))
	require.Contains(t, stableAliasSummary(aliases),
		"postgres-service/instance=gcp:sql/databaseInstance:DatabaseInstance::my-app-postgres-service-postgres")

	resources, err := resourceIdentitiesIn(deployment)
	require.NoError(t, err)
	resources = append(resources, mkIdentity(
		"urn:pulumi:beta::my-app::defang-gcp:index:Project$defang-gcp:index:Postgres$"+
			"gcp:sql/databaseInstance:DatabaseInstance::postgres-service",
		"urn:pulumi:beta::my-app::defang-gcp:index:Project$defang-gcp:index:Postgres::postgres-service"))
	plan := analyzeLegacyState(resources, mustDesiredServices(t, gcpManagedCompose), cloudGCP, configFor(cloudGCP))
	require.NotEmpty(t, plan.blockers)
	require.Contains(t, plan.blockers[0].reason, "already has a resource")
}

func TestConditionalCloudSQLChildrenMustHaveAnAliasTarget(t *testing.T) {
	deployment := fixtureWithoutTypes(t, "legacy-state-gcp.json", "cloudbuild:index:CloudBuild")
	composeWithoutChildren := `services:
  postgres-service:
    image: postgres:16
    x-defang-postgres: true
  redis-service:
    image: redis:7
    x-defang-redis: true
`
	aliases := program.ServiceAliases{}
	err := prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName,
		testStackName, []byte(composeWithoutChildren), cloudGCP, configFor(cloudGCP), aliases, true)
	require.ErrorContains(t, err, "will not register a resource to consume this alias")
	require.NotContains(t, aliases["postgres-service"], compose.AliasUser)
	require.NotContains(t, aliases["postgres-service"], compose.AliasDatabase)
	require.Contains(t, aliases["postgres-service"], compose.AliasInstance)
}

func TestCloudDetectionRejectsProviderSwitchesAndLegacyAzure(t *testing.T) {
	// A current GCP state is safe only for GCP. Cross-cloud up must not delete it.
	plan := analyzeLegacyState(
		fixtureIdentities(t, "new-state-gcp.json"), map[string]desiredService{}, cloudAWS, configFor(cloudAWS),
	)
	require.NotEmpty(t, plan.blockers)
	require.Contains(t, plan.blockers[0].reason, "selected AWS")

	azureLegacy := []resourceIdentity{
		mkIdentity("urn:pulumi:prod::app::pulumi:pulumi:Stack::app-prod", ""),
		mkIdentity(
			"urn:pulumi:prod::app::azure-native:dbforpostgresql:Server::postgres",
			"urn:pulumi:prod::app::pulumi:pulumi:Stack::app-prod",
		),
	}
	plan = analyzeLegacyState(azureLegacy, mustDesiredServices(t, awsManagedCompose), cloudAzure, configFor(cloudAzure))
	require.NotEmpty(t, plan.blockers)
	require.Contains(t, plan.blockers[0].reason, "no legacy Azure")
}

func TestAWSRedisEngineMismatchBlocksUnsafeTypeAdoption(t *testing.T) {
	deployment := fixtureWithoutTypes(t, "legacy-state-aws.json", "pulumi-nodejs:dynamic:Resource")
	resources, err := resourceIdentitiesIn(deployment)
	require.NoError(t, err)
	config := configFor(cloudAWS)
	config["defang-aws:redis-engine"] = configValue{Value: "memorydb"}
	plan := analyzeLegacyState(resources, mustDesiredServices(t, awsManagedCompose), cloudAWS, config)
	require.Condition(t, func() bool {
		return slices.ContainsFunc(plan.blockers, func(problem migrationProblem) bool {
			return strings.Contains(problem.reason, "different resource type")
		})
	})
}

func TestPreviewNeverBlocksButReceivesProvableAliases(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := stderrLogger
	stderrLogger = &logs
	t.Cleanup(func() { stderrLogger = originalLogger })

	aliases, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose, "", false)
	require.NoError(t, err)
	require.NotEmpty(t, aliases)
	require.Contains(t, logs.String(), "postgres-service/instance=gcp:sql/databaseInstance:DatabaseInstance::")
	require.Contains(t, logs.String(), "a real up will stop")
	require.Contains(t, logs.String(), "cloudbuild:index:CloudBuild::app-build")
	require.Contains(t, logs.String(), "does not contain the legacy private cloudbuild plugin")
}

func TestStateInspectionFailsClosedForUpAndDoesNotLeakSecrets(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := stderrLogger
	stderrLogger = &logs
	t.Cleanup(func() { stderrLogger = originalLogger })

	secretErr := errSecretExport
	err := prepareLegacyState(t.Context(), fakeStack{err: secretErr}, "", testProjectName, testStackName,
		[]byte(gcpManagedCompose), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, true)
	var inspectionErr *stateInspectionError
	require.ErrorAs(t, err, &inspectionErr)
	require.NotContains(t, err.Error(), "TOP-SECRET")
	require.NotContains(t, logs.String(), "TOP-SECRET")

	// Preview applies no infrastructure changes, so the same inspection failure
	// warns but does not prevent Pulumi from showing whatever plan it can produce.
	require.NoError(t, prepareLegacyState(t.Context(), fakeStack{err: secretErr}, "", testProjectName, testStackName,
		[]byte(gcpManagedCompose), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, false))
	require.NotContains(t, logs.String(), "TOP-SECRET")

	deployment := deploymentOf(`{"resources":[{"urn":"urn:pulumi:beta::my-app::gcp:storage/bucket:Bucket::data",` +
		`"type":"gcp:storage/bucket:Bucket","outputs":{"password":"TOP-SECRET"}}]}`)
	err = prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName, testStackName,
		[]byte(gcpManagedCompose), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, true)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "TOP-SECRET")
	require.NotContains(t, logs.String(), "TOP-SECRET")
}

func TestMalformedStateNeverPanicsAndFailsClosedForUp(t *testing.T) {
	tests := []apitype.UntypedDeployment{
		deploymentOf(`{{{`),
		deploymentOf(`{"resources":[{}]}`),
		deploymentOf(`{"resources":[null]}`),
		deploymentOf(`{"resources":[{"urn":"not-a-urn"}]}`),
		deploymentOf(`{"resources":[{"urn":"urn:pulumi:s::p::aws:rds/instance:Instance::db","parent":"garbage"}]}`),
		{Version: 1, Deployment: json.RawMessage(`{"latest":{"resources":[]}}`)},
	}
	for i, deployment := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			require.NotPanics(t, func() {
				err := prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName,
					testStackName, []byte(gcpManagedCompose), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, true)
				require.Error(t, err)
			})
		})
	}
}

func TestExactStackOverrideScope(t *testing.T) {
	tests := []struct {
		name   string
		recipe string
		allow  bool
	}{
		{"stack settings yaml", "config:\n  defang:allowLegacyStateTakeover: " + testTarget + "\n", true},
		{"flat json", `{"defang:allowLegacyStateTakeover":{"value":"` + testTarget + `"}}`, true},
		{"boolean is tenant-wide and rejected", "config:\n  defang:allowLegacyStateTakeover: true\n", false},
		{"other stack", "config:\n  defang:allowLegacyStateTakeover: my-app/prod\n", false},
		{"other project", "config:\n  defang:allowLegacyStateTakeover: other/beta\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose, tt.recipe, true)
			if tt.allow {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

	for _, value := range []string{"", "1", "true", "my-app/prod", "other/beta"} {
		t.Run("env="+value, func(t *testing.T) {
			t.Setenv(allowTakeoverEnv, value)
			_, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose, "", true)
			require.Error(t, err)
		})
	}
	t.Run("exact env", func(t *testing.T) {
		t.Setenv(allowTakeoverEnv, testTarget)
		_, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose, "", true)
		require.NoError(t, err)
	})
}

func TestExactStackOverrideLogsSanitizedBlockerDetails(t *testing.T) {
	var logs bytes.Buffer
	originalLogger := stderrLogger
	stderrLogger = &logs
	t.Cleanup(func() { stderrLogger = originalLogger })

	aliases, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose,
		"config:\n  defang:allowLegacyStateTakeover: "+testTarget+"\n", true)
	require.NoError(t, err)
	require.NotEmpty(t, aliases)
	require.Contains(t, logs.String(), "continuing despite")
	require.Contains(t, logs.String(), "cloudbuild:index:CloudBuild::app-build")
	require.Contains(t, logs.String(), "does not contain the legacy private cloudbuild plugin")
	require.NotContains(t, logs.String(), "POSTGRES_PASSWORD")
	require.NotContains(t, logs.String(), "secret")
}

func TestLegacyStateErrorIsActionableAndBounded(t *testing.T) {
	_, err := prepareFixture(t, "legacy-state-gcp.json", cloudGCP, gcpManagedCompose, "", true)
	var migrationErr *legacyStateError
	require.ErrorAs(t, err, &migrationErr)
	msg := err.Error()
	require.Contains(t, msg, "Pulumi alias")
	require.Contains(t, msg, "Nothing has been changed")
	require.Contains(t, msg, "preview")
	require.Contains(t, msg, "`down`")
	require.Contains(t, msg, migrationRunbook)
	require.NotContains(t, msg, allowTakeoverConfigKey)
	require.NotContains(t, msg, allowTakeoverEnv)
	require.LessOrEqual(t, strings.Count(msg, "\n"), 20)
}

func TestForeignResourceDetectionUsesCloudPackageAndParent(t *testing.T) {
	const (
		stackURN   = "urn:pulumi:gcp::cd-test::pulumi:pulumi:Stack::cd-test-gcp"
		projectURN = "urn:pulumi:gcp::cd-test::defang-gcp:index:Project::cd-test"
	)
	base := make([]resourceIdentity, 0, 3)
	base = append(base, mkIdentity(stackURN, ""), mkIdentity(projectURN, stackURN))
	for name := range thisCDTopLevelNames {
		t.Run(name, func(t *testing.T) {
			atRoot := mkIdentity("urn:pulumi:gcp::cd-test::gcp:storage/bucketObject:BucketObject::"+name, stackURN)
			underLegacy := mkIdentity(
				"urn:pulumi:gcp::cd-test::defang-mvp:legacy:Project$"+
					"gcp:storage/bucketObject:BucketObject::"+name,
				"urn:pulumi:gcp::cd-test::defang-mvp:legacy:Project::legacy")
			require.Empty(t, foreignResources(append(slices.Clone(base), atRoot), cloudGCP))
			require.Len(t, foreignResources(append(slices.Clone(base), underLegacy), cloudGCP), 1)
		})
	}

	// A custom resource under this CD's Project inherits the defang package in
	// its qualified type and is accepted; the same name at the root is not.
	underProject := mkIdentity("urn:pulumi:gcp::cd-test::defang-gcp:index:Project$custom:mod:Thing::x", projectURN)
	require.Empty(t, foreignResources(append(base, underProject), cloudGCP))
}

func TestLegacyFixturesRemainForeignToCurrentCD(t *testing.T) {
	for _, tt := range []struct {
		fixture string
		cloud   string
	}{
		{"legacy-state-aws.json", cloudAWS},
		{"legacy-state-gcp.json", cloudGCP},
	} {
		foreign := foreignResources(fixtureIdentities(t, tt.fixture), tt.cloud)
		require.NotEmpty(t, foreign)
		for _, res := range foreign {
			require.NotContains(t, string(res.urn), thisCDPackages[tt.cloud]+":")
		}
	}
}

// Guard the hand-maintained historical top-level name list against current
// registrations. The source scan is pinned so a refactor cannot silently make
// the test match nothing.
func TestThisCDTopLevelNamesCoversEveryRootRegistration(t *testing.T) {
	registration := regexp.MustCompile(`\.New([A-Za-z]+)\((?:ctx|pctx), ("[^"]*"|[\w.]+)[,)]`)
	assignment := regexp.MustCompile(`(?m)^\s*(?:const\s+|var\s+)?(\w+)\s*=\s*"([^"]*)"`)
	sources, err := filepath.Glob(filepath.Join("program", "*.go"))
	require.NoError(t, err)

	bodies, constants := map[string]string{}, map[string]string{}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		// Sources come from the fixed program/*.go glob immediately above.
		body, err := os.ReadFile(source) //nolint:gosec
		require.NoError(t, err)
		bodies[source] = string(body)
		for _, match := range assignment.FindAllStringSubmatch(string(body), -1) {
			constants[match[1]] = match[2]
		}
	}

	found := 0
	for source, body := range bodies {
		for _, match := range registration.FindAllStringSubmatch(body, -1) {
			kind, arg := match[1], match[2]
			if kind == "Provider" || kind == "Project" {
				continue
			}
			name, err := strconv.Unquote(arg)
			if err != nil {
				name = constants[arg]
				require.NotEmptyf(t, name, "%s names a resource with %q, which this test cannot resolve", source, arg)
			}
			found++
			require.Truef(t, thisCDTopLevelNames[name], "%s registers unclassified top-level resource %q", source, name)
		}
	}
	require.Equal(t, 8, found)
}

func TestTopLevelNameHistoryAndOverrideArePinned(t *testing.T) {
	for _, name := range []string{"project-pb", "self-destruct", "defang-self-destruct", "self-destruct-starter"} {
		require.True(t, thisCDTopLevelNames[name])
	}
	env := program.SelfDestructEnv([]string{"PROJECT=app", allowTakeoverEnv + "=" + testTarget})
	require.NotContains(t, env, allowTakeoverEnv)
}

func TestResourceIdentitiesInEmptyDeployment(t *testing.T) {
	fresh := apitype.UntypedDeployment{Version: apitype.DeploymentSchemaVersionCurrent, Deployment: json.RawMessage(
		`{"manifest":{"time":"2026-08-28T00:00:00Z","magic":"m","version":"v3.259.0"}}`)}
	resources, err := resourceIdentitiesIn(fresh)
	require.NoError(t, err)
	require.Empty(t, resources)
	resources, err = resourceIdentitiesIn(apitype.UntypedDeployment{})
	require.NoError(t, err)
	require.Empty(t, resources)
}

func mustDesiredServices(t *testing.T, yaml string) map[string]desiredService {
	t.Helper()
	services, err := desiredManagedServices([]byte(yaml))
	require.NoError(t, err)
	return services
}

func mkIdentity(urn, parent string) resourceIdentity {
	u := resource.URN(urn)
	return resourceIdentity{urn: u, typ: string(u.Type()), parent: resource.URN(parent)}
}

func deploymentOf(body string) apitype.UntypedDeployment {
	return apitype.UntypedDeployment{Version: apitype.DeploymentSchemaVersionCurrent, Deployment: json.RawMessage(body)}
}

func TestConfiguredAliasConflictBlocks(t *testing.T) {
	const conflictingAlias = `x-defang-postgres: true
    x-defang-aliases:
      instance: urn:pulumi:other::project::gcp:sql/databaseInstance:DatabaseInstance::other`
	composeYAML := strings.Replace(gcpManagedCompose, "x-defang-postgres: true",
		conflictingAlias, 1)
	deployment := fixtureWithoutTypes(t, "legacy-state-gcp.json", "cloudbuild:index:CloudBuild")
	err := prepareLegacyState(t.Context(), fakeStack{deployment: deployment}, "", testProjectName, testStackName,
		[]byte(composeYAML), cloudGCP, configFor(cloudGCP), program.ServiceAliases{}, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "x-defang-aliases")
}

func TestAliasKindsUsedByMigrationAreStable(t *testing.T) {
	require.Equal(t, []string{
		compose.AliasCluster,
		compose.AliasDatabase,
		compose.AliasInstance,
		compose.AliasParameterGroup,
		compose.AliasSecurityGroup,
		compose.AliasSubnetGroup,
		compose.AliasUser,
	}, []string{"cluster", "database", "instance", "parameter-group", "security-group", "subnet-group", "user"})
}
