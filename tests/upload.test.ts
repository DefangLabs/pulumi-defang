import { assert } from "chai";
import { rm, rmdir } from "fs/promises";
import "mocha";
import { dirname, resolve } from "path";
import * as tar from "tar";
import * as upload from "../upload";

describe("#createTarball", () => {
  const expected = [
    "./",
    "./Dockerfile.test",
    "./modules/",
    "./modules/defang/",
    "./modules/defang/defang",
  ];
  const tests = [
    {
      dockerfile: undefined,
      error: "the specified dockerfile could not be read: Dockerfile",
    },
    {
      dockerfile: "",
      error: "the specified dockerfile could not be read: Dockerfile",
    },
    {
      dockerfile: "blah",
      error: "the specified dockerfile could not be read: blah",
    },
    { dockerfile: "Dockerfile.test" },
    { dockerfile: "./Dockerfile.test" },
  ];

  for (const tt of tests) {
    it(`create tarball for "${tt.dockerfile}"`, async () => {
      let actual: string[] = [];
      try {
        const tgz = await upload.createTarball(
          resolve(__dirname, "./test1"),
          tt.dockerfile
        );
        await tar.list({ file: tgz, onentry: (e) => actual.push(e.path) });
        await rm(tgz);
        await rmdir(dirname(tgz));
      } catch (err) {
        assert(err instanceof Error);
        assert.equal(err.message, tt.error);
        return;
      }
      if (tt.error) {
        assert.fail(`Expected failure for "${tt.dockerfile}"`);
      } else {
        assert.deepEqual(actual, expected);
      }
    });
  }
});
