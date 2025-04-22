const winston = require('winston');

// Get log level from environment or use default
const LOG_LEVEL = process.env.LOG_LEVEL || 'info';

// Configure Winston logger
const logger = winston.createLogger({
    level: LOG_LEVEL,
    format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json()
    ),
    defaultMeta: { service: 'kafka-integration' },
    transports: [
        new winston.transports.Console({
            format: winston.format.combine(
                winston.format.colorize(),
                winston.format.timestamp(),
                winston.format.printf(({ timestamp, level, message, ...rest }) => {
                    const meta = Object.keys(rest).length ? 
                        JSON.stringify(rest, null, 2) : '';
                    return `[${timestamp}] ${level}: ${message} ${meta}`;
                })
            )
        })
    ]
});

module.exports = { logger };