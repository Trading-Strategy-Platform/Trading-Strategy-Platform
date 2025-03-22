FROM confluentinc/cp-kafka:7.3.0

ARG CONFIG_DIR=./config

# Copy the server properties
COPY ${CONFIG_DIR}/server.properties /etc/kafka/server.properties

# Make the script executable locally before copying it
COPY ${CONFIG_DIR}/topics.sh /tmp/topics.sh
USER root
RUN chmod +x /tmp/topics.sh && \
    mv /tmp/topics.sh /usr/local/bin/setup-topics.sh

# Create startup script
RUN echo '#!/bin/bash \n\
# Start Kafka in background \n\
/etc/confluent/docker/run & \n\
\n\
# Wait for Kafka to start \n\
sleep 10 \n\
\n\
# Set up topics \n\
/usr/local/bin/setup-topics.sh \n\
\n\
# Wait for the Kafka process \n\
wait $!' > /usr/local/bin/startup.sh && \
chmod +x /usr/local/bin/startup.sh

# Override the entrypoint
ENTRYPOINT ["/usr/local/bin/startup.sh"]