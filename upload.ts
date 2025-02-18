import ignore from "@balena/dockerignore";
import { createReadStream, createWriteStream, promises, Stats } from "fs";
import { tmpdir } from "os";
import { join, normalize } from "path";
import { promises as stream } from "stream";
import * as tar from "tar";

const SOURCE_DATE_EPOCH = process.env["SOURCE_DATE_EPOCH"] ?? "315532800"; // defaults to 1980-01-01, same as nix-shell

// The default .dockerignore for projects that don't have one. Keep in sync with pkg/cli/context.go in CLI repo.
const defaultDockerIgnore = `# Default .dockerignore file for Defang
**/__pycache__
**/.direnv
**/.DS_Store
**/.envrc
**/.git
**/.github
**/.idea
**/.next
**/.vscode
**/compose.*.yaml
**/compose.*.yml
**/compose.yaml
**/compose.yml
**/docker-compose.*.yaml
**/docker-compose.*.yml
**/docker-compose.yaml
**/docker-compose.yml
**/node_modules
**/Thumbs.db
Dockerfile
*.Dockerfile
# Ignore our own binary, but only in the root to avoid ignoring subfolders
defang
defang.exe
# Ignore our project-level state
.defang`;

const extractMessageRegex = /<Message>(.*?)<\/Message>/;

async function tryReadIgnoreFile(
  cwd: string,
  ignorefile: string
): Promise<string | null> {
  try {
    const path = join(cwd, ignorefile);
    const patterns = await promises.readFile(path, "utf8");
    console.debug("Using", path);
    return patterns;
  } catch {
    return null;
  }
}

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
  let dockerignore = dockerfile + ".dockerignore";
  let patterns = await tryReadIgnoreFile(cwd, dockerignore);
  if (patterns === null) {
    dockerignore = ".dockerignore";
    patterns = await tryReadIgnoreFile(cwd, dockerignore);
    if (patterns === null) {
      console.debug("No .dockerignore file found; using defaults");
      patterns = defaultDockerIgnore;
    }
  }
  const filter = ignore({ ignorecase: false }).add(patterns).createFilter();

  const mtime = parseInt(SOURCE_DATE_EPOCH); // can be NaN
  const tempdir = await promises.mkdtemp(join(tmpdir(), "defang-build-"));
  console.debug(`Using temporary folder ${tempdir} for context ${cwd}`);
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
            if (!foundDockerfile && normalized === dockerfile) {
              return (foundDockerfile = true); // we need the Dockerfile, even if it's in the .dockerignore file
            }
            return normalized === dockerignore || filter(normalized); // we need the .dockerignore file too: it might ignore itself and/or the Dockerfile
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
