import React, { useState, useEffect } from 'react';
import { statsService, LogStats } from '../services/stats';
import Navbar from '../components/Navbar';
import { useNavigate } from 'react-router-dom';

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<LogStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    try {
      setIsLoading(true);
      const data = await statsService.getStats();
      setStats(data);
      setError('');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to fetch statistics');
    } finally {
      setIsLoading(false);
    }
  };

  const formatNumber = (num: number) => {
    return num.toLocaleString();
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleString();
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50">
        <Navbar />
        <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
            <p className="mt-2 text-gray-600">Loading dashboard...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <Navbar />
      <div className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <div className="px-4 py-6 sm:px-0">
          <div className="flex justify-between items-center mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Dashboard</h2>
            <button
              onClick={fetchStats}
              className="px-4 py-2 bg-indigo-600 text-white font-medium rounded-md hover:bg-indigo-700"
            >
              Refresh
            </button>
          </div>

          {error && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="text-sm text-red-800">{error}</div>
            </div>
          )}

          {stats && (
            <>
              {/* Overview Stats */}
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                <div className="bg-white rounded-lg shadow p-6">
                  <div className="text-sm font-medium text-gray-500 mb-1">Total Logs</div>
                  <div className="text-3xl font-bold text-gray-900">
                    {formatNumber(stats.total_logs)}
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow p-6">
                  <div className="text-sm font-medium text-gray-500 mb-1">Logs Today</div>
                  <div className="text-3xl font-bold text-blue-600">
                    {formatNumber(stats.logs_today)}
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow p-6">
                  <div className="text-sm font-medium text-gray-500 mb-1">This Week</div>
                  <div className="text-3xl font-bold text-indigo-600">
                    {formatNumber(stats.logs_this_week)}
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow p-6">
                  <div className="text-sm font-medium text-gray-500 mb-1">Errors</div>
                  <div className="text-3xl font-bold text-red-600">
                    {formatNumber(stats.error_count)}
                  </div>
                </div>
              </div>

              {/* Log Levels Breakdown */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
                <div className="bg-white rounded-lg shadow p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">
                    Logs by Level
                  </h3>
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 bg-red-500 rounded-full"></div>
                        <span className="text-sm text-gray-700">Error</span>
                      </div>
                      <span className="text-sm font-medium text-gray-900">
                        {formatNumber(stats.error_count)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 bg-yellow-500 rounded-full"></div>
                        <span className="text-sm text-gray-700">Warning</span>
                      </div>
                      <span className="text-sm font-medium text-gray-900">
                        {formatNumber(stats.warning_count)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 bg-blue-500 rounded-full"></div>
                        <span className="text-sm text-gray-700">Info</span>
                      </div>
                      <span className="text-sm font-medium text-gray-900">
                        {formatNumber(stats.info_count)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <div className="w-3 h-3 bg-gray-500 rounded-full"></div>
                        <span className="text-sm text-gray-700">Debug</span>
                      </div>
                      <span className="text-sm font-medium text-gray-900">
                        {formatNumber(stats.debug_count)}
                      </span>
                    </div>
                  </div>
                </div>

                {/* Top Sources */}
                <div className="bg-white rounded-lg shadow p-6">
                  <h3 className="text-lg font-semibold text-gray-900 mb-4">
                    Top Sources
                  </h3>
                  {stats.top_sources.length > 0 ? (
                    <div className="space-y-3">
                      {stats.top_sources.map((source, index) => (
                        <div key={index} className="flex items-center justify-between">
                          <span className="text-sm text-gray-700 truncate flex-1">
                            {source.source || 'Unknown'}
                          </span>
                          <span className="text-sm font-medium text-gray-900 ml-3">
                            {formatNumber(source.count)}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500">No data available</p>
                  )}
                </div>
              </div>

              {/* Recent Errors */}
              <div className="bg-white rounded-lg shadow overflow-hidden">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h3 className="text-lg font-semibold text-gray-900">Recent Errors</h3>
                </div>
                {stats.recent_errors.length > 0 ? (
                  <ul className="divide-y divide-gray-200">
                    {stats.recent_errors.map((error) => (
                      <li
                        key={error.id}
                        className="px-6 py-4 hover:bg-gray-50 cursor-pointer"
                        onClick={() => navigate('/explorer')}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <p className="text-sm text-gray-900 font-mono">
                              {error.message}
                            </p>
                            <div className="mt-1 flex items-center gap-3">
                              <span className="text-xs text-gray-500">
                                {formatTimestamp(error.timestamp)}
                              </span>
                              {error.source && (
                                <span className="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded">
                                  {error.source}
                                </span>
                              )}
                            </div>
                          </div>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="px-6 py-8 text-center text-gray-500">
                    No recent errors - all systems running smoothly!
                  </div>
                )}
              </div>

              {/* Quick Actions */}
              <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
                <button
                  onClick={() => navigate('/explorer')}
                  className="px-6 py-4 bg-white border border-gray-300 rounded-lg shadow-sm hover:shadow-md transition-shadow text-left"
                >
                  <div className="text-sm font-medium text-gray-500">View All Logs</div>
                  <div className="mt-1 text-lg font-semibold text-indigo-600">
                    Log Explorer →
                  </div>
                </button>

                <button
                  onClick={() => navigate('/live')}
                  className="px-6 py-4 bg-white border border-gray-300 rounded-lg shadow-sm hover:shadow-md transition-shadow text-left"
                >
                  <div className="text-sm font-medium text-gray-500">Monitor in Real-time</div>
                  <div className="mt-1 text-lg font-semibold text-green-600">
                    Live Logs →
                  </div>
                </button>

                <button
                  onClick={() => navigate('/api-keys')}
                  className="px-6 py-4 bg-white border border-gray-300 rounded-lg shadow-sm hover:shadow-md transition-shadow text-left"
                >
                  <div className="text-sm font-medium text-gray-500">Manage Access</div>
                  <div className="mt-1 text-lg font-semibold text-purple-600">
                    API Keys →
                  </div>
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
