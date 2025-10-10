# LogBuilder

This is the basic implementation for the Log Aggregation System. This github repo will be used in separate SDKs that can allow users to connect their applications to the log aggregation dashboard seamlessly.


## Running
In frontend/src, run **npm start**
In log_analytics_engine/, run **go run cmd/ingester/main.go**
In a separate terminal window in log_analytics_engine, run **go run cmd/processor/main.go**


## SDKs
In order for implementation into actual applications, a separate SDK was built that uses this LogBuilder application for log aggregation

### Steps to use the SDK:
1. Get API Key
Sign up at your LogBuilder dashboard
Create an API key and copy it
2. Install & Use
npm install logbuilder-sdk
const { LogBuilder } = require('logbuilder-sdk');

const logger = new LogBuilder({
  apiKey: 'your-api-key',
  projectID: 'my-app',
  environment: 'production'
});

logger.info('App started');
logger.error('Payment failed', { orderId: 123 });

// Shutdown gracefully
process.on('SIGTERM', async () => {
  await logger.shutdown();
  process.exit(0);
});
3. View Logs
Open your LogBuilder dashboard to search, analyze, and delete logs.
