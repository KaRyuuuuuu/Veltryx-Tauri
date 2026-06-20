import { spawn } from "node:child_process";
import process from "node:process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const webDir = path.resolve(__dirname, "..");
const repoRoot = path.resolve(webDir, "..");

const children = [];

function start(name, command, args, cwd) {
  const child = spawn(command, args, {
    cwd,
    shell: true,
    stdio: "inherit",
    env: process.env,
  });

  child.on("exit", (code) => {
    if (code && code !== 0) {
      console.error(`[${name}] exited with code ${code}`);
    }
  });

  children.push(child);
  return child;
}

function shutdown() {
  for (const child of children) {
    if (!child.killed) {
      child.kill();
    }
  }
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
process.on("exit", shutdown);

start("backend", "go", ["run", "./cmd/api"], repoRoot);
start("frontend", "npm", ["run", "dev", "--", "--strictPort"], webDir);

await new Promise(() => {});
