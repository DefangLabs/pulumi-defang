package aws

import (
	"fmt"

	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type ecrResult struct {
	repository *ecr.Repository
	repoURL    pulumix.Output[string]
}

// createECRRepo creates an ECR repository for built images.
func createECRRepo(
	ctx *pulumi.Context,
	name string,
	opts ...pulumi.ResourceOption,
) (*ecrResult, error) {
	repo, err := ecr.NewRepository(ctx, name, &ecr.RepositoryArgs{
		ForceDelete:        pulumi.Bool(true),
		ImageTagMutability: pulumi.String("MUTABLE"),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating ECR repository: %w", err)
	}

	return &ecrResult{
		repository: repo,
		repoURL:    pulumix.Output[string](repo.RepositoryUrl),
	}, nil
}
