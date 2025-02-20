package main

import (
	"example.com/pulumi-defang/sdk/go/defang"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := defang.NewProject(ctx, "myProject", &defang.ProjectArgs{
			ProviderID: pulumi.String("aws"),
			Name:       pulumi.String("my-project"),
			ConfigPaths: pulumi.StringArray{
				pulumi.String("../../compose.yaml.example"),
			},
		})
		if err != nil {
			return err
		}
		ctx.Export("output", nil)
		return nil
	})
}
