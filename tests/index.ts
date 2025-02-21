import { DefangService } from "@defang-io/pulumi-defang/lib";

const fromDir = new DefangService("test1", {
  build: {
    context: "./test1",
    dockerfile: "./Dockerfile.test",
    args: {
      DNS: `defang-test1.prod1.defang.dev`,
    },
  },
  healthcheck: {
    test: ["CMD", "true"],
    retries: 1,
    timeout: 3,
    interval: 5,
  },
  // networks: {
  //   "public": {},
  // },
  waitForSteadyState: true,
});

export const test1etag = fromDir.etag;

// const fromImage = new DefangService("test2", {
//   image: "nginx:latest",
//   ports: [{ target: 80, mode: "ingress" }],
// });

// export const test2url = fromImage.endpoints[0];
