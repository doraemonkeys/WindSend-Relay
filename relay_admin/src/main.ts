import './assets/base.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import App from './App.vue'
import router from './router'
import { apiClient } from './api/api'
import { useApiStore } from './stores/api'
import i18n from './i18n'

const app = createApp(App)
const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)
app.use(pinia)
app.use(router)
app.use(i18n)
app.mount('#app')



apiClient.onTokenChange = (token) => {
  if (token) {
    useApiStore().setAuthToken(token)
  } else {
    useApiStore().setAuthToken(null)
  }
}
apiClient.onUnauthorized = () => {
  router.push('/login')
}
apiClient.getAuthToken = () => {
  return useApiStore().authToken
}


