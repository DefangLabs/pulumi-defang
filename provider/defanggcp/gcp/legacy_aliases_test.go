package gcp

import (
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The names below were read out of a real stack that the legacy defang-mvp CD
// deployed (project "martekio-mig", stack "cyc"). If the reconstruction drifts
// from defang-mvp's pulumi/cd/gcp/gcpcd/safe_namings.go, migrations stop
// adopting and start replacing — so pin the exact strings.
func TestLegacyNamesMatchTheMvpCD(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		const project = "martekio-mig"

		assert.Equal(t, "martekio-mig-vpc", legacyNetworkName(project+"-vpc"))
		assert.Equal(t, "defang-martekio-mig-cyc-shared-subnet",
			legacyResourceName(ctx, project, "shared-subnet"))
		assert.Equal(t, "defang-martekio-mig-cyc-private-dns",
			legacyResourceName(ctx, project, "private-dns"))
		assert.Equal(t, "defang-martekio-mig-cyc-vpc-peering-ip",
			legacyResourceName(ctx, project, "vpc-peering-ip"))
		assert.Equal(t, "defang-martekio-mig-cyc-service-connection",
			legacyResourceName(ctx, project, "service-connection"))
		assert.Equal(t, "defang-martekio-mig-cyc", legacyResourceName(ctx, project))
		assert.Equal(t, "defang-martekio-mig-cyc-database-postgres",
			legacyResourceName(ctx, project, "database", "postgres"))
		assert.Equal(t, "martekio-mig-database-postgres-user",
			legacyPlainName(project, "database", "postgres", "user"))
		assert.Equal(t, "martekio-mig-database-postgres-db",
			legacyPlainName(project, "database", "postgres", "db"))
		return nil
	}, pulumi.WithMocks("martekio-mig", "cyc", testMocks{}))
	require.NoError(t, err)
}

// resourceName left 8 characters for Pulumi's suffix, so it trimmed at 55 with a
// 6-character base36 hash. A project name long enough to trip that must trim the
// same way or the alias names a URN that never existed.
func TestLegacyResourceNameTrimsLikeTheMvpCD(t *testing.T) {
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		got := legacyResourceName(ctx, "a-very-long-project-name-that-will-certainly-overflow", "shared-subnet")

		assert.Len(t, got, 55)
		// 49 characters of name, then a 6-character base36 hash of the remainder.
		assert.Equal(t, "defang-a-very-long-project-name-that-will-certain", got[:49])
		assert.Regexp(t, `^[a-z0-9]{6}$`, got[49:])
		return nil
	}, pulumi.WithMocks("proj", "somestack", testMocks{}))
	require.NoError(t, err)
}

// Uppercase and other stray characters are folded, matching replaceNonAlphaNumericOrDash.
func TestLegacyNameSanitization(t *testing.T) {
	assert.Equal(t, "my-proj-vpc", legacyNetworkName("My_Proj-vpc"))
	assert.Equal(t, "lead", legacySanitize("lead---"))
}

// A network name may not start with a non-letter; the legacy CD prefixed those.
func TestLegacyNetworkNamePrefixesNonLetters(t *testing.T) {
	assert.Equal(t, "net-9lives-vpc", legacyNetworkName("9lives-vpc"))
}
