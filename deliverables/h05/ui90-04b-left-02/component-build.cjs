const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { pathToFileURL } = require('node:url');

(async () => {
  const root = path.resolve(__dirname, '../../..');
  const { build } = await import(pathToFileURL(path.join(root, 'web/node_modules/vite/dist/node/index.js')));
  const { default: vue } = await import(pathToFileURL(path.join(root, 'web/node_modules/@vitejs/plugin-vue/dist/index.mjs')));
  const outDir = fs.mkdtempSync(path.join(os.tmpdir(), 'uol-left01-component-build-'));
  const result = { task: 'UI90-04B-LEFT-01', mode: 'isolated actual Vue component production build, not full application or host', outDir, pass: false };
  try {
    const output = await build({ configFile: false, root, plugins: [vue()], resolve: { dedupe: ['vue'], alias: { vue: path.join(root, 'web/node_modules/vue/dist/vue.esm-bundler.js') } }, define: { 'import.meta.env.VITE_USE_MOCK': 'false', 'import.meta.env.VITE_API_BASE_URL': JSON.stringify('/api/v1') }, build: { outDir, rollupOptions: { input: path.join(__dirname, 'geometry-fixture.html') } } });
    result.pass = true;
    result.outputs = output.output.map(item => item.fileName);
  } catch (error) {
    result.error = error.message;
    process.exitCode = 1;
  }
  fs.writeFileSync(path.join(__dirname, 'component-build-results.json'), JSON.stringify(result, null, 2) + String.fromCharCode(10));
  console.log(JSON.stringify(result));
})().catch(error => { console.error(error); process.exitCode = 1; });
