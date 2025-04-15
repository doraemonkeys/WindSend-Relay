<template>
  <div class="container mx-auto px-4 py-8">
    <!-- Back to Home Button -->
    <div class="mb-4">
      <router-link to="/" class="btn btn-sm btn-outline inline-flex items-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24"
          stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        {{ t('statisticView.backToHome') }}
      </router-link>
    </div>

    <!-- 页面标题 -->
    <div class="mb-8 text-center">
      <h1 class="text-3xl font-bold text-primary">{{ t('statisticView.title') }}</h1>
      <p class="text-gray-500 mt-2">{{ t('statisticView.subtitle') }}</p>
    </div>

    <!-- 筛选和排序控制 -->
    <div class="bg-base-100 rounded-xl shadow-md p-6 mb-6">
      <div class="flex flex-wrap gap-4 items-end">
        <!-- Sort By -->
        <div class="form-control">
          <label class="label">
            <span class="label-text text-sm font-medium">{{ t('statisticView.sort.fieldLabel') }}</span>
          </label>
          <select v-model="sortBy" class="select select-bordered select-sm w-full max-w-xs">
            <option value="">{{ t('statisticView.sort.options.default') }}</option>
            <option v-for="option in sortOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>

        <!-- Sort Order -->
        <div class="form-control">
          <label class="label">
            <span class="label-text text-sm font-medium">{{ t('statisticView.sort.orderLabel') }}</span>
          </label>
          <select v-model="sortType" class="select select-bordered select-sm w-full max-w-xs">
            <option value="asc">{{ t('statisticView.sort.orderOptions.asc') }}</option>
            <option value="desc">{{ t('statisticView.sort.orderOptions.desc') }}</option>
          </select>
        </div>

        <!-- Page Size -->
        <div class="form-control">
          <label class="label">
            <span class="label-text text-sm font-medium">{{ t('statisticView.pageSizeLabel') }}</span>
          </label>
          <select v-model="pageSize" class="select select-bordered select-sm w-full max-w-xs">
            <option v-for="size in pageSizeOptions" :key="size" :value="size">
              {{ t('statisticView.pageSizeOption', { count: size }) }}
            </option>
          </select>
        </div>

        <!-- Refresh Button -->
        <button @click="fetchData" class="btn btn-primary btn-sm">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 mr-1" fill="none" viewBox="0 0 24 24"
            stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          {{ t('statisticView.refreshData') }}
        </button>
      </div>
    </div>

    <!-- 数据卡片概览 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-6">
      <!-- Total Records -->
      <div class="stat bg-base-100 rounded-xl shadow-md">
        <div class="stat-figure text-primary">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
        </div>
        <div class="stat-title text-xs">{{ t('statisticView.stats.totalRecords') }}</div>
        <div class="stat-value text-primary">{{ statistics.total }}</div>
      </div>

      <!-- Total Relays -->
      <div class="stat bg-base-100 rounded-xl shadow-md">
        <div class="stat-figure text-secondary">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </div>
        <div class="stat-title text-xs">{{ t('statisticView.stats.totalRelayCount') }}</div>
        <div class="stat-value text-secondary">{{ totalStats.connections }}</div>
      </div>

      <!-- Total Errors -->
      <div class="stat bg-base-100 rounded-xl shadow-md">
        <div class="stat-figure text-error">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div class="stat-title text-xs">{{ t('statisticView.stats.totalErrors') }}</div>
        <div class="stat-value text-error">{{ totalStats.errors }}</div>
      </div>

      <!-- Total Offline -->
      <div class="stat bg-base-100 rounded-xl shadow-md">
        <div class="stat-figure text-info">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div class="stat-title text-xs">{{ t('statisticView.stats.totalOffline') }}</div>
        <div class="stat-value text-info">{{ totalStats.offline }}</div>
      </div>

      <!-- Total Traffic -->
      <div class="stat bg-base-100 rounded-xl shadow-md">
        <div class="stat-figure text-success">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
        </div>
        <div class="stat-title text-xs">{{ t('statisticView.stats.totalTraffic') }}</div>
        <div class="stat-value text-success">{{ formatBytes(totalStats.bytes) }}</div>
      </div>
    </div>

    <!-- 数据表格 -->
    <div class="bg-base-100 rounded-xl shadow-md overflow-x-auto">
      <table class="table w-full">
        <thead>
          <tr class="bg-base-200">
            <th class="text-xs">{{ t('statisticView.table.header.id') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.createdAt') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.updatedAt') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.relayCount') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.errorCount') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.offlineCount') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.totalDuration') }}</th>
            <th class="text-xs">{{ t('statisticView.table.header.totalTraffic') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="loading" class="hover">
            <td colspan="8" class="text-center py-8">
              <span class="loading loading-spinner loading-md"></span>
            </td>
          </tr>
          <tr v-else-if="statistics.list.length === 0">
            <td colspan="8" class="text-center py-8 text-gray-500">{{ t('statisticView.table.empty') }}</td>
          </tr>
          <tr v-else v-for="item in statistics.list" :key="item.id" class="hover">
            <td class="text-xs font-mono">{{ truncateId(item.id) }}</td>
            <td class="text-xs">{{ formatDate(item.createdAt) }}</td>
            <td class="text-xs">{{ formatDate(item.updatedAt) }}</td>
            <td class="text-xs">{{ item.totalRelayCount }}</td>
            <td class="text-xs">
              <span :class="item.totalRelayErrCount > 0 ? 'text-error' : ''">
                {{ item.totalRelayErrCount }}
              </span>
            </td>
            <td class="text-xs">
              <span :class="item.totalRelayOfflineCount > 0 ? 'text-warning' : ''">
                {{ item.totalRelayOfflineCount }}
              </span>
            </td>
            <td class="text-xs">{{ formatDuration(item.totalRelayMs) }}</td>
            <td class="text-xs">{{ formatBytes(item.totalRelayBytes) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 分页控制 -->
    <div class="flex justify-between items-center mt-6">
      <div class="text-sm text-gray-500">
        {{ t('statisticView.pagination.summary', {
          total: statistics.total,
          start: statistics.total === 0 ? 0 : (statistics.page - 1) * statistics.pageSize + 1,
          end: Math.min(statistics.page * statistics.pageSize, statistics.total)
        })
        }}
      </div>
      <div class="join">
        <button class="join-item btn btn-sm" :class="{ 'btn-disabled': statistics.page <= 1 }"
          @click="changePage(statistics.page - 1)">
          «
        </button>
        <button class="join-item btn btn-sm">{{ statistics.page }}</button>
        <button class="join-item btn btn-sm"
          :class="{ 'btn-disabled': statistics.page * statistics.pageSize >= statistics.total }"
          @click="changePage(statistics.page + 1)">
          »
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n'; // Import useI18n
import { apiClient, type RespHistoryStatistic } from '@/api/api';
import { formatBytes, formatDuration } from '@/utils/utils';

const { t, locale } = useI18n(); // Instantiate t and locale

// Loading state
const loading = ref(false);

// 分页和排序状态
const page = ref(1);
const pageSize = ref(10);
const sortBy = ref('');
const sortType = ref<'asc' | 'desc'>('desc');

// Options for dropdowns (reactive to locale changes)
const sortOptions = computed(() => [
  { value: 'totalRelayCount', label: t('statisticView.sort.options.totalRelayCount') },
  { value: 'totalRelayErrCount', label: t('statisticView.sort.options.totalRelayErrCount') },
  { value: 'totalRelayOfflineCount', label: t('statisticView.sort.options.totalRelayOfflineCount') },
  { value: 'totalRelayMs', label: t('statisticView.sort.options.totalRelayMs') },
  { value: 'totalRelayBytes', label: t('statisticView.sort.options.totalRelayBytes') },
  { value: 'createdAt', label: t('statisticView.sort.options.createdAt') }
]);

const pageSizeOptions = ref([10, 20, 50, 100]);

// 统计数据
const statistics = ref<RespHistoryStatistic>({
  list: [],
  total: 0,
  page: 1,
  pageSize: 10
});

// 计算总计数据
const totalStats = computed(() => {
  // Calculate totals only if list is not empty to avoid summing initial zeros
  if (!statistics.value.list || statistics.value.list.length === 0) {
    return { connections: 0, errors: 0, offline: 0, bytes: 0, ms: 0 };
  }
  return statistics.value.list.reduce((acc, curr) => {
    return {
      connections: acc.connections + (curr.totalRelayCount || 0),
      errors: acc.errors + (curr.totalRelayErrCount || 0),
      offline: acc.offline + (curr.totalRelayOfflineCount || 0),
      bytes: acc.bytes + (curr.totalRelayBytes || 0),
      ms: acc.ms + (curr.totalRelayMs || 0)
    };
  }, { connections: 0, errors: 0, offline: 0, bytes: 0, ms: 0 });
});

// 加载数据
const fetchData = async () => {
  loading.value = true; // Start loading
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      sortBy: sortBy.value,
      sortType: sortType.value
    };

    statistics.value = await apiClient.getConnectionStatistic(params);
  } catch (error) {
    console.error('Failed to fetch statistics:', error);
    // Reset data on error to avoid issues with computed properties or templates
    statistics.value = { list: [], total: 0, page: 1, pageSize: 10 };
    // Optionally: show an error message to the user using a toast notification system
    // toast.error(t('error.fetchConnections')); // Assuming you have a toast system
  } finally {
    loading.value = false; // End loading
  }
};

// 切换页码
const changePage = (newPage: number) => {
  if (newPage < 1 || (statistics.value.total > 0 && newPage > Math.ceil(statistics.value.total / pageSize.value))) {
    return;
  }
  page.value = newPage;
  fetchData();
};

// 格式化日期 based on current locale
const formatDate = (dateStr: string | Date | undefined | null): string => {
  if (!dateStr) return 'N/A'; // Handle null or undefined dates
  try {
    const date = new Date(dateStr);
    // Check if date is valid
    if (isNaN(date.getTime())) {
      return 'Invalid Date';
    }
    // Use locale from useI18n for dynamic formatting
    return date.toLocaleString(locale.value, { // Use current locale
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false // Use 24-hour format if preferred
    }); //.replace(/\//g, '-'); // Optional: Replace slashes with dashes - localeString handles this better
  } catch (e) {
    console.error("Error formatting date:", dateStr, e);
    return 'Date Error';
  }
};

// 截断ID显示
const truncateId = (id: string | undefined | null): string => {
  if (!id) return 'N/A';
  if (id.length <= 8) return id;
  return id.substring(0, 8) + '...';
};

// 监听排序和分页变化
watch([sortBy, sortType, pageSize], () => {
  page.value = 1; // Reset to first page when sorting or page size changes
  fetchData();
});

// 组件挂载时加载数据
onMounted(() => {
  fetchData();
});

// Watch for locale changes to potentially re-fetch or re-format if needed
// (formatDate already uses locale, so it will update automatically)
// watch(locale, () => {
//   // Optionally re-fetch if data itself is locale-dependent, though unlikely here
//   // Or trigger re-computation of locale-dependent computed properties if any
// });

</script>

<style scoped>
.stat {
  transition: all 0.3s ease;
}

.stat:hover {
  transform: translateY(-3px);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
}

.table thead tr {
  /* Ensure header background applies correctly */
  background-color: var(--b2, hsl(var(--b2) / var(--tw-bg-opacity, 1)));
}

.table tbody tr {
  transition: background-color 0.2s ease;
}

/* Ensure table cells handle text overflow gracefully */
.table td,
.table th {
  /* Optional: add padding or other styles as needed */
  white-space: nowrap;
  /* Prevent text wrapping if desired */
  /* overflow: hidden; */
  /* Consider if text should be hidden */
  /* text-overflow: ellipsis; */
  /* Add ellipsis for overflow */
}
</style>