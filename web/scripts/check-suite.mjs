// Contract checks may exercise isolated mock helpers, but that mode is never
// enabled by a product build. Set it only for this test process.
process.env.BHIS_WEB_TEST_MODE = '1'
process.env.VITE_USE_MOCK = 'true'
const checks = [
  './check-outpatient-redesign.mjs', './check-outpatient-context.mjs', './check-mock.mjs', './check-workbench.mjs',
  './check-refund-workspace.mjs', './check-access-control.mjs', './check-user-operations-log.mjs',
  './check-client-http.mjs', './check-patients.mjs', './check-register.mjs', './check-orders.mjs',
  './check-charges.mjs', './check-dispense.mjs', './check-master.mjs', './check-usability-auth.mjs',
  './check-usability-flow.mjs', './check-usability-clinical.mjs', './check-usability-clinical-api.mjs',
  './check-usability-master.mjs',
]
for (const check of checks) {
  // A few legacy contract scripts perform mock calls before their own env
  // setup. Seed the isolated test process; real-API sections override it.
  process.env.VITE_USE_MOCK = 'true'
  await import(check)
}
