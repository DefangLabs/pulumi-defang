import pulumi
import pulumi_defang as defang

my_project = defang.Project("myProject",
    provider_id="aws",
    config_paths=["../../compose.yaml.example"])
pulumi.export("output", {
    "albArn": my_project.alb_arn,
    "etag": my_project.etag,
})
