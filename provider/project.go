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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DefangLabs/defang/src/cmd/cli/command"
	"github.com/DefangLabs/defang/src/pkg"
	"github.com/DefangLabs/defang/src/pkg/cli"
	"github.com/DefangLabs/defang/src/pkg/cli/client"
	"github.com/DefangLabs/defang/src/pkg/cli/compose"
	"github.com/DefangLabs/defang/src/pkg/logs"
	"github.com/DefangLabs/defang/src/pkg/term"
	defangTypes "github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
	"github.com/DefangLabs/pulumi-defang/provider/types"
	"github.com/compose-spec/compose-go/v2/loader"
	composeTypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/pulumi/pulumi-go-provider/infer"
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
	ConfigPaths []string       `pulumi:"configPaths,optional"`
	Config      *ProjectConfig `pulumi:"config,optional"`
}

type ServiceState struct {
	ID       string  `pulumi:"id,omitempty"`
	TaskRole *string `pulumi:"task_role"`
}

// Each resource has a state, describing the fields that exist on the created resource.
type ProjectState struct {
	// It is generally a good idea to embed args in outputs, but it isn't strictly necessary.
	ProjectArgs
	Etag     defangTypes.ETag         `pulumi:"etag"`
	AlbArn   string                   `pulumi:"albArn"`
	Services map[string]*ServiceState `pulumi:"services"`
}

var errNoProjectUpdate = errors.New("no project update found")

var errNilProjectOutputs = errors.New("project update outputs are nil")

// All resources must implement Create at a minimum.
func (Project) Create(ctx context.Context, name string, input ProjectArgs, preview bool) (string, ProjectState, error) {
	state := ProjectState{ProjectArgs: input}
	if preview {
		return name, state, nil
	}

	config := infer.GetConfig[Config](ctx)
	term.SetDebug(config.Debug)
	var providerID client.ProviderID
	err := providerID.Set(config.CloudProviderID)
	if err != nil {
		providerID = client.ProviderAuto
	}

	project, err := loadProject(ctx, name, input)
	if err != nil {
		return name, state, fmt.Errorf("failed to load project: %w", err)
	}

	driver, err := NewDriver(ctx, providerID)
	if err != nil {
		return name, state, fmt.Errorf("failed to create driver: %w", err)
	}

	err = Authenticate(ctx, driver)
	if err != nil {
		return name, state, fmt.Errorf("failed to authenticate: %w", err)
	}

	err = configureProviderCdImage(ctx, driver, name, providerID)
	if err != nil {
		return name, state, fmt.Errorf("failed to configure provider CD image: %w", err)
	}

	deploy, err := deployProject(ctx, driver.GetFabricClient(), driver.GetProvider(), project, preview)
	if err != nil {
		return name, state, fmt.Errorf("failed to deploy project: %w", err)
	}

	etag := deploy.GetEtag()
	projectUpdate, err := getProjectUpdate(ctx, driver.GetProvider(), project.Name, etag)
	if err != nil {
		return name, state, fmt.Errorf("failed to get projectUpdate: %w", err)
	}

	state, err = getProjectState(etag, projectUpdate)
	if err != nil {
		return name, state, fmt.Errorf("failed to get project state: %w", err)
	}

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

	err = cli.TailAndMonitor(ctx, project, provider, -1, cli.TailOptions{
		Deployment: deploy.GetEtag(),
		Since:      deployTime,
		Verbose:    true,
		LogType:    logs.LogTypeAll,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to tail: %w", err)
	}

	return deploy, nil
}

func getProjectUpdate(
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

type V1DefangProjectOutputs struct {
	Services map[string]V1DefangServiceOutputs `json:"services"`
}

type V1DefangServiceOutputs struct {
	ID       string  `json:"id,omitempty"        pulumi:"id"`
	TaskRole *string `json:"task_role,omitempty" pulumi:"task_role"`
}

func getProjectState(etag string, projectUpdate *defangv1.ProjectUpdate) (ProjectState, error) {
	state := ProjectState{}
	if projectUpdate == nil {
		return state, errNoProjectUpdate
	}

	state.Etag = etag
	state.AlbArn = projectUpdate.GetAlbArn()
	projectOutputs := projectUpdate.GetProjectOutputs()

	if projectOutputs == nil {
		return state, errNilProjectOutputs
	}

	decodedProjectOutputs, err := base64.StdEncoding.DecodeString(string(projectOutputs))
	if err != nil {
		return state, fmt.Errorf("failed to base64 decode project outputs: %w", err)
	}

	var v1DefangProjectOutputs V1DefangProjectOutputs
	err = json.Unmarshal(decodedProjectOutputs, &v1DefangProjectOutputs)
	if err != nil {
		return state, fmt.Errorf("failed to unmarshal project update outputs: %w", err)
	}

	services := make(map[string]*ServiceState, len(projectUpdate.GetServices()))
	for _, serviceOutputs := range v1DefangProjectOutputs.Services {
		services[serviceOutputs.ID] = &ServiceState{
			ID:       serviceOutputs.ID,
			TaskRole: serviceOutputs.TaskRole,
		}
	}
	state.Services = services

	return state, nil
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
