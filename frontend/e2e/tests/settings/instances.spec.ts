import { test, expect, type Page } from '@playwright/test'

interface MockInstance {
  id: string
  name: string
  status: string
  is_default: boolean
  auto_read_receipt: boolean
  organization_id: string
  created_at: string
  updated_at: string
}

const HEALTH_PAYLOAD = {
  uptime_seconds: 0,
  messages_sent_today: 0,
  messages_received_today: 0,
  messages_failed_today: 0,
  error_rate_percent: 0,
  queue_depth: 0,
}

type MockHooks = {
  onCreate?: (payload: Record<string, unknown>) => void
  onUpdate?: (id: string, payload: Record<string, unknown>) => void
  onDelete?: (id: string) => void
  onConnect?: (id: string) => void
  onDisconnect?: (id: string) => void
  onReconnect?: (id: string) => void
  onPairPhone?: (id: string, payload: Record<string, unknown>) => void
}

async function loginAsSuperAdmin(page: Page) {
  await page.goto('/login')
  await page.locator('input[name="email"], input[type="email"]').fill('admin@admin.com')
  await page.locator('input[name="password"], input[type="password"]').fill('admin')
  await page.locator('button[type="submit"]').click()
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 10000 })
}

async function mockInstancesApi(page: Page, instances: MockInstance[], hooks: MockHooks = {}) {
  await page.route('**/api/instances**', async route => {
    const request = route.request()
    const method = request.method()
    const pathname = new URL(request.url()).pathname

    if (method === 'GET' && pathname.endsWith('/api/instances')) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: instances,
        }),
      })
      return
    }

    if (method === 'POST' && pathname.endsWith('/api/instances')) {
      const payload = (request.postDataJSON() || {}) as Record<string, unknown>
      hooks.onCreate?.(payload)

      const now = new Date().toISOString()
      const createdInstance: MockInstance = {
        id: `e2e-instance-created-${Date.now()}`,
        name: typeof payload.name === 'string' && payload.name.trim() ? payload.name : `E2E Instance ${instances.length + 1}`,
        status: 'disconnected',
        is_default: false,
        auto_read_receipt: true,
        organization_id: 'e2e-org-id',
        created_at: now,
        updated_at: now,
      }
      instances.push(createdInstance)

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: createdInstance,
        }),
      })
      return
    }

    const instanceRootMatch = pathname.match(/\/api\/instances\/([^/]+)$/)
    if (instanceRootMatch && method === 'PUT') {
      const instanceID = instanceRootMatch[1]
      const payload = (request.postDataJSON() || {}) as Record<string, unknown>
      hooks.onUpdate?.(instanceID, payload)

      const index = instances.findIndex(instance => instance.id === instanceID)
      const current = index >= 0 ? instances[index] : null
      const nextInstance: MockInstance = {
        ...(current || {
          id: instanceID,
          name: 'Unnamed Instance',
          status: 'disconnected',
          is_default: false,
          auto_read_receipt: true,
          organization_id: 'e2e-org-id',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }),
        ...(typeof payload.name === 'string' ? { name: payload.name } : {}),
        updated_at: new Date().toISOString(),
      }
      if (index >= 0) {
        instances[index] = nextInstance
      } else {
        instances.push(nextInstance)
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: nextInstance,
        }),
      })
      return
    }

    if (instanceRootMatch && method === 'DELETE') {
      const instanceID = instanceRootMatch[1]
      hooks.onDelete?.(instanceID)
      const index = instances.findIndex(instance => instance.id === instanceID)
      if (index >= 0) {
        instances.splice(index, 1)
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: null,
        }),
      })
      return
    }

    const actionMatch = pathname.match(/\/api\/instances\/([^/]+)\/(connect|disconnect|reconnect|pair-phone)$/)
    if (actionMatch && method === 'POST') {
      const [, instanceID, action] = actionMatch
      if (action === 'connect') {
        hooks.onConnect?.(instanceID)
      } else if (action === 'disconnect') {
        hooks.onDisconnect?.(instanceID)
      } else if (action === 'reconnect') {
        hooks.onReconnect?.(instanceID)
      } else if (action === 'pair-phone') {
        const payload = (request.postDataJSON() || {}) as Record<string, unknown>
        hooks.onPairPhone?.(instanceID, payload)
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: action === 'pair-phone'
            ? {
                pairing_code: '123-456',
                phone_number: '+15551234567',
                timeout_seconds: 30,
              }
            : {},
        }),
      })
      return
    }

    if (method === 'GET' && /\/api\/instances\/[^/]+\/health$/.test(pathname)) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'success',
          data: HEALTH_PAYLOAD,
        }),
      })
      return
    }

    await route.fallback()
  })
}

test.describe('WhatsApp Instances', () => {
  test('should create a new instance from Add Account dialog', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const instances: MockInstance[] = []
    const newInstanceName = `E2E Instance ${Date.now()}`
    let receivedCreatePayload: { name?: string } | null = null

    await mockInstancesApi(page, instances, {
      onCreate: payload => {
        receivedCreatePayload = payload as { name?: string }
      },
    })

    await page.goto('/settings/instances')
    await expect(page.getByRole('heading', { name: /WhatsApp Instances/i })).toBeVisible()

    await page.getByRole('button', { name: /Add Account/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Add WhatsApp Account')).toBeVisible()
    await dialog.locator('input#name').fill(newInstanceName)
    await dialog.getByRole('button', { name: 'Create' }).click()

    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Instance created successfully' })).toBeVisible()
    await expect(page.locator('h3').filter({ hasText: newInstanceName })).toBeVisible()
    expect(receivedCreatePayload).toEqual({ name: newInstanceName })
  })

  test('should edit an instance name from the instance card', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const now = new Date().toISOString()
    const originalInstance: MockInstance = {
      id: 'e2e-instance-edit-id',
      name: `Original Instance ${Date.now()}`,
      status: 'disconnected',
      is_default: false,
      auto_read_receipt: true,
      organization_id: 'e2e-org-id',
      created_at: now,
      updated_at: now,
    }
    const updatedInstanceName = `${originalInstance.name} Updated`
    const instances: MockInstance[] = [originalInstance]
    let receivedUpdatePayload: { name?: string } | null = null

    await mockInstancesApi(page, instances, {
      onUpdate: (_, payload) => {
        receivedUpdatePayload = payload as { name?: string }
      },
    })

    await page.goto('/settings/instances')
    await expect(page.locator('h3').filter({ hasText: originalInstance.name })).toBeVisible()

    await page.getByRole('button', { name: /Edit instance name/i }).first().click()

    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Edit Account Name')).toBeVisible()
    await dialog.locator('input#edit-name').fill(updatedInstanceName)
    await dialog.getByRole('button', { name: 'Update' }).click()

    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Instance updated successfully' })).toBeVisible()
    await expect(page.locator('h3').filter({ hasText: updatedInstanceName })).toBeVisible()
    expect(receivedUpdatePayload).toEqual({ name: updatedInstanceName })
  })

  test('should delete an instance from the instance card', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const now = new Date().toISOString()
    const deletingInstance: MockInstance = {
      id: 'e2e-instance-delete-id',
      name: `Delete Me ${Date.now()}`,
      status: 'disconnected',
      is_default: false,
      auto_read_receipt: true,
      organization_id: 'e2e-org-id',
      created_at: now,
      updated_at: now,
    }
    const instances: MockInstance[] = [deletingInstance]
    let deletedInstanceID = ''

    await mockInstancesApi(page, instances, {
      onDelete: id => {
        deletedInstanceID = id
      },
    })

    await page.goto('/settings/instances')
    await expect(page.locator('h3').filter({ hasText: deletingInstance.name })).toBeVisible()

    await page.getByRole('button', { name: /Delete instance/i }).first().click()

    const deleteDialog = page.getByRole('alertdialog')
    await expect(deleteDialog.getByText('Delete WhatsApp Account')).toBeVisible()
    await deleteDialog.getByRole('button', { name: 'Delete Account' }).click()

    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Instance deleted successfully' })).toBeVisible()
    await expect(page.locator('h3').filter({ hasText: deletingInstance.name })).toHaveCount(0)
    expect(deletedInstanceID).toBe(deletingInstance.id)
  })

  test('should initiate connect flow and request QR regeneration', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const now = new Date().toISOString()
    const instance: MockInstance = {
      id: 'e2e-instance-connect-id',
      name: `Connect Flow ${Date.now()}`,
      status: 'disconnected',
      is_default: false,
      auto_read_receipt: true,
      organization_id: 'e2e-org-id',
      created_at: now,
      updated_at: now,
    }
    const instances: MockInstance[] = [instance]
    const connectCalls: string[] = []
    const reconnectCalls: string[] = []

    await mockInstancesApi(page, instances, {
      onConnect: id => {
        connectCalls.push(id)
      },
      onReconnect: id => {
        reconnectCalls.push(id)
      },
    })

    await page.goto('/settings/instances')
    await expect(page.locator('h3').filter({ hasText: instance.name })).toBeVisible()

    await page.getByRole('button', { name: /Connect \/ Scan QR/i }).first().click()
    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Connection initiated. waiting for QR code...' })).toBeVisible()
    await expect(page.getByRole('dialog').getByText('Link WhatsApp Device')).toBeVisible()
    expect(connectCalls).toEqual([instance.id])

    await page.getByRole('button', { name: /Regenerate QR/i }).click()
    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Requesting a new QR code...' })).toBeVisible()
    expect(reconnectCalls).toEqual([instance.id])
  })

  test('should disconnect a connected instance from the instance card', async ({ page }) => {
    await loginAsSuperAdmin(page)

    const now = new Date().toISOString()
    const instance: MockInstance = {
      id: 'e2e-instance-disconnect-id',
      name: `Disconnect Flow ${Date.now()}`,
      status: 'connected',
      is_default: false,
      auto_read_receipt: true,
      organization_id: 'e2e-org-id',
      created_at: now,
      updated_at: now,
    }
    const instances: MockInstance[] = [instance]
    const disconnectCalls: string[] = []

    await mockInstancesApi(page, instances, {
      onDisconnect: id => {
        disconnectCalls.push(id)
      },
    })

    await page.goto('/settings/instances')
    await expect(page.locator('h3').filter({ hasText: instance.name })).toBeVisible()

    await page.getByRole('button', { name: /^Disconnect$/i }).first().click()

    await expect(page.locator('[data-sonner-toast]').filter({ hasText: 'Instance logged out' })).toBeVisible()
    await expect(page.getByRole('button', { name: /Connect \/ Scan QR/i }).first()).toBeVisible()
    expect(disconnectCalls).toEqual([instance.id])
  })
})
