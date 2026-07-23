import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'
import { basicAuthHeader } from '@/lib/api-error'
import { normalizeBaseUrl, sameOriginBaseUrl } from '@/lib/url'

export type ConnectionStatus =
  | 'booting'
  | 'unconfigured'
  | 'connected'
  | 'unauthorized'
  | 'unreachable'

export type TestResult = 'ok' | 'unauthorized' | 'not-gowa' | 'unreachable'

export interface AppInfo {
  version: string
  os: string
  base_path?: string
  max_file_size: number
  max_video_size: number
  max_image_size: number
  chatwoot_enabled?: boolean
}

/**
 * Probe a server without the shared axios instance (no redirect side effects)
 */
async function probeServer(
  baseUrl: string,
  username?: string,
  password?: string
): Promise<TestResult> {
  try {
    const headers: Record<string, string> = { Accept: 'application/json' }
    const auth = basicAuthHeader(username, password)
    if (auth) headers['Authorization'] = auth

    const response = await axios.get(`${baseUrl}/devices`, {
      timeout: 8000,
      validateStatus: () => true,
      headers
    })

    if (response.status === 401) return 'unauthorized'

    const body = response.data
    if (
      response.status === 200 &&
      typeof body === 'object' &&
      body !== null &&
      'code' in body
    ) {
      return 'ok'
    }

    return 'not-gowa'
  } catch {
    return 'unreachable'
  }
}

/**
 * Fetch server info (/app/info)
 */
export async function fetchAppInfo(
  baseUrl: string,
  username?: string,
  password?: string
): Promise<AppInfo> {
  const headers: Record<string, string> = { Accept: 'application/json' }
  const auth = basicAuthHeader(username, password)
  if (auth) headers['Authorization'] = auth

  const response = await axios.get(`${baseUrl}/app/info`, {
    timeout: 8000,
    headers
  })
  const data = response.data
  if (data?.results) return data.results
  if (data?.data) return data.data
  return data
}

export const useConnectionStore = defineStore('connection', () => {
  const baseUrl = ref<string | null>(localStorage.getItem('gowa_base_url'))
  const username = ref<string | null>(localStorage.getItem('gowa_username'))
  const password = ref<string | null>(localStorage.getItem('gowa_password'))
  const status = ref<ConnectionStatus>('booting')

  function save() {
    if (baseUrl.value) localStorage.setItem('gowa_base_url', baseUrl.value)
    else localStorage.removeItem('gowa_base_url')

    if (username.value) localStorage.setItem('gowa_username', username.value)
    else localStorage.removeItem('gowa_username')

    if (password.value) localStorage.setItem('gowa_password', password.value)
    else localStorage.removeItem('gowa_password')
  }

  async function connect(
    rawUrl: string,
    userVal?: string,
    passVal?: string
  ): Promise<TestResult> {
    const normUrl = normalizeBaseUrl(rawUrl)
    const result = await probeServer(normUrl, userVal, passVal)
    if (result === 'ok') {
      baseUrl.value = normUrl
      username.value = userVal || null
      password.value = passVal || null
      status.value = 'connected'
      save()
    } else if (result === 'unauthorized') {
      status.value = 'unauthorized'
    } else {
      status.value = 'unreachable'
    }
    return result
  }

  async function boot(): Promise<void> {
    if (baseUrl.value) {
      const stored = await probeServer(
        baseUrl.value,
        username.value || undefined,
        password.value || undefined
      )
      if (stored === 'ok') {
        status.value = 'connected'
        return
      }
      status.value = stored === 'unauthorized' ? 'unauthorized' : 'unreachable'
      return
    }

    const origin = sameOriginBaseUrl()
    if ((await probeServer(origin)) === 'ok') {
      baseUrl.value = origin
      status.value = 'connected'
      save()
      return
    }

    status.value = 'unconfigured'
  }

  function disconnect() {
    baseUrl.value = null
    username.value = null
    password.value = null
    status.value = 'unconfigured'
    save()
  }

  return {
    baseUrl,
    username,
    password,
    status,
    connect,
    boot,
    disconnect
  }
})
