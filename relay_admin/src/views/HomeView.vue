<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue';
import { useApiStore } from '@/stores/api';
import router from '@/router';
import { apiClient, type ActiveConnection } from '@/api/api';
import { formatBytes } from '@/utils/utils';
import { useI18n } from 'vue-i18n';

// --- i18n ---
const { t, locale } = useI18n();

// --- Define available languages ---
const availableLocales = ref([
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
]);

// --- State ---
const apiStore = useApiStore();
const connections = ref<ActiveConnection[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const updateError = ref<string | null>(null); // Specific error for updates
const searchQuery = ref('');
const refreshing = ref(false);
const lastRefreshed = ref(new Date());
const editingConnectionId = ref<string | null>(null); // Track which connection is being edited
const editingName = ref(''); // Temporary storage for the name being edited
const editInputRef = ref<HTMLInputElement | null>(null); // Ref for the input field to focus it

// --- Computed Properties ---
const filteredConnections = computed(() => {
  if (!searchQuery.value) return connections.value;
  const query = searchQuery.value.toLowerCase();
  return connections.value.filter(conn =>
    conn.reqAddr.toLowerCase().includes(query) ||
    (conn.customName && conn.customName.toLowerCase().includes(query)) ||
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
  try {
    return new Intl.DateTimeFormat(locale.value, {
      month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false
    }).format(date);
  } catch (e) {
    console.warn("Intl.DateTimeFormat failed for locale:", locale.value, e);
    return date.toLocaleString();
  }
};

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


// --- API Calls & Actions ---
const fetchConnections = async () => {
  loading.value = true;
  error.value = null;
  updateError.value = null; // Clear update error on fetch
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
    await fetchConnections(); // Refresh list after closing
  } catch (err) {
    console.error('closeConnection error:', err);
    error.value = t('error.closeConnectionFailed', { id });
  }
};

const startEditingName = (conn: ActiveConnection) => {
  editingConnectionId.value = conn.id;
  editingName.value = conn.customName || ''; // Start with current name or empty string
  updateError.value = null; // Clear previous update errors
  // Focus the input field after it's rendered
  nextTick(() => {
    editInputRef.value?.focus();
  });
};

const cancelEditName = () => {
  editingConnectionId.value = null;
  editingName.value = '';
  updateError.value = null;
};

const saveEditName = async (id: string) => {
  if (editingConnectionId.value !== id) return; // Should not happen, but safety check
  const newName = editingName.value.trim(); // Trim whitespace
  updateError.value = null; // Clear previous error

  // Optional: Add validation if needed (e.g., prevent saving empty name if desired)
  // if (!newName) {
  //   updateError.value = t('error.customNameCannotBeEmpty'); // Add translation key
  //   return;
  // }

  try {
    await apiClient.updateConnectionName(id, newName);
    editingConnectionId.value = null; // Exit edit mode
    editingName.value = '';
    await fetchConnections(); // Refresh data to show the updated name
  } catch (err) {
    console.error('updateConnectionName error:', err);
    const message = t('error.updateNameFailed');
    updateError.value = `${t('error.updateFailedPrefix')}: ${message}`;
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
          :placeholder="t('search.placeholderConnections')" />
      </div>
    </div>

    <!-- Loading Indicator -->
    <div v-if="loading" class="flex justify-center my-12">
      <div class="loading loading-spinner loading-lg text-indigo-500"></div>
    </div>

    <!-- Error Message (General Fetch Error) -->
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

    <!-- Update Error Message -->
    <div v-if="updateError" class="alert alert-warning shadow-lg mb-6">
      <div>
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none"
          viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <span>{{ updateError }}</span>
      </div>
      <div class="flex-none">
        <button @click="updateError = null" class="btn btn-sm btn-ghost">{{ t('button.dismiss') }}</button>
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
            <!-- Keep dropdown if other actions are needed, otherwise it can be removed -->
            <div class="dropdown dropdown-end">
              <label tabindex="0" class="btn btn-ghost btn-xs">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                  stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                </svg>
              </label>
              <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                <li><a @click="startEditingName(conn)">{{ t('action.editName') }}</a></li>
                <li><a @click="closeConnection(conn.id)">{{ t('action.closeConnection') }}</a></li>
              </ul>
            </div>
          </div>

          <!-- Card Title (Custom Name / Req Addr) & Edit Input -->
          <div class="mt-2 min-h-[60px]">
            <div v-if="editingConnectionId !== conn.id" class="flex items-center group">
              <!-- Display Custom Name if available, otherwise Req Addr -->
              <h2 class="card-title text-lg font-medium text-gray-700 truncate flex-grow"
                :title="conn.customName || conn.reqAddr">
                {{ conn.customName || conn.reqAddr }}
              </h2>
              <!-- Edit Icon - shows on hover -->
              <button @click="startEditingName(conn)"
                class="btn btn-ghost btn-xs ml-1 opacity-0 group-hover:opacity-100 transition-opacity"
                :aria-label="t('action.editName')">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24"
                  stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
            </div>
            <!-- Edit Name Input Form -->
            <div v-else class="flex items-center gap-2">
              <input ref="editInputRef" type="text" v-model="editingName" @keyup.enter="saveEditName(conn.id)"
                @keyup.esc="cancelEditName" class="input input-sm flex-grow"
                :placeholder="t('placeholder.customName')" />
              <button @click="saveEditName(conn.id)" class="btn btn-success btn-xs"
                :aria-label="t('button.save')">✓</button>
              <button @click="cancelEditName" class="btn btn-ghost btn-xs" :aria-label="t('button.cancel')">✕</button>
            </div>
            <!-- Subtitle: Req Addr (if custom name exists) and ID -->
            <div class="text-xs text-gray-500 mt-1 truncate" :title="conn.id">
              <span v-if="conn.customName">{{ conn.reqAddr }}<br /></span>
              <!-- Show ReqAddr below if custom name exists -->
              ID: {{ conn.id }}
            </div>
          </div>


          <!-- Connection Details Grid -->
          <div class="grid grid-cols-2 gap-2 text-sm mt-3"> <!-- Adjusted margin -->
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

/* Ensure edit icon is vertically centered */
.group .btn-ghost {
  vertical-align: middle;
}
</style>