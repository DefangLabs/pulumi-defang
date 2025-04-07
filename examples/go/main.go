package main

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		ctx.Export("output", pulumi.AnyMap{
			"albArn": myProject.AlbArn,
			"etag":   myProject.Etag,
		})
		return nil
	})
}
