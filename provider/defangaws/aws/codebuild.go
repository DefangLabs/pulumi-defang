package aws

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/codebuild"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

// codeBuildResult holds the outputs of creating a CodeBuild project.
type codeBuildResult struct {
	project     *codebuild.Project
	destination pulumix.Output[string] // ECR image URL (repo:tag) where the built image is pushed
}

// codeBuildComputeType maps shm_size (in bytes) to CodeBuild compute type.
// Matches TS: undefined→LARGE, <=4096MiB→SMALL, <=8192→MEDIUM, <=16384→LARGE, <=65536→XLARGE, else 2XLARGE.
func codeBuildComputeType(shmSizeBytes int) string {
	mib := shmSizeBytes / (1024 * 1024)
	if mib <= 0 {
		return "BUILD_GENERAL1_LARGE"
	}
	switch {
	case mib <= 4096:
		return "BUILD_GENERAL1_SMALL"
	case mib <= 8192:
		return "BUILD_GENERAL1_MEDIUM"
	case mib <= 16384:
		return "BUILD_GENERAL1_LARGE"
	case mib <= 65536:
		return "BUILD_GENERAL1_XLARGE"
	default:
		return "BUILD_GENERAL1_2XLARGE"
	}
}

// platformToArch extracts architecture from a platform string.
// Matches TS platformToArch.
func platformToArch(platform string) string {
	if strings.Contains(platform, "arm64") {
		return "arm64"
	}
	return "x86_64"
}

// getBuildSpec generates the CodeBuild buildspec YAML for a Docker image build.
// Matches TS getBuildSpec: pre_build sets up buildx, build runs docker buildx build --push.
func getBuildSpec(build compose.BuildConfig, destination string) string {
	dockerfile := build.GetDockerfile()

	// Build args in deterministic order (matches TS: Object.keys(buildArgs).sort())
	var buildArgsStr string
	if len(build.Args) > 0 {
		keys := make([]string, 0, len(build.Args))
		for k := range build.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("--build-arg \"%s\"", k))
		}
		buildArgsStr = strings.Join(parts, " ")
	}

	var targetArg string
	if target := build.GetTarget(); target != "" {
		targetArg = fmt.Sprintf("--target %s", target)
	}

	preBuildCommands := []string{
		"aws ecr get-login-password --region $AWS_DEFAULT_REGION | docker login --username AWS --password-stdin $(aws sts get-caller-identity --query Account --output text).dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com", // //nolint:lll
		"aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws",
		"docker buildx create --use --driver=docker-container --use",
	}

	buildCmd := fmt.Sprintf("docker buildx build -t %s -f %s --push %s %s $CODEBUILD_SRC_DIR",
		destination, dockerfile, buildArgsStr, targetArg)

	spec := map[string]interface{}{
		"version": 0.2,
		"phases": map[string]interface{}{
			"pre_build": map[string]interface{}{
				"commands": preBuildCommands,
			},
			"build": map[string]interface{}{
				"commands": []string{
					"echo Building the Docker image...",
					strings.TrimSpace(buildCmd),
				},
			},
		},
	}

	b, _ := json.Marshal(spec)
	return string(b)
}

// createCodeBuildProject creates an AWS CodeBuild project for building container images.
// Matches TS createCodeBuildImage: creates project with S3 source, ECR push, privileged mode.
func createCodeBuildProject(
	ctx *pulumi.Context,
	name string,
	build compose.BuildConfig,
	platform string,
	codeBuildRole *iam.Role,
	logGroup *cloudwatch.LogGroup,
	ecrRepoURL pulumix.Output[string],
	region string,
	opts ...pulumi.ResourceOption,
) (*codeBuildResult, error) {
	arch := platformToArch(platform)

	envType := "LINUX_CONTAINER"
	if arch == "arm64" {
		envType = "ARM_CONTAINER"
	}

	// Base image: Amazon Linux (matches TS AMAZON_LINUX_*_IMAGE)
	baseImage := "aws/codebuild/amazonlinux-x86_64-standard:5.0"
	if arch == "arm64" {
		baseImage = "aws/codebuild/amazonlinux-aarch64-standard:3.0"
	}

	computeType := codeBuildComputeType(build.GetShmSizeBytes())

	// Destination: repo:tag where we push the built image
	destination := pulumix.Apply(ecrRepoURL, func(url string) string {
		return url + ":latest"
	})

	// The buildspec needs the destination at apply time
	buildspecOutput := pulumix.Apply(destination, func(dest string) string {
		return getBuildSpec(build, dest)
	})

	// Build environment variables (build args become env vars)
	envVars := codebuild.ProjectEnvironmentEnvironmentVariableArray{
		&codebuild.ProjectEnvironmentEnvironmentVariableArgs{
			Name:  pulumi.String("AWS_DEFAULT_REGION"),
			Value: pulumi.String(region),
		},
	}
	for k, v := range build.Args {
		envVars = append(envVars, &codebuild.ProjectEnvironmentEnvironmentVariableArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(v),
		})
	}

	// Context must be an S3 URL
	sourceType := "S3"
	sourceLocation := pulumix.Apply(pulumix.Output[string](build.Context.ToStringOutput()), func(s string) string {
		return strings.TrimPrefix(s, "s3://")
	})

	project, err := codebuild.NewProject(ctx, name, &codebuild.ProjectArgs{
		Description: pulumi.Sprintf("Build image for %s", name),
		ServiceRole: codeBuildRole.Arn,
		Artifacts: &codebuild.ProjectArtifactsArgs{
			Type: pulumi.String("NO_ARTIFACTS"),
		},
		Cache: &codebuild.ProjectCacheArgs{
			Type:  pulumi.String("LOCAL"),
			Modes: pulumi.StringArray{pulumi.String("LOCAL_DOCKER_LAYER_CACHE"), pulumi.String("LOCAL_SOURCE_CACHE")},
		},
		Environment: &codebuild.ProjectEnvironmentArgs{
			ComputeType:              pulumi.String(computeType),
			Image:                    pulumi.String(baseImage),
			ImagePullCredentialsType: pulumi.String("CODEBUILD"),
			PrivilegedMode:           pulumi.Bool(true), // Required for Docker builds
			Type:                     pulumi.String(envType),
			EnvironmentVariables:     envVars,
		},
		LogsConfig: &codebuild.ProjectLogsConfigArgs{
			CloudwatchLogs: &codebuild.ProjectLogsConfigCloudwatchLogsArgs{
				GroupName:  logGroup.Name,
				StreamName: pulumi.String(name),
			},
		},
		Source: &codebuild.ProjectSourceArgs{
			Type:      pulumi.String(sourceType),
			Location:  pulumi.StringOutput(sourceLocation),
			Buildspec: pulumi.StringOutput(buildspecOutput),
		},
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild project: %w", err)
	}

	return &codeBuildResult{
		project:     project,
		destination: destination,
	}, nil
}

// createCodeBuildRole creates an IAM role for CodeBuild with permissions for
// CloudWatch Logs, S3 source, ECR push, and ECR public login.
// Matches TS createCodeBuildRole.
func createCodeBuildRole(
	ctx *pulumi.Context,
	name string,
	logGroup *cloudwatch.LogGroup,
	ecrRepo *ecr.Repository,
	opts ...pulumi.ResourceOption,
) (*iam.Role, error) {
	assumeRolePolicy, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Action": "sts:AssumeRole",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "codebuild.amazonaws.com",
				},
			},
		},
	})

	role, err := iam.NewRole(ctx, name, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(assumeRolePolicy)),
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild role: %w", err)
	}

	// Inline policy with CloudWatch Logs, S3, ECR permissions
	// Matches TS createCodeBuildRole policy statements
	policyDoc := pulumix.Apply2Err(logGroup.Arn, ecrRepo.Arn, func(logGroupArn, ecrRepoArn string) (string, error) {
		policy := map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Sid":    "AllowCloudWatchLogs",
					"Effect": "Allow",
					"Action": []string{
						"logs:CreateLogGroup",
						"logs:CreateLogStream",
						"logs:PutLogEvents",
					},
					"Resource": []string{logGroupArn + ":*"},
				},
				{
					"Sid":    "AllowS3",
					"Effect": "Allow",
					"Action": []string{
						"s3:PutObject",
						"s3:PutObjectAcl",
						"s3:GetObject",
						"s3:ListBucket",
					},
					"Resource": []string{"*"},
				},
				{
					"Sid":      "AllowECRLogin",
					"Effect":   "Allow",
					"Action":   []string{"ecr:GetAuthorizationToken"},
					"Resource": []string{"*"},
				},
				{
					"Sid":    "AllowECR",
					"Effect": "Allow",
					"Action": []string{
						"ecr:BatchCheckLayerAvailability",
						"ecr:BatchGetImage",
						"ecr:CompleteLayerUpload",
						"ecr:GetDownloadUrlForLayer",
						"ecr:InitiateLayerUpload",
						"ecr:PutImage",
						"ecr:UploadLayerPart",
					},
					"Resource": []string{ecrRepoArn},
				},
				{
					"Sid":    "AllowECRPublicLogin",
					"Effect": "Allow",
					"Action": []string{
						"ecr-public:GetAuthorizationToken",
						"sts:GetServiceBearerToken",
					},
					"Resource": "*",
				},
			},
		}

		b, err := json.Marshal(policy)
		return string(b), err
	})

	_, err = iam.NewRolePolicy(ctx, name+"-policy", &iam.RolePolicyArgs{
		Role:   role.Name,
		Policy: policyDoc,
	}, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating CodeBuild role policy: %w", err)
	}

	return role, nil
}
