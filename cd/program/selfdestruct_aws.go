package program

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/DefangLabs/pulumi-defang/provider/compose"
	defangaws "github.com/DefangLabs/pulumi-defang/provider/defangaws/aws"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v7/go/aws/scheduler"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// On AWS the CD runs as a CodeBuild build (see AwsCodeBuild.Run in the CLI),
// so the self-destruct trigger is an EventBridge Scheduler one-time schedule
// (`at(...)`) whose universal target calls codebuild:StartBuild with the same
// overrides the CLI passes: this image, a buildspec running `/app/cd down`,
// and (a filtered copy of) this run's environment.
const (
	// codebuildStartBuildTargetArn is the EventBridge Scheduler universal
	// target for the CodeBuild StartBuild API.
	codebuildStartBuildTargetArn = "arn:aws:scheduler:::aws-sdk:codebuild:startBuild"

	// selfDestructBuildspec mirrors buildspec() in the CLI's codebuild
	// package: CodeBuild ignores the image's ENTRYPOINT/WORKDIR, so the spec
	// recreates them. The paths match this repo's Dockerfile (WORKDIR /app,
	// ENTRYPOINT /app/cd) — necessarily the layout of the very image this
	// code runs in, which is also the image the trigger re-runs.
	selfDestructBuildspec = `version: "0.2"
phases:
  build:
    commands:
      - mkdir -p /app && cd /app && /app/cd down
`
)

// createAWSSelfDestruct schedules this stack's own `defang cd down`. Unlike
// the Azure variant it deliberately does NOT depend on the project component:
// the schedule has no resource-level dependency on it, and arming the trigger
// before the services finish deploying means even a deploy that fails halfway
// still cleans itself up at the TTL.
func createAWSSelfDestruct(pctx *pulumi.Context, cf *compose.Project, ttl time.Duration, opts ...pulumi.ResourceOption) error {
	buildArn := os.Getenv("CODEBUILD_BUILD_ARN")
	image := os.Getenv("CODEBUILD_BUILD_IMAGE")
	if buildArn == "" || image == "" {
		// Local debug runs (DEFANG_PULUMI_DIR) have no CodeBuild identity to
		// clone; there is nothing to schedule a down against.
		return fmt.Errorf("defang:ttl requires the CD to run in CodeBuild (CODEBUILD_BUILD_ARN/CODEBUILD_BUILD_IMAGE not set)")
	}
	projectName, projectArn, err := codebuildProjectFromBuildArn(buildArn)
	if err != nil {
		return err
	}

	fireAt := time.Now().Add(ttl)
	input, err := awsSelfDestructInput(projectName, image, os.Environ())
	if err != nil {
		return err
	}

	assumeRole, err := json.Marshal(defangaws.PolicyDocument{
		Version: iam.PolicyDocumentVersion_2012_10_17,
		Statement: []defangaws.PolicyStatement{{
			Effect:    iam.PolicyStatementEffectALLOW,
			Principal: map[string]any{"Service": "scheduler.amazonaws.com"},
			Action:    "sts:AssumeRole",
		}},
	})
	if err != nil {
		return err
	}
	policy, err := json.Marshal(defangaws.PolicyDocument{
		Version: iam.PolicyDocumentVersion_2012_10_17,
		Statement: []defangaws.PolicyStatement{{
			Effect:   iam.PolicyStatementEffectALLOW,
			Action:   "codebuild:StartBuild",
			Resource: projectArn,
		}},
	})
	if err != nil {
		return err
	}

	role, err := iam.NewRole(pctx, "self-destruct", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(assumeRole),
		InlinePolicies: iam.RoleInlinePolicyArray{
			iam.RoleInlinePolicyArgs{
				Name:   pulumi.String("start-cd-down"),
				Policy: pulumi.String(policy),
			},
		},
	}, opts...)
	if err != nil {
		return err
	}

	_ = pctx.Log.Info(fmt.Sprintf("self-destruct: this stack will run `defang cd down` on itself at %s (ttl %s); redeploying extends it",
		fireAt.UTC().Format(time.RFC3339), ttl), nil)

	_, err = scheduler.NewSchedule(pctx, "self-destruct", &scheduler.ScheduleArgs{
		Description: pulumi.Sprintf("defang self-destruct for %s/%s", cf.Name, pctx.Stack()),
		// One-time schedule, evaluated in UTC (the provider default). AWS
		// rejects a fire time in the past at create — the CLI's minTTL floor
		// keeps that from happening for real deploys, but a short TTL from a
		// test or manual CD invocation can still hit it if this resource
		// isn't created until after fireAt has passed.
		ScheduleExpression: pulumi.String(fmt.Sprintf("at(%s)", fireAt.UTC().Format("2006-01-02T15:04:05"))),
		FlexibleTimeWindow: scheduler.ScheduleFlexibleTimeWindowArgs{
			Mode: pulumi.String("OFF"),
		},
		Target: scheduler.ScheduleTargetArgs{
			Arn:     pulumi.String(codebuildStartBuildTargetArn),
			RoleArn: role.Arn,
			Input:   pulumi.String(input),
		},
	}, opts...)
	return err
}

// codebuildProjectFromBuildArn derives the CodeBuild project name and project
// ARN from a build ARN like
// arn:aws:codebuild:us-west-2:123456789012:build/defang-cd-abc:uuid.
func codebuildProjectFromBuildArn(buildArn string) (name, arn string, _ error) {
	prefix, resource, ok := strings.Cut(buildArn, ":build/")
	if !ok {
		return "", "", fmt.Errorf("unexpected CODEBUILD_BUILD_ARN %q", buildArn)
	}
	name, _, ok = strings.Cut(resource, ":")
	if !ok || name == "" {
		return "", "", fmt.Errorf("unexpected CODEBUILD_BUILD_ARN %q", buildArn)
	}
	return name, prefix + ":project/" + name, nil
}

// awsSelfDestructInput renders the codebuild:StartBuild request the schedule
// fires — the same overrides AwsCodeBuild.Run passes, with args ["down"].
// Universal-target inputs use the target API's own wire shape, which for
// CodeBuild is camelCase JSON.
func awsSelfDestructInput(projectName, image string, environ []string) (string, error) {
	env := SelfDestructEnv(environ)
	envOverrides := make([]startBuildEnvVar, 0, len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		envOverrides = append(envOverrides, startBuildEnvVar{Name: k, Value: env[k], Type: "PLAINTEXT"})
	}
	input := startBuildInput{
		ProjectName:                  projectName,
		ImageOverride:                image,
		BuildspecOverride:            selfDestructBuildspec,
		EnvironmentVariablesOverride: envOverrides,
	}
	// Mirrors AwsCodeBuild.Run: non-"aws/" images pull with the service role
	// (e.g. via an ECR pull-through cache).
	if !strings.HasPrefix(image, "aws/") {
		input.ImagePullCredentialsTypeOverride = "SERVICE_ROLE"
	}
	b, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// startBuildInput is the subset of the CodeBuild StartBuild request the
// schedule sends (universal-target inputs use the target API's own camelCase
// wire shape; see codebuild.StartBuildInput in the AWS SDK for the full set).
type startBuildInput struct {
	ProjectName                      string             `json:"projectName"`
	ImageOverride                    string             `json:"imageOverride"`
	BuildspecOverride                string             `json:"buildspecOverride"`
	EnvironmentVariablesOverride     []startBuildEnvVar `json:"environmentVariablesOverride"`
	ImagePullCredentialsTypeOverride string             `json:"imagePullCredentialsTypeOverride,omitempty"`
}

type startBuildEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Type  string `json:"type"`
}
