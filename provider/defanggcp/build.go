package defanggcp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1"
	cloudbuildpb "cloud.google.com/go/cloudbuild/apiv1/v2/cloudbuildpb"
	"cloud.google.com/go/longrunning"
	lroauto "cloud.google.com/go/longrunning/autogen"
	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/DefangLabs/pulumi-defang/provider/common"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/api/option"
	gtransport "google.golang.org/api/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"gopkg.in/yaml.v3"
)

type Build struct{}

var (
	errBuildFailed               = errors.New("build failed")
	errBuildResultsMissing       = errors.New("build results are missing")
	errBuildResultsMissingImages = errors.New("build results are missing images")
	errInvalidGCSURIPrefix       = errors.New("URI must start with 'gs://' prefix")
	errInvalidGCSURIFormat       = errors.New("URI must contain a bucket and an object path")
)

type BuildArgs struct {
	// Required fields
	ProjectId string `provider:"replaceOnChanges" pulumi:"projectId"`
	Location  string `provider:"replaceOnChanges" pulumi:"location"`
	Source    string `provider:"replaceOnChanges" pulumi:"source"`
	Steps     string `provider:"replaceOnChanges" pulumi:"steps"`

	// TODO: We should be able to use ETAG from object metadata as digest in Diff func
	// to determine a new build is necessary
	SourceDigest   *string           `provider:"replaceOnChanges"     pulumi:"sourceDigest,optional"`
	Images         []string          `provider:"replaceOnChanges"     pulumi:"images,optional"`
	ServiceAccount *string           `provider:"replaceOnChanges"     pulumi:"serviceAccount,optional"`
	Tags           []string          `pulumi:"tags,optional"`
	MachineType    *string           `pulumi:"machineType,optional"`
	DiskSizeGb     *int64            `pulumi:"diskSizeGb,optional"`
	Substitutions  map[string]string `pulumi:"substitutions,optional"`

	// Max wait time in seconds (default: common.DefaultBuildMaxWaitTime).
	// Sets the server-side build timeout in Cloud Build.
	MaxWaitTime *int `pulumi:"maxWaitTime,optional"`
}

type MachineType = cloudbuildpb.BuildOptions_MachineType

const (
	UNSPECIFIED   MachineType = cloudbuildpb.BuildOptions_UNSPECIFIED
	N1_HIGHCPU_8  MachineType = cloudbuildpb.BuildOptions_N1_HIGHCPU_8
	N1_HIGHCPU_32 MachineType = cloudbuildpb.BuildOptions_N1_HIGHCPU_32
	E2_HIGHCPU_8  MachineType = cloudbuildpb.BuildOptions_E2_HIGHCPU_8
	E2_HIGHCPU_32 MachineType = cloudbuildpb.BuildOptions_E2_HIGHCPU_32
	E2_MEDIUM     MachineType = cloudbuildpb.BuildOptions_E2_MEDIUM
)

type BuildState struct {
	BuildArgs
	BuildId     string `pulumi:"buildId"`
	ImageDigest string `pulumi:"imageDigest"`
}

// Create runs a Cloud Build job and returns the build ID and image digest.
func (*Build) Create(
	ctx context.Context,
	req infer.CreateRequest[BuildArgs],
) (infer.CreateResponse[BuildState], error) {
	state := BuildState{BuildArgs: req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[BuildState]{ID: req.Name, Output: state}, nil
	}
	buildId, imageDigest, err := runCloudBuild(ctx, req.Inputs)
	if err != nil {
		return infer.CreateResponse[BuildState]{ID: req.Name, Output: state}, err
	}
	state.BuildId = buildId
	state.ImageDigest = imageDigest
	return infer.CreateResponse[BuildState]{ID: req.Name, Output: state}, nil
}

func runCloudBuild(ctx context.Context, args BuildArgs) (string, string, error) {
	client, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Cloud Build client: %w", err)
	}
	defer func() { _ = client.Close() }()

	// steps := createSteps(args)
	var steps []*cloudbuildpb.BuildStep
	if err := yaml.Unmarshal([]byte(args.Steps), &steps); err != nil {
		return "", "", fmt.Errorf("failed to parse cloudbuild steps: %w, steps are:\n%v", err, args.Steps)
	}

	// Extract bucket and object from the source
	bucket, object, err := parseGCSURI(args.Source)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse source URI: %w", err)
	}

	// TODO: Implement secrets with a global `availableSecrets` and per-step `secretEnv`
	// See: https://cloud.google.com/build/docs/securing-builds/use-secrets
	// TODO: Use inline secret for environment variables since there is no other way to pass env vars to build steps
	// var secrets *cloudbuildpb.Secrets

	maxWait := common.DefaultBuildMaxWaitTime
	if args.MaxWaitTime != nil {
		maxWait = *args.MaxWaitTime
	}

	// Create a build request
	build := &cloudbuildpb.Build{
		Substitutions: args.Substitutions,
		Steps:         steps,
		// TODO: Support NPM or Python packages using Artifacts field
		Source: &cloudbuildpb.Source{
			Source: &cloudbuildpb.Source_StorageSource{
				StorageSource: &cloudbuildpb.StorageSource{
					Bucket: bucket,
					Object: object,
				},
			},
		},
		// AvailableSecrets: secrets,
		Options: &cloudbuildpb.BuildOptions{
			MachineType: GetMachineType(args.MachineType),
			DiskSizeGb:  GetDiskSize(args.DiskSizeGb),
			Logging:     cloudbuildpb.BuildOptions_CLOUD_LOGGING_ONLY,
		},
		Timeout: durationpb.New(time.Duration(maxWait) * time.Second),
		Tags:    args.Tags,
		Images:  args.Images,
	}

	if args.ServiceAccount != nil {
		build.ServiceAccount = fmt.Sprintf("projects/%s/serviceAccounts/%s", args.ProjectId, *args.ServiceAccount)
	}

	// Trigger the build
	pbop, err := client.CreateBuild(ctx, &cloudbuildpb.CreateBuildRequest{
		ProjectId: args.ProjectId, // Replace with your GCP project ID
		// Current API endpoint does not support location
		// Parent:    fmt.Sprintf("projects/%s/locations/%s", args.ProjectId, args.Location),
		Build: build,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to create build: %w", err)
	}

	op, err := NewCloudBuildOperation(ctx, pbop)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Cloud Build operation: %w", err)
	}
	defer func() { _ = op.Close() }()

	build, err = op.Wait(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to wait for build to complete: %w", err)
	}
	if build.GetStatus() != cloudbuildpb.Build_SUCCESS {
		return "", "", fmt.Errorf("%w: %v: %v", errBuildFailed, build.GetStatus().String(), build.GetStatusDetail())
	}
	if build.GetResults() == nil {
		return "", "", errBuildResultsMissing
	}
	if len(build.GetResults().GetImages()) == 0 {
		return "", "", errBuildResultsMissingImages
	}
	return build.GetId(), build.GetResults().GetImages()[0].GetDigest(), nil
}

type CloudBuildOperation struct {
	lro    *longrunning.Operation
	client *lroauto.OperationsClient
}

func NewCloudBuildOperation(ctx context.Context, op *longrunningpb.Operation) (*CloudBuildOperation, error) {
	clientOptions := []option.ClientOption{
		option.WithEndpoint("cloudbuild.googleapis.com:443"),
		option.WithScopes("https://www.googleapis.com/auth/cloud-platform"),
		option.WithGRPCDialOption(grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(math.MaxInt32))),
	}

	connPool, err := gtransport.DialPool(ctx, clientOptions...)
	if err != nil {
		return nil, err
	}

	lroClient, err := lroauto.NewOperationsClient(ctx, gtransport.WithConnPool(connPool))
	if err != nil {
		_ = connPool.Close()
		return nil, fmt.Errorf("failed to create Operations client: %w", err)
	}
	lro := longrunning.InternalNewOperation(lroClient, op)

	return &CloudBuildOperation{lro: lro, client: lroClient}, nil
}

func (o *CloudBuildOperation) Wait(ctx context.Context) (*cloudbuildpb.Build, error) {
	var resp cloudbuildpb.Build
	err := o.lro.Wait(ctx, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, err
}

// Close releases the underlying gRPC connection pool. Without this the pool
// dialed in NewCloudBuildOperation leaks for the lifetime of the process —
// each Build resource Create call opens a new pool that's never reaped.
func (o *CloudBuildOperation) Close() error {
	if o.client == nil {
		return nil
	}
	return o.client.Close()
}

func GetMachineType(machineType *string) MachineType {
	if machineType == nil {
		return UNSPECIFIED
	}
	m, ok := cloudbuildpb.BuildOptions_MachineType_value[*machineType]
	if !ok {
		return UNSPECIFIED
	}
	return MachineType(m)
}

func GetDiskSize(diskSizeGb *int64) int64 {
	if diskSizeGb == nil {
		return 0
	}
	return *diskSizeGb
}

func parseGCSURI(uri string) (string, string, error) {
	if !strings.HasPrefix(uri, "gs://") {
		return "", "", errInvalidGCSURIPrefix
	}

	parts := strings.SplitN(uri[5:], "/", 2)
	if len(parts) < 2 {
		return "", "", errInvalidGCSURIFormat
	}
	obj, err := url.QueryUnescape(parts[1]) // Because the base 64 encoding may contain '='
	if err != nil {
		return "", "", fmt.Errorf("failed to unescape object path: %w", err)
	}

	return parts[0], obj, nil
}
