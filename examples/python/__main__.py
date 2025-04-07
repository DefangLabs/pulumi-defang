import pulumi

pulumi.export("output", {
    "albArn": my_project["albArn"],
    "etag": my_project["etag"],
})
