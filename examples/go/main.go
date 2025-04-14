package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		myProject, err := defang.NewProject(ctx, "myProject", &defang.ProjectArgs{
			ConfigPaths: pulumi.StringArray{
				pulumi.String("../../compose.yaml.example"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("output", pulumi.Map{
			"albArn": myProject.AlbArn,
			"etag":   myProject.Etag,
			"services": pulumi.StringMapMap{
				"service1": pulumi.StringMap{
					"resource_name": myProject.Services.ApplyT(func(services map[string]defang.ServiceState) (*string, error) {
						return &services.Service1.Resource_name, nil
					}).(pulumi.StringPtrOutput),
					"task_role": myProject.Services.ApplyT(func(services map[string]defang.ServiceState) (*string, error) {
						return &services.Service1.Task_role, nil
					}).(pulumi.StringPtrOutput),
				},
			},
		})
		return nil
	})
}
