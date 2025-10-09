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
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [currentQueryParams, setCurrentQueryParams] = useState<any>(null);
  const [deleteSuccess, setDeleteSuccess] = useState<string>('');

  const parseNaturalLanguageQuery = (question: string): any => {
    const params: any = { limit: 100 };
    const lowerQuestion = question.toLowerCase();

    // Time range parsing
    if (lowerQuestion.includes('today')) {
      params.last_days = 1;
    } else if (lowerQuestion.includes('yesterday')) {
      params.last_days = 2;
    } else if (lowerQuestion.match(/last (\d+) days?/)) {
      const match = lowerQuestion.match(/last (\d+) days?/);
      params.last_days = parseInt(match![1]);
    } else if (lowerQuestion.match(/last (\d+) hours?/)) {
      const match = lowerQuestion.match(/last (\d+) hours?/);
      params.last_hours = parseInt(match![1]);
    } else if (lowerQuestion.match(/last (\d+) minutes?/)) {
      const match = lowerQuestion.match(/last (\d+) minutes?/);
      params.last_minutes = parseInt(match![1]);
    } else if (lowerQuestion.includes('this week')) {
      params.last_days = 7;
    } else if (lowerQuestion.includes('this month')) {
      params.last_days = 30;
    }

    // Log level parsing
    if (lowerQuestion.includes('error')) {
      params.level = 'ERROR';
    } else if (lowerQuestion.includes('warning') || lowerQuestion.includes('warn')) {
      params.level = 'WARN';
    } else if (lowerQuestion.includes('info')) {
      params.level = 'INFO';
    } else if (lowerQuestion.includes('debug')) {
      params.level = 'DEBUG';
    }

    // Extract quoted text for message search
    const quotedMatch = question.match(/"([^"]+)"/);
    if (quotedMatch) {
      params.message_contains = quotedMatch[1];
    } else {
      // If no specific filters were found, search in message
      const skipWords = ['show', 'me', 'all', 'logs', 'from', 'get', 'find', 'search', 'for', 'the'];
      const words = question.toLowerCase().split(' ').filter(word =>
        !skipWords.includes(word) &&
        !word.match(/today|yesterday|error|warning|warn|info|debug|last|days?|hours?|minutes?|week|month/)
      );
      if (words.length > 0) {
        params.message_contains = words.join(' ');
      }
    }

    return params;
  };

  const handleQuery = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!question.trim()) return;

    setError('');
    setDeleteSuccess('');
    setIsLoading(true);

    try {
      // Parse natural language query into parameters
      const params = parseNaturalLanguageQuery(question);
      setCurrentQueryParams(params);

      const response = await logsService.queryLogs(params);
      setLogs(response.logs);
      setQueryInfo({
        query: question,
        count: response.total_count,
        executionTime: 0, // Not provided by backend
      });
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to query logs. Please try again.');
      setLogs([]);
      setQueryInfo(null);
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteClick = () => {
    if (!queryInfo || queryInfo.count === 0) {
      setError('No logs to delete. Please run a query first.');
      return;
    }
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = async () => {
    setShowDeleteConfirm(false);
    setError('');
    setDeleteSuccess('');
    setIsLoading(true);

    try {
      // Remove limit from params for deletion to delete all matching logs
      const deleteParams = { ...currentQueryParams };
      delete deleteParams.limit;
      delete deleteParams.offset;

      const response = await logsService.deleteLogs(deleteParams);
      setDeleteSuccess(`Successfully deleted ${response.deleted_count} log(s)`);

      // Clear the current query results
      setLogs([]);
      setQueryInfo(null);
      setQuestion('');
      setCurrentQueryParams(null);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to delete logs. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancelDelete = () => {
    setShowDeleteConfirm(false);
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

          {deleteSuccess && (
            <div className="mb-4 rounded-md bg-green-50 p-4">
              <div className="text-sm text-green-800">{deleteSuccess}</div>
            </div>
          )}

          {queryInfo && (
            <div className="mb-4 rounded-md bg-blue-50 p-4">
              <div className="flex items-center justify-between">
                <div className="text-sm text-blue-800">
                  <strong>Query:</strong> {queryInfo.query} | <strong>Results:</strong>{' '}
                  {queryInfo.count} logs found
                </div>
                <button
                  onClick={handleDeleteClick}
                  disabled={isLoading}
                  className="px-4 py-2 bg-red-600 text-white text-sm font-medium rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 disabled:bg-red-400"
                >
                  Clear These Logs
                </button>
              </div>
            </div>
          )}

          {/* Delete Confirmation Modal */}
          {showDeleteConfirm && (
            <div className="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50 flex items-center justify-center">
              <div className="relative bg-white rounded-lg shadow-xl p-6 m-4 max-w-md">
                <h3 className="text-lg font-bold text-gray-900 mb-4">Confirm Deletion</h3>
                <p className="text-sm text-gray-700 mb-6">
                  Are you sure you want to delete <strong>{queryInfo?.count} log(s)</strong> matching this query?
                  <br />
                  <span className="text-red-600 font-medium">This action cannot be undone.</span>
                </p>
                <div className="flex gap-4 justify-end">
                  <button
                    onClick={handleCancelDelete}
                    className="px-4 py-2 bg-gray-200 text-gray-800 font-medium rounded-md hover:bg-gray-300 focus:outline-none focus:ring-2 focus:ring-gray-500"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleConfirmDelete}
                    className="px-4 py-2 bg-red-600 text-white font-medium rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                  >
                    Yes, Delete
                  </button>
                </div>
              </div>
            </div>
          )}

          {logs && logs.length > 0 ? (
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
                        {log.fields && Object.keys(log.fields).length > 0 && (
                          <div className="mt-2">
                            <details className="text-xs text-gray-600">
                              <summary className="cursor-pointer font-medium">
                                Fields
                              </summary>
                              <pre className="mt-2 bg-gray-50 p-2 rounded overflow-x-auto">
                                {JSON.stringify(log.fields, null, 2)}
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
