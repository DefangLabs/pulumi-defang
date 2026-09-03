package program

import (
	"reflect"
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	awscompose "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each to<Cloud>ServiceArgs copies a parsed compose service into its generated
// SDK args struct field by field. A field the copy forgets is dropped in
// silence: the provider sees a service that never asked for the feature, the
// deploy succeeds, and the feature is simply absent. Every unit and
// provider-mock test still passes, because those build the component inputs
// directly and never cross this boundary. That is how x-defang-s3 reached AWS
// as a MinIO ECS task with no bucket on its first end-to-end run (#519).

const (
	cloudGCP   = "gcp"
	cloudAzure = "azure"

	// A pre-migration ElastiCache URN, the shape x-defang-aliases carries.
	clusterURN = "urn:pulumi:prod::proj::aws:elasticache/cluster:Cluster::cache"
	// A volumes_from entry: a container name with the optional mode suffix.
	volumesFromRef = "sidecar:ro"
)

// fullService sets every field of compose.ServiceConfig to a distinctive
// non-zero value, so that converting it and converting the zero service differ
// in every field of the SDK args. It is deliberately not a service anyone would
// deploy — Postgres, Redis and ObjectStore are all set at once, and no real
// compose file does that. The converters validate nothing, so it is a legal
// probe of the copy and nothing more.
func fullService() compose.ServiceConfig {
	return compose.ServiceConfig{
		Aliases:       map[string]string{compose.AliasCluster: clusterURN},
		Autoscaling:   true,
		Build:         &compose.BuildConfig{Context: pulumi.String("./src"), Dockerfile: ptr("Dockerfile.prod")},
		Command:       []string{"serve"},
		ContainerName: ptr("app-main"),
		DependsOn:     compose.DependsOnConfig{"db": {Condition: "service_healthy", Required: true}},
		Deploy:        &compose.DeployConfig{Replicas: ptr(int32(3))},
		DomainName:    "app.example.com",
		Entrypoint:    []string{"/bin/sh", "-c"},
		Environment:   compose.Environment{"LOG_LEVEL": pulumi.String("debug")},
		HealthCheck:   &compose.HealthCheckConfig{Test: []string{"CMD", "true"}},
		Image:         pulumi.String("nginx:alpine"),
		LLM:           &compose.LlmConfig{},
		NetworkMode:   "service:app",
		Networks: map[compose.NetworkID]compose.ServiceNetworkConfig{
			compose.DefaultNetwork: {Aliases: []string{"web"}},
		},
		ObjectStore:     &compose.ObjectStoreConfig{Bucket: "proj-uploads"},
		Platform:        ptr("linux/arm64"),
		Policies:        compose.PolicyList{"arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess"},
		Ports:           []compose.ServicePortConfig{{Target: 8080, Mode: compose.PortModeIngress}},
		Postgres:        &compose.PostgresConfig{FromSnapshot: ptr("snap-1")},
		Redis:           &compose.RedisConfig{FromSnapshot: ptr("snap-2")},
		Restart:         "no",
		StopGracePeriod: ptr("90s"),
		Volumes:         []compose.ServiceVolumeConfig{{Source: "data", Target: "/var/lib/data", ReadOnly: true}},
		VolumesFrom:     []string{volumesFromRef},
		WorkingDir:      ptr("/srv"),
	}
}

// TestServiceArgsCopyEveryField is the guard that makes the next added field
// fail loudly instead of silently. It carries no list of fields to check: it
// converts the maximal service above and the zero service, then compares the
// two SDK args structs field by field. A field the converter never touches
// comes out identical from both, and the test names it.
//
// So a field added to the SDK schema starts out uncopied, and therefore starts
// out failing — nobody has to remember to extend this test. Adding the field to
// fullService and to the converters is what makes it pass.
func TestServiceArgsCopyEveryField(t *testing.T) {
	full := fullService()
	empty := compose.ServiceConfig{}
	tests := []struct {
		cloud     string
		converter string
		got       any
		zero      any
	}{
		{"aws", "toAWSServiceArgs", toAWSServiceArgs(full), toAWSServiceArgs(empty)},
		{cloudGCP, "toGCPServiceArgs", toGCPServiceArgs(full), toGCPServiceArgs(empty)},
		{cloudAzure, "toAzureServiceArgs", toAzureServiceArgs(full), toAzureServiceArgs(empty)},
	}
	for _, tt := range tests {
		t.Run(tt.cloud, func(t *testing.T) {
			got, zero := reflect.ValueOf(tt.got), reflect.ValueOf(tt.zero)
			require.Positive(t, got.NumField())
			for i := range got.NumField() {
				assert.NotEqual(t, zero.Field(i).Interface(), got.Field(i).Interface(),
					"ServiceConfigArgs.%s is not copied by %s: the compose field is dropped on "+
						"the way to the provider, and the feature behind it is inert. Copy it "+
						"there, and set it in fullService so this test can see it.",
					got.Type().Field(i).Name, tt.converter)
			}
		})
	}
}

// TestServiceArgsCopyValues pins what each field is copied *as*. The guard
// above proves a field was touched; these prove it arrived intact, and record
// the arg shape chosen for the fields where the compose type and the SDK type
// disagree (bool vs *bool, string vs *string).
func TestServiceArgsCopyValues(t *testing.T) {
	tests := []struct {
		name string
		svc  compose.ServiceConfig
		got  func(awscompose.ServiceConfigArgs) any
		want any
	}{
		{
			// x-defang-policies: attached to the task role by aws/ecs.go.
			// Copied raw — every consumer calls compose.NormalizePolicies
			// itself, because inputs also arrive from hand-written programs.
			name: "policies",
			svc:  compose.ServiceConfig{Policies: compose.PolicyList{"arn:aws:iam::aws:policy/ReadOnlyAccess", "Custom"}},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.Policies },
			want: pulumi.ToStringArray([]string{"arn:aws:iam::aws:policy/ReadOnlyAccess", "Custom"}),
		},
		{
			// EFS access points and mounts. ReadOnly is a plain bool in
			// compose and a *bool in the SDK: written unconditionally,
			// because false is the parsed value, not an absent one.
			name: "volumes",
			svc: compose.ServiceConfig{Volumes: []compose.ServiceVolumeConfig{
				{Source: "data", Target: "/var/lib/data", ReadOnly: true},
				{Source: "cache", Target: "/tmp/cache"},
			}},
			got: func(a awscompose.ServiceConfigArgs) any { return a.Volumes },
			want: awscompose.ServiceVolumeConfigArray{
				awscompose.ServiceVolumeConfigArgs{
					Source: pulumi.String("data"), Target: pulumi.String("/var/lib/data"), ReadOnly: pulumi.Bool(true),
				},
				awscompose.ServiceVolumeConfigArgs{
					Source: pulumi.String("cache"), Target: pulumi.String("/tmp/cache"), ReadOnly: pulumi.Bool(false),
				},
			},
		},
		{
			name: "volumes_from",
			svc:  compose.ServiceConfig{VolumesFrom: []string{volumesFromRef}},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.VolumesFrom },
			want: pulumi.ToStringArray([]string{volumesFromRef}),
		},
		{
			// network_mode is copied verbatim; the provider parses it with
			// SidecarParent() and owns the errors for a bad parent.
			name: "network_mode folds a sidecar into its parent",
			svc:  compose.ServiceConfig{NetworkMode: "service:app"},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.NetworkMode },
			want: pulumi.String("service:app"),
		},
		{
			name: "restart",
			svc:  compose.ServiceConfig{Restart: "no"},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.Restart },
			want: pulumi.String("no"),
		},
		{
			// x-defang-autoscaling: bool in compose, *bool in the SDK.
			// Written unconditionally for the same reason as ReadOnly.
			name: "autoscaling",
			svc:  compose.ServiceConfig{Autoscaling: true},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.Autoscaling },
			want: pulumi.Bool(true),
		},
		{
			name: "working_dir",
			svc:  compose.ServiceConfig{WorkingDir: ptr("/srv")},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.WorkingDir },
			want: pulumi.StringPtr("/srv"),
		},
		{
			// Read via GetContainerName: names the ECS container, and the
			// container a depends_on condition and a volumes_from source
			// resolve to.
			name: "container_name",
			svc:  compose.ServiceConfig{ContainerName: ptr("app-main")},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.ContainerName },
			want: pulumi.StringPtr("app-main"),
		},
		{
			// Read via GetStopGracePeriodSeconds: the ECS stopTimeout.
			name: "stop_grace_period",
			svc:  compose.ServiceConfig{StopGracePeriod: ptr("90s")},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.StopGracePeriod },
			want: pulumi.StringPtr("90s"),
		},
		{
			// x-defang-aliases: read via AliasOptions from aws/elasticache.go
			// and aws/memorydb.go, so a migrated cluster is adopted rather
			// than replaced. Dropped, the next up destroys it.
			name: "aliases adopt a pre-migration cluster",
			svc:  compose.ServiceConfig{Aliases: map[string]string{compose.AliasCluster: clusterURN}},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.Aliases },
			want: pulumi.ToStringMap(map[string]string{compose.AliasCluster: clusterURN}),
		},
		{
			// The drop that started #519: the store deployed as a MinIO ECS
			// task because the provider never saw the extension.
			name: "x-defang-s3",
			svc:  compose.ServiceConfig{ObjectStore: &compose.ObjectStoreConfig{Bucket: "proj-uploads"}},
			got:  func(a awscompose.ServiceConfigArgs) any { return a.ObjectStore },
			want: awscompose.ObjectStoreConfigArgs{Bucket: pulumi.String("proj-uploads")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got(toAWSServiceArgs(tt.svc)))
		})
	}
}

// A service that asks for none of this must not grow inputs it never declared:
// an empty ObjectStore would make the provider dispatch a bucket for an
// ordinary container service.
func TestServiceArgsLeaveUnsetExtensionsNil(t *testing.T) {
	svc := compose.ServiceConfig{Image: pulumi.String("nginx")}

	aws := toAWSServiceArgs(svc)
	assert.Nil(t, aws.ObjectStore)
	assert.Nil(t, aws.Postgres)
	assert.Nil(t, aws.Redis)
	assert.Nil(t, aws.Llm)
	assert.Nil(t, aws.Volumes)
	// Autoscaling is written even when off: compose has no "unset" bool, and
	// an explicit false is what the provider's `if svc.Autoscaling` reads.
	assert.Equal(t, pulumi.Bool(false), aws.Autoscaling)

	gcp := toGCPServiceArgs(svc)
	assert.Nil(t, gcp.ObjectStore)
	assert.Nil(t, gcp.Volumes)

	azure := toAzureServiceArgs(svc)
	assert.Nil(t, azure.ObjectStore)
	assert.Nil(t, azure.Volumes)
}

// The three converters are the same function over three generated packages.
// Fixing one and not the others is exactly how #519 happened, so assert they
// agree field for field on the same input.
func TestServiceArgsAgreeAcrossClouds(t *testing.T) {
	full := fullService()
	aws := reflect.ValueOf(toAWSServiceArgs(full))
	tests := []struct {
		cloud string
		args  any
	}{
		{cloudGCP, toGCPServiceArgs(full)},
		{cloudAzure, toAzureServiceArgs(full)},
	}
	for _, tt := range tests {
		t.Run(tt.cloud, func(t *testing.T) {
			other := reflect.ValueOf(tt.args)
			require.Equal(t, aws.NumField(), other.NumField())
			for i := range aws.NumField() {
				name := aws.Type().Field(i).Name
				require.Equal(t, name, other.Type().Field(i).Name)
				// The values are same-shaped types in different packages, so
				// compare whether each side set the field at all.
				assert.Equal(t, aws.Field(i).IsZero(), other.Field(i).IsZero(),
					"ServiceConfigArgs.%s is copied on aws but not on %s (or the reverse)", name, tt.cloud)
			}
		})
	}
}

func TestApplyServiceAliasesDoesNotMutateProjectUpdateSource(t *testing.T) {
	project, err := parseCompose([]byte("services:\n  db:\n    image: postgres\n    x-defang-postgres: true\n"), "proj")
	require.NoError(t, err)
	require.Nil(t, project.Services["db"].Aliases)

	aliases := ServiceAliases{"db": {compose.AliasInstance: clusterURN}}
	require.NoError(t, applyServiceAliases(project, aliases))
	require.Equal(t, clusterURN, project.Services["db"].Aliases[compose.AliasInstance])
	// The migration map remains a separate input. In production the source is
	// ProjectUpdate.Compose, which is later uploaded verbatim as project.pb.
	require.Equal(t, clusterURN, aliases["db"][compose.AliasInstance])
}

func TestApplyServiceAliasesRejectsUnknownServiceAndConflict(t *testing.T) {
	project, err := parseCompose([]byte(`services:
  db:
    image: postgres
    x-defang-postgres: true
    x-defang-aliases:
      instance: urn:pulumi:old
`), "proj")
	require.NoError(t, err)

	err = applyServiceAliases(project, ServiceAliases{"missing": {compose.AliasInstance: clusterURN}})
	require.ErrorContains(t, err, "unknown service")
	err = applyServiceAliases(project, ServiceAliases{"db": {compose.AliasInstance: clusterURN}})
	require.ErrorContains(t, err, "conflicts with x-defang-aliases")
}
