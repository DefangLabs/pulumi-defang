package gcp

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Adoption of resources created by the legacy defang-mvp CD.
//
// The two CDs register the same cloud resources under different logical names,
// and the legacy ones hang off the stack rather than off a component. Pulumi
// therefore sees no continuity: an `up` plans a create plus a delete instead of
// an update. For a Cloud SQL instance that delete is real — it carries no
// retainOnDelete — so it takes the data with it. For the VPC it is quieter but
// just as fatal: the adopted database keeps the private network it was created
// on, so a fresh VPC leaves every service without a route to it.
//
// The legacy names are deterministic, so they are reconstructed here rather
// than asked of the operator. Everything below mirrors defang-mvp's
// pulumi/cd/gcp/gcpcd/safe_namings.go; keep the two in step.

// legacyPrefix is DEFANG_PREFIX's default in the legacy CD. Sanitization
// lowercases it.
const legacyPrefix = "Defang"

// legacyPulumiSuffixLength is the room the legacy CD left for Pulumi's own
// random suffix when trimming a name.
const legacyPulumiSuffixLength = 8

var legacyNonNameChars = regexp.MustCompile(`[^a-z0-9-]`)

// legacyAlias declares the name a resource had under the legacy CD. NoParent is
// required: legacy resources were registered at stack level, whereas this CD
// nests them under the Project component, and an alias without it would inherit
// the current parent and so name a URN that never existed.
func legacyAlias(name string) pulumi.ResourceOption {
	return pulumi.Aliases([]pulumi.Alias{{
		Name:     pulumi.String(name),
		NoParent: pulumi.Bool(true),
	}})
}

// legacyResourceName mirrors resourceName(): "<prefix>-<project>-<stack>-<parts...>".
func legacyResourceName(ctx *pulumi.Context, projectName string, parts ...string) string {
	return legacyTrimmedName(ctx, 63-legacyPulumiSuffixLength, projectName, parts...)
}

// legacyRedisInstanceName mirrors redisInstanceName(), which built the same
// name but trimmed it at 40 characters rather than 63, for Memorystore's own
// limit.
func legacyRedisInstanceName(ctx *pulumi.Context, projectName, serviceName string) string {
	return legacyTrimmedName(ctx, 40-legacyPulumiSuffixLength, projectName, serviceName, "redis")
}

func legacyTrimmedName(ctx *pulumi.Context, maxLength int, projectName string, parts ...string) string {
	all := append([]string{legacyPrefix, projectName, ctx.Stack()}, parts...)
	return legacyHashTrim(legacySanitize(strings.Join(all, "-")), maxLength)
}

// legacyPlainName mirrors the places the legacy CD joined names by hand,
// without the prefix, the stack, or any trimming — the Cloud SQL user and
// database, for instance.
func legacyPlainName(parts ...string) string {
	return strings.Join(parts, "-")
}

// legacyNetworkName mirrors networkName(), which skips the prefix and stack and
// trims at the full 63 characters.
func legacyNetworkName(name string) string {
	name = legacySanitize(name)
	if name == "" || name[0] == '-' {
		name = "net" + name
	}
	if name[0] < 'a' || name[0] > 'z' {
		name = "net-" + name
	}
	return legacyHashTrim(name, 63)
}

func legacySanitize(name string) string {
	name = strings.ToLower(name)
	name = legacyNonNameChars.ReplaceAllLiteralString(name, "-")
	return strings.TrimRight(name, "-")
}

func legacyHashTrim(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}
	const hashLength = 6
	return name[:maxLength-hashLength] + legacyHashN(name[maxLength-hashLength:], hashLength)
}

func legacyHashN(str string, length int) string {
	hash := sha256.New()
	hash.Write([]byte(str))
	hashBase36 := strconv.FormatUint(binary.LittleEndian.Uint64(hash.Sum(nil)[:8]), 36)
	if len(hashBase36) > length {
		return hashBase36[:length]
	}
	return fmt.Sprintf("%0*s", length, hashBase36)
}
