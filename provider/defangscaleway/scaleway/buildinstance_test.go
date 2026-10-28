package scaleway

import (
	"testing"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectBuildInstanceType(t *testing.T) {
	tests := []struct {
		family string
		memMiB int
		want   string
	}{
		{"dev", compose.DefaultBuildMemoryMiB, "DEV1-L"},
		{"DEV", compose.DefaultBuildMemoryMiB, "DEV1-L"},
		{"dev", 512, "DEV1-S"},
		{"dev", 4096, "DEV1-M"},
		{"dev", 12289, "GP1-XS"},
		{"dev", 81920, "GP1-L"},
		{"dev", 1 << 20, "GP1-XL"},
		{"basic3", 12288, "BASIC3-X6C-12G"},
		{"basic3", 1 << 20, "BASIC3-X16C-64G"},
		{"standard3", 81920, "STANDARD3-X24C-96G"},
		{"pop2", 1 << 20, "POP2-64C-256G"},
		{"standard3", compose.DefaultBuildMemoryMiB, "STANDARD3-X2C-8G"},
		{"basic3", 4096, "BASIC3-X2C-4G"},
		{"pop2", 16384, "POP2-4C-16G"},
	}
	for _, tt := range tests {
		got, err := SelectBuildInstanceType(tt.family, tt.memMiB)
		require.NoError(t, err)
		assert.Equal(t, tt.want, got.Name, "family=%s memMiB=%d", tt.family, tt.memMiB)
	}
}

func TestSelectBuildInstanceTypeUnknownFamily(t *testing.T) {
	_, err := SelectBuildInstanceType("gpu", compose.DefaultBuildMemoryMiB)
	require.ErrorIs(t, err, errUnknownBuildInstanceFamily)
	assert.Contains(t, err.Error(), "basic3, dev, pop2, standard3")
}

func TestLookupBuildInstanceType(t *testing.T) {
	assert.Equal(t, 80, LookupBuildInstanceType("dev1-l").MaxLocalDiskGB)
	assert.Equal(t, BuildInstanceType{Name: "PRO2-XS"}, LookupBuildInstanceType("PRO2-XS"))
}

func TestBuildDiskSizeGB(t *testing.T) {
	devS, _ := SelectBuildInstanceType("dev", 2048)
	devL, _ := SelectBuildInstanceType("dev", 8192)
	gpM, _ := SelectBuildInstanceType("dev", 65536)
	gpXL, _ := SelectBuildInstanceType("dev", 262144)
	std, _ := SelectBuildInstanceType("standard3", 65536)
	assert.Equal(t, 16, BuildDiskSizeGB(devS, 2048))
	assert.Equal(t, 16, BuildDiskSizeGB(devL, 8192))
	assert.Equal(t, 128, BuildDiskSizeGB(gpM, 65536))
	assert.Equal(t, 128, BuildDiskSizeGB(std, 65536))
	assert.Equal(t, 20, BuildDiskSizeGB(devS, 16384), "capped at DEV1-S's 20 GB local storage")
	assert.Equal(t, 512, BuildDiskSizeGB(gpXL, 262144))
}
