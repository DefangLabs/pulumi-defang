package main

import (
	"github.com/DefangLabs/pulumi-defang/sdk/go/defang"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		proj, err := defang.NewProject(ctx, "myProject", &defang.ProjectArgs{
			ProviderId: "aws",
			Services: defang.ServiceInputMap{
				"web": defang.ServiceInputArgs{
					Image: pulumi.StringPtr("nginx:latest"),
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(80),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
				"app": defang.ServiceInputArgs{
					Build: defang.BuildInputArgs{
						Context: pulumi.String("s3://my-bucket/app-context.tar.gz"),
					},
					Ports: defang.PortConfigArray{
						defang.PortConfigArgs{
							Target:      pulumi.Int(8080),
							Mode:        pulumi.StringPtr("ingress"),
							AppProtocol: pulumi.StringPtr("http"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("endpoints", proj.Endpoints)
		ctx.Export("loadBalancerDns", proj.LoadBalancerDns)
		return nil
	})
}
