package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		myProject, err := defang.NewProject(ctx, "myProject", &defang.ProjectArgs{
			ProviderID: pulumi.String("aws"),
			ConfigPaths: pulumi.StringArray{
				pulumi.String("../../compose.yaml.example"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("output", pulumi.StringMap{
			"albArn": myProject.AlbArn,
			"etag":   myProject.Etag,
		})
		return nil
	})
}
