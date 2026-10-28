package scaleway

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

var errUnknownBuildInstanceFamily = errors.New("unknown build instance family")

// BuildInstanceType is a Compute Instance commercial type a build can run on.
type BuildInstanceType struct {
	Name   string
	MemMiB int
	// MaxLocalDiskGB is the type's local (l_ssd) storage limit, or 0 for a
	// block-only type. A local boot disk is deleted along with the instance,
	// so it is used whenever the type has one.
	MaxLocalDiskGB int
}

// buildInstanceFamilies are the families the build-instance-family recipe
// setting chooses between, each ordered by memory, listing only the cheapest
// type at each memory size. A static table rather than a lookup against
// Scaleway's product API, so the same compose.yaml and recipe always pick the
// same type.
var buildInstanceFamilies = map[string][]BuildInstanceType{
	// Scaleway's Development range: shared vCPU, local boot disk.
	"dev": {
		{"DEV1-S", 2048, 20},
		{"DEV1-M", 4096, 40},
		{"DEV1-L", 8192, 80},
		{"DEV1-XL", 12288, 120},
		{"GP1-XS", 16384, 150},
		{"GP1-S", 32768, 300},
		{"GP1-M", 65536, 600},
		{"GP1-L", 131072, 600},
		{"GP1-XL", 262144, 600},
	},
	"basic3": {
		{"BASIC3-X2C-4G", 4096, 0},
		{"BASIC3-X2C-8G", 8192, 0},
		{"BASIC3-X6C-12G", 12288, 0},
		{"BASIC3-X4C-16G", 16384, 0},
		{"BASIC3-X6C-24G", 24576, 0},
		{"BASIC3-X8C-32G", 32768, 0},
		{"BASIC3-X12C-48G", 49152, 0},
		{"BASIC3-X16C-64G", 65536, 0},
	},
	"standard3": {
		{"STANDARD3-X2C-8G", 8192, 0},
		{"STANDARD3-X4C-16G", 16384, 0},
		{"STANDARD3-X6C-24G", 24576, 0},
		{"STANDARD3-X8C-32G", 32768, 0},
		{"STANDARD3-X12C-48G", 49152, 0},
		{"STANDARD3-X16C-64G", 65536, 0},
		{"STANDARD3-X24C-96G", 98304, 0},
		{"STANDARD3-X32C-128G", 131072, 0},
		{"STANDARD3-X48C-192G", 196608, 0},
		{"STANDARD3-X64C-256G", 262144, 0},
	},
	"pop2": {
		{"POP2-2C-8G", 8192, 0},
		{"POP2-4C-16G", 16384, 0},
		{"POP2-6C-24G", 24576, 0},
		{"POP2-8C-32G", 32768, 0},
		{"POP2-12C-48G", 49152, 0},
		{"POP2-16C-64G", 65536, 0},
		{"POP2-24C-96G", 98304, 0},
		{"POP2-32C-128G", 131072, 0},
		{"POP2-48C-192G", 196608, 0},
		{"POP2-64C-256G", 262144, 0},
	},
}

// SelectBuildInstanceType returns the smallest type in family with at least
// memMiB of memory. A request beyond the family's largest type gets that type,
// as AWS and GCP cap at their largest build machine; pick a family that goes
// higher (standard3 and pop2 reach 256 GiB) for bigger builds.
func SelectBuildInstanceType(family string, memMiB int) (BuildInstanceType, error) {
	types, ok := buildInstanceFamilies[strings.ToLower(family)]
	if !ok {
		return BuildInstanceType{}, fmt.Errorf("%w %q: use one of %s",
			errUnknownBuildInstanceFamily, family, strings.Join(slices.Sorted(maps.Keys(buildInstanceFamilies)), ", "))
	}
	for _, t := range types {
		if t.MemMiB >= memMiB {
			return t, nil
		}
	}
	return types[len(types)-1], nil
}

// LookupBuildInstanceType returns the catalog entry for a commercial type, or
// a block-only entry for a type the catalog doesn't list.
func LookupBuildInstanceType(name string) BuildInstanceType {
	for _, types := range buildInstanceFamilies {
		for _, t := range types {
			if strings.EqualFold(t.Name, name) {
				return t
			}
		}
	}
	return BuildInstanceType{Name: name}
}

// BuildDiskSizeGB sizes the build instance's boot disk the way GCP sizes Cloud
// Build's: twice the build memory, at least 16 GB, within the type's local
// storage limit when it has one.
func BuildDiskSizeGB(t BuildInstanceType, memMiB int) int {
	gb := max(memMiB*2/1024, 16)
	if t.MaxLocalDiskGB > 0 {
		gb = min(gb, t.MaxLocalDiskGB)
	}
	return gb
}
