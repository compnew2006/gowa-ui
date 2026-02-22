import { defineStore } from 'pinia'
import { ref } from 'vue'
import { instancesService } from '@/services/api'
import type { InstanceHealth, WhatsAppInstance } from '@/types/whatsmeow'
import { toast } from 'vue-sonner'

export const useInstancesStore = defineStore('instances', () => {
  const instances = ref<WhatsAppInstance[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const healthByInstance = ref<Record<string, InstanceHealth>>({})
  let healthPollTimer: number | null = null

  async function fetchInstances() {
    loading.value = true
    error.value = null
    try {
      const response = await instancesService.list()
      const list = (response.data.data || response.data) as WhatsAppInstance[]
      instances.value = list.map(instance => ({
        ...instance,
        health: healthByInstance.value[instance.id]
      }))
    } catch (err: any) {
      const message = err.response?.data?.message || 'Failed to fetch instances'
      error.value = message
      toast.error(message)
    } finally {
      loading.value = false
    }
  }

  async function fetchInstanceHealth(id: string) {
    try {
      const response = await instancesService.health(id)
      const health = (response.data.data || response.data) as InstanceHealth
      healthByInstance.value[id] = health
      const instance = instances.value.find(item => item.id === id)
      if (instance) {
        instance.health = health
      }
      return health
    } catch (err) {
      return null
    }
  }

  async function fetchAllHealth() {
    if (instances.value.length === 0) {
      return
    }
    await Promise.all(instances.value.map(instance => fetchInstanceHealth(instance.id)))
  }

  async function fetchInstance(id: string) {
    try {
      const response = await instancesService.get(id)
      const payload = (response.data.data || response.data) as WhatsAppInstance
      const index = instances.value.findIndex(instance => instance.id === id)
      const existing = index !== -1 ? instances.value[index] : undefined
      const nextInstance: WhatsAppInstance = {
        ...existing,
        ...payload,
        health: healthByInstance.value[id] || existing?.health
      }

      if (index === -1) {
        instances.value.push(nextInstance)
      } else {
        instances.value[index] = nextInstance
      }

      return nextInstance
    } catch (err: any) {
      const message = err.response?.data?.message || 'Failed to fetch instance'
      error.value = message
      return null
    }
  }

  function startHealthPolling(intervalMs = 30000) {
    stopHealthPolling()
    healthPollTimer = window.setInterval(() => {
      fetchAllHealth()
    }, intervalMs)
  }

  function stopHealthPolling() {
    if (healthPollTimer !== null) {
      clearInterval(healthPollTimer)
      healthPollTimer = null
    }
  }

  async function createInstance(data: { name: string; is_default?: boolean; auto_read_receipt?: boolean }) {
    loading.value = true
    try {
      const response = await instancesService.create(data)
      const newInstance = (response.data.data || response.data) as WhatsAppInstance
      instances.value.push(newInstance)
      toast.success('Instance created successfully')
      return newInstance
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to create instance'
      toast.error(msg)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateInstance(id: string, data: Partial<WhatsAppInstance>) {
    loading.value = true
    try {
      const response = await instancesService.update(id, data)
      const updatedInstance = (response.data.data || response.data) as WhatsAppInstance
      const index = instances.value.findIndex(instance => instance.id === id)
      if (index !== -1) {
        const current = instances.value[index]
        instances.value[index] = {
          ...current,
          ...updatedInstance,
          ...data,
          settings: data.settings !== undefined ? data.settings : updatedInstance.settings || current.settings,
          health: healthByInstance.value[id]
        }
      }
      toast.success('Instance updated successfully')
      return updatedInstance
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to update instance'
      toast.error(msg)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteInstance(id: string) {
    loading.value = true
    try {
      await instancesService.delete(id)
      instances.value = instances.value.filter(instance => instance.id !== id)
      delete healthByInstance.value[id]
      toast.success('Instance deleted successfully')
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to delete instance'
      toast.error(msg)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function connectInstance(id: string) {
    try {
      await instancesService.connect(id)
      updateInstanceStatus(id, 'connecting')
      toast.info('Connection initiated. waiting for QR code...')
      return true
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to initiate connection'
      toast.error(msg)
      return false
    }
  }

  async function disconnectInstance(id: string) {
    loading.value = true
    try {
      await instancesService.disconnect(id)
      toast.success('Instance logged out')
      const index = instances.value.findIndex(instance => instance.id === id)
      if (index !== -1) {
        instances.value[index].status = 'logged_out'
      }
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to disconnect instance'
      toast.error(msg)
    } finally {
      loading.value = false
    }
  }

  async function reconnectInstance(id: string) {
    try {
      await instancesService.reconnect(id)
      toast.info('Requesting a new QR code...')
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to regenerate QR code'
      toast.error(msg)
      throw err
    }
  }

  async function requestPhonePairCode(id: string, phoneNumber: string) {
    try {
      const response = await instancesService.pairPhone(id, {
        phone_number: phoneNumber
      })
      const payload = response.data.data || response.data
      toast.success('Pairing code generated. Enter it in WhatsApp linked devices.')
      return payload
    } catch (err: any) {
      const msg = err.response?.data?.message || 'Failed to request pairing code'
      toast.error(msg)
      throw err
    }
  }

  function updateInstanceStatus(id: string, status: string, phoneNumber?: string) {
    const instance = instances.value.find(item => item.id === id)
    if (instance) {
      instance.status = status
      if (phoneNumber) {
        instance.phone_number = phoneNumber
      }
    }
  }

  return {
    instances,
    loading,
    error,
    healthByInstance,
    fetchInstances,
    fetchInstanceHealth,
    fetchInstance,
    fetchAllHealth,
    startHealthPolling,
    stopHealthPolling,
    createInstance,
    updateInstance,
    deleteInstance,
    connectInstance,
    requestPhonePairCode,
    reconnectInstance,
    disconnectInstance,
    updateInstanceStatus
  }
})
