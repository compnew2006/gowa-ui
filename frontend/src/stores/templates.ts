import { defineStore } from 'pinia'
import { ref } from 'vue'
import { templatesService, type Template } from '@/services/api'

export interface CreateTemplateData {
  name: string
  display_name?: string
  language: string
  category: string
  whatsapp_account: string
  header_type?: string
  header_content?: string
  body_content: string
  footer_content?: string
  buttons?: any[]
}

export interface UpdateTemplateData extends Partial<CreateTemplateData> {}

export interface FetchTemplatesParams {
  search?: string
  category?: string
  account?: string
  page?: number
  limit?: number
}

export interface FetchTemplatesResponse {
  templates: Template[]
  total: number
  page: number
  limit: number
}

export const useTemplatesStore = defineStore('templates', () => {
  const templates = ref<Template[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchTemplates(params?: FetchTemplatesParams): Promise<FetchTemplatesResponse> {
    loading.value = true
    error.value = null
    try {
      const response = await templatesService.list(params)
      const data = (response.data as any).data || response.data
      templates.value = data.templates || []
      return {
        templates: data.templates || [],
        total: data.total ?? templates.value.length,
        page: data.page ?? 1,
        limit: data.limit ?? 50
      }
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to fetch templates'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createTemplate(data: CreateTemplateData): Promise<Template> {
    loading.value = true
    error.value = null
    try {
      const response = await templatesService.create(data)
      const newTemplate = (response.data as any).data || response.data
      templates.value.unshift(newTemplate)
      return newTemplate
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to create template'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateTemplate(id: string, data: UpdateTemplateData): Promise<Template> {
    loading.value = true
    error.value = null
    try {
      const response = await templatesService.update(id, data)
      const updatedTemplate = (response.data as any).data || response.data
      const index = templates.value.findIndex(t => t.id === id)
      if (index !== -1) {
        templates.value[index] = updatedTemplate
      }
      return updatedTemplate
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to update template'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteTemplate(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await templatesService.delete(id)
      templates.value = templates.value.filter(t => t.id !== id)
    } catch (err: any) {
      error.value = err.response?.data?.message || 'Failed to delete template'
      throw err
    } finally {
      loading.value = false
    }
  }

  function getTemplateById(id: string): Template | undefined {
    return templates.value.find(t => t.id === id)
  }

  return {
    templates,
    loading,
    error,
    fetchTemplates,
    createTemplate,
    updateTemplate,
    deleteTemplate,
    getTemplateById
  }
})
