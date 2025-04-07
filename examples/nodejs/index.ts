import * as pulumi from "@pulumi/pulumi";

export const output = {
    albArn: myProject.albArn,
    etag: myProject.etag,
};
