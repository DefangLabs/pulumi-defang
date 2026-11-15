import * as pulumi from "@pulumi/pulumi";
import * as defang_scaleway from "@defang-io/pulumi-defang-scaleway";

const config = new pulumi.Config();
const postgresPassword = config.requireSecret("POSTGRES_PASSWORD");
const databaseUrl = config.requireSecret("DATABASE_URL");
const apiUrl = config.require("API_URL");
const apiImage = config.require("API_IMAGE");
const webImage = config.require("WEB_IMAGE");
const db = new defang_scaleway.Postgres("db", {
    projectName: "scaleway-e2e",
    postgres: {},
    environment: {
        POSTGRES_DB: "defang",
        POSTGRES_USER: "defang",
        POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}",
    },
});
const api = new defang_scaleway.Service("api", {
    projectName: "scaleway-e2e-api",
    image: apiImage,
    ports: [{
        target: 8080,
        mode: "ingress",
        appProtocol: "http",
    }],
    environment: {
        DATABASE_URL: "${DATABASE_URL}",
    },
});
const web = new defang_scaleway.Service("web", {
    projectName: "scaleway-e2e-web",
    image: webImage,
    ports: [{
        target: 8080,
        mode: "ingress",
        appProtocol: "http",
    }],
    environment: {
        API_URL: "${API_URL}",
    },
});
export const dbEndpoint = db.endpoint;
export const databaseUrl0 = db.connectionUrl;
export const apiEndpoint = api.endpoint;
export const webEndpoint = web.endpoint;
