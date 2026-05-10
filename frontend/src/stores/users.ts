import { defineStore } from 'pinia'
import { ref } from 'vue'
import { usersService } from '@/services/api'
import type { User } from '@/types/auth'
import { unwrapResponse } from '@/lib/api-utils'

export interface CreateUserData {
  email: string
  password: string
  full_name: string
  role_id?: string
  is_super_admin?: boolean
}

export interface UpdateUserData {
  email?: string
  password?: string
  full_name?: string
  role_id?: string
  is_active?: boolean
  is_super_admin?: boolean
}

export interface FetchUsersParams {
  search?: string
  page?: number
  limit?: number
}

export interface FetchUsersResponse {
  users: User[]
  total: number
  page: number
  limit: number
}

export const useUsersStore = defineStore('users', () => {
  const users = ref<User[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchUsers(params?: FetchUsersParams): Promise<FetchUsersResponse> {
    loading.value = true
    error.value = null
    try {
      const response = await usersService.list(params)
      const data = unwrapResponse<FetchUsersResponse>(response)
      users.value = data.users || []
      return {
        users: data.users || [],
        total: data.total ?? users.value.length,
        page: data.page ?? 1,
        limit: data.limit ?? 50
      }
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch users'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createUser(data: CreateUserData): Promise<User> {
    loading.value = true
    error.value = null
    try {
      const response = await usersService.create(data)
      const newUser = unwrapResponse<User>(response)
      users.value.unshift(newUser)
      return newUser
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to create user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateUser(id: string, data: UpdateUserData): Promise<User> {
    loading.value = true
    error.value = null
    try {
      const response = await usersService.update(id, data)
      const updatedUser = unwrapResponse<User>(response)
      const index = users.value.findIndex(u => u.id === id)
      if (index !== -1) {
        users.value[index] = updatedUser
      }
      return updatedUser
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to update user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function fetchAllUsers(params?: Omit<FetchUsersParams, 'page' | 'limit'>): Promise<User[]> {
    loading.value = true
    error.value = null
    try {
      const pageSize = 100
      let page = 1
      let allUsers: User[] = []
      let total = 0

      do {
        const response = await usersService.list({ ...params, page, limit: pageSize })
        const data = unwrapResponse<FetchUsersResponse>(response)
        const fetched = data.users || []
        allUsers = allUsers.concat(fetched)
        total = data.total ?? allUsers.length
        page++
      } while (allUsers.length < total)

      users.value = allUsers
      return allUsers
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch users'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteUser(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await usersService.delete(id)
      users.value = users.value.filter(u => u.id !== id)
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to delete user'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    users,
    loading,
    error,
    fetchUsers,
    fetchAllUsers,
    createUser,
    updateUser,
    deleteUser
  }
})
