import React, { useState, useEffect } from 'react';
import { apiKeysService, APIKey } from '../services/apiKeys';
import Navbar from '../components/Navbar';

const APIKeys: React.FC = () => {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [isCreating, setIsCreating] = useState(false);

  useEffect(() => {
    fetchAPIKeys();
  }, []);

  const fetchAPIKeys = async () => {
    try {
      setIsLoading(true);
      const keys = await apiKeysService.getAPIKeys();
      console.log("KEYS", keys);
      setApiKeys(keys);
      setError('');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch API keys');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newKeyName.trim()) return;

    try {
      setIsCreating(true);
      const response = await apiKeysService.createAPIKey(newKeyName);
      setCreatedKey(response.api_key);
      setNewKeyName('');
      await fetchAPIKeys();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create API key');
    } finally {
      setIsCreating(false);
    }
  };

  const handleDeleteKey = async (id: number, name: string) => {
    if (!window.confirm(`Are you sure you want to delete the API key "${name}"?`)) {
      return;
    }

    try {
      await apiKeysService.deleteAPIKey(id);
      await fetchAPIKeys();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete API key');
    }
  };

  const closeCreateModal = () => {
    setShowCreateModal(false);
    setCreatedKey(null);
    setNewKeyName('');
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const maskKey = (key: string | undefined | null) => {
    if (!key) return 'N/A';
    if (key.length <= 8) return key;
    return `${key.substring(0, 4)}...${key.substring(key.length - 4)}`;
  };
  
  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">API Keys</h2>
            <button
              onClick={() => setShowCreateModal(true)}
              className="px-4 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
            >
              Create New API Key
            </button>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{error}</div>
            </div>
          )}

          {isLoading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
              <p className="mt-2 text-gray-600">Loading API keys...</p>
            </div>
          ) : apiKeys.length > 0 ? (
            <div className="bg-white shadow overflow-hidden sm:rounded-md">
              <ul className="divide-y divide-gray-200">
                {apiKeys.map((apiKey) => {
                  if (!apiKey || !apiKey.api_key) {
                    console.warn('Invalid API key data:', apiKey);
                    return null; // Skip rendering this key
                  }
                  return (
                    <li key={apiKey.id} className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <h3 className="text-sm font-medium text-gray-900">{apiKey.name}</h3>
                        <div className="mt-1 flex items-center gap-4">
                          <code className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                            {maskKey(apiKey.api_key)}
                          </code>
                          <span className="text-xs text-gray-500">
                            Created: {formatDate(apiKey.created_at)}
                          </span>
                          {apiKey.last_used_at && (
                            <span className="text-xs text-gray-500">
                              Last used: {formatDate(apiKey.last_used_at)}
                            </span>
                          )}
                        </div>
                      </div>
                      <button
                        onClick={() => handleDeleteKey(apiKey.id, apiKey.name)}
                        className="ml-4 px-3 py-1 text-sm text-red-600 hover:text-red-800 hover:bg-red-50 rounded-md"
                      >
                        Delete
                      </button>
                    </div>
                  </li>
                  )
                })}
              </ul>
            </div>
          ) : (
            <div className="text-center py-12 bg-white rounded-md shadow">
              <p className="text-gray-500">No API keys found. Create one to get started.</p>
            </div>
          )}
        </div>
      </div>

      {/* Create API Key Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
            {createdKey ? (
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">API Key Created!</h3>
                <div className="mb-4 p-4 bg-yellow-50 border border-yellow-200 rounded-md">
                  <p className="text-sm text-yellow-800 mb-2">
                    <strong>Important:</strong> Copy this API key now. You won't be able to see it again!
                  </p>
                  <div className="mt-2 p-2 bg-white rounded border border-yellow-300">
                    <code className="text-xs break-all">{createdKey}</code>
                  </div>
                  <button
                    onClick={() => copyToClipboard(createdKey)}
                    className="mt-2 w-full px-3 py-2 bg-indigo-600 text-white text-sm rounded-md hover:bg-indigo-700"
                  >
                    Copy to Clipboard
                  </button>
                </div>
                <button
                  onClick={closeCreateModal}
                  className="w-full px-4 py-2 bg-gray-200 text-gray-800 font-medium rounded-md hover:bg-gray-300"
                >
                  Close
                </button>
              </div>
            ) : (
              <form onSubmit={handleCreateKey}>
                <h3 className="text-lg font-medium text-gray-900 mb-4">Create New API Key</h3>
                <div className="mb-4">
                  <label htmlFor="keyName" className="block text-sm font-medium text-gray-700 mb-2">
                    Key Name
                  </label>
                  <input
                    type="text"
                    id="keyName"
                    value={newKeyName}
                    onChange={(e) => setNewKeyName(e.target.value)}
                    placeholder="e.g., Production Server"
                    className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    required
                  />
                </div>
                <div className="flex gap-3">
                  <button
                    type="button"
                    onClick={closeCreateModal}
                    className="flex-1 px-4 py-2 bg-gray-200 text-gray-800 font-medium rounded-md hover:bg-gray-300"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={isCreating}
                    className="flex-1 px-4 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700 disabled:bg-indigo-400"
                  >
                    {isCreating ? 'Creating...' : 'Create'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default APIKeys;
