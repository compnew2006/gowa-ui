import posthog from 'posthog-js'
import type { App } from 'vue'

export const initPostHog = (app: App) => {
  const apiKey = import.meta.env.VITE_POSTHOG_KEY
  const apiHost = import.meta.env.VITE_POSTHOG_HOST || 'https://app.posthog.com'

  if (!apiKey) {
    if (import.meta.env.DEV) {
      console.warn('PostHog API key is missing. Analytics will be disabled in development.')
    }
    return
  }

  posthog.init(apiKey, {
    api_host: apiHost,
    person_profiles: 'identified_only', // or 'always' depending on your requirements
    capture_pageview: false, // We'll handle this manually via the router if needed, or set to true for auto-capture
    loaded: (ph) => {
      if (import.meta.env.DEV) ph.debug()
    }
  })

  // Add posthog to global properties if needed
  app.config.globalProperties.$posthog = posthog
}

export default posthog
