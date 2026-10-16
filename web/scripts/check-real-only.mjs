import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve(process.cwd())
const config = fs.readFileSync(path.join(root, 'src/api/config.js'), 'utf8')
const env = fs.readFileSync(path.join(root, '.env'), 'utf8')
if (!/export function useMock\(\)\s*\{[\s\S]*return false/.test(config)) {
  throw new Error('website runtime must force real HTTP API mode')
}
if (/VITE_USE_MOCK\s*=\s*true/.test(env)) {
  throw new Error('web/.env must not enable the memory-backed mock runtime')
}
console.log('website real-api mode checks passed')
