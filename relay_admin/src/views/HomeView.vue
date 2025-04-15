<script setup lang="ts">
import { ref, onMounted, computed } from 'vue';
import { useApiStore } from '@/stores/api';
import router from '@/router';
import { apiClient, type ActiveConnection } from '@/api/api';
import { formatBytes } from '@/utils/utils';
import { useI18n } from 'vue-i18n';

// --- i18n ---
// Destructure locale along with t
const { t, locale } = useI18n();

// --- Define available languages ---
// Make sure these codes match the locales configured in your i18n setup
const availableLocales = ref([
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
  // Add other supported languages here
]);

// --- State ---
const apiStore = useApiStore();
const connections = ref<ActiveConnection[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const searchQuery = ref('');
const refreshing = ref(false);
const lastRefreshed = ref(new Date());

// --- Computed Properties ---
const filteredConnections = computed(() => {
  if (!searchQuery.value) return connections.value;
  const query = searchQuery.value.toLowerCase();
  return connections.value.filter(conn =>
    conn.reqAddr.toLowerCase().includes(query) ||
    conn.id.toLowerCase().includes(query)
  );
});

const stats = computed(() => {
  const total = connections.value.length;
  const active = connections.value.filter(conn => conn.relaying).length;
  const inactive = total - active;
  return { total, active, inactive };
});

// --- Formatting ---
const formatDateTime = (dateStr: string | Date) => {
  const date = new Date(dateStr);
  // Use Intl for better locale-aware formatting based on the i18n locale
  try {
    // Pass the current i18n locale to ensure consistency
    return new Intl.DateTimeFormat(locale.value, {
      month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false // Adjust options as needed
    }).format(date);
  } catch (e) {
    // Fallback if Intl fails or locale is unsupported by browser's Intl
    console.warn("Intl.DateTimeFormat failed for locale:", locale.value, e);
    return date.toLocaleString(); // Use basic fallback
  }
};

// const formatDateTime = (dateStr: string | Date) => {
//   const date = new Date(dateStr);
//   // Consider using locale-aware formatting for better i18n
//   return date.toLocaleString();
// };

const formatDuration2 = (dateStr: string | Date): string => {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();

  const seconds = Math.floor(diffMs / 1000);
  if (seconds < 60) return `${seconds} ${t('unit.second', seconds)}`;

  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} ${t('unit.minute', minutes)}`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} ${t('unit.hour', hours)}`;

  const days = Math.floor(hours / 24);
  return `${days} ${t('unit.day', days)}`;
};

const selectedLocale = ref(locale.value);

const switchLocale = (code: string) => {
  locale.value = code;
  localStorage.setItem('locale', code);
}


// --- API Calls ---
const fetchConnections = async () => {
  loading.value = true;
  error.value = null;
  try {
    connections.value = await apiClient.getConnectionStatus();
    lastRefreshed.value = new Date();
  } catch (err) {
    error.value = t('error.fetchConnections');
    console.error(err);
  } finally {
    loading.value = false;
  }
};

const refreshData = async () => {
  refreshing.value = true;
  await fetchConnections();
  refreshing.value = false;
};

const closeConnection = async (id: string) => {
  try {
    await apiClient.closeConnection(id);
    await fetchConnections();
  } catch (err) {
    console.error('closeConnection error:', err);
    error.value = t('error.closeConnectionFailed', { id }); // Ensure this key exists
  }
};

// --- Navigation ---
const goToStatistics = () => {
  router.push('/statistic');
};

// --- Lifecycle ---
onMounted(() => {
  if (!apiStore.authToken) {
    router.push('/login');
  } else {
    fetchConnections();
  }
});

</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-blue-50 to-purple-50 p-6">
    <!-- Page Title & Controls -->
    <div class="flex flex-col md:flex-row justify-between items-center mb-8 gap-4">
      <!-- Left Side: Title & Subtitle -->
      <div class="flex-grow">
        <h1 class="text-3xl font-bold text-indigo-600 mb-2">{{ t('activeConnections') }}</h1>
        <p class="text-gray-500">
          {{ t('viewAndManageCurrentActiveConnections') }}
          <span class="text-xs ml-2 block sm:inline mt-1 sm:mt-0">
            {{ t('lastUpdated') }}: {{ formatDateTime(lastRefreshed) }}
          </span>
        </p>
      </div>

      <!-- Right Side: Controls -->
      <div class="flex items-center gap-3 flex-shrink-0">
        <!-- Language Selector -->
        <select v-model="selectedLocale" @change="switchLocale(selectedLocale)"
          class="select select-bordered select-sm max-w-xs">
          <option v-for="lang in availableLocales" :key="lang.code" :value="lang.code">
            {{ lang.name }}
          </option>
        </select>

        <!-- Action Buttons -->
        <button @click="refreshData" class="btn btn-primary btn-sm gap-2" :class="{ 'loading': refreshing }">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ t('button.refresh') }}
        </button>
        <button @click="goToStatistics" class="btn btn-outline btn-sm gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          {{ t('button.statistics') }}
        </button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
      <!-- Total Connections Card -->
      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-blue-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-blue-500" fill="none" viewBox="0 0 24 24"
                stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.totalConnections') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.total }}</div>
            </div>
          </div>
        </div>
      </div>
      <!-- Active Connections Card -->
      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-green-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24"
                stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.active') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.active }}</div>
            </div>
          </div>
        </div>
      </div>
      <!-- Idle Connections Card -->
      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-amber-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-amber-500" fill="none" viewBox="0 0 24 24"
                stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.idle') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.inactive }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>


    <!-- Search Box -->
    <div class="mb-6">
      <div class="relative">
        <div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
          <svg class="w-4 h-4 text-gray-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none"
            viewBox="0 0 20 20">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="m19 19-4-4m0-7A7 7 0 1 1 1 8a7 7 0 0 1 14 0Z" />
          </svg>
        </div>
        <input v-model="searchQuery" type="search"
          class="block w-full p-3 pl-10 text-sm text-gray-900 border border-gray-300 rounded-lg bg-white focus:ring-indigo-500 focus:border-indigo-500"
          :placeholder="t('search.placeholder')" />
      </div>
    </div>

    <!-- Loading Indicator -->
    <div v-if="loading" class="flex justify-center my-12">
      <div class="loading loading-spinner loading-lg text-indigo-500"></div>
    </div>

    <!-- Error Message -->
    <div v-else-if="error" class="alert alert-error shadow-lg mb-6">
      <div>
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none"
          viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>{{ error }}</span>
      </div>
      <div class="flex-none">
        <button @click="fetchConnections" class="btn btn-sm">{{ t('button.retry') }}</button>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredConnections.length === 0" class="text-center py-12">
      <div class="flex flex-col items-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24"
          stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <h3 class="text-lg font-medium text-gray-500">
          {{ searchQuery ? t('emptyState.noMatch') : t('emptyState.noConnections') }}
        </h3>
        <p class="text-gray-400 mt-2">
          {{ searchQuery ? t('emptyState.tryOtherSearch') : t('emptyState.allClosed') }}
        </p>
      </div>
    </div>

    <!-- Connections List -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="conn in filteredConnections" :key="conn.id"
        class="card bg-white shadow-md hover:shadow-lg transition-all"
        :class="{ 'border-2 border-green-300': conn.relaying }">
        <div class="card-body p-5">
          <!-- Card Header: Status Badge & Actions Dropdown -->
          <div class="flex justify-between items-start">
            <div class="badge" :class="conn.relaying ? 'badge-success' : 'badge-warning'">
              {{ conn.relaying ? t('status.active') : t('status.idle') }}
            </div>
            <div class="dropdown dropdown-end">
              <label tabindex="0" class="btn btn-ghost btn-xs">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                  stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                </svg>
              </label>
              <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                <li><a @click="closeConnection(conn.id)">{{ t('action.closeConnection') }}</a></li>
              </ul>
            </div>
          </div>

          <!-- Card Title & ID -->
          <h2 class="card-title text-lg font-medium text-gray-700 mt-2 truncate" :title="conn.reqAddr">
            {{ conn.reqAddr }}
          </h2>
          <div class="text-xs text-gray-500 mb-3 truncate" :title="conn.id">
            ID: {{ conn.id }}
          </div>

          <!-- Connection Details Grid -->
          <div class="grid grid-cols-2 gap-2 text-sm">
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.connectTime') }}</span>
              <span class="font-medium">{{ formatDateTime(conn.connectTime) }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.duration') }}</span>
              <span class="font-medium">{{ formatDuration2(conn.connectTime) }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.lastActive') }}</span>
              <span class="font-medium">{{ formatDateTime(conn.lastActive) }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.traffic') }}</span>
              <span class="font-medium">{{ formatBytes(conn.history.totalRelayBytes) }}</span>
            </div>
          </div>

          <!-- Card Footer: Stats & Close Button -->
          <div class="mt-4 flex justify-between items-center">
            <!-- Stats Badges -->
            <div class="flex items-center gap-2">
              <div class="tooltip" :data-tip="t('tooltip.totalRequests')">
                <div class="badge badge-outline gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                  </svg>
                  {{ conn.history.totalRelayCount }}
                </div>
              </div>
              <div class="tooltip" :data-tip="t('tooltip.errorCount')">
                <div class="badge badge-outline gap-1"
                  :class="{ 'text-red-500 border-red-300': conn.history.totalRelayErrCount > 0 }">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24"
                    stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {{ conn.history.totalRelayErrCount }}
                </div>
              </div>
            </div>
            <!-- Close Button -->
            <button @click="closeConnection(conn.id)" class="btn btn-xs btn-outline btn-error">
              {{ t('button.close') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card {
  transition: all 0.3s ease;
}

.card:hover {
  transform: translateY(-3px);
}
</style>