import { expect, request as playwrightRequest, test } from '@playwright/test'

const panelBaseURL = process.env.BAKARAY_PANEL_BASE_URL || 'http://localhost:8500'
const apiBaseURL = process.env.BAKARAY_API_BASE_URL || `${panelBaseURL}/api`
const wslHostIP = process.env.WSL_HOST_IP || '127.0.0.1'
const adminUsername = process.env.BAKARAY_ADMIN_USER || 'admin'
const adminPassword = process.env.BAKARAY_ADMIN_PASS || 'admin123'
const nodeSecret = process.env.BAKARAY_NODE_SECRET || 'e2e-node-secret'

const gostAdminPort = Number(process.env.BAKARAY_GOST_NODE_ADMIN_PORT || '18081')
const iptAdminPort = Number(process.env.BAKARAY_IPT_NODE_ADMIN_PORT || '28081')
const gostListenPort = Number(process.env.BAKARAY_GOST_LISTEN_PORT || '18080')
const iptListenPort = Number(process.env.BAKARAY_IPT_LISTEN_PORT || '28080')
const gostTargetPort = Number(process.env.BAKARAY_GOST_TARGET_PORT || '19081')
const iptTargetPort = Number(process.env.BAKARAY_IPT_TARGET_PORT || '29081')

type NodeSummary = {
  id: number
  name: string
  status: string
}

type UserSummary = {
  id: number
  username: string
}

type RuleOptions = {
  name: string
  protocolTitle: string
  nodeName: string
  listenPort: number
  targetHost: string
  targetPort: number
}

type Page = import('@playwright/test').Page
type APIRequestContext = import('@playwright/test').APIRequestContext
type BrowserContext = import('@playwright/test').BrowserContext

async function login(page: Page, path: string, usernameLabel: string, username: string, password: string, submitName: string) {
  await page.goto(path)
  await page.getByLabel(usernameLabel).fill(username)
  await page.getByRole('textbox', { name: /^密码$/ }).fill(password)
  await page.getByRole('button', { name: submitName }).click()
}

async function buildAuthApi(page: Page) {
  const token = await page.evaluate(() => localStorage.getItem('token'))
  expect(token).toBeTruthy()
  return playwrightRequest.newContext({
    baseURL: apiBaseURL.endsWith('/') ? apiBaseURL : `${apiBaseURL}/`,
    extraHTTPHeaders: {
      Authorization: `Bearer ${token}`
    }
  })
}

async function createNodeViaUI(page: Page, name: string, host: string, port: number) {
  await page.goto('/admin/nodes')
  await page.getByRole('button', { name: '添加节点' }).click()
  await page.getByLabel('节点名称').fill(name)
  await page.getByLabel('节点地址').fill(host)
  await page.getByLabel('管理端口').fill(String(port))
  await page.getByRole('textbox', { name: /^认证密钥$/ }).fill(nodeSecret)
  const responsePromise = page.waitForResponse((response) =>
    response.request().method() === 'POST' && response.url().includes('/api/admin/nodes')
  )
  await page.getByRole('button', { name: '保存' }).click()
  const response = await responsePromise
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  expect(body?.data?.id).toBeTruthy()
  return Number(body.data.id)
}

async function chooseSelect(page: Page, label: string, optionName: string) {
  await page.getByRole('combobox', { name: new RegExp(`^${label}$`) }).click({ force: true })
  await page.getByRole('option', { name: optionName, exact: true }).click()
}

async function createRuleViaUI(page: Page, opts: RuleOptions) {
  await page.goto('/rules')
  await page.getByRole('button', { name: '创建规则' }).click()
  await page.getByLabel('规则名称').fill(opts.name)
  await chooseSelect(page, '协议类型', opts.protocolTitle)
  await chooseSelect(page, '选择节点', opts.nodeName)
  await page.getByLabel('监听端口').fill(String(opts.listenPort))
  await page.getByLabel('目标地址').fill(opts.targetHost)
  await page.getByLabel('目标端口').fill(String(opts.targetPort))
  const responsePromise = page.waitForResponse((response) =>
    response.request().method() === 'POST' && response.url().includes('/api/rules')
  )
  await page.getByRole('button', { name: '保存' }).click()
  const response = await responsePromise
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  expect(body?.data?.id).toBeTruthy()
  return Number(body.data.id)
}

async function listAdminNodes(api: APIRequestContext) {
  const response = await api.get('admin/nodes?page=1&page_size=200')
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  return (body?.data?.list || []) as NodeSummary[]
}

async function listAdminUsers(api: APIRequestContext) {
  const response = await api.get('admin/users?page=1&page_size=200')
  expect(response.ok()).toBeTruthy()
  const body = await response.json()
  return (body?.data?.list || []) as UserSummary[]
}

async function waitForNodePresence(api: APIRequestContext, names: string[]) {
  await expect.poll(async () => {
    const nodes = await listAdminNodes(api)
    return names.every((name) => nodes.some((node) => node.name === name))
  }, { timeout: 30_000 }).toBeTruthy()
}

async function waitForNodesOnline(api: APIRequestContext, names: string[]) {
  await expect.poll(async () => {
    const nodes = await listAdminNodes(api)
    return nodes.filter((node) => names.includes(node.name) && node.status === 'online').length
  }, { timeout: 90_000 }).toBe(names.length)
}

async function waitForUserPresence(api: APIRequestContext, username: string) {
  await expect.poll(async () => {
    const users = await listAdminUsers(api)
    return users.find((user) => user.username === username)?.id ?? 0
  }, { timeout: 30_000 }).toBeGreaterThan(0)
}

async function fetchBody(request: APIRequestContext, url: string) {
  try {
    const response = await request.get(url, { timeout: 10_000 })
    if (!response.ok()) {
      return `status:${response.status()}`
    }
    return response.text()
  } catch (error) {
    return `error:${error instanceof Error ? error.message : String(error)}`
  }
}

async function bestEffortDelete(api: APIRequestContext | undefined, path: string, label: string) {
  if (!api) {
    return
  }

  try {
    const response = await api.delete(path)
    if (!response.ok() && response.status() !== 404) {
      console.warn(`cleanup failed for ${label}: ${response.status()}`)
    }
  } catch (error) {
    console.warn(`cleanup failed for ${label}: ${error instanceof Error ? error.message : String(error)}`)
  }
}

test('WSL zero-deploy panel + two native nodes forwarding E2E', async ({ browser, page, request }) => {
  test.setTimeout(5 * 60 * 1000)

  const suffix = Date.now().toString().slice(-8)
  const gostNodeName = `wsl-gost-${suffix}`
  const iptNodeName = `wsl-iptables-${suffix}`
  const gostRuleName = `gost-${suffix}`
  const iptRuleName = `iptables-${suffix}`
  const username = `e2e_${suffix}`
  const password = 'e2e_pass123'
  const nodeNames = [gostNodeName, iptNodeName]
  const createdNodeIds: number[] = []
  const createdRuleIds: number[] = []
  let adminAPI: APIRequestContext | undefined
  let userAPI: APIRequestContext | undefined
  let userContext: BrowserContext | undefined

  try {
    await login(page, '/admin/login', '管理员用户名', adminUsername, adminPassword, '管理后台登录')
    await expect(page).toHaveURL(/\/admin$/)

    adminAPI = await buildAuthApi(page)
    createdNodeIds.push(await createNodeViaUI(page, gostNodeName, wslHostIP, gostAdminPort))
    await waitForNodePresence(adminAPI, [gostNodeName])

    createdNodeIds.push(await createNodeViaUI(page, iptNodeName, wslHostIP, iptAdminPort))
    await waitForNodePresence(adminAPI, nodeNames)
    await waitForNodesOnline(adminAPI, nodeNames)

    userContext = await browser.newContext({ baseURL: panelBaseURL })
    const userPage = await userContext.newPage()

    await userPage.goto('/register')
    await userPage.getByLabel('用户名').fill(username)
    await userPage.getByRole('textbox', { name: /^密码$/ }).fill(password)
    await userPage.getByLabel('确认密码').fill(password)
    await userPage.getByRole('button', { name: '注册' }).click()
    await expect(userPage).toHaveURL(/\/login$/)

    await login(userPage, '/login', '用户名', username, password, '登录')
    await expect(userPage).toHaveURL(/\/$/)

    userAPI = await buildAuthApi(userPage)
    await waitForUserPresence(adminAPI, username)

    createdRuleIds.push(await createRuleViaUI(userPage, {
      name: gostRuleName,
      protocolTitle: 'Gost 用户态转发',
      nodeName: gostNodeName,
      listenPort: gostListenPort,
      targetHost: wslHostIP,
      targetPort: gostTargetPort
    }))

    createdRuleIds.push(await createRuleViaUI(userPage, {
      name: iptRuleName,
      protocolTitle: '内核转发 (iptables)',
      nodeName: iptNodeName,
      listenPort: iptListenPort,
      targetHost: wslHostIP,
      targetPort: iptTargetPort
    }))

    await expect.poll(async () => fetchBody(request, `http://${wslHostIP}:${gostListenPort}`), {
      timeout: 90_000,
      intervals: [2_000, 3_000, 5_000]
    }).toContain('backend-node-1')

    await expect.poll(async () => fetchBody(request, `http://${wslHostIP}:${iptListenPort}`), {
      timeout: 90_000,
      intervals: [2_000, 3_000, 5_000]
    }).toContain('backend-node-2')
  } finally {
    for (const ruleId of [...createdRuleIds].reverse()) {
      await bestEffortDelete(userAPI, `rules/${ruleId}`, `rule ${ruleId}`)
    }
    for (const nodeId of [...createdNodeIds].reverse()) {
      await bestEffortDelete(adminAPI, `admin/nodes/${nodeId}`, `node ${nodeId}`)
    }

    if (adminAPI) {
      const users = await listAdminUsers(adminAPI).catch(() => [] as UserSummary[])
      const createdUser = users.find((user) => user.username === username)
      if (createdUser) {
        await bestEffortDelete(adminAPI, `admin/users/${createdUser.id}`, `user ${username}`)
      }
    }

    await userAPI?.dispose()
    await adminAPI?.dispose()
    await userContext?.close()
  }
})
