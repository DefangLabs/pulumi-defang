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
	"github.com/DefangLabs/defang/src/pkg/types"
	defangv1 "github.com/DefangLabs/defang/src/protos/io/defang/v1"
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

// Each resource has an input struct, defining what arguments it accepts.
type ProjectArgs struct {
	// Fields projected into Pulumi must be public and hava a `pulumi:"..."` tag.
	// The pulumi tag doesn't need to match the field name, but it's generally a
	// good idea.
	ProviderID  client.ProviderID `pulumi:"providerID"`
	Name        string            `pulumi:"name"`
	ConfigPaths []string          `pulumi:"configPaths"`
}

// Each resource has a state, describing the fields that exist on the created resource.
type ProjectState struct {
	// It is generally a good idea to embed args in outputs, but it isn't strictly necessary.
	ProjectArgs
	Etag     types.ETag              `pulumi:"etag"`
	AlbArn   string                  `pulumi:"albArn"`
	Services []*defangv1.ServiceInfo `pulumi:"services"`
}

var errNoProjectUpdate = errors.New("no project update found")

// All resources must implement Create at a minimum.
func (Project) Create(ctx context.Context, name string, input ProjectArgs, preview bool) (string, ProjectState, error) {
	state := ProjectState{ProjectArgs: input}
	if preview {
		return name, state, nil
	}

	loader := compose.NewLoader(compose.WithProjectName(input.Name), compose.WithPath(input.ConfigPaths...))
	project, err := loader.LoadProject(ctx)
	if err != nil {
		return name, state, fmt.Errorf("failed to load project: %w", err)
	}

	resp, err := fabricClient.CanIUse(ctx, &defangv1.CanIUseRequest{
		Project:  input.Name,
		Provider: input.ProviderID.EnumValue(),
	})
	if err != nil {
		return name, state, fmt.Errorf("failed to get CanIUse: %w", err)
	}

	// Allow local override of the CD image
	cdImage := pkg.Getenv("DEFANG_CD_IMAGE", resp.GetCdImage())
	providerClient.SetCDImage(cdImage)

	deploy, err := deployProject(ctx, cdImage, project)
	if err != nil {
		return name, state, fmt.Errorf("failed to deploy project: %w", err)
	}

	etag := deploy.GetEtag()

	projectUpdate, err := getProjectOutputs(ctx, providerClient, project.Name, etag)
	if err != nil {
		return name, state, fmt.Errorf("failed to get project outputs: %w", err)
	}

	if projectUpdate == nil {
		return name, state, errNoProjectUpdate
	}

	state.Etag = etag
	state.AlbArn = projectUpdate.GetAlbArn()
	state.Services = projectUpdate.GetServices()

	return name, state, nil
}

func deployProject(ctx context.Context, cdImage string, project *compose.Project) (*defangv1.DeployResponse, error) {
	upload := compose.UploadModeDigest
	mode := command.Mode(defangv1.DeploymentMode_DEVELOPMENT)
	deployTime := time.Now()
	deploy, _, err := cli.ComposeUp(ctx, project, fabricClient, providerClient, upload, mode.Value())
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %w", err)
	}

	err = cli.WaitAndTail(ctx, project, fabricClient, providerClient, deploy, 60*time.Minute, deployTime, true)
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
