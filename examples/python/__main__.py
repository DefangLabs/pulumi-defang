import pulumi
import pulumi_defang as defang

my_project = defang.Project("myProject",
    provider_id="aws",
    name="my-project",
    config_paths=["../../compose.yaml.example"])
pulumi.export("output", None)
