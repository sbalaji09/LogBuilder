import api from './api';

export interface Log {
  id: string;
  user_id: number;
  level: string;
  message: string;
  timestamp: string;
  source: string;
  metadata?: Record<string, any>;
}

export interface QueryRequest {
  question: string;
}

export interface QueryResponse {
  logs: Log[];
  query: string;
  count: number;
  execution_time_ms: number;
}

export const logsService = {
  async queryLogs(question: string): Promise<QueryResponse> {
    const response = await api.post<QueryResponse>('/logs/query', {
      question,
    });
    return response.data;
  },

  async getLogs(params?: {
    level?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ logs: Log[]; total: number }> {
    const response = await api.get('/logs', { params });
    return response.data;
  },
};
