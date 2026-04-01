import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'

import App from './App.vue'
import router from './router'
import { i18n } from './i18n'
import { initPostHog } from './lib/posthog'

import './assets/fonts.css'
import './assets/index.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(VueQueryPlugin)
app.use(i18n)

initPostHog(app)

router.afterEach((to) => {
  const posthog = app.config.globalProperties.$posthog
  if (posthog) {
    posthog.capture('$pageview', {
      $current_url: window.location.origin + to.fullPath
    })
  }
})

app.mount('#app')
