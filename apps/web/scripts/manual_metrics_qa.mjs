import { chromium } from 'playwright'

const baseUrl = 'http://127.0.0.1:4173/status/payments'

const categoryPayload = {
  prefix: 'payments',
  name: 'Payments',
  description: 'Payments API',
  aggregateStatus: 'operational',
  uptime90d: 99.95,
  services: [
    {
      id: 'monitor-payments-api',
      name: 'Payments API',
      description: 'Primary payment endpoint',
      status: 'operational',
      uptime90d: 99.9,
      uptimeHistory: [],
    },
  ],
  incidents: [],
}

const settingsPayload = {
  head: { title: '', description: '', keywords: '', faviconUrl: '', metaTags: {} },
  branding: { siteName: 'StatusForge', logoUrl: '', backgroundImageUrl: '', heroImageUrl: '' },
  theme: { preset: 'default' },
  layout: { variant: 'classic' },
  footer: { text: '', showPoweredBy: false },
  customCss: '',
  updatedAt: new Date().toISOString(),
  createdAt: new Date().toISOString(),
}

const metricsPayload = {
  latency: { p90: 2739, p99: 6556 },
  availability: { last30Days: 98.7 },
  history: [
    {
      month: 'February 2026',
      latency: { p90: 2495, p99: 5547 },
      availability: 98.7,
    },
  ],
}

async function run() {
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage()

  await page.route('**/api/v1/status/category/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(categoryPayload),
    })
  })

  await page.route('**/api/status/settings', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(settingsPayload),
    })
  })

  await page.route('**/api/v1/monitors/*/metrics', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(metricsPayload),
    })
  })

  await page.goto(baseUrl, { waitUntil: 'networkidle' })

  const latencyTitle = await page.getByText('Latency').first().textContent()
  const summary = await page
    .locator('section:has-text("Latency") p')
    .first()
    .textContent()

  const detailsButton = page.getByRole('button', { name: 'Historical latency and availability' })
  const collapsedByDefault = (await page.getByText('February 2026').count()) === 0

  await detailsButton.click()
  await page.getByText('February 2026').waitFor({ state: 'visible' })

  const rowText = await page.locator('tbody tr').first().textContent()
  const indicatorClass = await page.locator('tbody tr .rounded-full').first().getAttribute('class')

  console.log('[QA] Latency section title:', latencyTitle?.trim())
  console.log('[QA] Summary text:', summary?.replace(/\s+/g, ' ').trim())
  console.log('[QA] Collapsed by default:', collapsedByDefault)
  console.log('[QA] Historical row:', rowText?.replace(/\s+/g, ' ').trim())
  console.log('[QA] Availability indicator classes:', indicatorClass)

  await browser.close()
}

run().catch((error) => {
  console.error(error)
  process.exit(1)
})
