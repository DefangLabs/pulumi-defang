import ignore from "@balena/dockerignore";
import { createReadStream, createWriteStream, promises, Stats } from "fs";
import { tmpdir } from "os";
import { join, normalize } from "path";
import { promises as stream } from "stream";
import * as tar from "tar";

const SOURCE_DATE_EPOCH = process.env["SOURCE_DATE_EPOCH"] ?? "315532800"; // defaults to 1980-01-01, same as nix-shell
const defaultDockerIgnore = `# Default .dockerignore file for Defang
**/.DS_Store
**/.direnv
**/.envrc
**/.git
**/.github
**/.idea
**/.vscode
**/__pycache__
**/compose.yaml
**/compose.yml
**/defang.exe
**/docker-compose.yaml
**/docker-compose.yml
**/node_modules
**/Thumbs.db
# Ignore our own binary, but only in the root to avoid ignoring subfolders
defang`;

const extractMessageRegex = /<Message>(.*?)<\/Message>/;

export async function createTarball(
  cwd: string,
  dockerfile?: string
): Promise<string> {
  if (!dockerfile) {
    dockerfile = "Dockerfile";
  } else {
    dockerfile = normalize(dockerfile);
  }

  // A Dockerfile-specific ignore-file takes precedence over the .dockerignore file at the root of the build context if both exist.
  let patterns: string | undefined;
  try {
    const dockerignore = join(cwd, dockerfile + ".dockerignore");
    patterns = await promises.readFile(dockerignore, "utf8");
    console.debug("Using", dockerignore);
  } catch {
    try {
      const dockerignore = join(cwd, ".dockerignore");
      patterns = await promises.readFile(dockerignore, "utf8");
      console.debug("Using", dockerignore);
    } catch {
      console.debug("No .dockerignore file found; using defaults");
      patterns = defaultDockerIgnore;
    }
  }
  const filter = ignore({ ignorecase: false }).add(patterns).createFilter();

  const mtime = parseInt(SOURCE_DATE_EPOCH); // can be NaN
  const tempdir = await promises.mkdtemp(join(tmpdir(), "defang-build-"));
  console.debug(`Using temporary folder ${tempdir}`);
  const temppath = join(tempdir, "context.tar.gz");
  try {
    let foundDockerfile = false;

    // Using stream.pipeline() instead of .pipe() to correctly handle errors
    await stream.pipeline(
      tar.create(
        {
          cwd,
          filter: (p: string, stat: Stats) => {
            // Docker converts absolute source paths to relative paths (relative to the "build context") prior to pattern matching.
            const normalized = normalize(p);
            foundDockerfile ||= normalized === dockerfile;
            return filter(normalized);
          },
          gzip: true,
          mtime: isNaN(mtime) ? undefined : new Date(mtime * 1000), // seconds -> milliseconds
          portable: true,
          strict: true,
        } as tar.PackOptions, // https://github.com/DefinitelyTyped/DefinitelyTyped/issues/67775
        ["."]
      ),
      createWriteStream(temppath)
    );

    if (!foundDockerfile) {
      throw new Error(
        `the specified dockerfile could not be read: ${dockerfile}`
      );
    }
  } catch (err) {
    await promises.rm(temppath);
    await promises.rmdir(tempdir);
    throw err;
  }
  return temppath;
}

export async function uploadTarball(
  putUrl: string,
  temppath: string
): Promise<void> {
  const contentLength = (await promises.stat(temppath)).size;
  const fetch = (await import("node-fetch")).default; // ESM
  const res = await fetch(putUrl, {
    method: "PUT",
    headers: {
      "Content-Length": `${contentLength}`, // required by S3 presigned URLs
      "Content-Type": "application/gzip",
    },
    body: createReadStream(temppath),
  });
  if (!res.ok) {
    const errorBody = await res.text();
    // console.debug(errorBody); TODO: check debug flag
    const message = errorBody.match(extractMessageRegex)?.[1];
    throw new Error(
      `Failed to upload build context: ${message || res.statusText}`
    );
  }
}
