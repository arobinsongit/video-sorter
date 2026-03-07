import { build } from 'esbuild';
import { cpSync, mkdirSync } from 'fs';

const staticDir = 'cmd/media-sorter/static';
mkdirSync(staticDir, { recursive: true });

await build({
  entryPoints: ['frontend/src/main.js'],
  bundle: true,
  minify: true,
  outfile: `${staticDir}/app.min.js`,
  format: 'iife',
  target: 'es2020',
});

cpSync('frontend/index.html', `${staticDir}/index.html`);
cpSync('favicon.svg', `${staticDir}/favicon.svg`);

console.log(`Frontend build complete → ${staticDir}/`);
