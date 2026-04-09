package common

import (
	"errors"
	"regexp"
)

var ErrNoImageOrBuildConfig = errors.New("no image or build config")

// imgRE parses a Docker image reference into registry, repo, tag, and digest parts.
// Mirrors the regex used by DefangLabs/defang/src/pkg/dockerhub.
//
//nolint:lll
var imgRE = regexp.MustCompile(`^((?:((?:[0-9a-z](?:[0-9a-z-]{0,61}[0-9a-z])?\.)+[a-z]{2,63})\/)?(.{1,127}?))(?::(\w[\w.-]{0,127}))?(?:@(sha256:[0-9a-f]{64}))?$`)

type ImageInfo struct {
	Registry string
	Repo     string
	Tag      string
	Digest   string
}

func ParseImage(image string) ImageInfo {
	m := imgRE.FindStringSubmatch(image)
	if m == nil {
		return ImageInfo{Repo: image}
	}
	return ImageInfo{Registry: m[2], Repo: m[3], Tag: m[4], Digest: m[5]}
}

func (i ImageInfo) FullImage() string {
	s := i.Repo
	if i.Registry != "" {
		s = i.Registry + "/" + s
	}
	if i.Tag != "" {
		s += ":" + i.Tag
	}
	if i.Digest != "" {
		s += "@" + i.Digest
	}
	return s
}
