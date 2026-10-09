import { randomUUID } from "node:crypto";
import { mkdir, open, rm } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));

// setInputFiles only takes files inside the project root and refuses hidden
// paths such as `.e2e`, so generated uploads live in a visible, git-ignored
// folder of their own.
const uploadsRoot = path.join(repositoryRoot, "tests", "generated-uploads");

export interface GeneratedUploads {
  /**
   * Creates `name` with exactly `bytes` bytes and returns its project-relative
   * path. The file is sparse zeros, so a 100 MB upload costs no disk.
   */
  create(name: string, bytes: number): Promise<string>;
  /** Deletes everything this instance created. */
  remove(): Promise<void>;
}

export function generatedUploads(): GeneratedUploads {
  const directory = path.join(uploadsRoot, randomUUID());

  return {
    async create(name, bytes) {
      if (path.basename(name) !== name || name.startsWith(".")) {
        throw new Error("Generated upload names must be plain, visible file names.");
      }
      await mkdir(directory, { recursive: true });
      const file = path.join(directory, name);
      const handle = await open(file, "w");
      try {
        await handle.truncate(bytes);
      } finally {
        await handle.close();
      }
      return path.relative(repositoryRoot, file).split(path.sep).join("/");
    },
    async remove() {
      await rm(directory, { recursive: true, force: true });
    },
  };
}
