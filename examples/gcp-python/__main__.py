import pulumi
import pulumi_defang_gcp as defang_gcp

gcp_yaml = defang_gcp.defanggcp.Project("gcp-yaml", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})
pulumi.export("endpoints", gcp_yaml.endpoints)
