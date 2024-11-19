import "mocha";
import { newClient } from "../client";
import * as pb from "../protos/io/defang/v1/fabric_pb";

describe("#newClient", () => {
  it("connect to fabric-prod1.defang.dev", function (done) {
    const client = newClient("fabric-prod1.defang.dev", "");

    const trackRequest = new pb.TrackRequest();
    trackRequest.setAnonId("6046178C-AD7C-45E2-957A-5DA4BF6D2A8C"); // avoid segment error
    trackRequest.setEvent("Pulumi-Defang Test");
    client.track(trackRequest, (err) => (err ? done(err) : done()));
  });
});
