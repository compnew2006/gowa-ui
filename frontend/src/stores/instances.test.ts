// @vitest-environment happy-dom

import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { WhatsAppInstance } from '@/types/whatsmeow'

const mocks = vi.hoisted(() => ({
  instancesService: {
    list: vi.fn(),
    get: vi.fn(),
    health: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    connect: vi.fn(),
    disconnect: vi.fn(),
    reconnect: vi.fn(),
    pairPhone: vi.fn(),
  },
  toast: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock('@/services/api', () => ({
  instancesService: mocks.instancesService,
}))

vi.mock('vue-sonner', () => ({
  toast: mocks.toast,
}))

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      t: (key: string) => key,
    },
  },
}))

import { useInstancesStore } from './instances'

function makeInstance(overrides: Partial<WhatsAppInstance> = {}): WhatsAppInstance {
  const now = new Date().toISOString()
  return {
    id: overrides.id ?? 'instance-1',
    name: overrides.name ?? 'Instance 1',
    status: overrides.status ?? 'disconnected',
    is_default: false,
    auto_read_receipt: true,
    organization_id: overrides.organization_id ?? 'org-1',
    settings: overrides.settings,
    health: overrides.health,
    created_at: now,
    updated_at: now,
    send_block_reason: overrides.send_block_reason,
    send_blocked_until: overrides.send_blocked_until,
    jid: overrides.jid,
    phone_number: overrides.phone_number,
  }
}

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useInstancesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('merges updates without dropping existing settings when payload has no settings', async () => {
    const store = useInstancesStore()
    store.instances = [
      makeInstance({
        id: 'instance-merge',
        settings: {
          auto_sync_history: true,
          custom_existing_setting: 'keep-me',
        },
      }),
    ]

    mocks.instancesService.update.mockResolvedValueOnce({
      data: {
        data: {
          id: 'instance-merge',
          name: 'Renamed',
          status: 'connected',
        },
      },
    })

    await store.updateInstance('instance-merge', { name: 'Renamed' })

    expect(store.instances[0].name).toBe('Renamed')
    expect(store.instances[0].status).toBe('connected')
    expect(store.instances[0].settings).toEqual({
      auto_sync_history: true,
      custom_existing_setting: 'keep-me',
    })
  })

  it('updates status transitions for connect, disconnect, reconnect', async () => {
    const store = useInstancesStore()
    store.instances = [makeInstance({ id: 'instance-state', status: 'disconnected' })]

    mocks.instancesService.connect.mockResolvedValueOnce({ data: { status: 'success' } })
    const connected = await store.connectInstance('instance-state')
    expect(connected).toBe(true)
    expect(store.instances[0].status).toBe('connecting')

    mocks.instancesService.disconnect.mockResolvedValueOnce({ data: { status: 'success' } })
    await store.disconnectInstance('instance-state')
    expect(store.instances[0].status).toBe('logged_out')

    mocks.instancesService.reconnect.mockResolvedValueOnce({ data: { status: 'success' } })
    await store.reconnectInstance('instance-state')
    expect(store.instances[0].status).toBe('connecting')
  })

  it('guards health polling when a previous tick is still in flight', async () => {
    vi.useFakeTimers()
    const store = useInstancesStore()

    const deferredList = createDeferred<any>()
    mocks.instancesService.list.mockReturnValue(deferredList.promise)

    store.startHealthPolling(10, { refreshInstances: true })

    await vi.advanceTimersByTimeAsync(10)
    await vi.advanceTimersByTimeAsync(10)

    expect(mocks.instancesService.list).toHaveBeenCalledTimes(1)

    deferredList.resolve({ data: { data: [makeInstance({ id: 'instance-poll' })] } })
    await Promise.resolve()
    store.stopHealthPolling()
  })

  it('sets error and emits toast when fetching instances fails', async () => {
    const store = useInstancesStore()

    mocks.instancesService.list.mockRejectedValueOnce({
      response: {
        data: {
          message: 'backend failed',
        },
      },
    })

    await store.fetchInstances()

    expect(store.error).toBe('backend failed')
    expect(mocks.toast.error).toHaveBeenCalledWith('backend failed')
  })

  it('returns false and shows toast when connect fails', async () => {
    const store = useInstancesStore()
    store.instances = [makeInstance({ id: 'instance-connect-fail', status: 'disconnected' })]

    mocks.instancesService.connect.mockRejectedValueOnce({
      response: {
        data: {
          message: 'cannot connect',
        },
      },
    })

    const ok = await store.connectInstance('instance-connect-fail')

    expect(ok).toBe(false)
    expect(mocks.toast.error).toHaveBeenCalledWith('cannot connect')
    expect(store.instances[0].status).toBe('disconnected')
  })
})
