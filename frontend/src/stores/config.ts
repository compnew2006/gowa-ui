import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/services/api'

export interface FeatureFlags {
    templates: boolean
    flows: boolean
    catalog: boolean
    business_profile: boolean
    campaigns: boolean
    meta_insights: boolean
}

export interface AppConfig {
    whatsapp_provider: 'meta' | 'whatsmeow'
    features: FeatureFlags
}

/**
 * useConfigStore exposes the active WhatsApp provider and feature flags.
 * Fetched once on app load; components use `features` to hide Meta-only UI.
 */
export const useConfigStore = defineStore('config', () => {
    const config = ref<AppConfig | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)

    const provider = computed(() => config.value?.whatsapp_provider ?? 'meta')
    const features = computed<FeatureFlags>(() => config.value?.features ?? {
        templates: true,
        flows: true,
        catalog: true,
        business_profile: true,
        campaigns: true,
        meta_insights: true,
    })

    const isWhatsmeow = computed(() => provider.value === 'whatsmeow')
    const isMeta = computed(() => provider.value === 'meta')

    async function fetchConfig() {
        if (config.value) return // already loaded
        loading.value = true
        error.value = null
        try {
            const resp = await api.get('/config')
            config.value = resp.data?.data ?? resp.data
        } catch (e: any) {
            error.value = e?.message ?? 'Failed to load config'
            // Fallback to meta defaults so the UI isn't broken
            config.value = {
                whatsapp_provider: 'meta',
                features: {
                    templates: true,
                    flows: true,
                    catalog: true,
                    business_profile: true,
                    campaigns: true,
                    meta_insights: true,
                },
            }
        } finally {
            loading.value = false
        }
    }

    return {
        config,
        loading,
        error,
        provider,
        features,
        isWhatsmeow,
        isMeta,
        fetchConfig,
    }
})
