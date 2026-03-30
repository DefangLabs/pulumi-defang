import pulumi
import pulumi_defang_aws as defang_aws

aws_demo = defang_aws.Project("aws-demo", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", aws_demo.endpoints)
