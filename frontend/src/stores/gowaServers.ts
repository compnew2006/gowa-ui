import { defineStore } from 'pinia'
import { ref } from 'vue'
import { gowaServersService, type GowaServer, type GowaDevice } from '@/services/api'

export interface CreateGowaServerData {
  name: string
  base_url: string
  username: string
  password: string
  webhook_url?: string
  is_active?: boolean
}

export const useGowaServersStore = defineStore('gowaServers', () => {
  const servers = ref<GowaServer[]>([])
  const currentServer = ref<GowaServer | null>(null)
  const devices = ref<GowaDevice[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchServers(): Promise<GowaServer[]> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.list()
      const data = (response.data as any).data || response.data
      servers.value = data.instances || []
      return servers.value
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch GOWA servers'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchServer(id: string): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.get(id)
      const data = (response.data as any).data || response.data
      currentServer.value = data.instance
      return data.instance
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createServer(data: CreateGowaServerData): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.create(data)
      const server = (response.data as any).data?.instance || (response.data as any).data
      servers.value.unshift(server)
      return server
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to create GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateServer(id: string, data: Partial<CreateGowaServerData>): Promise<GowaServer> {
    loading.value = true
    error.value = null
    try {
      const response = await gowaServersService.update(id, data)
      const server = (response.data as any).data?.instance || (response.data as any).data
      const index = servers.value.findIndex(s => s.id === id)
      if (index !== -1) servers.value[index] = server
      if (currentServer.value?.id === id) currentServer.value = server
      return server
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to update GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteServer(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await gowaServersService.delete(id)
      servers.value = servers.value.filter(s => s.id !== id)
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to delete GOWA server'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchDevices(serverId: string): Promise<GowaDevice[]> {
    try {
      const response = await gowaServersService.listDevices(serverId)
      const data = (response.data as any).data || response.data
      devices.value = data.devices || []
      return devices.value
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch devices'
      throw err
    }
  }

  return {
    servers,
    currentServer,
    devices,
    loading,
    error,
    fetchServers,
    fetchServer,
    createServer,
    updateServer,
    deleteServer,
    fetchDevices,
  }
})
