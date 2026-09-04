package aws

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Adoption of the managed databases the legacy defang-mvp CD created.
//
// The two CDs register the same cloud resources under different logical names,
// and the legacy ones hang off the stack rather than off a component. Pulumi
// therefore sees no continuity: an `up` plans a create plus a delete instead of
// an update, and for an RDS instance or an ElastiCache replication group that
// delete takes the data with it.
//
// The legacy names are deterministic, so they are reconstructed here rather
// than asked of the operator. Everything below mirrors defang-mvp's
// pulumi/shared/config.ts (stack()) and pulumi/shared/aws/safe_namings.ts
// (awsIdentifierSafe(), truncate()); keep the two in step.
//
// Two things differ from the GCP side (see defanggcp/gcp/legacy_aliases.go):
//
//   - The legacy program was the Pulumi project "cd" -- the name in
//     pulumi/cd/Pulumi.yaml -- while this CD's project is the compose project
//     name. Every legacy URN therefore carries "cd" as its project, and the
//     alias has to say so; an alias without it names a URN that never existed.
//     The legacy GCP CD passed the compose project to the automation API, so
//     its aliases need no such override.
//   - The legacy names embed that same "cd" rather than the compose project,
//     because shared/config.ts built its prefix from pulumi.getProject().

// legacyProject is the Pulumi project of the legacy CD program, both in the
// URNs it wrote and in the names it generated.
const legacyProject = "cd"

// legacyRdsInstanceNameMaxLength is MAX_DB_INSTANCE_IDENTITIER_LENGTH minus the
// 8 characters the legacy CD reserved for Pulumi's own random suffix.
const legacyRdsInstanceNameMaxLength = 63 - 8

// legacyIdentifierInvalidChars mirrors the character class awsIdentifierSafe
// folds. The + matters: a run of invalid characters collapses to one hyphen.
var legacyIdentifierInvalidChars = regexp.MustCompile(`[^a-z0-9-]+`)

// legacyAlias declares the name a resource had under the legacy CD. NoParent is
// required: legacy resources were registered at stack level, whereas this CD
// nests them under a component, and an alias without it would inherit the
// current parent and so name a URN that never existed.
func legacyAlias(name string) pulumi.ResourceOption {
	return pulumi.Aliases([]pulumi.Alias{{
		Name:     pulumi.String(name),
		Project:  pulumi.String(legacyProject),
		NoParent: pulumi.Bool(true),
	}})
}

// legacyStackName mirrors stack(): "<prefix>-cd-<stack>-<parts...>", with the
// prefix omitted on the playground stacks that never set one.
func legacyStackName(ctx *pulumi.Context, parts ...string) string {
	all := make([]string, 0, len(parts)+3)
	if prefix := common.Prefix.Get(ctx); prefix != "" {
		all = append(all, prefix)
	}
	all = append(all, legacyProject, ctx.Stack())
	return strings.Join(append(all, parts...), "-")
}

// legacyRdsInstanceName mirrors the one legacy name that was not used verbatim:
// awsIdentifierSafe(stack(service), MAX_DB_INSTANCE_IDENTITIER_LENGTH-8).
func legacyRdsInstanceName(ctx *pulumi.Context, serviceName string) string {
	return legacyTruncate(legacySanitize(legacyStackName(ctx, serviceName)), legacyRdsInstanceNameMaxLength)
}

// legacySanitize mirrors awsIdentifierSafe()'s folding: lowercase, then every
// run of characters outside [a-z0-9-] replaced by a single hyphen.
func legacySanitize(name string) string {
	return legacyIdentifierInvalidChars.ReplaceAllLiteralString(strings.ToLower(name), "-")
}

// legacyTruncate mirrors truncate(): the head of the name plus a hex hash of
// the whole of it, with no separator between them.
func legacyTruncate(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}
	const hashLength = 6
	digest := sha256.Sum256([]byte(name))
	return name[:maxLength-hashLength] + hex.EncodeToString(digest[:])[:hashLength]
}
