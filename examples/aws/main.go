package main

import (
	defangaws "github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/defangaws"
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang-aws/shared"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		proj, err := defangaws.NewProject(ctx, "myProject", &defangaws.ProjectArgs{
			Services: shared.ServiceInputMap{
				"app": shared.ServiceInputArgs{
					Build: shared.BuildInputArgs{
						Context:    pulumi.String("."),
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
						"DJANGO_SETTINGS_MODULE": pulumi.String("config.settings"),
						// TODO: interpolate POSTGRES_PASSWORD from config/secret
						"DATABASE_URL":          pulumi.String("postgres://postgres:@postgres:5432/postgres?sslmode=require"),
						"REDIS_URL":             pulumi.String("redis://redis:6379/0"),
						"CELERY_BROKER_URL":     pulumi.String("redis://redis:6379/0"),
						"CELERY_RESULT_BACKEND": pulumi.String("redis://redis:6379/0"),
						"DJANGO_SECRET_KEY":     pulumi.String(""), // set via config/secret
						"SSL_MODE":              pulumi.String(""), // set via config/secret
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
				"worker": shared.ServiceInputArgs{
					Build: shared.BuildInputArgs{
						Context:    pulumi.String("."),
						Dockerfile: pulumi.StringPtr("Dockerfile"),
					},
					Command: pulumi.StringArray{
						pulumi.String("celery"),
						pulumi.String("-A"),
						pulumi.String("config"),
						pulumi.String("worker"),
						pulumi.String("-l"),
						pulumi.String("info"),
					},
					Environment: pulumi.StringMap{
						"DJANGO_SETTINGS_MODULE": pulumi.String("config.settings"),
						// TODO: interpolate POSTGRES_PASSWORD from config/secret
						"DATABASE_URL":      pulumi.String("postgres://postgres:@postgres:5432/postgres?sslmode=require"),
						"REDIS_URL":         pulumi.String("redis://redis:6379/0"),
						"OPENAI_API_KEY":    pulumi.String("defang"),
						"DJANGO_SECRET_KEY": pulumi.String(""), // set via config/secret
						"SSL_MODE":          pulumi.String(""), // set via config/secret
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
						"POSTGRES_PASSWORD": pulumi.String(""), // set via config/secret
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
				// x-defang-redis: Redis managed service not yet supported in Go SDK;
				// deployed as a regular container image.
				"redis": shared.ServiceInputArgs{
					Image: pulumi.StringPtr("redis:6.2"),
					// Redis: shared.RedisInputArgs{},
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
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)

		return nil
	})
}
