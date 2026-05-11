import pulumi
import pulumi_defang_scaleway as defang_scaleway

scaleway_demo = defang_scaleway.Project("scaleway-demo", services={
    "app": {
        "image": "nginx",
        "ports": [{
            "target": 80,
            "mode": "ingress",
            "app_protocol": "http",
        }],
    },
})

pulumi.export("endpoints", scaleway_demo.endpoints)
