package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/DefangLabs/pulumi-defang/provider/compose"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/codebuild"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/ecr"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

var (
	errCodeBuildS3NoObjectKey = errors.New("CodeBuild S3 source URL has no object key")
	errCodeBuildNotS3URL      = errors.New("CodeBuild source must be an S3 URL")
	errCodeBuildS3Incomplete  = errors.New("CodeBuild source must include an S3 bucket and object key")
)

// s3HostPattern matches real AWS S3 HTTPS endpoint hostnames -- both
// virtual-hosted ("bucket.s3[-.]<region>.amazonaws.com") and path-style
// ("s3[-.]<region>.amazonaws.com") -- anchored on both ends so a lookalike
// like "bucket.s3.evil.example" or "s3.evil.example" cannot be mistaken for
// one (a naive substring/prefix check on the hostname would accept both).
//
// The optional "dualstack" label covers the IPv6 dual-stack endpoints
// ("s3.dualstack.<region>.amazonaws.com" and its virtual-hosted form), which
// the AWS SDK emits when it is configured with AWS_USE_DUALSTACK_ENDPOINT --
// without it a dual-stack presigned context URL would fail this check and
// surface as a confusing "must be an S3 URL" error at project-creation time.
var s3HostPattern = regexp.MustCompile(
	`^(?:(?P<bucket>[^.]+)\.)?s3(?:\.dualstack)?(?:[.-][a-z0-9-]+)?\.amazonaws\.com$`)

func normalizeCodeBuildS3Location(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing CodeBuild source URL: %w", err)
	}

	object := strings.TrimPrefix(u.EscapedPath(), "/")
	var bucket string
	switch u.Scheme {
	case "s3":
		bucket = u.Host
	case "http", "https":
		m := s3HostPattern.FindStringSubmatch(u.Hostname())
		if m == nil {
			return "", fmt.Errorf("%w: got %q", errCodeBuildNotS3URL, rawURL)
		}
		if bucket = m[s3HostPattern.SubexpIndex("bucket")]; bucket == "" {
			// Path style: https://s3.region.amazonaws.com/bucket/key.
			var found bool
			bucket, object, found = strings.Cut(object, "/")
			if !found {
				return "", fmt.Errorf("%w: %q", errCodeBuildS3NoObjectKey, rawURL)
			}
		}
	default:
		return "", fmt.Errorf("%w: got %q", errCodeBuildNotS3URL, rawURL)
	}

	if bucket == "" || object == "" {
		return "", fmt.Errorf("%w: got %q", errCodeBuildS3Incomplete, rawURL)
	}
	return bucket + "/" + object, nil
}

// codeBuildExtractArchiveCommand returns the pre_build shell command that extracts the
// uploaded build context, or "" if the context is a .zip -- CodeBuild's S3 source only
// auto-extracts .zip, so a .zip context arrives in $CODEBUILD_SRC_DIR already extracted
// and there is nothing to do.
//
// The CLI uploads the build context as .tar.gz (or .tgz) for every non-Railpack build on
// every provider (GCP's Cloud Build extracts that natively, CodeBuild does not), so a
// non-zip source otherwise lands here as a single untouched archive file with no Dockerfile
// anywhere. CodeBuild downloads a non-zip S3 source into $CODEBUILD_SRC_DIR under the same
// filename as the object key, so that name is known up front from the source URL -- extract
// that exact file rather than glob-discovering it (matches the legacy TS CD's use of the
// known contextFile path). The archive is removed afterward so a Dockerfile that COPYs the
// whole build context doesn't also pick up its own multi-megabyte source tarball.
func codeBuildExtractArchiveCommand(contextURL string) (string, error) {
	loc, err := normalizeCodeBuildS3Location(contextURL)
	if err != nil {
		return "", err
	}
	archive := path.Base(loc)
	if !strings.HasSuffix(archive, ".tar.gz") && !strings.HasSuffix(archive, ".tgz") {
		return "", nil
	}
	quoted := "'" + strings.ReplaceAll(archive, "'", `'\''`) + "'"
	return fmt.Sprintf("tar xzf %s && rm %s", quoted, quoted), nil
}

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

type ArchType string

const Arm64 ArchType = "arm64"
const X86_64 ArchType = "x86_64"

// platformToArch extracts architecture from a platform string.
// Matches TS platformToArch.
func platformToArch(platform string) ArchType {
	if strings.Contains(strings.ToLower(platform), "arm") {
		return Arm64
	}
	return X86_64 // default to x86_64; TODO: revisit this default
}

// dockerhubMirrorContainerName is the name of the local pull-through mirror container.
const dockerhubMirrorContainerName = "dockerhub-ecr-mirror"

// publicEcrProxyURL is where the local pull-through mirror listens on the CodeBuild host.
// Both the Docker daemon (registry-mirrors) and BuildKit (buildkitd.toml) point at it.
const publicEcrProxyURL = "http://localhost:5000"

// publicEcrProxyNginxConf is the nginx config for the Docker Hub -> public.ecr.aws pull-through
// mirror. Docker Hub images are mirrored by AWS under public.ecr.aws/docker/<repo>, so /v2/<rest>
// maps to /v2/docker/<rest>; anything the mirror does not have (404) falls back to Docker Hub
// itself. The __TOKEN__ line is replaced at build time with the public-ECR bearer token.
// Matches TS getPublicEcrProxySteps' nginxConf.
//
// The backslash-escaped $ are for the unquoted shell heredoc that writes this file: they must
// reach nginx as literal nginx variables, not be expanded by the shell.
const publicEcrProxyNginxConf = `events {}
http {
    server {

        set \$token "";
__TOKEN__

        listen 80;
        location = /v2/ {
            proxy_pass https://public.ecr.aws/v2/;
            proxy_intercept_errors on;
            proxy_set_header Authorization "Bearer \$token";
            proxy_ssl_server_name on;

            error_page 404 = @fallback;
        }

        location ~ ^/v2/(.+) {
            # Capture the rest of the path
            set \$rest \$1;

            # Build upstream path: /v2/docker/<rest>
            proxy_pass https://public.ecr.aws/v2/docker/\$rest;
            proxy_intercept_errors on;
            proxy_set_header Authorization "Bearer \$token";
            proxy_ssl_server_name on;

            error_page 404 = @fallback;
        }

        location @fallback {
            proxy_pass https://registry-1.docker.io;
            proxy_ssl_server_name on;
        }
    }
}
`

// getPublicEcrProxySteps returns the pre_build commands that start a local nginx container
// proxying Docker Hub pulls to public.ecr.aws, so builds are not subject to Docker Hub's
// anonymous pull rate limits. Matches TS getPublicEcrProxySteps.
//
// The bearer token is fetched with docker-credential-ecr-login (valid 12h, see
// https://docs.aws.amazon.com/AmazonECRPublic/latest/APIReference/API_GetAuthorizationToken.html)
// and split into 1024-char chunks, because nginx cannot hold a token that long in one directive.
func getPublicEcrProxySteps() []string {
	return []string{
		"cat > /tmp/nginx.conf.tmpl << EOF\n" + publicEcrProxyNginxConf + "\nEOF",
		`TOKEN_BLOCK=$(echo "https://public.ecr.aws" | docker-credential-ecr-login get | ` +
			`jq -j '"\(.Username):\(.Secret)"' | base64 -w 1024 | ` +
			`awk '{ print "set $token \"${token}" $0 "\";" }')`,
		`awk -v token_block="$TOKEN_BLOCK" '{ if ($0 ~ /__TOKEN__/) { print token_block } else { print } }' ` +
			`/tmp/nginx.conf.tmpl > /tmp/nginx.conf`,
		// CodeBuild reuses warm hosts (we enable LOCAL_* caches), so a mirror container from an
		// earlier build can still be around under this name; remove it before creating our own.
		"docker rm -f " + dockerhubMirrorContainerName + " || true",
		// NOTE: intentionally NO restart policy on this container -- do not add --restart=always.
		// The container only has to live for one build. With a restart policy, the Docker daemon
		// on a reused host resurrects the previous build's container from its stale
		// /tmp/nginx.conf bind-mount spec, and Docker creates a missing bind-mount source as a
		// *directory*, which then breaks this build's own `awk ... > /tmp/nginx.conf` redirect.
		// See https://github.com/DefangLabs/defang-mvp/issues/2869
		"docker run -d --rm -p 5000:80 --name " + dockerhubMirrorContainerName +
			" -v /tmp/nginx.conf:/etc/nginx/nginx.conf:ro public.ecr.aws/nginx/nginx:stable-alpine",
		"sleep 3", // give the mirror some time to start
	}
}

// getSetupMirrorSteps returns the pre_build commands that point the Docker daemon and BuildKit at
// the given docker.io mirrors. Matches TS getSetupMirrorSteps.
func getSetupMirrorSteps(mirrors []string) ([]string, error) {
	if len(mirrors) == 0 {
		return nil, nil
	}

	daemonJSON, err := json.Marshal(map[string][]string{"registry-mirrors": mirrors})
	if err != nil {
		return nil, fmt.Errorf("marshaling docker daemon.json: %w", err)
	}

	// BuildKit's `mirrors` entries are bare host:port (no scheme), with a separate `http = true`
	// flag for a plaintext mirror -- unlike Docker's own daemon.json, which wants the full URL.
	// https://docs.docker.com/build/buildkit/toml-configuration/
	allHTTP := true
	quoted := make([]string, 0, len(mirrors))
	for _, m := range mirrors {
		host, isHTTP := strings.CutPrefix(m, "http://")
		if !isHTTP {
			host = strings.TrimPrefix(m, "https://")
			allHTTP = false
		}
		quoted = append(quoted, `"`+host+`"`)
	}
	var httpLine string
	if allHTTP {
		httpLine = "\n  http = true"
	}
	buildkitdToml := "\n[registry.\"docker.io\"]\n  mirrors = [\n    " +
		strings.Join(quoted, ", ") + "\n  ]" + httpLine + "\n"

	// Written via single-quoted heredocs, not `echo '...'`: a mirror value containing a single
	// quote would otherwise break out of the shell string. These values are internal constants
	// today, not user input, but this avoids the class of bug entirely rather than relying on that.
	return []string{
		"mkdir -p /etc/docker/\ncat > /etc/docker/daemon.json <<'EOF'\n" + string(daemonJSON) + "\nEOF",
		"cat > /tmp/buildkitd.toml <<'EOF'" + buildkitdToml + "EOF",
		// dockerd only picks up a changed registry-mirrors list on SIGHUP.
		"kill -HUP $(pidof dockerd)",
	}, nil
}

// getBuildSpec generates the CodeBuild buildspec YAML for a Docker image build.
// contextURL is the same S3 location used as the CodeBuild project's source, so the
// generated extraction step can name the archive exactly rather than glob-discovering it.
// Matches TS getBuildSpec: pre_build sets up mirrors and buildx, build runs docker buildx build --push.
func getBuildSpec(build compose.BuildConfig, destination, contextURL string) (string, error) {
	dockerfile := build.GetDockerfile()

	extractArchiveCmd, err := codeBuildExtractArchiveCommand(contextURL)
	if err != nil {
		return "", err
	}

	// Build args in deterministic order (matches TS: Object.keys(buildArgs).sort())
	var buildArgsStr string
	if len(build.Args) > 0 {
		var parts []string
		for k := range common.Sorted(build.Args) {
			parts = append(parts, fmt.Sprintf("--build-arg %q", k))
		}
		buildArgsStr = strings.Join(parts, " ")
	}

	var targetArg string
	if target := build.GetTarget(); target != "" {
		targetArg = "--target " + target
	}

	// Matches TS prepareBuildSteps: --platform on both buildx create and buildx build,
	// cache_from/cache_to passed through verbatim.
	var platformArg string
	if len(build.Platforms) > 0 {
		platformArg = "--platform " + strings.Join(build.Platforms, ",")
	}
	cacheArgs := make([]string, 0, len(build.CacheFrom)+len(build.CacheTo))
	for _, c := range build.CacheFrom {
		cacheArgs = append(cacheArgs, "--cache-from="+c)
	}
	for _, c := range build.CacheTo {
		cacheArgs = append(cacheArgs, "--cache-to="+c)
	}

	preBuildCommands := make([]string, 0, 13)
	if extractArchiveCmd != "" {
		preBuildCommands = append(preBuildCommands, extractArchiveCmd)
	}
	preBuildCommands = append(preBuildCommands,
		"aws ecr get-login-password --region $AWS_DEFAULT_REGION | docker login --username AWS --password-stdin $(aws sts get-caller-identity --query Account --output text).dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com", //nolint:lll
		"aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws",
	)

	// Route docker.io through a local pull-through mirror backed by public.ecr.aws, so builds do
	// not hit Docker Hub's anonymous pull rate limits. Matches TS prepareBuildSteps, minus the
	// hasDockerhubAuth() opt-out: unlike TS, this provider has no way to configure explicit Docker
	// Hub credentials, so there is never a reason to skip the mirror.
	// TODO: also honor a user-supplied registry mirror here (TS BuildSpecArgs.registryMirror).
	mirrors := []string{publicEcrProxyURL}
	mirrorSteps, err := getSetupMirrorSteps(mirrors)
	if err != nil {
		return "", err
	}
	preBuildCommands = append(preBuildCommands, mirrorSteps...)
	preBuildCommands = append(preBuildCommands, getPublicEcrProxySteps()...)

	// BuildKit runs in its own container, so it needs both the mirror config and host networking
	// to be able to reach the mirror on localhost:5000.
	var buildkitdCfgArg string
	if len(mirrors) > 0 {
		buildkitdCfgArg = "--buildkitd-config=/tmp/buildkitd.toml"
	}
	buildxCreate := "docker buildx create --use --driver=docker-container " + buildkitdCfgArg +
		" --driver-opt network=host --use " + platformArg
	preBuildCommands = append(preBuildCommands, strings.Join(strings.Fields(buildxCreate), " "))

	buildCmd := fmt.Sprintf("docker buildx build %s %s -t %s -f %s --push %s %s $CODEBUILD_SRC_DIR",
		platformArg, strings.Join(cacheArgs, " "), destination, dockerfile, buildArgsStr, targetArg)
	buildCmd = strings.Join(strings.Fields(buildCmd), " ") // collapse empty args

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

	b, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshaling buildspec: %w", err)
	}
	return string(b), nil
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
	// build.platforms takes precedence over the service platform. Matches TS:
	// ARM_CONTAINER only when building for exactly one platform and it's arm64
	// (multi-platform builds emulate via buildx, so run on the x86 fleet).
	arch := platformToArch(platform)
	if len(build.Platforms) > 0 {
		arch = X86_64
		if len(build.Platforms) == 1 {
			arch = platformToArch(build.Platforms[0])
		}
	}

	envType := "LINUX_CONTAINER"
	if arch == Arm64 {
		envType = "ARM_CONTAINER"
	}

	// Base image: Amazon Linux (matches TS AMAZON_LINUX_*_IMAGE)
	baseImage := "aws/codebuild/amazonlinux-x86_64-standard:5.0"
	if arch == Arm64 {
		baseImage = "aws/codebuild/amazonlinux-aarch64-standard:3.0"
	}

	computeType := codeBuildComputeType(build.GetShmSizeBytes())

	// Destination: repo:tag where we push the built image
	destination := pulumix.Apply(ecrRepoURL, func(url string) string {
		return url + ":latest"
	})

	contextOutput := pulumix.Output[string](build.Context.ToStringOutput())

	// The buildspec needs the destination and the resolved build-context URL (so its
	// extraction step can name the uploaded archive exactly) at apply time.
	buildspecOutput := pulumix.Apply2Err(destination, contextOutput, func(dest, contextURL string) (string, error) {
		return getBuildSpec(build, dest, contextURL)
	})

	// Build environment variables (build args become env vars)
	envVars := codebuild.ProjectEnvironmentEnvironmentVariableArray{
		&codebuild.ProjectEnvironmentEnvironmentVariableArgs{
			Name:  pulumi.String("AWS_DEFAULT_REGION"),
			Value: pulumi.String(region),
		},
	}
	for k, v := range common.Sorted(build.Args) {
		envVars = append(envVars, &codebuild.ProjectEnvironmentEnvironmentVariableArgs{
			Name:  pulumi.String(k),
			Value: pulumi.String(v),
		})
	}

	// CodeBuild expects S3 sources as bucket/key, while the CLI supplies either
	// an s3:// URL or the query-free HTTPS URL returned by S3's presigner.
	sourceType := "S3"
	sourceLocation := pulumix.ApplyErr(contextOutput, normalizeCodeBuildS3Location)

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
				GroupName:  logGroup.Name, // FIXME: separate logGroup for builds
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
	assumeRolePolicyBytes, err := json.Marshal(map[string]interface{}{
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
	if err != nil {
		return nil, fmt.Errorf("marshaling assume role policy: %w", err)
	}
	assumeRolePolicy := string(assumeRolePolicyBytes)

	role, err := iam.NewRole(ctx, name, &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(assumeRolePolicy),
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
