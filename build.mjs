import { build } from 'esbuild';
import { cpSync, mkdirSync } from 'fs';

mkdirSync('static', { recursive: true });

await build({
  entryPoints: ['frontend/src/main.js'],
  bundle: true,
  minify: true,
  outfile: 'static/app.min.js',
  format: 'iife',
  target: 'es2020',
});

cpSync('frontend/index.html', 'static/index.html');
cpSync('favicon.svg', 'static/favicon.svg');

console.log('Frontend build complete → static/');
