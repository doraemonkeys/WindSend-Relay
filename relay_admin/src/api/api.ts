import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosError } from 'axios';

export interface PageInfo {
  page: number;
  pageSize: number;
}

export interface PageInfoSort {
  sortBy?: string;              // Optional: field name, empty means no sort
  sortType?: 'asc' | 'desc'; // Optional: sort direction
}


export interface PaginatedData<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}


export interface HistoryStatistic {
  id: string;
  customName: string;
  createdAt: string | Date;
  updatedAt: string | Date;
  totalRelayCount: number;
  totalRelayErrCount: number;
  totalRelayOfflineCount: number;
  totalRelayMs: number;
  totalRelayBytes: number;
}


export interface ActiveConnection {
  id: string;
  customName: string;
  reqAddr: string;
  connectTime: string | Date;
  lastActive: string | Date;
  relaying: boolean;
  history: HistoryStatistic;
}


export interface ReqHistoryStatistic extends PageInfo, PageInfoSort { }


export type RespHistoryStatistic = PaginatedData<HistoryStatistic>;


export interface LoginResponse {
  token?: string;
}


export interface LoginRequest {
  username: string;
  password: string;
}


export interface ReqUpdateConnection {
  id: string;
  customName: string;
}


export class ApiClient {
  private axiosInstance: AxiosInstance;
  getAuthToken: (() => string | null) = () => null;
  onTokenChange: ((token: string | null) => void) | null = null;
  onUnauthorized: (() => void) | null = null;

  constructor(baseURL: string, config?: AxiosRequestConfig) {
    this.axiosInstance = axios.create({
      baseURL: baseURL,
      ...config,
    });

    // Optional: Interceptor to automatically add the auth token
    this.axiosInstance.interceptors.request.use((config) => {
      if (this.getAuthToken()) {
        config.headers = config.headers ?? {};
        config.headers.Authorization = `Bearer ${this.getAuthToken()}`; // Adjust if using a different scheme
      }
      return config;
    });

    // Optional: Add response interceptor for centralized error handling
    this.axiosInstance.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        // Handle errors globally here if needed
        console.error('API call failed:', error.response?.status, error.message);
        // You might want to check for 401 Unauthorized and trigger logout/re-login
        if (error.response?.status === 401) {
          this.clearAuthToken();
        }
        return Promise.reject(error);
      }
    );
  }

  /**
   * Sets the authentication token to be used for subsequent requests.
   * @param token The authentication token (e.g., JWT). Pass null to clear.
   */
  setAuthToken(token: string | null): void {
    if (this.onTokenChange) {
      this.onTokenChange(token);
    }
  }

  /**
   * Clears the stored authentication token.
   */
  clearAuthToken(): void {
    if (this.onUnauthorized) {
      this.onUnauthorized();
    }
    if (this.onTokenChange) {
      this.onTokenChange(null);
    }
  }


  async login(params: LoginRequest): Promise<LoginResponse> {
    try {
      const formData = new FormData();
      formData.append('username', params.username);
      formData.append('password', params.password);
      const response = await this.axiosInstance.post<LoginResponse>('/login', formData);
      const token = response.data.token;
      if (token) {
        this.setAuthToken(token);
      }
      return response.data;
    } catch (error) {
      console.error('Login failed:', error);
      throw error;
    }
  }

  /**
   * Fetches paginated connection statistics. Requires authentication.
   * Corresponds to GET /api/conn/statistic
   * @param params Pagination and sorting parameters (ReqHistoryStatistic).
   * @param sortBy: string;  totalRelayCount | totalRelayMs | totalRelayBytes etc.
   * @returns Promise resolving to paginated history statistics.
   */
  async getConnectionStatistic(params: ReqHistoryStatistic): Promise<RespHistoryStatistic> {
    if (!this.getAuthToken()) { // Basic check, relies on server for actual auth
      console.warn('getConnectionStatistic called without an auth token set.');
      // Or throw new Error('Authentication token not set');
    }
    try {
      const response = await this.axiosInstance.get<RespHistoryStatistic>('/conn/statistic', {
        params: params, // Axios automatically handles query parameters for GET
      });
      return response.data;
    } catch (error) {
      console.error('Failed to get connection statistics:', error);
      throw error;
    }
  }

  /**
   * Fetches the status of active connections. Requires authentication.
   * Corresponds to GET /api/conn/status
   * Assumes it returns an array of ActiveConnection. Adjust if the structure differs.
   * @returns Promise resolving to an array of active connections.
   */
  async getConnectionStatus(): Promise<ActiveConnection[]> {
    if (!this.getAuthToken()) {
      console.warn('getConnectionStatus called without an auth token set.');
    }
    try {
      // Assuming the endpoint returns a raw array of connections
      const response = await this.axiosInstance.get<ActiveConnection[]>('/conn/status');
      // Optional: Parse date strings to Date objects if needed
      // return response.data.map(conn => ({
      //   ...conn,
      //   connectTime: new Date(conn.connectTime),
      //   lastActive: new Date(conn.lastActive),
      // }));
      return response.data;
    } catch (error) {
      console.error('Failed to get connection status:', error);
      throw error;
    }
  }

  /**
   * Closes a specific connection by its ID. Requires authentication.
   * Corresponds to GET /api/conn/close/:id
   * @param id The ID of the connection to close.
   * @returns Promise resolving when the request is complete (void).
   */
  async closeConnection(id: string): Promise<void> {
    if (!this.getAuthToken()) {
      console.warn('closeConnection called without an auth token set.');
    }
    try {
      // Using GET as defined in the router, though DELETE might be more conventional
      await this.axiosInstance.get(`/conn/close/${id}`);
      // No return value expected on success typically
    } catch (error) {
      console.error(`Failed to close connection ${id}:`, error);
      throw error;
    }
  }

  async updateConnectionName(id: string, customName: string): Promise<void> {
    try {
      const params: ReqUpdateConnection = {
        id,
        customName,
      }
      await this.axiosInstance.post('/conn/update', params);
    } catch (error) {
      console.error(`Failed to update connection ${id}:`, error);
      throw error;
    }
  }
}


export const apiClient = new ApiClient('/api');
// export const apiClient = new ApiClient('http://127.0.0.1:16780/api');
