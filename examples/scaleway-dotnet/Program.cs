using System.Collections.Generic;
using System.Linq;
using Pulumi;
using DefangScaleway = DefangLabs.DefangScaleway;

return await Deployment.RunAsync(() => 
{
    var config = new Config();
    var postgresPassword = config.RequireSecret("POSTGRES_PASSWORD");
    var databaseUrl = config.RequireSecret("DATABASE_URL");
    var apiUrl = config.Require("API_URL");
    var apiImage = config.Require("API_IMAGE");
    var webImage = config.Require("WEB_IMAGE");
    var db = new DefangScaleway.Postgres("db", new()
    {
        ProjectName = "scaleway-e2e",
        Postgres = new() { },
        Environment = 
        {
            { "POSTGRES_DB", "defang" },
            { "POSTGRES_USER", "defang" },
            { "POSTGRES_PASSWORD", "${POSTGRES_PASSWORD}" },
        },
    });

    var api = new DefangScaleway.Service("api", new()
    {
        ProjectName = "scaleway-e2e-api",
        Image = apiImage,
        Ports = new[]
        {
            new DefangScaleway.Compose.Inputs.ServicePortConfigArgs
            {
                Target = 8080,
                Mode = "ingress",
                AppProtocol = "http",
            },
        },
        Environment = 
        {
            { "DATABASE_URL", "${DATABASE_URL}" },
        },
    });

    var web = new DefangScaleway.Service("web", new()
    {
        ProjectName = "scaleway-e2e-web",
        Image = webImage,
        Ports = new[]
        {
            new DefangScaleway.Compose.Inputs.ServicePortConfigArgs
            {
                Target = 8080,
                Mode = "ingress",
                AppProtocol = "http",
            },
        },
        Environment = 
        {
            { "API_URL", "${API_URL}" },
        },
    });

    return new Dictionary<string, object?>
    {
        ["dbEndpoint"] = db.Endpoint,
        ["databaseUrl"] = db.ConnectionUrl,
        ["apiEndpoint"] = api.Endpoint,
        ["webEndpoint"] = web.Endpoint,
    };
});

