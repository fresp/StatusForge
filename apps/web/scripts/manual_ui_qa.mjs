import { chromium } from 'playwright'

const BASE_API = 'http://localhost:8080/api'
const BASE_WEB = 'http://localhost:3000'

async function apiRequest(path, { method = 'GET', token, body } = {}) {
  const headers = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const response = await fetch(`${BASE_API}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const text = await response.text()
  let data = null
  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = text
  }

  return { ok: response.ok, status: response.status, data }
}

async function getAdminToken() {
  const login = await apiRequest('/auth/login', {
    method: 'POST',
    body: { email: 'admin@statusplatform.com', password: 'admin123' },
  })

  if (!login.ok || !login.data?.token) {
    throw new Error(`Admin login failed: ${login.status} ${JSON.stringify(login.data)}`)
  }

  return {
    token: login.data.token,
    profile: login.data.admin,
  }
}

async function provisionOperator(adminToken) {
  const stamp = Date.now()
  const email = `operator.qa.${stamp}@example.com`
  const username = `operatorqa${stamp}`
  const password = 'OperatorQA!123'

  const invite = await apiRequest('/admins/invitations', {
    method: 'POST',
    token: adminToken,
    body: { email, role: 'operator' },
  })

  if (!invite.ok || !invite.data?.token) {
    throw new Error(`Operator invitation failed: ${invite.status} ${JSON.stringify(invite.data)}`)
  }

  const activate = await apiRequest('/admins/invitations/activate', {
    method: 'POST',
    body: { token: invite.data.token, username, password },
  })

  if (!activate.ok) {
    throw new Error(`Operator activation failed: ${activate.status} ${JSON.stringify(activate.data)}`)
  }

  const operatorLogin = await apiRequest('/auth/login', {
    method: 'POST',
    body: { email, password },
  })

  if (!operatorLogin.ok || !operatorLogin.data?.token) {
    throw new Error(`Operator login failed: ${operatorLogin.status} ${JSON.stringify(operatorLogin.data)}`)
  }

  return {
    email,
    password,
    token: operatorLogin.data.token,
    profile: operatorLogin.data.admin,
  }
}

async function verifyApiRbac(operatorToken, adminToken) {
  const operatorAdmins = await apiRequest('/admins', { token: operatorToken })
  const operatorIncidents = await apiRequest('/incidents', { token: operatorToken })
  const adminAdmins = await apiRequest('/admins', { token: adminToken })

  if (operatorAdmins.status !== 403) {
    throw new Error(`Expected operator /admins 403, got ${operatorAdmins.status}`)
  }
  if (operatorIncidents.status !== 200) {
    throw new Error(`Expected operator /incidents 200, got ${operatorIncidents.status}`)
  }
  if (adminAdmins.status !== 200) {
    throw new Error(`Expected admin /admins 200, got ${adminAdmins.status}`)
  }

  return {
    operatorAdminsStatus: operatorAdmins.status,
    operatorIncidentsStatus: operatorIncidents.status,
    adminAdminsStatus: adminAdmins.status,
  }
}

async function runUiChecks(adminAuth, operatorCreds) {
  const browser = await chromium.launch({ headless: true })

  try {
    const adminContext = await browser.newContext()
    const adminPage = await adminContext.newPage()

    await adminPage.goto(BASE_WEB, { waitUntil: 'networkidle' })
    await adminPage.evaluate((auth) => {
      localStorage.setItem('admin_token', auth.token)
      localStorage.setItem('admin_profile', JSON.stringify(auth.profile))
    }, adminAuth)

    await adminPage.goto(`${BASE_WEB}/admin/members`, { waitUntil: 'networkidle' })
    await adminPage.waitForTimeout(500)
    if (!adminPage.url().includes('/admin/members')) {
      throw new Error(`Expected admin members URL, got ${adminPage.url()}`)
    }
    await adminPage.waitForSelector('text=Manage admin accounts, roles, and status', { timeout: 15000 })
    await adminPage.waitForSelector('button:has-text("Members")', { timeout: 15000 })
    await adminPage.waitForSelector('button:has-text("Invitations")', { timeout: 15000 })
    await adminPage.screenshot({ path: '/tmp/admin-members-members-tab.png', fullPage: true })

    await adminPage.click('button:has-text("Invitations")')
    await adminPage.waitForSelector('h2:has-text("Invited Members")', { timeout: 15000 })
    await adminPage.screenshot({ path: '/tmp/admin-members-invitations-tab.png', fullPage: true })

    const operatorContext = await browser.newContext()
    const operatorPage = await operatorContext.newPage()

    await operatorPage.goto(BASE_WEB, { waitUntil: 'networkidle' })
    await operatorPage.evaluate((auth) => {
      localStorage.setItem('admin_token', auth.token)
      localStorage.setItem('admin_profile', JSON.stringify(auth.profile))
    }, operatorCreds)

    await operatorPage.goto(`${BASE_WEB}/admin/incidents`, { waitUntil: 'networkidle' })
    await operatorPage.waitForSelector('a:has-text("Incidents")', { timeout: 15000 })
    await operatorPage.waitForSelector('a:has-text("Maintenance")', { timeout: 15000 })

    const forbiddenLabels = ['Dashboard', 'Components', 'Sub-Components', 'Monitors', 'Subscribers', 'Members']
    for (const label of forbiddenLabels) {
      const match = operatorPage.locator(`a:has-text("${label}")`)
      if ((await match.count()) > 0) {
        throw new Error(`Operator sidebar should not include ${label}`)
      }
    }

    await operatorPage.screenshot({ path: '/tmp/operator-nav.png', fullPage: true })

    await operatorPage.goto(`${BASE_WEB}/admin/members`, { waitUntil: 'networkidle' })
    await operatorPage.waitForURL('**/admin/incidents', { timeout: 15000 })
    await operatorPage.screenshot({ path: '/tmp/operator-members-redirect.png', fullPage: true })

    await adminContext.close()
    await operatorContext.close()

    return {
      adminMembersScreenshot: '/tmp/admin-members-members-tab.png',
      adminInvitationsScreenshot: '/tmp/admin-members-invitations-tab.png',
      operatorNavScreenshot: '/tmp/operator-nav.png',
      operatorRedirectScreenshot: '/tmp/operator-members-redirect.png',
    }
  } finally {
    await browser.close()
  }
}

async function main() {
  const adminAuth = await getAdminToken()
  const operatorCreds = await provisionOperator(adminAuth.token)
  const apiEvidence = await verifyApiRbac(operatorCreds.token, adminAuth.token)
  const uiEvidence = await runUiChecks(adminAuth, operatorCreds)

  const result = {
    createdOperator: {
      email: operatorCreds.email,
      password: operatorCreds.password,
    },
    apiEvidence,
    uiEvidence,
    completedAt: new Date().toISOString(),
  }

  console.log(JSON.stringify(result, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
