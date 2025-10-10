# LogBuilder

This is the basic implementation for the Log Aggregation System. This github repo will be used in separate SDKs that can allow users to connect their applications to the log aggregation dashboard seamlessly.

## Running
In frontend/src, run **npm start**
In log_analytics_engine/, run **go run cmd/ingester/main.go**
In a separate terminal window in log_analytics_engine, run **go run cmd/processor/main.go**
