const http = require('http');
const fs = require('fs');

// Simple healthcheck to verify the service is running
// and the log file is accessible

function checkLogFileExists() {
    const LOG_FILE = '/var/log/nginx/audit_events.log';
    try {
        fs.accessSync(LOG_FILE, fs.constants.R_OK);
        return true;
    } catch (err) {
        return false;
    }
}

// Check if environment variables are set properly
function checkConfig() {
    const requiredEnvVars = ['KAFKA_BROKERS'];
    const missingVars = requiredEnvVars.filter(
        envVar => !process.env[envVar]
    );
    
    return missingVars.length === 0;
}

// Exit with success (0) or failure (1) status
if (checkLogFileExists() && checkConfig()) {
    console.log('Health check passed');
    process.exit(0);
} else {
    console.error('Health check failed');
    process.exit(1);
}