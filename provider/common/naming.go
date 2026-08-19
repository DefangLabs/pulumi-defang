package common

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const logicalNameHashLength = 8

// BoundedName joins a prefix and stable suffix while keeping the result within
// maxLength. When shortening is necessary, it keeps the suffix visible when it
// fits and adds a hash of the complete unshortened name so two long prefixes
// with the same leading characters cannot collapse to one Pulumi logical name.
func BoundedName(prefix, suffix string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}
	name := prefix + suffix
	if len(name) <= maxLength {
		return name
	}

	digest := sha256.Sum256([]byte(name))
	hash := hex.EncodeToString(digest[:])[:logicalNameHashLength]
	separatorAndHash := "-" + hash
	prefixLength := maxLength - len(suffix) - len(separatorAndHash)
	if prefixLength < 0 {
		// This fallback is only for callers whose fixed suffix alone exceeds the
		// requested bound. Preserve a collision-resistant identity even though
		// the human-readable role cannot fit in full.
		return hash[:min(maxLength, len(hash))]
	}

	trimmedPrefix := strings.TrimRight(prefix[:prefixLength], "-")
	if trimmedPrefix == "" {
		return hash + suffix
	}
	return trimmedPrefix + separatorAndHash + suffix
}
