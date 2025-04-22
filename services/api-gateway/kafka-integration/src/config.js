const { Kafka } = require('kafkajs');
const { logger } = require('./logger');

// Get Kafka brokers from environment or use default
const KAFKA_BROKERS = process.env.KAFKA_BROKERS ? 
    process.env.KAFKA_BROKERS.split(',') : 
    ['kafka:9092'];

// Topic mapping configuration
const TOPIC_MAPPING = {
    'api/v1/auth': 'user-events',
    'api/v1/admin': 'user-events',
    'api/v1/users': 'user-events',
    'api/v1/strategies': 'strategy-events',
    'api/v1/indicators': 'strategy-events',
    'api/v1/marketplace': 'marketplace-events',
    'api/v1/backtest': 'backtest-events',
    'api/v1/market-data': 'historical-events'
};

// Default topic for events that don't match specific patterns
const DEFAULT_TOPIC = 'user-events';

// Initialize Kafka
const kafka = new Kafka({
    clientId: 'nginx-audit-processor',
    brokers: KAFKA_BROKERS,
    retry: {
        initialRetryTime: 100,
        retries: 8
    }
});

// Get Kafka producer with connection
async function getKafkaProducer() {
    const producer = kafka.producer();
    try {
        await producer.connect();
        return producer;
    } catch (error) {
        logger.error('Failed to connect to Kafka', { error: error.message });
        throw error;
    }
}

// Determine which Kafka topic to use based on the URI
function determineTopic(uri) {
    for (const [pattern, topic] of Object.entries(TOPIC_MAPPING)) {
        if (uri.includes(pattern)) {
            return topic;
        }
    }
    return DEFAULT_TOPIC;
}

module.exports = {
    getKafkaProducer,
    determineTopic,
    KAFKA_BROKERS
};