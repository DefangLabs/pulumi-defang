using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangGcp = DefangLabs.DefangGcp;

return await Deployment.RunAsync(() => 
{
    var gcpDemo = new DefangGcp.Project("gcp-demo", new()
    {
        Services = 
        {
            { "app", new DefangGcp.Compose.Inputs.ServiceConfigArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangGcp.Compose.Inputs.ServicePortConfigArgs
                    {
                        Target = 80,
                        Mode = "ingress",
                        AppProtocol = "http",
                    },
                },
            } },
        },
    });

    return new Dictionary<string, object?>
    {
        ["endpoints"] = gcpDemo.Endpoints,
    };
});

