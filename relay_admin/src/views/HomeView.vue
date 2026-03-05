<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue';
import { useApiStore } from '@/stores/api';
import router from '@/router';
import { apiClient, type ActiveConnection } from '@/api/api';
import { formatBytes } from '@/utils/utils';
import { useI18n } from 'vue-i18n';

type ConnectionViewState = 'denied' | 'active' | 'probing' | 'idle';

interface ConnectionStatusMeta {
  labelKey: string;
  badgeClass: string;
  borderClass: string;
}

const connectionStatusMetaMap: Record<ConnectionViewState, ConnectionStatusMeta> = {
  denied: {
    labelKey: 'status.denied',
    badgeClass: 'badge-error',
    borderClass: 'border-2 border-red-300',
  },
  active: {
    labelKey: 'status.active',
    badgeClass: 'badge-success',
    borderClass: 'border-2 border-green-300',
  },
  probing: {
    labelKey: 'status.probing',
    badgeClass: 'badge-info',
    borderClass: 'border-2 border-blue-300',
  },
  idle: {
    labelKey: 'status.idle',
    badgeClass: 'badge-warning',
    borderClass: 'border-2 border-amber-300',
  },
};

const { t, locale } = useI18n();

const availableLocales = ref([
  { code: 'en', name: 'English' },
  { code: 'zh', name: '中文' },
]);

const apiStore = useApiStore();
const connections = ref<ActiveConnection[]>([]);
const loading = ref(true);
const error = ref<string | null>(null);
const updateError = ref<string | null>(null);
const searchQuery = ref('');
const refreshing = ref(false);
const lastRefreshed = ref(new Date());
const editingConnectionId = ref<string | null>(null);
const editingName = ref('');
const editInputRef = ref<HTMLInputElement | null>(null);
const pendingActionConnectionId = ref<string | null>(null);

const getConnectionState = (conn: ActiveConnection): ConnectionViewState => {
  if (conn.denied) {
    return 'denied';
  }
  if (conn.activeCount > 0) {
    return 'active';
  }
  if (conn.probingCount > 0) {
    return 'probing';
  }
  return 'idle';
};

const getConnectionStatusMeta = (conn: ActiveConnection): ConnectionStatusMeta => {
  return connectionStatusMetaMap[getConnectionState(conn)];
};

const isActionPending = (id: string): boolean => pendingActionConnectionId.value === id;

const filteredConnections = computed(() => {
  if (!searchQuery.value) return connections.value;
  const query = searchQuery.value.toLowerCase();
  return connections.value.filter((conn) => {
    const customName = conn.customName ?? '';
    return customName.toLowerCase().includes(query) || conn.id.toLowerCase().includes(query);
  });
});

const stats = computed(() => {
  return connections.value.reduce(
    (acc, conn) => {
      acc.total += 1;
      acc.idle += conn.idleCount;
      acc.active += conn.activeCount;
      acc.probing += conn.probingCount;
      if (conn.denied) {
        acc.denied += 1;
      }
      return acc;
    },
    { total: 0, idle: 0, active: 0, probing: 0, denied: 0 }
  );
});

const formatDateTime = (dateStr: string | Date | undefined | null) => {
  if (!dateStr) {
    return '-';
  }
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime()) || date.getUTCFullYear() < 1971) {
    return '-';
  }
  try {
    return new Intl.DateTimeFormat(locale.value, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    }).format(date);
  } catch (e) {
    console.warn('Intl.DateTimeFormat failed for locale:', locale.value, e);
    return date.toLocaleString();
  }
};

const selectedLocale = ref(locale.value);

const switchLocale = (code: string) => {
  locale.value = code;
  localStorage.setItem('locale', code);
};

const loadConnections = async (silent = false) => {
  if (!silent) {
    loading.value = true;
  }
  error.value = null;
  try {
    connections.value = await apiClient.getConnectionStatus();
    lastRefreshed.value = new Date();
  } catch (err) {
    error.value = t('error.fetchConnections');
    console.error(err);
  } finally {
    if (!silent) {
      loading.value = false;
    }
  }
};

const fetchConnections = async () => {
  await loadConnections(false);
};

const refreshData = async () => {
  refreshing.value = true;
  await loadConnections(true);
  refreshing.value = false;
};

const withConnectionAction = async (
  id: string,
  action: () => Promise<void>,
  errorKey: 'error.closeConnectionFailed' | 'error.allowConnectionFailed'
) => {
  pendingActionConnectionId.value = id;
  error.value = null;
  try {
    await action();
    await loadConnections(true);
  } catch (err) {
    console.error('connection action error:', err);
    error.value = t(errorKey, { id });
  } finally {
    if (pendingActionConnectionId.value === id) {
      pendingActionConnectionId.value = null;
    }
  }
};

const closeConnection = async (id: string) => {
  await withConnectionAction(id, () => apiClient.closeConnection(id), 'error.closeConnectionFailed');
};

const allowConnection = async (id: string) => {
  await withConnectionAction(id, () => apiClient.allowConnection(id), 'error.allowConnectionFailed');
};

const startEditingName = (conn: ActiveConnection) => {
  editingConnectionId.value = conn.id;
  editingName.value = conn.customName || '';
  updateError.value = null;
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
  if (editingConnectionId.value !== id) return;
  const newName = editingName.value.trim();
  updateError.value = null;

  try {
    await apiClient.updateConnectionName(id, newName);
    editingConnectionId.value = null;
    editingName.value = '';
    await loadConnections(true);
  } catch (err) {
    console.error('updateConnectionName error:', err);
    const message = t('error.updateNameFailed');
    updateError.value = `${t('error.updateFailedPrefix')}: ${message}`;
  }
};

const goToStatistics = () => {
  router.push('/statistic');
};

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
          <span class="text-xs ml-2 block sm:inline mt-1 sm:mt-0"> {{ t('lastUpdated') }}: {{ formatDateTime(lastRefreshed) }} </span>
        </p>
      </div>

      <!-- Right Side: Controls -->
      <div class="flex items-center gap-3 flex-shrink-0">
        <!-- Language Selector -->
        <select v-model="selectedLocale" @change="switchLocale(selectedLocale)" class="select select-bordered select-sm max-w-xs">
          <option v-for="lang in availableLocales" :key="lang.code" :value="lang.code">
            {{ lang.name }}
          </option>
        </select>

        <!-- Action Buttons -->
        <button @click="refreshData" class="btn btn-primary btn-sm gap-2" :class="{ loading: refreshing }" :disabled="refreshing">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          {{ t('button.refresh') }}
        </button>
        <button @click="goToStatistics" class="btn btn-outline btn-sm gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
            />
          </svg>
          {{ t('button.statistics') }}
        </button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-blue-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
                />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.totalConnections') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.total }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-green-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v12m6-6H6" />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.active') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.active }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-amber-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v8m-4-4h8" />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.idle') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.idle }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-cyan-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-cyan-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M13 10V3L4 14h7v7l9-11h-7z"
                />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.probing') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.probing }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="card bg-white shadow-md hover:shadow-lg transition-shadow">
        <div class="card-body p-4">
          <div class="flex items-center">
            <div class="rounded-full bg-red-100 p-3 mr-4">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
            <div>
              <div class="text-sm text-gray-500">{{ t('stats.denied') }}</div>
              <div class="text-2xl font-bold text-gray-700">{{ stats.denied }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Search Box -->
    <div class="mb-6">
      <div class="relative">
        <div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
          <svg class="w-4 h-4 text-gray-500" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 20 20">
            <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m19 19-4-4m0-7A7 7 0 1 1 1 8a7 7 0 0 1 14 0Z" />
          </svg>
        </div>
        <input
          v-model="searchQuery"
          type="search"
          class="block w-full p-3 pl-10 text-sm text-gray-900 border border-gray-300 rounded-lg bg-white focus:ring-indigo-500 focus:border-indigo-500"
          :placeholder="t('search.placeholderConnections')"
        />
      </div>
    </div>

    <!-- Loading Indicator -->
    <div v-if="loading" class="flex justify-center my-12">
      <div class="loading loading-spinner loading-lg text-indigo-500"></div>
    </div>

    <!-- Error Message (General Fetch Error) -->
    <div v-else-if="error" class="alert alert-error shadow-lg mb-6">
      <div>
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"
          />
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
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current flex-shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
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
        <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
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
      <div
        v-for="conn in filteredConnections"
        :key="conn.id"
        class="card bg-white shadow-md hover:shadow-lg transition-all"
        :class="getConnectionStatusMeta(conn).borderClass"
      >
        <div class="card-body p-5">
          <!-- Card Header: Status Badge & Actions Dropdown -->
          <div class="flex justify-between items-start">
            <div class="badge" :class="getConnectionStatusMeta(conn).badgeClass">
              {{ t(getConnectionStatusMeta(conn).labelKey) }}
            </div>
            <div class="dropdown dropdown-end">
              <label tabindex="0" class="btn btn-ghost btn-xs">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
                  />
                </svg>
              </label>
              <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-52">
                <li><button @click="startEditingName(conn)">{{ t('action.editName') }}</button></li>
                <li v-if="conn.denied">
                  <button @click="allowConnection(conn.id)" :disabled="isActionPending(conn.id)">
                    {{ t('action.allowConnection') }}
                  </button>
                </li>
                <li v-else>
                  <button @click="closeConnection(conn.id)" :disabled="isActionPending(conn.id)">
                    {{ t('action.closeConnection') }}
                  </button>
                </li>
              </ul>
            </div>
          </div>

          <!-- Card Title (Custom Name / ID) & Edit Input -->
          <div class="mt-2 min-h-[60px]">
            <div v-if="editingConnectionId !== conn.id" class="flex items-center group">
              <h2 class="card-title text-lg font-medium text-gray-700 truncate flex-grow" :title="conn.customName || conn.id">
                {{ conn.customName || conn.id }}
              </h2>
              <button
                @click="startEditingName(conn)"
                class="btn btn-ghost btn-xs ml-1 opacity-0 group-hover:opacity-100 transition-opacity"
                :aria-label="t('action.editName')"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
                  />
                </svg>
              </button>
            </div>

            <div v-else class="flex items-center gap-2">
              <input
                ref="editInputRef"
                type="text"
                v-model="editingName"
                @keyup.enter="saveEditName(conn.id)"
                @keyup.esc="cancelEditName"
                class="input input-sm flex-grow"
                :placeholder="t('placeholder.customName')"
              />
              <button @click="saveEditName(conn.id)" class="btn btn-success btn-xs" :aria-label="t('button.save')">✓</button>
              <button @click="cancelEditName" class="btn btn-ghost btn-xs" :aria-label="t('button.cancel')">✕</button>
            </div>

            <div class="text-xs text-gray-500 mt-1 truncate" :title="conn.id">ID: {{ conn.id }}</div>
          </div>

          <!-- Connection Details Grid -->
          <div class="grid grid-cols-2 gap-2 text-sm mt-3">
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.createdAt') }}</span>
              <span class="font-medium">{{ formatDateTime(conn.history.createdAt) }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.updatedAt') }}</span>
              <span class="font-medium">{{ formatDateTime(conn.history.updatedAt) }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.idleCount') }}</span>
              <span class="font-medium">{{ conn.idleCount }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.activeCount') }}</span>
              <span class="font-medium">{{ conn.activeCount }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.probingCount') }}</span>
              <span class="font-medium">{{ conn.probingCount }}</span>
            </div>
            <div class="flex flex-col">
              <span class="text-gray-500">{{ t('connection.traffic') }}</span>
              <span class="font-medium">{{ formatBytes(conn.history.totalRelayBytes) }}</span>
            </div>
          </div>

          <!-- Card Footer: Stats & Action Button -->
          <div class="mt-4 flex justify-between items-center">
            <div class="flex items-center gap-2">
              <div class="tooltip" :data-tip="t('tooltip.totalRequests')">
                <div class="badge badge-outline gap-1">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                    />
                  </svg>
                  {{ conn.history.totalRelayCount }}
                </div>
              </div>
              <div class="tooltip" :data-tip="t('tooltip.errorCount')">
                <div class="badge badge-outline gap-1" :class="{ 'text-red-500 border-red-300': conn.history.totalRelayErrCount > 0 }">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  {{ conn.history.totalRelayErrCount }}
                </div>
              </div>
            </div>

            <button
              v-if="conn.denied"
              @click="allowConnection(conn.id)"
              class="btn btn-xs btn-outline btn-success"
              :class="{ loading: isActionPending(conn.id) }"
              :disabled="isActionPending(conn.id)"
            >
              {{ t('button.allow') }}
            </button>
            <button
              v-else
              @click="closeConnection(conn.id)"
              class="btn btn-xs btn-outline btn-error"
              :class="{ loading: isActionPending(conn.id) }"
              :disabled="isActionPending(conn.id)"
            >
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

.group .btn-ghost {
  vertical-align: middle;
}
</style>
