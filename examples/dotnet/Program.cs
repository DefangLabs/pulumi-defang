using System.Collections.Generic;
using System.Linq;
using Pulumi;
using Defang = DefangLabs.Defang;

return await Deployment.RunAsync(() => 
{
    var myProject = new Defang.Project("myProject", new()
    {
        ProviderID = "aws",
        ConfigPaths = new[]
        {
            "../../compose.yaml.example",
        },
    });

    return new Dictionary<string, object?>
    {
        ["output"] = 
        {
            { "albArn", myProject.AlbArn },
            { "etag", myProject.Etag },
        },
    };
});

