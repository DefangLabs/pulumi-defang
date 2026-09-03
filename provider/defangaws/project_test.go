package defangaws

import (
	"maps"
	"slices"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
)

func TestApexServiceName(t *testing.T) {
	ingress := compose.ServicePortConfig{Target: 80, Mode: compose.PortModeIngress}
	defaultMode := compose.ServicePortConfig{Target: 80} // empty mode defaults to ingress
	host := compose.ServicePortConfig{Target: 5432, Mode: compose.PortModeHost}

	tests := []struct {
		name     string
		services compose.Services
		want     string
	}{
		{
			name:     "no services",
			services: compose.Services{},
			want:     "",
		},
		{
			name: "single service single ingress port",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress}},
			},
			want: "web",
		},
		{
			name: "empty mode counts as ingress",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{defaultMode}},
			},
			want: "web",
		},
		{
			name: "one ingress service alongside managed datastore and worker",
			services: compose.Services{
				"web":    {Ports: []compose.ServicePortConfig{ingress}},
				"db":     {Postgres: &compose.PostgresConfig{}, Ports: []compose.ServicePortConfig{defaultMode}},
				"cache":  {Redis: &compose.RedisConfig{}, Ports: []compose.ServicePortConfig{defaultMode}},
				"worker": {}, // no ports
			},
			want: "web",
		},
		{
			name: "host-only port does not qualify",
			services: compose.Services{
				"internal": {Ports: []compose.ServicePortConfig{host}},
			},
			want: "",
		},
		{
			name: "two ingress services -> unbound",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress}},
				"api": {Ports: []compose.ServicePortConfig{ingress}},
			},
			want: "",
		},
		{
			name: "single service with two ingress ports -> unbound",
			services: compose.Services{
				"web": {Ports: []compose.ServicePortConfig{ingress, {Target: 443, Mode: compose.PortModeIngress}}},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := apexServiceName(tt.services); got != tt.want {
				t.Errorf("apexServiceName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestObjectStoreGrants(t *testing.T) {
	const uploads, backups = "uploads", "backups"
	stores := map[string]pulumi.StringInput{
		uploads: pulumi.String("arn:aws:s3:::proj-uploads"),
		backups: pulumi.String("arn:aws:s3:::proj-backups"),
	}
	dependsOn := func(names ...string) compose.DependsOnConfig {
		cfg := compose.DependsOnConfig{}
		for _, name := range names {
			cfg[name] = compose.ServiceDependency{Required: true}
		}
		return cfg
	}

	tests := []struct {
		name   string
		svc    compose.ServiceConfig
		stores map[string]pulumi.StringInput
		want   []string
	}{
		{
			name: "no stores in the project",
			svc:  compose.ServiceConfig{DependsOn: dependsOn(uploads)},
			want: nil,
		},
		{
			name:   "no depends_on",
			svc:    compose.ServiceConfig{},
			stores: stores,
			want:   nil,
		},
		{
			name:   "depends_on a store",
			svc:    compose.ServiceConfig{DependsOn: dependsOn(uploads)},
			stores: stores,
			want:   []string{uploads},
		},
		{
			name:   "depends_on a plain service is not a grant",
			svc:    compose.ServiceConfig{DependsOn: dependsOn("db")},
			stores: stores,
			want:   nil,
		},
		{
			name:   "depends_on two stores",
			svc:    compose.ServiceConfig{DependsOn: dependsOn(uploads, backups, "db")},
			stores: stores,
			want:   []string{backups, uploads},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := objectStoreGrants(tt.svc, tt.stores)
			assert.ElementsMatch(t, tt.want, slices.Sorted(maps.Keys(got)))
		})
	}
}

// A sidecar shares its parent's task definition and task role, so its
// depends_on is the parent's: it decides when the parent is created and which
// buckets the parent's task role reaches. Edges back into the sidecar group
// are container-level, not resources to order against.
func TestFoldSidecarDeps(t *testing.T) {
	const app, store, db = "app", "uploads", "db"
	dep := compose.ServiceDependency{Required: true}

	standalone := compose.Services{
		app:   {DependsOn: compose.DependsOnConfig{db: dep}},
		store: {ObjectStore: &compose.ObjectStoreConfig{Bucket: "proj-uploads"}},
		db:    {Postgres: &compose.PostgresConfig{}},
	}
	sidecars := map[string]map[string]compose.ServiceConfig{
		app: {
			"log":  {DependsOn: compose.DependsOnConfig{store: dep, app: dep}},
			"prox": {DependsOn: compose.DependsOnConfig{"log": dep}},
		},
	}

	graph := foldSidecarDeps(standalone, sidecars)
	assert.ElementsMatch(t, []string{db, store}, slices.Sorted(maps.Keys(graph[app].DependsOn)))
	// The input map is left alone: the loop still dispatches the original
	// service config.
	assert.ElementsMatch(t, []string{db}, slices.Sorted(maps.Keys(standalone[app].DependsOn)))
	// No sidecars, no copy.
	assert.Nil(t, foldSidecarDeps(standalone, nil)[store].DependsOn)
}
