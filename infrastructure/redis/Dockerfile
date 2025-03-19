FROM redis:7

ARG CONFIG_FILE=./redis.conf

# Copy the custom configuration file
COPY ${CONFIG_FILE} /usr/local/etc/redis/redis.conf

# Run Redis with the custom configuration
CMD ["redis-server", "/usr/local/etc/redis/redis.conf"]

# Expose Redis port
EXPOSE 6379