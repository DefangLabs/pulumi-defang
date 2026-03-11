using System.Collections.Generic;
using Pulumi;
using Defang = DefangLabs.Defang;

return await Deployment.RunAsync(() =>
{
    var myProject = new Defang.Project("myProject", new()
    {
        ProviderId = "aws",
        Services =
        {
            ["web"] = new Defang.Inputs.ServiceInputArgs
            {
                Image = "nginx:latest",
                Ports =
                {
                    new Defang.Inputs.PortConfigArgs
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
        ["endpoints"] = myProject.Endpoints,
        ["loadBalancerDns"] = myProject.LoadBalancerDns,
    };
});
