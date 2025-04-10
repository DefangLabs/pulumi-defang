// Copyright 2016-2023, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DefangLabs/defang/src/cmd/cli/command"
	"github.com/DefangLabs/defang/src/pkg"
	"github.com/DefangLabs/defang/src/pkg/cli"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/cli/compose"
	defangTypes "github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/types"
	"github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/pulumi/pulumi-go-provider/infer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Each resource has a controlling struct.
// Resource behavior is determined by implementing methods on the controlling struct.
// The `Create` method is mandatory, but other methods are optional.
// - Check: Remap inputs before they are typed.
// - Diff: Change how instances of a resource are compared.
// - Update: Mutate a resource in place.
// - Read: Get the state of a resource from the backing provider.
// - Delete: Custom logic when the resource is deleted.
// - Annotate: Describe fields and set defaults for a resource.
// - WireDependencies: Control how outputs and secrets flows through values.
type Project struct{}

type ProjectConfig struct {
	types.Project
}

// Each resource has an input struct, defining what arguments it accepts.
type ProjectArgs struct {
	// Fields projected into Pulumi must be public and hava a `pulumi:"..."` tag.
	// The pulumi tag doesn't need to match the field name, but it's generally a
	// good idea.
	CloudProviderID client.ProviderID `pulumi:"providerID"`
	ConfigPaths     []string          `pulumi:"configPaths,optional"`
	Config          *ProjectConfig    `pulumi:"config,optional"`
}

type ServiceInfo struct {
	defangv1.ServiceInfo
	Endpoints   []string               `json:"endpoints,omitempty"     pulumi:"endpoints,omitempty"`
	Project     string                 `json:"project,omitempty"       pulumi:"project,omitempty"`
	Etag        string                 `json:"etag,omitempty"          pulumi:"etag,omitempty"`
	Status      string                 `json:"status,omitempty"        pulumi:"status,omitempty"`
	NatIps      []string               `json:"nat_ips,omitempty"       pulumi:"nat_ips,omitempty"`
	LbIps       []string               `json:"lb_ips,omitempty"        pulumi:"lb_ips,omitempty"`
	PrivateFqdn string                 `json:"private_fqdn,omitempty"  pulumi:"private_fqdn,omitempty"`
	PublicFqdn  string                 `json:"public_fqdn,omitempty"   pulumi:"public_fqdn,omitempty"`
	CreatedAt   *timestamppb.Timestamp `json:"created_at,omitempty"    pulumi:"created_at,omitempty"`
	UpdatedAt   *timestamppb.Timestamp `json:"updated_at,omitempty"    pulumi:"updated_at,omitempty"`
	ZoneId      string                 `json:"zone_id,omitempty"       pulumi:"zone_id,omitempty"` //nolint:stylecheck
	UseAcmeCert bool                   `json:"use_acme_cert,omitempty" pulumi:"use_acme_cert,omitempty"`
	Domainname  string                 `json:"domainname,omitempty"    pulumi:"domanname,omitempty"`
	LbDnsName   string                 `json:"lb_dns_name,omitempty"   pulumi:"lb_dns_name,omitempty"` //nolint:stylecheck
	TaskRole    *string                `json:"task_role,omitempty"     pulumi:"task_role"`
}

// Each resource has a state, describing the fields that exist on the created resource.
type ProjectState struct {
	// It is generally a good idea to embed args in outputs, but it isn't strictly necessary.
	ProjectArgs
	Etag     defangTypes.ETag        `pulumi:"etag"`
	AlbArn   string                  `pulumi:"albArn"`
	Services map[string]*ServiceInfo `pulumi:"services"`
}

var errNoProjectUpdate = errors.New("no project update found")

// All resources must implement Create at a minimum.
func (Project) Create(ctx context.Context, name string, input ProjectArgs, preview bool) (string, ProjectState, error) {
	state := ProjectState{ProjectArgs: input}
	if preview {
		return name, state, nil
	}

	project, err := loadProject(ctx, name, input)
	if err != nil {
		return name, state, fmt.Errorf("failed to load project: %w", err)
	}

	driver, err := NewDriver(ctx, input.CloudProviderID)
	if err != nil {
		return name, state, fmt.Errorf("failed to create driver: %w", err)
	}

	err = Authenticate(ctx, driver)
	if err != nil {
		return name, state, fmt.Errorf("failed to authenticate: %w", err)
	}

	err = configureProviderCdImage(ctx, driver, name, input.CloudProviderID)
	if err != nil {
		return name, state, fmt.Errorf("failed to configure provider CD image: %w", err)
	}

	deploy, err := deployProject(ctx, driver.GetFabricClient(), driver.GetProvider(), project, preview)
	if err != nil {
		return name, state, fmt.Errorf("failed to deploy project: %w", err)
	}

	etag := deploy.GetEtag()

	projectUpdate, err := getProjectOutputs(ctx, driver.GetProvider(), project.Name, etag)
	if err != nil {
		return name, state, fmt.Errorf("failed to get project outputs: %w", err)
	}

	if projectUpdate == nil {
		return name, state, errNoProjectUpdate
	}

	state.Etag = etag
	state.AlbArn = projectUpdate.GetAlbArn()
	state.Services = makeServices(projectUpdate)

	return name, state, nil
}

func loadProject(ctx context.Context, name string, input ProjectArgs) (*compose.Project, error) {
	if input.Config == nil {
		l := compose.NewLoader(compose.WithProjectName(name), compose.WithPath(input.ConfigPaths...))
		project, err := l.LoadProject(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load project from paths %q: %w", input.ConfigPaths, err)
		}

		return project, nil
	}

	if input.Config.Name == "" {
		input.Config.Name = name
	}

	// HACK: we need to convert types.Project into compose.Project. the easiest way
	// to do that AFAICT is to marshal the types.Project to YAML and then parse it
	// back into a compose.Project. this is ineffecient, but it works for now.
	// We should avoid marshalling to YAML only to parse it again. instead try to
	// cast types.Project to map[string]interface{} and then pass it to
	// LoadWithContext as ConfigFile.Config.
	content, err := input.Config.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal project: %w", err)
	}

	configDetails := composeTypes.ConfigDetails{
		ConfigFiles: []composeTypes.ConfigFile{
			{
				Content: content,
			},
		},
	}

	project, err := loader.LoadWithContext(ctx, configDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to load project data: %w", err)
	}

	return project, nil
}

func configureProviderCdImage(
	ctx context.Context,
	driver IDriver,
	projectName string,
	providerID client.ProviderID,
) error {
	resp, err := driver.GetFabricClient().CanIUse(ctx, &defangv1.CanIUseRequest{
		Project:  projectName,
		Provider: providerID.EnumValue(),
	})
	if err != nil {
		return fmt.Errorf("failed to get CanIUse: %w", err)
	}

	// Allow local override of the CD image
	cdImage := pkg.Getenv("DEFANG_CD_IMAGE", resp.GetCdImage())
	driver.GetProvider().SetCanIUseConfig(&defangv1.CanIUseResponse{
		CdImage: cdImage,
	})

	return nil
}

func deployProject(
	ctx context.Context,
	fabric client.FabricClient,
	provider client.Provider,
	project *compose.Project,
	preview bool,
) (*defangv1.DeployResponse, error) {
	var upload compose.UploadMode
	if preview {
		upload = compose.UploadModeIgnore
	} else {
		upload = compose.UploadModeDigest
	}
	config := infer.GetConfig[Config](ctx)
	deploymentMode := defangv1.DeploymentMode_value[config.DeploymentMode]
	deployTime := time.Now()
	deploy, _, err := cli.ComposeUp(ctx, project, fabric, provider, upload, command.Mode(deploymentMode).Value())
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}

	err = cli.WaitAndTail(ctx, project, fabric, provider, deploy, -1, deployTime, true)
	if err != nil {
		return nil, fmt.Errorf("failed to tail: %w", err)
	}

	return deploy, nil
}

func getProjectOutputs(
	ctx context.Context,
	providerClient client.Provider,
	name string,
	etag string,
) (*defangv1.ProjectUpdate, error) {
	getProjectUpdateMaxRetries := 10
	var projectUpdate *defangv1.ProjectUpdate
	var err error
	for range getProjectUpdateMaxRetries {
		projectUpdate, err = providerClient.GetProjectUpdate(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get project update: %w", err)
		}
		allMatch := true
		for _, si := range projectUpdate.GetServices() {
			if si.GetEtag() != etag {
				allMatch = false
			}
		}
		if allMatch {
			break
		}

		err = pkg.SleepWithContext(ctx, 1*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to sleep: %w", err)
		}
	}
	return projectUpdate, nil
}

func makeServices(projectUpdate *defangv1.ProjectUpdate) map[string]*ServiceInfo {
	services := make(map[string]*ServiceInfo, len(projectUpdate.GetServices()))
	for _, serviceInfo := range projectUpdate.GetServices() {
		services[serviceInfo.GetService().GetName()] = &ServiceInfo{
			Endpoints:   serviceInfo.GetEndpoints(),
			Project:     serviceInfo.GetProject(),
			Etag:        serviceInfo.GetEtag(),
			Status:      serviceInfo.GetStatus(),
			NatIps:      serviceInfo.GetNatIps(),
			LbIps:       serviceInfo.GetLbIps(),
			PrivateFqdn: serviceInfo.GetPrivateFqdn(),
			PublicFqdn:  serviceInfo.GetPublicFqdn(),
			CreatedAt:   serviceInfo.GetCreatedAt(),
			UpdatedAt:   serviceInfo.GetUpdatedAt(),
			ZoneId:      serviceInfo.GetZoneId(),
			UseAcmeCert: serviceInfo.GetUseAcmeCert(),
			Domainname:  serviceInfo.GetDomainname(),
			LbDnsName:   serviceInfo.GetLbDnsName(),
		}
	}
	return services
}

func Authenticate(ctx context.Context, driver IDriver) error {
	_, err := cli.Whoami(ctx, driver.GetFabricClient(), driver.GetProvider())
	if err == nil {
		return nil
	}

	err = cli.NonInteractiveLogin(ctx, driver.GetFabricClient(), cli.DefangFabric)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	return nil
}
