import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/services/api'
import type { User } from '@/types/auth'
import { unwrapResponse } from '@/lib/api-utils'

export interface AuthState {
  user: User | null
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const breakStartedAt = ref<string | null>(null)

  const isAuthenticated = computed(() => !!user.value)
  const userRole = computed(() => user.value?.role?.name || 'agent')
  const organizationId = computed(() => user.value?.organization_id || '')
  const userSettings = computed(() => user.value?.settings || {})
  const isAvailable = computed(() => user.value?.is_available ?? true)

  function setAuth(authData: { user: User }) {
    user.value = authData.user
    localStorage.setItem('user', JSON.stringify(authData.user))
  }

  function clearAuth() {
    user.value = null

    // Clean up localStorage (including legacy token keys)
    localStorage.removeItem('user')
    localStorage.removeItem('auth_token')
    localStorage.removeItem('refresh_token')
  }

  async function restoreSession(): Promise<boolean> {
    // Remove legacy token keys if present
    if (localStorage.getItem('auth_token')) {
      localStorage.removeItem('auth_token')
    }
    if (localStorage.getItem('refresh_token')) {
      localStorage.removeItem('refresh_token')
    }

    const refreshed = await refreshUserData()
    if (!refreshed) {
      clearAuth()
      return false
    }

    return true
  }

  // Fetch fresh user data from API (including updated permissions)
  async function refreshUserData(): Promise<boolean> {
    try {
      const response = await api.get('/me')
      const freshUser = unwrapResponse<User>(response)
      user.value = freshUser
      localStorage.setItem('user', JSON.stringify(freshUser))
      return true
    } catch {
      // If unauthorized, clear auth
      return false
    }
  }

  async function login(email: string, password: string): Promise<void> {
    const response = await api.post('/auth/login', { email, password })
    // Server sets cookies; response body has { user, expires_in }
    const payload = unwrapResponse<{ user: User }>(response)
    setAuth({ user: payload.user })
  }

  async function register(data: {
    email: string
    password: string
    full_name: string
    organization_id?: string
    invitation_token: string
  }): Promise<void> {
    await api.post('/auth/register', data)
  }

  async function switchOrg(organizationId: string): Promise<void> {
    const response = await api.post('/auth/switch-org', { organization_id: organizationId })
    const payload = unwrapResponse<{ user: User }>(response)
    setAuth({ user: payload.user })
    // Update localStorage org override
    localStorage.setItem('selected_organization_id', organizationId)
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/auth/logout', {})
    } catch {
      // Ignore logout errors
    } finally {
      clearAuth()
    }
  }

  function setAvailability(available: boolean, breakStart?: string | null) {
    if (user.value) {
      user.value = { ...user.value, is_available: available }
      localStorage.setItem('user', JSON.stringify(user.value))
    }
    // Track break start time
    if (!available && breakStart) {
      breakStartedAt.value = breakStart
      localStorage.setItem('break_started_at', breakStart)
    } else if (available) {
      breakStartedAt.value = null
      localStorage.removeItem('break_started_at')
    }
  }

  function restoreBreakTime() {
    const stored = localStorage.getItem('break_started_at')
    if (stored && !isAvailable.value) {
      breakStartedAt.value = stored
    }
  }

  // Check if user has a specific permission
  function hasPermission(resource: string, action: string = 'read'): boolean {
    // Super admins have all permissions
    if (user.value?.is_super_admin) {
      return true
    }

    const permissions = user.value?.role?.permissions
    if (!permissions || permissions.length === 0) {
      return false
    }

    // Handle both string keys (e.g., "resource:read") and objects (e.g., { resource: "resource", action: "read" })
    const targetKey = `${resource}:${action}`

    return permissions.some(p => {
      if (typeof p === 'string') {
        return p === targetKey || p === `${resource}:manage` || p === '*:*'
      }
      return (
        (p.resource === resource && (p.action === action || p.action === 'manage')) ||
        (p.resource === '*' && p.action === '*')
      )
    })
  }

  return {
    user,
    breakStartedAt,
    isAuthenticated,
    userRole,
    organizationId,
    userSettings,
    isAvailable,
    setAuth,
    clearAuth,
    restoreSession,
    restoreBreakTime,
    refreshUserData,
    login,
    register,
    switchOrg,
    logout,
    setAvailability,
    hasPermission
  }
})
