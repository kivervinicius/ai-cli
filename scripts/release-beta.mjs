import { readFile, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";

const versionPath = new URL("../VERSION", import.meta.url);
const current = (await readFile(versionPath, "utf8")).trim();
const match = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+)\.(\d+))?$/.exec(current);

if (!match) {
  throw new Error(`VERSION inválida: ${current}`);
}

const [, major, minor, patch, pre, preNumber] = match;
const next = pre === "beta"
  ? `${major}.${minor}.${patch}-beta.${Number(preNumber) + 1}`
  : `${major}.${Number(minor) + 1}.0-beta.0`;

await writeFile(versionPath, `${next}\n`);
console.log(`Gerando versão beta ${next} (anterior: ${current})`);

const child = spawn("make", ["build"], { stdio: "inherit", cwd: new URL("..", import.meta.url) });
const exitCode = await new Promise((resolve) => {
  child.on("close", (code) => resolve(code ?? 1));
  child.on("error", () => resolve(1));
});

if (exitCode !== 0) {
  await writeFile(versionPath, `${current}\n`);
  console.error(`Build falhou; VERSION restaurada para ${current}`);
  process.exit(exitCode);
}

console.log(`Versão beta ${next} compilada e instalada.`);
