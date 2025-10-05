import React, { useState, useEffect } from 'react';
import { logsService, Log } from '../services/logs';
import Navbar from '../components/Navbar';

const LogExplorer: React.FC = () => {
  const [logs, setLogs] = useState<Log[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [levelFilter, setLevelFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState('');
  const [limit, setLimit] = useState(50);
  const [totalCount, setTotalCount] = useState(0);

  useEffect(() => {
    fetchLogs();
  }, [levelFilter, limit]);

  const fetchLogs = async () => {
    try {
      setIsLoading(true);
      setError('');
      const response = await logsService.getLogs({
        level: levelFilter || undefined,
        limit,
      });
      setLogs(response.logs);
      setTotalCount(response.total);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch logs');
      setLogs([]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!searchQuery.trim()) {
      fetchLogs();
      return;
    }

    try {
      setIsLoading(true);
      setError('');
      const response = await logsService.queryLogs({
        message_contains: searchQuery,
        limit,
      });
      setLogs(response.logs);
      setTotalCount(response.total_count);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to search logs');
      setLogs([]);
    } finally {
      setIsLoading(false);
    }
  };

  const handleClearFilters = () => {
    setSearchQuery('');
    setLevelFilter('');
    setSourceFilter('');
    setLimit(50);
  };

  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error':
        return 'bg-red-100 text-red-800';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800';
      case 'info':
        return 'bg-blue-100 text-blue-800';
      case 'debug':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  const filteredLogs = logs.filter((log) => {
    if (sourceFilter && !log.source.toLowerCase().includes(sourceFilter.toLowerCase())) {
      return false;
    }
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Log Explorer</h2>

          {/* Search and Filters */}
          <div className="bg-white rounded-lg shadow p-6 mb-6">
            <form onSubmit={handleSearch} className="mb-4">
              <div className="flex gap-4">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search logs (e.g., 'Show me all errors from today')"
                  className="flex-1 px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
                <button
                  type="submit"
                  disabled={isLoading}
                  className="px-6 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:bg-indigo-400"
                >
                  {isLoading ? 'Searching...' : 'Search'}
                </button>
              </div>
            </form>

            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Level
                </label>
                <select
                  value={levelFilter}
                  onChange={(e) => setLevelFilter(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value="">All Levels</option>
                  <option value="error">Error</option>
                  <option value="warning">Warning</option>
                  <option value="info">Info</option>
                  <option value="debug">Debug</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Source
                </label>
                <input
                  type="text"
                  value={sourceFilter}
                  onChange={(e) => setSourceFilter(e.target.value)}
                  placeholder="Filter by source"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Limit
                </label>
                <select
                  value={limit}
                  onChange={(e) => setLimit(Number(e.target.value))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value={10}>10</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                  <option value={500}>500</option>
                </select>
              </div>

              <div className="flex items-end">
                <button
                  onClick={handleClearFilters}
                  className="w-full px-4 py-2 bg-gray-200 text-gray-800 font-medium rounded-md hover:bg-gray-300"
                >
                  Clear Filters
                </button>
              </div>
            </div>

            <div className="mt-4 text-sm text-gray-600">
              Showing {filteredLogs.length} of {totalCount} logs
            </div>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{error}</div>
            </div>
          )}

          {/* Logs List */}
          {isLoading ? (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
              <p className="mt-2 text-gray-600">Loading logs...</p>
            </div>
          ) : filteredLogs.length > 0 ? (
            <div className="bg-white shadow overflow-hidden sm:rounded-md">
              <ul className="divide-y divide-gray-200">
                {filteredLogs.map((log) => (
                  <li key={log.id} className="px-6 py-4 hover:bg-gray-50">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <span
                            className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getLevelColor(
                              log.level
                            )}`}
                          >
                            {log.level.toUpperCase()}
                          </span>
                          <span className="text-xs text-gray-500">
                            {formatTimestamp(log.timestamp)}
                          </span>
                          {log.source && (
                            <span className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                              {log.source}
                            </span>
                          )}
                        </div>
                        <p className="text-sm text-gray-900 font-mono">{log.message}</p>
                        {log.fields && Object.keys(log.fields).length > 0 && (
                          <div className="mt-2">
                            <details className="text-xs">
                              <summary className="cursor-pointer font-medium text-indigo-600 hover:text-indigo-800">
                                View Fields
                              </summary>
                              <pre className="mt-2 bg-gray-50 p-3 rounded overflow-x-auto text-xs border border-gray-200">
                                {JSON.stringify(log.fields, null, 2)}
                              </pre>
                            </details>
                          </div>
                        )}
                      </div>
                      <div className="ml-4 text-xs text-gray-400 font-mono">
                        {log.id}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <div className="text-center py-12 bg-white rounded-md shadow">
              <p className="text-gray-500">
                No logs found. Try adjusting your filters or search query.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LogExplorer;
