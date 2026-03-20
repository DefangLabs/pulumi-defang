import pulumi
import pulumi_defang_azure as defang_azure

azure_yaml = defang_azure.defangazure.Project("azure-yaml", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", azure_yaml.endpoints)
