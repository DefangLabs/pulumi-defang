import pulumi
import pulumi_defang_aws as defang_aws

project = defang_aws.Project("aws-python",
    services={
        "app": defang_aws.shared.ServiceInputArgs(
            image="nginx",
            ports=[defang_aws.shared.PortConfigArgs(
                target=80,
                mode="ingress",
                app_protocol="http",
            )],
        ),
    },
)

pulumi.export("endpoints", project.endpoints)
