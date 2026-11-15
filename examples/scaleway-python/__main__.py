import pulumi
import pulumi_defang_scaleway as defang_scaleway

config = pulumi.Config()
postgres_password = config.require_secret("POSTGRES_PASSWORD")
database_url = config.require_secret("DATABASE_URL")
api_url = config.require("API_URL")
api_image = config.require("API_IMAGE")
web_image = config.require("WEB_IMAGE")
db = defang_scaleway.Postgres("db",
    project_name="scaleway-e2e",
    postgres={},
    environment={
        "POSTGRES_DB": "defang",
        "POSTGRES_USER": "defang",
        "POSTGRES_PASSWORD": "${POSTGRES_PASSWORD}",
    })
api = defang_scaleway.Service("api",
    project_name="scaleway-e2e-api",
    image=api_image,
    ports=[{
        "target": 8080,
        "mode": "ingress",
        "app_protocol": "http",
    }],
    environment={
        "DATABASE_URL": "${DATABASE_URL}",
    })
web = defang_scaleway.Service("web",
    project_name="scaleway-e2e-web",
    image=web_image,
    ports=[{
        "target": 8080,
        "mode": "ingress",
        "app_protocol": "http",
    }],
    environment={
        "API_URL": "${API_URL}",
    })
pulumi.export("dbEndpoint", db.endpoint)
pulumi.export("databaseUrl", db.connection_url)
pulumi.export("apiEndpoint", api.endpoint)
pulumi.export("webEndpoint", web.endpoint)
