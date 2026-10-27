using System.Collections.Generic;
using Pulumi;
using DefangLabs.DefangScaleway;
using DefangLabs.DefangScaleway.Compose.Inputs;

return await Deployment.RunAsync(() =>
{
    var scalewayDemo = new Project("scaleway-demo", new()
    {
        Services =
        {
            ["app"] = new ServiceConfigArgs
            {
                Image = "nginx",
                Ports =
                {
                    new ServicePortConfigArgs
                    {
                        Target = 80,
                        Mode = "ingress",
                        AppProtocol = "http",
                    },
                },
            },
        },
    });

    return new Dictionary<string, object?>
    {
        ["endpoints"] = scalewayDemo.Endpoints,
    };
});
