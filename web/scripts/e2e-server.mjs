import { spawn } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

// Every suite gets an isolated database; never edit a developer's dashboard.
const directory = mkdtempSync(join(tmpdir(), "glance-e2e-"));
const binary =
  process.env.GLANCE_E2E_BINARY ||
  fileURLToPath(new URL("../../build/glance", import.meta.url));
const server = spawn(
  binary,
  [
    "dashboard",
    "--listen",
    "127.0.0.1:8091",
    "--db",
    join(directory, "dashboard.db"),
  ],
  {
    stdio: "inherit",
    env: {
      ...process.env,
      GLANCE_DASHBOARD_TOKEN: "",
      GLANCE_DASHBOARD_TOKEN_FILE: "",
      GLANCE_CONNECTION_KEY_FILE: "",
      GLANCE_DASHBOARD_PUBLIC_URL: "",
    },
  },
);
for (const signal of ["SIGINT", "SIGTERM"])
  process.on(signal, () => server.kill(signal));
server.on("error", (error) => {
  console.error(error);
  rmSync(directory, { recursive: true, force: true });
  process.exit(1);
});
server.on("exit", (code) => {
  rmSync(directory, { recursive: true, force: true });
  process.exit(code || 0);
});
