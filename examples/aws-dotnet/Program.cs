using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangAws = DefangLabs.DefangAws;

return await Deployment.RunAsync(() => 
{
    var awsYaml = new DefangAws.Project("aws-yaml", new()
    {
        Services = 
        {
            { "app", new DefangAws.Shared.Inputs.ServiceInputArgs
            {
                Image = "nginx",
                Ports = new[]
                {
                    new DefangAws.Shared.Inputs.ServicePortConfigArgs
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
        ["endpoints"] = awsYaml.Endpoints,
    };
});

