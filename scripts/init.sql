-- Create users table first (required by other tables)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(60) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

-- Create the main logs table with partitioning
CREATE TABLE logs (
    id BIGSERIAL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source VARCHAR(255) NOT NULL,
    level VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    service VARCHAR(255),
    fields JSONB,
    raw_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id INTEGER REFERENCES users(id)
) PARTITION BY RANGE (timestamp);

-- Create initial partitions for current and next month
-- This will be automated later, but we need at least one partition to start
CREATE TABLE logs_2024_01 PARTITION OF logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE logs_2024_02 PARTITION OF logs
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

CREATE TABLE logs_2024_03 PARTITION OF logs
    FOR VALUES FROM ('2024-03-01') TO ('2024-04-01');

CREATE TABLE logs_2024_04 PARTITION OF logs
    FOR VALUES FROM ('2024-04-01') TO ('2024-05-01');

CREATE TABLE logs_2024_05 PARTITION OF logs
    FOR VALUES FROM ('2024-05-01') TO ('2024-06-01');

CREATE TABLE logs_2024_06 PARTITION OF logs
    FOR VALUES FROM ('2024-06-01') TO ('2024-07-01');

-- Create table for storing alert rules
CREATE TABLE alert_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    condition_query TEXT NOT NULL,
    threshold_value NUMERIC,
    threshold_operator VARCHAR(10) CHECK (threshold_operator IN ('>', '<', '>=', '<=', '=', '!=')),
    time_window_minutes INTEGER DEFAULT 5,
    notification_channels JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create table for alert history
CREATE TABLE alert_history (
    id SERIAL PRIMARY KEY,
    alert_rule_id INTEGER REFERENCES alert_rules(id),
    triggered_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    trigger_value NUMERIC,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMPTZ,
    details JSONB
);

-- Create table for saved queries
CREATE TABLE saved_queries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query_text TEXT NOT NULL,
    created_by VARCHAR(255),
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

-- Create table for system metrics (for monitoring your own system)
CREATE TABLE system_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(255) NOT NULL,
    metric_value NUMERIC NOT NULL,
    metric_type VARCHAR(50), -- counter, gauge, histogram
    labels JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    api_key TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true
);

-- Insert some sample alert rules
INSERT INTO alert_rules (name, description, condition_query, threshold_value, threshold_operator, time_window_minutes, notification_channels) VALUES
('High Error Rate', 'Alert when error rate exceeds 5% in 10 minutes', 
 'SELECT COUNT(*) FROM logs WHERE level = ''ERROR'' AND timestamp > NOW() - INTERVAL ''10 minutes''', 
 50, '>', 10, 
 '{"slack": {"channel": "#alerts"}, "email": {"recipients": ["dev-team@company.com"]}}'),

('Database Connection Errors', 'Alert on database connection failures',
 'SELECT COUNT(*) FROM logs WHERE message ILIKE ''%database%connection%'' AND level = ''ERROR'' AND timestamp > NOW() - INTERVAL ''5 minutes''',
 3, '>', 5,
 '{"slack": {"channel": "#database-alerts"}}'),

('High Log Volume', 'Alert when log volume is unusually high',
 'SELECT COUNT(*) FROM logs WHERE timestamp > NOW() - INTERVAL ''1 minute''',
 1000, '>', 1,
 '{"email": {"recipients": ["ops@company.com"]}}');

-- Insert some sample saved queries
INSERT INTO saved_queries (name, description, query_text, created_by, is_public) VALUES
('Recent Errors', 'Show all errors from the last hour', 
 'SELECT timestamp, source, service, message FROM logs WHERE level = ''ERROR'' AND timestamp > NOW() - INTERVAL ''1 hour'' ORDER BY timestamp DESC LIMIT 100', 
 'system', true),

('Database Issues', 'Find all database-related errors', 
 'SELECT timestamp, source, message FROM logs WHERE (message ILIKE ''%database%'' OR message ILIKE ''%sql%'' OR message ILIKE ''%connection%'') AND level IN (''ERROR'', ''FATAL'') ORDER BY timestamp DESC', 
 'system', true),

('Service Health Check', 'Check error rates by service in last 24 hours',
 'SELECT service, COUNT(*) as total_logs, COUNT(*) FILTER (WHERE level = ''ERROR'') as errors, ROUND(COUNT(*) FILTER (WHERE level = ''ERROR'') * 100.0 / COUNT(*), 2) as error_rate FROM logs WHERE timestamp > NOW() - INTERVAL ''1 day'' GROUP BY service ORDER BY error_rate DESC',
 'system', true);

-- Enable row-level security (optional, for multi-tenant scenarios)
-- ALTER TABLE logs ENABLE ROW LEVEL SECURITY;

-- Create a function to automatically create partitions (you'll call this from your Go code)
CREATE OR REPLACE FUNCTION create_monthly_partition(target_date DATE)
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    start_date := date_trunc('month', target_date)::DATE;
    end_date := (start_date + INTERVAL '1 month')::DATE;
    partition_name := 'logs_' || to_char(start_date, 'YYYY_MM');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF logs FOR VALUES FROM (%L) TO (%L)',
                   partition_name, start_date, end_date);
    
    -- Add indexes to the new partition
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %I USING BRIN (timestamp)',
                   partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_level ON %I (level)',
                   partition_name, partition_name);
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%s_source ON %I (source)',
                   partition_name, partition_name);
END;
$$ LANGUAGE plpgsql;