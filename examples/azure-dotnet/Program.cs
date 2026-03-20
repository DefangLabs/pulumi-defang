using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangAzure = DefangLabs.DefangAzure;

return await Deployment.RunAsync(() => 
{
    var azureYaml = new DefangAzure.Defangazure.Project("azure-yaml", new()
    {
        Services = 
        {
            { "app", new DefangAzure.Shared.Inputs.ServiceInputArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangAzure.Shared.Inputs.PortConfigArgs
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
        ["endpoints"] = azureYaml.Endpoints,
    };
});

