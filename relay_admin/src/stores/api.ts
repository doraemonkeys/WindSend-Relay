import { ref, computed } from 'vue'
import { defineStore } from 'pinia'



export const useApiStore = defineStore('api', () => {
  const authToken = ref<string | null>(null);
  const setAuthToken = (token: string | null) => {
    authToken.value = token;
  };
  return { authToken, setAuthToken };
}, {
  persist: true,
});
