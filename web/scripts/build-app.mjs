import { spawn } from "node:child_process";
import fs from "node:fs/promises";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, "..");
const repoRoot = path.resolve(webDir, "..");
const sidecarDir = path.join(webDir, "src-tauri", "bin");
const sidecarName = process.platform === "win32" ? "api.exe" : "api";
const sidecarPath = path.join(sidecarDir, sidecarName);

function run(command, args, cwd) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd,
      shell: true,
      stdio: "inherit",
      env: process.env,
    });

    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`${command} ${args.join(" ")} failed with exit code ${code ?? "unknown"}`));
    });
  });
}

await fs.mkdir(sidecarDir, { recursive: true });
await run("go", ["build", "-o", sidecarPath, "./cmd/api"], repoRoot);
await run("npm", ["run", "build"], webDir);
