const fs = require('fs');
const { Tail } = require('tail');
const { logger } = require('./logger');
const { getKafkaProducer, determineTopic } = require('./config');

const AUDIT_LOG_PATH = '/var/log/nginx/audit_events.log';
const MAX_RETRY_COUNT = 30;
const RETRY_INTERVAL = 5000; // 5 seconds

// Function to check if log file exists
async function checkLogFileExists(retryCount = 0) {
    try {
        await fs.promises.access(AUDIT_LOG_PATH, fs.constants.R_OK);
        logger.info(`Log file ${AUDIT_LOG_PATH} is accessible`);
        return true;
    } catch (error) {
        if (retryCount >= MAX_RETRY_COUNT) {
            logger.error(`Log file ${AUDIT_LOG_PATH} not found after ${MAX_RETRY_COUNT} retries`);
            return false;
        }
        
        logger.info(`Waiting for log file ${AUDIT_LOG_PATH} to be created (attempt ${retryCount + 1}/${MAX_RETRY_COUNT})...`);
        await new Promise(resolve => setTimeout(resolve, RETRY_INTERVAL));
        return checkLogFileExists(retryCount + 1);
    }
}

// Main function to start watching
async function startWatching() {
    try {
        // Check log file exists or wait for it to be created
        const logFileExists = await checkLogFileExists();
        if (!logFileExists) {
            throw new Error(`Log file ${AUDIT_LOG_PATH} not accessible after maximum retries`);
        }
        
        // Initialize Kafka producer
        const producer = await getKafkaProducer();
        logger.info('Connected to Kafka');

        // Watch the log file for new entries
        const tail = new Tail(AUDIT_LOG_PATH);

        // Handle new log lines
        tail.on('line', async (line) => {
            try {
                // Parse JSON log
                const logEntry = JSON.parse(line);
                
                // Get appropriate Kafka topic
                const topic = determineTopic(logEntry.uri);
                
                // Send to Kafka
                await producer.send({
                    topic,
                    messages: [
                        { 
                            key: logEntry.user_id || 'anonymous',
                            value: line
                        }
                    ],
                });
                
                logger.debug(`Sent audit event to topic ${topic}`, { 
                    uri: logEntry.uri,
                    method: logEntry.request_method,
                    status: logEntry.status
                });
            } catch (error) {
                logger.error('Error processing log line', { error: error.message, line });
            }
        });

        // Handle errors
        tail.on('error', (error) => {
            logger.error('Error watching log file', { error: error.message });
        });

        logger.info(`Watching for audit events in ${AUDIT_LOG_PATH}`);
        
        // Keep the process running
        process.on('SIGINT', async () => {
            logger.info('Caught SIGINT, shutting down...');
            await producer.disconnect();
            tail.unwatch();
            process.exit(0);
        });
        
        process.on('SIGTERM', async () => {
            logger.info('Caught SIGTERM, shutting down...');
            await producer.disconnect();
            tail.unwatch();
            process.exit(0);
        });
    } catch (error) {
        logger.error('Failed to start Kafka integration', { error: error.message });
        process.exit(1);
    }
}

// Start the application
startWatching().catch(error => {
    logger.error('Unhandled error in main process', { error: error.message });
    process.exit(1);
});