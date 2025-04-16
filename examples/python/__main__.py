import pulumi
import pulumi_defang as defang

my_project = defang.Project("myProject", config_paths=["../../compose.yaml.example"])
pulumi.export("output", {
    "albArn": my_project.alb_arn,
    "etag": my_project.etag,
    "services": {
        "service1": {
            "id": my_project.services["service1"],
            "task_role": my_project.services["service1"],
        },
    },
})
