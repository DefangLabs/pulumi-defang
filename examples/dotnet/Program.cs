using System.Collections.Generic;
using System.Linq;
using Pulumi;

return await Deployment.RunAsync(() => 
{
    return new Dictionary<string, object?>
    {
        ["output"] = 
        {
            { "albArn", myProject.AlbArn },
            { "etag", myProject.Etag },
        },
    };
});

