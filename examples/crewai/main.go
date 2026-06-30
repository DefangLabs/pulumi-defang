package main

import (
	"fmt"

	defangaws "github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws"
	"github.com/DefangLabs/pulumi-defang/sdk/v2/go/defang-aws/compose"
	"github.com/aws/smithy-go/ptr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/resourcegroups"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		awsConfig := config.New(ctx, "aws")
		awsProvider, err := aws.NewProvider(ctx, "aws", &aws.ProviderArgs{
			Region:  pulumi.String(awsConfig.Require("region")),
			Profile: pulumi.String(awsConfig.Require("profile")),
			DefaultTags: &aws.ProviderDefaultTagsArgs{
				Tags: pulumi.StringMap{
					"defang:project": pulumi.String(ctx.Project()),
					"defang:stack":   pulumi.String(ctx.Stack()),
				},
			},
		})
		if err != nil {
			return err
		}

		providerOpt := pulumi.Provider(awsProvider)

		// Create resource group for the tags
		_, err = resourcegroups.NewGroup(ctx, "group", &resourcegroups.GroupArgs{
			// Name: pulumi.String("crewai-resources"),
			ResourceQuery: &resourcegroups.GroupResourceQueryArgs{
				Query: pulumi.JSONMarshal(map[string]any{
					"ResourceTypeFilters": []string{"AWS::AllSupported"},
					"TagFilters": []map[string]any{
						{
							"Key":    "defang:project",
							"Values": []string{ctx.Project()},
						},
						{
							"Key":    "defang:stack",
							"Values": []string{ctx.Stack()},
						},
					},
				}),
			},
		}, providerOpt)
		if err != nil {
			return err
		}

		// Builds happens in the cloud, so we cannot use folder references.
		var contextUrl pulumi.StringInput = pulumi.String("https://download-directory.github.io/?url=https%3A%2F%2Fgithub.com%2FDefangLabs%2Fsamples%2Ftree%2Fmain%2Fsamples%2Fcrewai%2Fapp")
		if appFolder := config.New(ctx, "").Get("appFolder"); appFolder != "" {
			// create an s3 bucket
			buildBucket, err := s3.NewBucket(ctx, "build-context", &s3.BucketArgs{
				ForceDestroy: pulumi.Bool(true),
			}, providerOpt)
			if err != nil {
				return err
			}

			// s3 bucket object for build context
			buildContext, err := s3.NewBucketObject(ctx, "build-context-object", &s3.BucketObjectArgs{
				Bucket: buildBucket.ID(),
				Key:    pulumi.String("uploads/sample.zip"),
				Source: pulumi.NewFileArchive("/Users/llunesu/dev/samples/samples/crewai/app"),
			}, providerOpt)
			if err != nil {
				return err
			}
			contextUrl = pulumi.Sprintf("s3://%s/%s", buildBucket.Bucket, buildContext.Key)
		}

		// Look up (or create) the public DNS zone — zone lifecycle is managed here, not inside the defang-aws provider
		var publicZoneID pulumi.StringPtrInput
		var projectDomain, appDomain pulumi.StringPtrInput
		if pd := config.New(ctx, "").Get("projectDomain"); pd != "" {
			publicZone, err := route53.LookupZone(ctx, &route53.LookupZoneArgs{
				Name:        &pd,
				PrivateZone: ptr.Bool(false),
			}, pulumi.Provider(awsProvider))
			if err != nil {
				return err
			}
			publicZoneID = pulumi.StringPtr(publicZone.Id)
			projectDomain = pulumi.StringPtr(pd)
			appDomain = pulumi.StringPtr(fmt.Sprintf("%s-%s.%s", ctx.Project(), ctx.Stack(), pd))
		}

		const sslMode = "require"                                                  // "${SSL_MODE}"
		postgresPassword := config.New(ctx, "").RequireSecret("postgres_password") // "${POSTGRES_PASSWORD}"
		secretKey := config.New(ctx, "").RequireSecret("secret_key")               // "${SECRET_KEY}"

		proj, err := defangaws.NewProject(ctx, "crewai", &defangaws.ProjectArgs{
			Services: compose.ServiceConfigMap{
				"app": compose.ServiceConfigArgs{
					DomainName: appDomain,
					Build: compose.BuildConfigArgs{
						Context:    contextUrl,
						Dockerfile: pulumi.StringPtr("Dockerfile"),
					},
					DependsOn: compose.ServiceDependencyMap{
						"postgres": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
						"redis": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
					},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target:      pulumi.Int(8000),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
					Environment: pulumi.StringMap{
						"CELERY_BROKER_URL":      pulumi.String("redis://redis:6379/0"),
						"CELERY_RESULT_BACKEND":  pulumi.String("redis://redis:6379/0"),
						"DATABASE_URL":           pulumi.Sprintf("postgres://postgres:%s@postgres:5432/postgres?sslmode=%s", postgresPassword, sslMode),
						"DJANGO_SECRET_KEY":      secretKey,
						"DJANGO_SETTINGS_MODULE": pulumi.String("config.settings"),
						"REDIS_URL":              pulumi.String("redis://redis:6379/0"),
					},
					HealthCheck: compose.HealthCheckConfigArgs{
						Test:               pulumi.StringArray{pulumi.String("CMD"), pulumi.String("curl"), pulumi.String("-f"), pulumi.String("http://localhost:8000/")},
						IntervalSeconds:    pulumi.IntPtr(5),
						TimeoutSeconds:     pulumi.IntPtr(2),
						Retries:            pulumi.IntPtr(10),
						StartPeriodSeconds: pulumi.IntPtr(10),
					},
					Networks: compose.ServiceNetworkConfigMap{
						"default": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"postgres": compose.ServiceConfigArgs{
					Image:    pulumi.StringPtr("pgvector/pgvector:pg16"),
					Postgres: compose.PostgresConfigArgs{},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(5432),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Environment: pulumi.StringMap{
						"POSTGRES_PASSWORD": postgresPassword,
					},
					Networks: compose.ServiceNetworkConfigMap{
						"default": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"redis": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("redis:6.2"),
					Redis: compose.RedisConfigArgs{},
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(6379),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Networks: compose.ServiceNetworkConfigMap{
						"default": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
				"chat": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("defangio/openai-access-gateway"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(80),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Environment: pulumi.StringMap{
						"OPENAI_API_KEY": pulumi.String("defang"),
					},
					Networks: compose.ServiceNetworkConfigMap{
						"model_provider_private": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
					Llm: compose.LlmConfigArgs{},
				},
				"embedding": compose.ServiceConfigArgs{
					Image: pulumi.StringPtr("defangio/openai-access-gateway"),
					Ports: compose.ServicePortConfigArray{
						compose.ServicePortConfigArgs{
							Target: pulumi.Int(80),
							Mode:   pulumi.StringPtr("host"),
						},
					},
					Environment: pulumi.StringMap{
						"OPENAI_API_KEY": pulumi.String("defang"),
					},
					Networks: compose.ServiceNetworkConfigMap{
						"model_provider_private": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
					Llm: compose.LlmConfigArgs{},
				},
				"worker": compose.ServiceConfigArgs{
					Build: compose.BuildConfigArgs{
						Context:    contextUrl,
						Dockerfile: pulumi.StringPtr("Dockerfile"),
					},
					Command: pulumi.StringArray{pulumi.String("celery"), pulumi.String("-A"), pulumi.String("config"), pulumi.String("worker"), pulumi.String("-l"), pulumi.String("info")},
					DependsOn: compose.ServiceDependencyMap{
						"chat": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
						"embedding": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
						"postgres": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
						"redis": compose.ServiceDependencyArgs{
							Condition: pulumi.StringPtr("service_started"),
							Required:  pulumi.BoolPtr(true),
						},
					},
					Environment: pulumi.StringMap{
						"CHAT_MODEL":             pulumi.String("chat-default"),
						"CHAT_URL":               pulumi.String("http://chat/api/v1/"),
						"DATABASE_URL":           pulumi.Sprintf("postgres://postgres:%s@postgres:5432/postgres?sslmode=%s", postgresPassword, sslMode),
						"DJANGO_SECRET_KEY":      secretKey,
						"DJANGO_SETTINGS_MODULE": pulumi.String("config.settings"),
						"EMBEDDING_MODEL":        pulumi.String("embedding-default"),
						"EMBEDDING_URL":          pulumi.String("http://embedding/api/v1/"),
						"OPENAI_API_KEY":         pulumi.String("defang"),
						"REDIS_URL":              pulumi.String("redis://redis:6379/0"),
					},
					Networks: compose.ServiceNetworkConfigMap{
						"default":                compose.ServiceNetworkConfigArgs{},
						"model_provider_private": compose.ServiceNetworkConfigArgs{},
					},
					Deploy: compose.DeployConfigArgs{
						Resources: compose.ResourcesArgs{
							Reservations: compose.ResourceConfigArgs{
								Cpus:   pulumi.Float64Ptr(0.5),
								Memory: pulumi.StringPtr("512M"),
							},
						},
					},
				},
			},
			Aws: &defangaws.AWSConfigArgs{
				ProjectDomain: projectDomain,
				PublicZoneId:  publicZoneID,
			},
		}, providerOpt)
		if err != nil {
			return err
		}
		ctx.Export("endpoints", proj.Endpoints)
		return nil
	})
}
