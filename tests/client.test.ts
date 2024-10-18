import "mocha";
import { newClient } from "../client";
import * as pb from "../protos/io/defang/v1/fabric_pb";

describe("#newClient", () => {
  it("connect to fabric-prod1.defang.dev", function (done) {
    const client = newClient("fabric-prod1.defang.dev", "");

    const trackRequest = new pb.TrackRequest();
    trackRequest.setEvent("Pulumi-Defang Test");
    client.track(trackRequest, (err) => (err ? done(err) : done()));
  });
});
