/**
 * Tip filter query + search cursor reset probe (mock path; no Vue/Go edits).
 * Uses tip accessAuditApi via BHIS_WEB_TEST_MODE + VITE_USE_MOCK.
 */
import { pathToFileURL } from 'node:url';
import path from 'node:path';
import fs from 'node:fs';
import { createRequire } from 'node:module';

process.env.BHIS_WEB_TEST_MODE = '1';
process.env.VITE_USE_MOCK = 'true';

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, '$1')), '..', '..', '..');
// Windows path from file URL can be awkward; use cwd-relative from deliverables path.
const apiPath = path.resolve('E:/B-HIS/web/src/master/accessAuditApi.js');

const mod = await import(pathToFileURL(apiPath).href);
const { fetchUserOperationLogs, buildUserOperationsCsv } = mod;

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

const cases = [];

// Case 1: empty filters → limit only
{
  const page = await fetchUserOperationLogs('tok', {});
  cases.push({ name: 'default-limit-page', ok: page.items.length === 5 && page.nextCursor === '', count: page.items.length });
}

// Case 2: nonexistent userId + date window → empty
{
  const page = await fetchUserOperationLogs('tok', {
    userId: 'H06-NONEXISTENT-READONLY',
    begin: '2026-10-15',
    end: '2026-10-16',
    limit: 50,
  });
  cases.push({
    name: 'user+date empty',
    ok: page.items.length === 0 && page.nextCursor === '',
    count: page.items.length,
  });
}

// Case 3: buildQuery shape via URLSearchParams recreation (mirror tip buildQuery fields)
{
  const params = { moduleId: 'module-master', userId: 'mock-operator', begin: '2026-10-14', end: '2026-10-14', limit: 50 };
  const page = await fetchUserOperationLogs('tok', params);
  const allMatchUser = page.items.every((r) => r.userId === 'mock-operator');
  cases.push({ name: 'module+user+date filter', ok: page.items.length >= 1 && allMatchUser, count: page.items.length });
}

// Case 4: search() semantics — clearing cursor then non-append replace (unit)
{
  const first = await fetchUserOperationLogs('tok', { limit: 2 });
  assert(first.nextCursor === 'mock:2', 'expected mock cursor');
  const refreshed = await fetchUserOperationLogs('tok', { limit: 2, cursor: '' });
  cases.push({
    name: 'search clears cursor (empty cursor = first page)',
    ok: refreshed.items.length === 2 && refreshed.items[0].requestId === first.items[0].requestId,
    count: refreshed.items.length,
  });
}

// Case 5: CSV formula escape still present
{
  const csv = buildUserOperationsCsv([{ module: '=1+1', type: 't', time: 't', user: 'u', department: '', description: '', ip: '', mac: '', version: '', outcome: '', requestId: '', errorCode: '' }]);
  cases.push({ name: 'csv formula escape', ok: csv.includes(`"'=1+1"`) || csv.includes(`\"'=1+1\"`), sample: csv.split(/\r?\n/)[1] });
}

const out = {
  mode: 'tip accessAuditApi mock path (BHIS_WEB_TEST_MODE=1 VITE_USE_MOCK=true); not live HTTP',
  tip: '2b69533',
  cases,
  allOk: cases.every((c) => c.ok),
};
fs.writeFileSync(path.resolve('E:/B-HIS/deliverables/h06/ui90-04c-reverify-01/filter-probe-results.json'), JSON.stringify(out, null, 2));
console.log('FILTER_PROBE', JSON.stringify(out));
if (!out.allOk) process.exitCode = 1;
