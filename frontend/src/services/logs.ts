import api from './api';

export interface Log {
  id: number;
  user_id: number;
  level: string;
  message: string;
  timestamp: string;
  source: string;
  service?: string;
  fields?: Record<string, any>;
  raw_message?: string;
  created_at: string;
}

export interface QueryRequest {
  level?: string;
  source?: string;
  service?: string;
  message_contains?: string;
  levels?: string[];
  sources?: string[];
  start_time?: string;
  end_time?: string;
  last_minutes?: number;
  last_hours?: number;
  last_days?: number;
  limit?: number;
  offset?: number;
  sort_by?: string;
  sort_order?: string;
}

export interface QueryResponse {
  logs: Log[];
  total_count: number;
  limit: number;
  offset: number;
  executed_at: string;
}

export const logsService = {
  async queryLogs(params: QueryRequest): Promise<QueryResponse> {
    const response = await api.post<QueryResponse>('/logs/query', params);
    return response.data;
  },

  async getLogs(params?: {
    level?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ logs: Log[]; total: number }> {
    // Use the query endpoint with filters
    const queryParams: QueryRequest = {
      limit: params?.limit || 100,
      offset: params?.offset || 0,
    };

    if (params?.level) {
      queryParams.level = params.level.toUpperCase();
    }

    const response = await api.post<QueryResponse>('/logs/query', queryParams);
    return {
      logs: response.data.logs,
      total: response.data.total_count,
    };
  },

  async getRecentLogs(): Promise<Log[]> {
    const response = await api.get<{ logs: Log[]; count: number }>('/logs/recent');
    return response.data.logs;
  },
};
