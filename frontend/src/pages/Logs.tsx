import React, { useState } from 'react';
import { logsService, Log } from '../services/logs';
import Navbar from '../components/Navbar';

const Logs: React.FC = () => {
  const [question, setQuestion] = useState('');
  const [logs, setLogs] = useState<Log[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [queryInfo, setQueryInfo] = useState<{
    query: string;
    count: number;
    executionTime: number;
  } | null>(null);

  const handleQuery = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!question.trim()) return;

    setError('');
    setIsLoading(true);

    try {
      const response = await logsService.queryLogs(question);
      setLogs(response.logs);
      setQueryInfo({
        query: response.query,
        count: response.count,
        executionTime: response.execution_time_ms,
      });
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to query logs. Please try again.');
      setLogs([]);
      setQueryInfo(null);
    } finally {
      setIsLoading(false);
    }
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

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="mb-6">
            <h2 className="text-2xl font-bold text-gray-900 mb-4">Query Logs</h2>
            <form onSubmit={handleQuery} className="flex gap-4">
              <input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                placeholder="Ask a question about your logs (e.g., 'Show me all errors from today')"
                className="flex-1 px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
              />
              <button
                type="submit"
                disabled={isLoading}
                className="px-6 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 disabled:bg-indigo-400"
              >
                {isLoading ? 'Searching...' : 'Search'}
              </button>
            </form>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{error}</div>
            </div>
          )}

          {queryInfo && (
            <div className="mb-4 rounded-md bg-green-50 p-4">
              <div className="text-sm text-green-800">
                <strong>Query:</strong> {queryInfo.query} | <strong>Results:</strong>{' '}
                {queryInfo.count} | <strong>Time:</strong> {queryInfo.executionTime}ms
              </div>
            </div>
          )}

          {logs.length > 0 ? (
            <div className="bg-white shadow overflow-hidden sm:rounded-md">
              <ul className="divide-y divide-gray-200">
                {logs.map((log) => (
                  <li key={log.id} className="px-6 py-4 hover:bg-gray-50">
                    <div className="flex items-center justify-between">
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
                        <p className="text-sm text-gray-900">{log.message}</p>
                        {log.metadata && Object.keys(log.metadata).length > 0 && (
                          <div className="mt-2">
                            <details className="text-xs text-gray-600">
                              <summary className="cursor-pointer font-medium">
                                Metadata
                              </summary>
                              <pre className="mt-2 bg-gray-50 p-2 rounded overflow-x-auto">
                                {JSON.stringify(log.metadata, null, 2)}
                              </pre>
                            </details>
                          </div>
                        )}
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            !isLoading && (
              <div className="text-center py-12 bg-white rounded-md shadow">
                <p className="text-gray-500">
                  {queryInfo
                    ? 'No logs found matching your query.'
                    : 'Enter a query above to search your logs.'}
                </p>
              </div>
            )
          )}

          {isLoading && (
            <div className="text-center py-12">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
              <p className="mt-2 text-gray-600">Searching logs...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Logs;
