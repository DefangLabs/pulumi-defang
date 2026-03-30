using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangAzure = DefangLabs.DefangAzure;

return await Deployment.RunAsync(() => 
{
    var azureDemo = new DefangAzure.Project("azure-demo", new()
    {
        Services = 
        {
            { "app", new DefangAzure.Compose.Inputs.ServiceConfigArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangAzure.Compose.Inputs.ServicePortConfigArgs
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
        ["endpoints"] = azureDemo.Endpoints,
    };
});

