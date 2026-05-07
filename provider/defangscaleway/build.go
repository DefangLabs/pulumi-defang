package defangscaleway

import (
	"context"
	"fmt"

	"github.com/pulumi/pulumi-go-provider/infer"
)

// Build is a schema placeholder for Scaleway image builds.
type Build struct{}

type BuildInputs struct {
	Image string `pulumi:"image"`
}

type BuildState struct {
	BuildInputs
	ImageURI string `pulumi:"imageUri"`
}

func (*Build) Create(
	ctx context.Context, req infer.CreateRequest[BuildInputs],
) (infer.CreateResponse[BuildState], error) {
	input := req.Inputs
	if input.Image == "" {
		return infer.CreateResponse[BuildState]{}, fmt.Errorf("image is required")
	}
	state := BuildState{BuildInputs: input, ImageURI: input.Image}
	return infer.CreateResponse[BuildState]{ID: req.Name, Output: state}, nil
}
