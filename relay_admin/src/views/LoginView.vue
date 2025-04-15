<!-- src/views/LoginView.vue -->
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { apiClient } from '@/api/api';
import { useI18n } from 'vue-i18n';
import { sha256hex } from '@/utils/utils';

const { t } = useI18n(); // Make sure t is available

const username = ref('');
const password = ref('');
const loading = ref(false);
const error = ref('');
const isHttpProtocol = ref(false);
const showPassword = ref(false);
const router = useRouter();

onMounted(() => {
  // Check if the current protocol is HTTP
  isHttpProtocol.value = window.location.protocol === 'http:';
});

const handleLogin = async () => {
  if (!username.value || !password.value) {
    error.value = t('error.missingCredentials'); // Use translation key
    return;
  }

  error.value = '';
  loading.value = true;

  try {
    await apiClient.login({
      username: username.value,
      password: await sha256hex(password.value)
    });
    router.push('/');

  } catch (e) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err = e as any;
    console.error(err);
    // Prioritize server message, fallback to translated generic error
    error.value = err.response?.data?.message || t('error.loginFailedFallback');
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-purple-100 p-4">
    <div class="w-full max-w-md">
      <!-- 不安全提醒 -->
      <div v-if="isHttpProtocol" class="mb-6 p-4 bg-amber-50 border-l-4 border-amber-500 rounded-lg shadow-md">
        <div class="flex items-center">
          <div class="flex-shrink-0">
            <svg class="h-5 w-5 text-amber-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"
              fill="currentColor">
              <path fill-rule="evenodd"
                d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                clip-rule="evenodd" />
            </svg>
          </div>
          <div class="ml-3">
            <p class="text-sm text-amber-700">
              {{ t('login.httpWarning') }} <!-- Use translation key -->
            </p>
          </div>
        </div>
      </div>

      <!-- 登录卡片 -->
      <div class="bg-white rounded-2xl shadow-xl overflow-hidden">
        <!-- 顶部装饰 -->
        <div class="h-3 bg-gradient-to-r from-cyan-400 via-blue-500 to-purple-500"></div>

        <div class="p-8">
          <div class="text-center mb-8">
            <h2 class="text-3xl font-bold text-gray-800 mb-2">{{ t('login.welcomeBack') }}</h2>
            <!-- Use translation key -->
            <p class="text-gray-500">{{ t('login.prompt') }}</p> <!-- Use translation key -->
          </div>

          <form @submit.prevent="handleLogin" class="space-y-6">
            <!-- 错误提示 -->
            <div v-if="error" class="p-3 bg-red-50 border border-red-200 text-red-600 rounded-lg text-sm">
              {{ error }} <!-- Error message is already dynamic -->
            </div>

            <!-- 用户名输入 -->
            <div class="space-y-2">
              <label for="username" class="block text-sm font-medium text-gray-700">{{ t('login.label.username')
              }}</label> <!-- Use translation key -->
              <div class="relative rounded-md">
                <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <svg class="h-5 w-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </div>
                <input id="username" v-model="username" type="text" autocomplete="username"
                  class="block w-full pl-10 pr-3 py-3 border border-gray-300 rounded-lg focus:ring-blue-500 focus:border-blue-500 transition duration-150 ease-in-out"
                  :placeholder="t('login.placeholder.username')" required /> <!-- Use translation key -->
              </div>
            </div>

            <!-- 密码输入 -->
            <div class="space-y-2">
              <label for="password" class="block text-sm font-medium text-gray-700">{{ t('login.label.password')
              }}</label> <!-- Use translation key -->
              <div class="relative rounded-md">
                <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <svg class="h-5 w-5 text-gray-400" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                  </svg>
                </div>
                <input id="password" v-model="password" :type="showPassword ? 'text' : 'password'"
                  autocomplete="current-password"
                  class="block w-full pl-10 pr-10 py-3 border border-gray-300 rounded-lg focus:ring-blue-500 focus:border-blue-500 transition duration-150 ease-in-out"
                  :placeholder="t('login.placeholder.password')" required /> <!-- Use translation key -->
                <div class="absolute inset-y-0 right-0 pr-3 flex items-center">
                  <button type="button" @click="showPassword = !showPassword"
                    class="text-gray-400 hover:text-gray-600 focus:outline-none">
                    <!-- SVGs for show/hide password don't need translation -->
                    <svg v-if="showPassword" class="h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none"
                      viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                    </svg>
                    <svg v-else class="h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                      stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                        d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            <!-- 登录按钮 -->
            <button type="submit"
              class="w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-white bg-gradient-to-r from-blue-500 to-purple-500 hover:from-blue-600 hover:to-purple-600 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition duration-150 ease-in-out"
              :disabled="loading">
              <svg v-if="loading" class="animate-spin -ml-1 mr-2 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg"
                fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                </path>
              </svg>
              {{ loading ? t('button.loggingIn') : t('button.login') }} <!-- Use translation keys -->
            </button>
          </form>

          <!-- 底部装饰 -->
          <div class="mt-8 flex justify-center">
            <div class="inline-flex space-x-1">
              <span class="h-2 w-2 rounded-full bg-blue-400 animate-bounce" style="animation-delay: 0ms"></span>
              <span class="h-2 w-2 rounded-full bg-purple-400 animate-bounce" style="animation-delay: 150ms"></span>
              <span class="h-2 w-2 rounded-full bg-pink-400 animate-bounce" style="animation-delay: 300ms"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- 底部版权信息 -->
      <div class="mt-6 text-center text-sm text-gray-500">
        <!-- Use translation key with interpolation -->
        {{ t('login.copyright', { year: new Date().getFullYear(), appName: t('appName') }) }}
      </div>
    </div>
  </div>
</template>

<style scoped>
@keyframes float {
  0% {
    transform: translateY(0px);
  }

  50% {
    transform: translateY(-10px);
  }

  100% {
    transform: translateY(0px);
  }
}

.animate-bounce {
  animation: bounce 1s infinite;
}

@keyframes bounce {

  0%,
  100% {
    transform: translateY(0);
  }

  50% {
    transform: translateY(-5px);
  }
}
</style>