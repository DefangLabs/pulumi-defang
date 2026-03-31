import pulumi
import pulumi_defang_gcp as defang_gcp

gcp_demo = defang_gcp.Project("gcp-demo", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", gcp_demo.endpoints)
