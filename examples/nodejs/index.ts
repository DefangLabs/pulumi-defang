import * as pulumi from "@pulumi/pulumi";
import * as defang from "@pulumi/defang";

const myRandomResource = new defang.Random("myRandomResource", {length: 24});
const myRandomComponent = new defang.RandomComponent("myRandomComponent", {length: 24});
export const output = {
    value: myRandomResource.result,
};
