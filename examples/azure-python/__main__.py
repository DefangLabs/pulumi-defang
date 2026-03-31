import pulumi
import pulumi_defang_azure as defang_azure

azure_demo = defang_azure.Project("azure-demo", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", azure_demo.endpoints)
