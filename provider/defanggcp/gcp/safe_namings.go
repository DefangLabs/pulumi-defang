package gcp

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func fullDefangResourceName(project string, config *GlobalConfig, names ...string) string {
	var parts []string
	if config.Prefix != "" {
		parts = append(parts, config.Prefix)
	}
	parts = append(parts, project, config.Stack)
	parts = append(parts, names...)
	return strings.Join(parts, "-")
}

const pulumiSuffixLength = 8

func cloudrunServiceName(project string, config *GlobalConfig, parts ...string) string {
	fullName := fullDefangResourceName(project, config, parts...)
	fullName = replaceNonAlphaNumericOrDash(fullName)
	// Less than 50 characters is 49 characters
	return hashTrim(fullName, 49-pulumiSuffixLength)
}

func resourceName(project string, config *GlobalConfig, parts ...string) string {
	fullName := fullDefangResourceName(project, config, parts...)
	fullName = replaceNonAlphaNumericOrDash(fullName)
	return hashTrim(fullName, 63-pulumiSuffixLength)
}

var nonLowerAlphaNumericOrDashRe = regexp.MustCompile(`[^a-z0-9-]`)

func replaceNonAlphaNumericOrDash(name string) string {
	name = strings.ToLower(name)
	name = nonLowerAlphaNumericOrDashRe.ReplaceAllLiteralString(name, "-")
	return strings.TrimRight(name, "-")
}

// - Lowercase letters, numbers, hyphens allowed
// - Name must start with a lowercase letter
// - Name must end with a lowercase letter or a number
// - Max length 63 characters
func networkName(name string) string {
	name = replaceNonAlphaNumericOrDash(name)
	if len(name) == 0 || name[0] == '-' {
		name = "net" + name
	}
	if name[0] < 'a' || name[0] > 'z' {
		name = "net-" + name
	}
	return hashTrim(name, 63)
}

func hashTrim(name string, maxLength int) string {
	if len(name) <= maxLength {
		return name
	}

	const hashLength = 6
	prefix := name[:maxLength-hashLength]
	suffix := name[maxLength-hashLength:]
	return prefix + hashn(suffix, hashLength)
}

func hashn(str string, length int) string {
	hash := sha256.New()
	hash.Write([]byte(str))
	hashInt := binary.LittleEndian.Uint64(hash.Sum(nil)[:8])
	hashBase36 := strconv.FormatUint(hashInt, 36) // base 36 string
	// truncate if the hash is too long
	if len(hashBase36) > length {
		return hashBase36[:length]
	}
	// if the hash is too short, pad with leading zeros
	return fmt.Sprintf("%0*s", length, hashBase36)
}
