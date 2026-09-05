import { build } from 'esbuild';
import { cp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { existsSync, readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import * as sass from 'sass';

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = resolve(webDir, '..');
const distDir = resolve(webDir, 'dist');
const embeddedDir = resolve(repoDir, 'internal/control/web/dist');

await rm(distDir, { recursive: true, force: true });
await mkdir(distDir, { recursive: true });

const tailwindCli = resolve(webDir, 'node_modules/@tailwindcss/cli/dist/index.mjs');
const css = spawnSync(process.execPath, [tailwindCli, '-i', 'src/index.css', '-o', 'dist/bundle.css', '--minify'], { cwd: webDir, stdio: 'inherit' });
if (css.status !== 0) process.exit(css.status ?? 1);

const collectedStyles = [];

const sassPlugin = {
  name: 'sass-modules',
  setup(buildInstance) {
    // Global SCSS
    buildInstance.onLoad({ filter: /\.scss$/ }, (args) => {
      if (args.path.includes('.module.')) return;
      const result = sass.compile(args.path);
      collectedStyles.push({ path: args.path, css: result.css });
      return {
        contents: '',
        loader: 'js',
      };
    });

    // SCSS Modules and CSS Modules
    buildInstance.onLoad({ filter: /\.module\.(scss|css)$/ }, (args) => {
      const isScss = args.path.endsWith('.scss');
      const compiledCss = isScss ? sass.compile(args.path).css : readFileSync(args.path, 'utf8');
      const fileHash = createHash('md5').update(args.path).digest('hex').slice(0, 6);

      const classMap = {};
      const transformedCss = compiledCss.replace(/\.([a-zA-Z_][a-zA-Z0-9_-]*)/g, (match, className) => {
        const scoped = `${className}_${fileHash}`;
        classMap[className] = scoped;
        return `.${scoped}`;
      });

      collectedStyles.push({ path: args.path, css: transformedCss });

      return {
        contents: `export default ${JSON.stringify(classMap)};`,
        loader: 'js',
      };
    });
  },
};

await build({
  absWorkingDir: webDir,
  entryPoints: { bundle: 'src/index.tsx' },
  outdir: 'dist',
  chunkNames: 'chunks/[name]-[hash]',
  bundle: true,
  splitting: true,
  treeShaking: true,
  minify: true,
  format: 'esm',
  platform: 'browser',
  target: ['es2022'],
  jsx: 'automatic',
  plugins: [sassPlugin],
  define: { 'process.env.NODE_ENV': '"production"' },
  logLevel: 'info',
});

if (collectedStyles.length > 0) {
  const currentBundleCss = await readFile(resolve(distDir, 'bundle.css'), 'utf8');
  const stableStyles = collectedStyles
    .toSorted((left, right) => left.path.localeCompare(right.path))
    .map((entry) => entry.css)
    .join('\n');
  await writeFile(resolve(distDir, 'bundle.css'), currentBundleCss + '\n' + stableStyles);
}

await cp(resolve(webDir, 'index.html'), resolve(distDir, 'index.html'));
const publicDir = resolve(webDir, 'public');
if (existsSync(publicDir)) {
  await cp(publicDir, distDir, { recursive: true });
}
const logo = existsSync(resolve(webDir, 'public/logo.png')) ? resolve(webDir, 'public/logo.png') : resolve(repoDir, 'logo.png');
if (existsSync(logo)) await cp(logo, resolve(distDir, 'logo.png'));

const manifest = {
  files: ['index.html', 'bundle.css', 'bundle.js', ...(existsSync(resolve(distDir, 'logo.png')) ? ['logo.png'] : [])],
  builder: 'node+esbuild+tailwind+code-splitting',
};
await writeFile(resolve(distDir, 'build-manifest.json'), JSON.stringify(manifest, null, 2) + '\n');

await rm(embeddedDir, { recursive: true, force: true });
await mkdir(embeddedDir, { recursive: true });
await cp(distDir, embeddedDir, { recursive: true });

const index = await readFile(resolve(distDir, 'index.html'), 'utf8');
if (!index.includes('./bundle.js') || !index.includes('./bundle.css')) throw new Error('index.html is missing bundle references');
console.log(`Nexus web build complete: ${distDir}`);
