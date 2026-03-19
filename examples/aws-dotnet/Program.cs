using System.Collections.Generic;
using Pulumi;
using DefangLabs.DefangAws;
using DefangLabs.DefangAws.Shared.Inputs;

return await Deployment.RunAsync(() =>
{
    var project = new Project("aws-dotnet", new ProjectArgs
    {
        Services =
        {
            ["app"] = new ServiceInputArgs
            {
                Image = "nginx",
                Ports =
                {
                    new PortConfigArgs { Target = 80, Mode = "ingress", AppProtocol = "http" },
                },
            },
        },
    });

    return new Dictionary<string, object?>
    {
        ["endpoints"] = project.Endpoints,
    };
});
