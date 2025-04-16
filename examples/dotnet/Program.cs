using System.Collections.Generic;
using System.Linq;
using Pulumi;
using Defang = DefangLabs.Defang;

return await Deployment.RunAsync(() => 
{
    var myProject = new Defang.Project("myProject", new()
    {
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
            { "services", 
            {
                { "service1", 
                {
                    { "id", myProject.Services.Apply(services => services.Service1.Id) },
                    { "task_role", myProject.Services.Apply(services => services.Service1.Task_role) },
                } },
            } },
        },
    };
});

