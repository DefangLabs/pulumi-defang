package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/defangaws"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/shared"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// create an s3 bucket
		bucket, err := s3.NewBucket(ctx, "build-context", &s3.BucketArgs{})
		if err != nil {
			return err
		}

		// s3 bucket object for build context
		buildContext, err := s3.NewBucketObjectv2(ctx, "build-context-object", &s3.BucketObjectv2Args{
			Bucket: bucket.ID(),
			Key:    pulumi.String("uploads/sample"),
			Source: pulumi.NewFileArchive("."),
		})
		if err != nil {
			return err
		}

		proj, err := defangaws.NewProject(ctx, "aws-go", &defangaws.ProjectArgs{
			Services: shared.ServiceInputMap{
				"app": shared.ServiceInputArgs{
					Build: shared.BuildInputArgs{
						Context:    pulumi.Sprintf("s3://%s/%s", buildContext.Bucket, buildContext.Key),
						Dockerfile: pulumi.StringPtr("Dockerfile"),
					},
					Command: pulumi.StringArray{pulumi.String("./run.sh")},
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target:      pulumi.Int(8000),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
					Environment: pulumi.StringMap{
						"DATABASE_URL": pulumi.String("postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/postgres?sslmode=require"),
						"REDIS_URL":    pulumi.String("redis://redis:6379/0"),
					},
					HealthCheck: shared.HealthCheckConfigArgs{
						Test:               pulumi.StringArray{pulumi.String("CMD"), pulumi.String("curl"), pulumi.String("-f"), pulumi.String("http://localhost:8000/")},
						IntervalSeconds:    pulumi.IntPtr(5),
						TimeoutSeconds:     pulumi.IntPtr(2),
						Retries:            pulumi.IntPtr(10),
						StartPeriodSeconds: pulumi.IntPtr(10),
					},
					Deploy: shared.DeployConfigArgs{
						Resources: shared.ResourcesConfigArgs{
							Reservations: shared.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"postgres": shared.ServiceInputArgs{
					Image:    pulumi.StringPtr("pgvector/pgvector:pg16"),
					Postgres: shared.PostgresInputArgs{},
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target: pulumi.Int(5432),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Environment: pulumi.StringMap{
						"POSTGRES_PASSWORD": pulumi.String("${POSTGRES_PASSWORD}"), // set via config/secret
					},
					Deploy: shared.DeployConfigArgs{
						Resources: shared.ResourcesConfigArgs{
							Reservations: shared.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"redis": shared.ServiceInputArgs{
					Image: pulumi.StringPtr("redis:6.2"),
					Redis: shared.RedisInputArgs{},
					Ports: shared.PortConfigArray{
						shared.PortConfigArgs{
							Target: pulumi.Int(6379),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Deploy: shared.DeployConfigArgs{
						Resources: shared.ResourcesConfigArgs{
							Reservations: shared.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"chat": shared.ServiceInputArgs{
					Provider: shared.ProviderInputArgs{
						Type: pulumi.String("openai"),
						Options: shared.ProviderOptionsArgs{
							Model: pulumi.String("chat-default"),
						},
					},
					Environment: pulumi.StringMap{
						"OPENAI_API_KEY": pulumi.String("defang"),
					},
					Deploy: shared.DeployConfigArgs{
						Resources: shared.ResourcesConfigArgs{
							Reservations: shared.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)

		return nil
	})
}
