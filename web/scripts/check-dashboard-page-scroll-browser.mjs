import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.BHIS_BROWSER_DRIVER || 'playwright')
const base = process.env.BHIS_UI_URL || 'http://127.0.0.1:8888'
const output = resolve(process.env.BHIS_UI_REPORT || '.codex/dashboard-page-scroll-browser')
await mkdir(output, { recursive: true })

const browser = await chromium.launch({ headless: true, ...(process.env.BHIS_BROWSER_EXECUTABLE ? { executablePath: process.env.BHIS_BROWSER_EXECUTABLE } : {}) })
const page = await browser.newPage({ viewport: { width: 1920, height: 954 } })
const errors = []
page.on('pageerror', (error) => errors.push(error.message))

try {
  await page.goto(`${base}/login?redirect=/app`)
  await page.locator('input[name="username"]').fill('system')
  await page.locator('input[name="password"]').fill('1')
  await page.getByRole('button', { name: '登录' }).click()
  await page.getByRole('heading', { name: /工作台/ }).waitFor()
  await page.locator('.ops').waitFor()
  await page.waitForTimeout(750)

  for (const viewport of [{ width: 1880, height: 954 }, { width: 1440, height: 900 }, { width: 1280, height: 720 }]) {
    await page.setViewportSize(viewport)
    const metrics = await page.evaluate(() => {
      const main = document.querySelector('.shell-main')
      const dashboard = document.querySelector('.dashboard')
      const ops = document.querySelector('.ops')
      const metricsArea = document.querySelector('.ops-metrics')
      return {
        viewport: { width: innerWidth, height: innerHeight },
        document: {
          clientHeight: document.documentElement.clientHeight,
          scrollHeight: document.documentElement.scrollHeight,
          scrollY,
        },
        main: main ? { clientHeight: main.clientHeight, scrollHeight: main.scrollHeight } : null,
        dashboard: dashboard ? { clientHeight: dashboard.clientHeight, scrollHeight: dashboard.scrollHeight } : null,
        sections: dashboard ? Array.from(dashboard.children, (element) => ({
          className: element.className,
          top: Math.round(element.getBoundingClientRect().top),
          bottom: Math.round(element.getBoundingClientRect().bottom),
          height: Math.round(element.getBoundingClientRect().height),
        })) : [],
        ops: ops ? { clientHeight: ops.clientHeight, scrollHeight: ops.scrollHeight } : null,
        metricsArea: metricsArea ? {
          clientHeight: metricsArea.clientHeight,
          scrollHeight: metricsArea.scrollHeight,
          overflowY: getComputedStyle(metricsArea).overflowY,
        } : null,
      }
    })

    await page.screenshot({ path: resolve(output, `home-dashboard-${viewport.width}x${viewport.height}.png`) })
    assert.ok(metrics.document.scrollHeight <= metrics.document.clientHeight + 1, `document scrolls: ${JSON.stringify(metrics)}`)
    assert.equal(metrics.document.scrollY, 0, 'home must not use document scrolling')
    assert.ok(metrics.main && metrics.main.scrollHeight <= metrics.main.clientHeight + 1, `workspace scrolls: ${JSON.stringify(metrics)}`)
    assert.ok(metrics.dashboard && metrics.dashboard.scrollHeight <= metrics.dashboard.clientHeight + 1, `dashboard content exceeds its viewport: ${JSON.stringify(metrics)}`)
    assert.ok(metrics.ops && metrics.ops.clientHeight > 0, `operations card is collapsed: ${JSON.stringify(metrics)}`)
    assert.ok(metrics.metricsArea && metrics.metricsArea.clientHeight >= 48, `metrics panel is too short: ${JSON.stringify(metrics)}`)
    assert.ok(metrics.metricsArea.scrollHeight > metrics.metricsArea.clientHeight, `overflow is not contained by the metrics panel: ${JSON.stringify(metrics)}`)
    assert.equal(metrics.metricsArea.overflowY, 'auto', 'metrics panel owns its vertical scroll')
    const metricsRegion = page.locator('.ops-metrics')
    await metricsRegion.hover()
    await page.mouse.wheel(0, 1000)
    await page.waitForTimeout(100)
    const metricsScroll = await metricsRegion.evaluate((element) => ({
      scrollTop: element.scrollTop,
      scrollHeight: element.scrollHeight,
      clientHeight: element.clientHeight,
      rect: element.getBoundingClientRect().toJSON(),
    }))
    assert.ok(metricsScroll.scrollTop > 0, `metrics cannot scroll at ${viewport.width}x${viewport.height}: ${JSON.stringify(metricsScroll)}`)
    assert.equal(await page.locator('.shell-main').evaluate((element) => element.scrollTop), 0, 'mouse wheel scrolling must remain inside the card')
    assert.equal(await page.evaluate(() => scrollY), 0, 'mouse wheel scrolling must not move the document')
    await page.screenshot({ path: resolve(output, `home-dashboard-${viewport.width}x${viewport.height}-card-scrolled.png`) })
    console.log(`PASS dashboard ${viewport.width}x${viewport.height} ${JSON.stringify(metrics)}`)
  }
  assert.deepEqual(errors, [])
  console.log('DASHBOARD_PAGE_SCROLL_BROWSER_OK')
} finally {
  await browser.close()
}
