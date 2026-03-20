using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangGcp = DefangLabs.DefangGcp;

return await Deployment.RunAsync(() => 
{
    var gcpYaml = new DefangGcp.Defanggcp.Project("gcp-yaml", new()
    {
        Services = 
        {
            { "app", new DefangGcp.Shared.Inputs.ServiceInputArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangGcp.Shared.Inputs.ServicePortConfigArgs
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
        ["endpoints"] = gcpYaml.Endpoints,
    };
});

