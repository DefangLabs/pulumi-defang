import pulumi
import pulumi_defang_aws as defang_aws

aws_yaml = defang_aws.Project("aws-yaml", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", aws_yaml.endpoints)
