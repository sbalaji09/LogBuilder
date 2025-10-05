import api from './api';

export interface LogStats {
  total_logs: number;
  error_count: number;
  warning_count: number;
  info_count: number;
  debug_count: number;
  logs_today: number;
  logs_this_week: number;
  top_sources: { source: string; count: number }[];
  recent_errors: Array<{
    id: string;
    message: string;
    timestamp: string;
    source: string;
  }>;
}

export const statsService = {
  async getStats(): Promise<LogStats> {
    // Since we don't have a dedicated stats endpoint, we'll aggregate from logs
    const [allLogs, errors, warnings, info, debug] = await Promise.all([
      api.get('/logs', { params: { limit: 1000 } }),
      api.get('/logs', { params: { level: 'error', limit: 100 } }),
      api.get('/logs', { params: { level: 'warning', limit: 100 } }),
      api.get('/logs', { params: { level: 'info', limit: 100 } }),
      api.get('/logs', { params: { level: 'debug', limit: 100 } }),
    ]);

    const logs = allLogs.data.logs || [];
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);

    const logsToday = logs.filter(
      (log: any) => new Date(log.timestamp) >= today
    ).length;

    const logsThisWeek = logs.filter(
      (log: any) => new Date(log.timestamp) >= weekAgo
    ).length;

    // Count sources
    const sourceCounts: { [key: string]: number } = {};
    logs.forEach((log: any) => {
      if (log.source) {
        sourceCounts[log.source] = (sourceCounts[log.source] || 0) + 1;
      }
    });

    const topSources = Object.entries(sourceCounts)
      .map(([source, count]) => ({ source, count: count as number }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 5);

    const recentErrors = (errors.data.logs || [])
      .slice(0, 5)
      .map((log: any) => ({
        id: log.id,
        message: log.message,
        timestamp: log.timestamp,
        source: log.source,
      }));

    return {
      total_logs: allLogs.data.total || 0,
      error_count: errors.data.total || 0,
      warning_count: warnings.data.total || 0,
      info_count: info.data.total || 0,
      debug_count: debug.data.total || 0,
      logs_today: logsToday,
      logs_this_week: logsThisWeek,
      top_sources: topSources,
      recent_errors: recentErrors,
    };
  },
};
