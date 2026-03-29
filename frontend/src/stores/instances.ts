import { defineStore } from 'pinia'
import { ref } from 'vue'
import { instancesService } from '@/services/api'
import type { InstanceHealth, WhatsAppInstance } from '@/types/whatsmeow'
import { toast } from 'vue-sonner'
import { i18n } from '@/i18n'
import { unwrapResponse } from '@/lib/api-utils'

export const useInstancesStore = defineStore('instances', () => {
  const t = i18n.global.t
  const instances = ref<WhatsAppInstance[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const healthByInstance = ref<Record<string, InstanceHealth>>({})
  let healthPollTimer: number | null = null
  let healthPollInFlight = false

  async function fetchInstances() {
    loading.value = true
    error.value = null
    try {
      const response = await instancesService.list()
      const list = unwrapResponse<WhatsAppInstance[]>(response)
      instances.value = list.map(instance => ({
        ...instance,
        health: healthByInstance.value[instance.id]
      }))
    } catch (err: any) {
      const message = err.response?.data?.message || t('instances.toast.fetchFailed')
      error.value = message
      toast.error(message)
    } finally {
      loading.value = false
    }
  }

  async function fetchInstanceHealth(id: string) {
    try {
      const response = await instancesService.health(id)
      const health = unwrapResponse<InstanceHealth>(response)
      healthByInstance.value[id] = health
      const instance = instances.value.find(item => item.id === id)
      if (instance) {
        instance.health = health
      }
      return health
    } catch (err) {
      const nextHealth = { ...healthByInstance.value }
      delete nextHealth[id]
      healthByInstance.value = nextHealth

      const instance = instances.value.find(item => item.id === id)
      if (instance) {
        instance.health = undefined
      }

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
      const payload = unwrapResponse<WhatsAppInstance>(response)
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
      const message = err.response?.data?.message || t('instances.toast.fetchOneFailed')
      error.value = message
      return null
    }
  }

  async function runHealthPollingTick(refreshInstances = false) {
    if (healthPollInFlight) {
      return
    }

    healthPollInFlight = true
    try {
      if (refreshInstances) {
        await fetchInstances()
      }
      await fetchAllHealth()
    } finally {
      healthPollInFlight = false
    }
  }

  function startHealthPolling(intervalMs = 30000, options: { refreshInstances?: boolean } = {}) {
    stopHealthPolling()
    const refreshInstances = options.refreshInstances === true
    healthPollTimer = window.setInterval(() => {
      void runHealthPollingTick(refreshInstances)
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
      toast.success(t('instances.toast.createSuccess'))
      return newInstance
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.createFailed')
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
      toast.success(t('instances.toast.updateSuccess'))
      return updatedInstance
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.updateFailed')
      toast.error(msg)
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteInstance(id: string, options?: { deleteChats?: boolean }) {
    loading.value = true
    try {
      await instancesService.delete(id, options)
      instances.value = instances.value.filter(instance => instance.id !== id)
      delete healthByInstance.value[id]
      toast.success(t('instances.toast.deleteSuccess'))
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.deleteFailed')
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
      toast.info(t('instances.toast.connectInitiated'))
      return true
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.connectFailed')
      toast.error(msg)
      return false
    }
  }

  async function disconnectInstance(id: string) {
    loading.value = true
    try {
      await instancesService.disconnect(id)
      toast.success(t('instances.toast.disconnectSuccess'))
      const index = instances.value.findIndex(instance => instance.id === id)
      if (index !== -1) {
        instances.value[index].status = 'logged_out'
      }
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.disconnectFailed')
      toast.error(msg)
    } finally {
      loading.value = false
    }
  }

  async function reconnectInstance(id: string) {
    try {
      await instancesService.reconnect(id)
      updateInstanceStatus(id, 'connecting')
      toast.info(t('instances.toast.reconnectRequested'))
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.reconnectFailed')
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
      toast.success(t('instances.toast.pairingCodeGenerated'))
      return payload
    } catch (err: any) {
      const msg = err.response?.data?.message || t('instances.toast.pairingCodeFailed')
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
