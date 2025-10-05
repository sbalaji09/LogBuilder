import React, { useState, useEffect, useRef } from 'react';
import { logsService, Log } from '../services/logs';
import Navbar from '../components/Navbar';

const LiveLogs: React.FC = () => {
  const [logs, setLogs] = useState<Log[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState('');
  const [autoScroll, setAutoScroll] = useState(true);
  const [maxLogs, setMaxLogs] = useState(100);
  const [levelFilter, setLevelFilter] = useState('');
  const logsEndRef = useRef<HTMLDivElement>(null);
  const pollInterval = useRef<NodeJS.Timeout | null>(null);

  const scrollToBottom = () => {
    if (autoScroll) {
      logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  };

  useEffect(() => {
    scrollToBottom();
  }, [logs]);

  const fetchRecentLogs = async () => {
    try {
      const response = await logsService.getLogs({
        level: levelFilter || undefined,
        limit: 20,
      });

      setLogs((prevLogs) => {
        const newLogs = [...response.logs, ...prevLogs];
        // Keep only the most recent logs up to maxLogs
        return newLogs.slice(0, maxLogs);
      });
      setError('');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch logs');
    }
  };

  const startStreaming = () => {
    setIsStreaming(true);
    setLogs([]);

    // Fetch initial logs
    fetchRecentLogs();

    // Poll for new logs every 2 seconds
    pollInterval.current = setInterval(() => {
      fetchRecentLogs();
    }, 2000);
  };

  const stopStreaming = () => {
    setIsStreaming(false);
    if (pollInterval.current) {
      clearInterval(pollInterval.current);
      pollInterval.current = null;
    }
  };

  useEffect(() => {
    return () => {
      if (pollInterval.current) {
        clearInterval(pollInterval.current);
      }
    };
  }, []);

  const handleClearLogs = () => {
    setLogs([]);
  };

  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case 'error':
        return 'bg-red-100 text-red-800 border-red-300';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'info':
        return 'bg-blue-100 text-blue-800 border-blue-300';
      case 'debug':
        return 'bg-gray-100 text-gray-800 border-gray-300';
      default:
        return 'bg-gray-100 text-gray-800 border-gray-300';
    }
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString() + '.' + date.getMilliseconds().toString().padStart(3, '0');
  };

  const filteredLogs = logs.filter((log) => {
    if (levelFilter && log.level.toLowerCase() !== levelFilter.toLowerCase()) {
      return false;
    }
    return true;
  });

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Live Logs</h2>
            <div className="flex gap-3">
              {isStreaming ? (
                <>
                  <div className="flex items-center gap-2 px-3 py-2 bg-green-100 text-green-800 rounded-md">
                    <div className="w-2 h-2 bg-green-600 rounded-full animate-pulse"></div>
                    <span className="text-sm font-medium">Streaming</span>
                  </div>
                  <button
                    onClick={stopStreaming}
                    className="px-4 py-2 bg-red-600 text-white font-medium rounded-md hover:bg-red-700"
                  >
                    Stop
                  </button>
                </>
              ) : (
                <button
                  onClick={startStreaming}
                  className="px-4 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700"
                >
                  Start Streaming
                </button>
              )}
            </div>
          </div>

          {/* Controls */}
          <div className="bg-white rounded-lg shadow p-4 mb-4">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Level Filter
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
                  Max Logs
                </label>
                <select
                  value={maxLogs}
                  onChange={(e) => setMaxLogs(Number(e.target.value))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500"
                >
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                  <option value={200}>200</option>
                  <option value={500}>500</option>
                </select>
              </div>

              <div className="flex items-end">
                <label className="flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={autoScroll}
                    onChange={(e) => setAutoScroll(e.target.checked)}
                    className="mr-2 w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                  />
                  <span className="text-sm text-gray-700">Auto-scroll</span>
                </label>
              </div>

              <div className="flex items-end">
                <button
                  onClick={handleClearLogs}
                  className="w-full px-4 py-2 bg-gray-200 text-gray-800 font-medium rounded-md hover:bg-gray-300"
                >
                  Clear Logs
                </button>
              </div>
            </div>

            <div className="mt-2 text-sm text-gray-600">
              {filteredLogs.length} logs displayed
            </div>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{error}</div>
            </div>
          )}

          {/* Logs Console */}
          <div className="bg-gray-900 rounded-lg shadow-lg overflow-hidden">
            <div className="h-[600px] overflow-y-auto p-4 font-mono text-sm">
              {filteredLogs.length > 0 ? (
                <div className="space-y-1">
                  {filteredLogs.map((log, index) => (
                    <div
                      key={`${log.id}-${index}`}
                      className={`flex items-start gap-3 p-2 rounded border ${getLevelColor(log.level)}`}
                    >
                      <span className="text-xs opacity-75 whitespace-nowrap">
                        {formatTimestamp(log.timestamp)}
                      </span>
                      <span className="text-xs font-bold uppercase whitespace-nowrap">
                        [{log.level}]
                      </span>
                      {log.source && (
                        <span className="text-xs opacity-75 whitespace-nowrap">
                          {log.source}:
                        </span>
                      )}
                      <span className="text-xs flex-1 break-words">
                        {log.message}
                      </span>
                    </div>
                  ))}
                  <div ref={logsEndRef} />
                </div>
              ) : (
                <div className="text-center py-12 text-gray-500">
                  {isStreaming
                    ? 'Waiting for logs...'
                    : 'Click "Start Streaming" to begin monitoring logs in real-time'}
                </div>
              )}
            </div>
          </div>

          {isStreaming && (
            <div className="mt-4 text-center text-sm text-gray-500">
              Refreshing every 2 seconds...
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default LiveLogs;
