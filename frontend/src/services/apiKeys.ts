import api from './api';

export interface APIKey {
  id: number;
  user_id: number;
  key: string;
  name: string;
  last_used_at?: string;
  created_at: string;
}

export interface CreateAPIKeyRequest {
  name: string;
}

export interface CreateAPIKeyResponse {
  id: number;
  key: string;
  name: string;
  created_at: string;
}

export const apiKeysService = {
  async createAPIKey(name: string): Promise<CreateAPIKeyResponse> {
    const response = await api.post<CreateAPIKeyResponse>('/api-keys', { name });
    return response.data;
  },

  async getAPIKeys(): Promise<APIKey[]> {
    const response = await api.get<{ api_keys: APIKey[] }>('/api-keys');
    return response.data.api_keys || [];
  },

  async deleteAPIKey(id: number): Promise<void> {
    await api.delete(`/api-keys/${id}`);
  },
};
