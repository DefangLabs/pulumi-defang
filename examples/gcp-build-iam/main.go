// Reproduction case for https://github.com/DefangLabs/defang-mvp/issues/2773
//
// The bug: when `pulumi destroy` is run, if the build service account is deleted
// before its IAM bindings are fully removed, GCP returns an error:
//   "Error retrieving IAM policy for artifactregistry repository"
//
// Root cause: `pulumi.DeletedWith(bsa)` is used on the RepositoryIamBinding,
// BucketIAMMember, and ProjectIAMMember resources. When the service account is
// destroyed first, the IAM policy still contains "deleted:serviceAccount:...@..."
// entries, which causes GCP to reject subsequent IAM policy reads/updates.
//
// To reproduce:
//   1. pulumi up        -- deploys the project and creates build infra
//   2. pulumi destroy   -- observe "Error retrieving IAM policy for artifactregistry repository"
package main

import (
	defanggcp "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-gcp/compose"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")
		domain := cfg.Get("domain")

		project, err := defanggcp.NewProject(ctx, "gcp-build-iam", &defanggcp.ProjectArgs{
			Domain: &domain,
			Services: compose.ServiceConfigMap{
				// This service has a build config, which triggers createBuildInfra:
				// a build service account + Artifact Registry repo + IAM bindings
				// using pulumi.DeletedWith(bsa). On destroy, GCP may fail to read
				// the Artifact Registry IAM policy after the service account is gone.
				"app": compose.ServiceConfigArgs{
					Build: &compose.BuildConfigArgs{
						Context: pulumi.String("./app"),
					},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("endpoints", project.Endpoints)
		return nil
	})
}
