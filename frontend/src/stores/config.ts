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

export interface TenantConfig {
    subdomain_locked: boolean
    organization_slug?: string
    organization_name?: string
}

export interface AppConfig {
    whatsapp_provider: 'meta' | 'whatsmeow'
    features: FeatureFlags
    tenant?: TenantConfig
}

const CHAT_SHOW_PRINT_BUTTONS_KEY = 'chat.showPrintButtons'
const CHAT_SHOW_DOWNLOAD_BUTTONS_KEY = 'chat.showDownloadButtons'

function readStoredBoolean(key: string, fallback: boolean): boolean {
    if (typeof window === 'undefined') return fallback
    try {
        const raw = localStorage.getItem(key)
        if (raw === null) return fallback
        if (raw === '1') return true
        if (raw === '0') return false
        return raw.toLowerCase() === 'true'
    } catch {
        return fallback
    }
}

function writeStoredBoolean(key: string, value: boolean): void {
    if (typeof window === 'undefined') return
    try {
        localStorage.setItem(key, value ? '1' : '0')
    } catch {
        // Ignore persistence failures.
    }
}

/**
 * useConfigStore exposes the active WhatsApp provider and feature flags.
 * Fetched once on app load; components use `features` to hide Meta-only UI.
 */
export const useConfigStore = defineStore('config', () => {
    const config = ref<AppConfig | null>(null)
    const loading = ref(false)
    const error = ref<string | null>(null)
    const showPrintButtons = ref<boolean>(
        readStoredBoolean(CHAT_SHOW_PRINT_BUTTONS_KEY, true),
    )
    const showDownloadButtons = ref<boolean>(
        readStoredBoolean(CHAT_SHOW_DOWNLOAD_BUTTONS_KEY, true),
    )

    const provider = computed(() => config.value?.whatsapp_provider ?? 'meta')
    const features = computed<FeatureFlags>(() => config.value?.features ?? {
        templates: true,
        flows: true,
        catalog: true,
        business_profile: true,
        campaigns: true,
        meta_insights: true,
    })
    const tenant = computed<TenantConfig>(() => config.value?.tenant ?? {
        subdomain_locked: false,
    })

    const isWhatsmeow = computed(() => provider.value === 'whatsmeow')
    const isMeta = computed(() => provider.value === 'meta')

    function setShowPrintButtons(value: boolean) {
        showPrintButtons.value = Boolean(value)
        writeStoredBoolean(CHAT_SHOW_PRINT_BUTTONS_KEY, showPrintButtons.value)
    }

    function setShowDownloadButtons(value: boolean) {
        showDownloadButtons.value = Boolean(value)
        writeStoredBoolean(
            CHAT_SHOW_DOWNLOAD_BUTTONS_KEY,
            showDownloadButtons.value,
        )
    }

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
                tenant: {
                    subdomain_locked: false,
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
        tenant,
        isWhatsmeow,
        isMeta,
        showPrintButtons,
        showDownloadButtons,
        setShowPrintButtons,
        setShowDownloadButtons,
        fetchConfig,
    }
})
