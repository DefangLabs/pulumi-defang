using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangAws = DefangLabs.DefangAws;

return await Deployment.RunAsync(() => 
{
    var awsDemo = new DefangAws.Project("aws-demo", new()
    {
        Services = 
        {
            { "app", new DefangAws.Compose.Inputs.ServiceConfigArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangAws.Compose.Inputs.ServicePortConfigArgs
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
        ["endpoints"] = awsDemo.Endpoints,
    };
});

