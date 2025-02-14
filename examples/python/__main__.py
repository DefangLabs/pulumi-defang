import pulumi
import pulumi_defang as defang

my_random_resource = defang.Random("myRandomResource", length=24)
my_random_component = defang.RandomComponent("myRandomComponent", length=24)
pulumi.export("output", {
    "value": my_random_resource.result,
})
