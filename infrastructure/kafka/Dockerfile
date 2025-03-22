FROM confluentinc/cp-kafka:7.3.0

ARG CONFIG_DIR=./config

# Copy the server properties
COPY ${CONFIG_DIR}/server.properties /etc/kafka/server.properties

# Copy the topics script 
COPY ${CONFIG_DIR}/topics.sh /usr/local/bin/setup-topics.sh
USER root
RUN chmod +x /usr/local/bin/setup-topics.sh

# Create startup script with proper line endings
RUN echo '#!/bin/bash' > /usr/local/bin/startup.sh && \
    echo '/etc/confluent/docker/run &' >> /usr/local/bin/startup.sh && \
    echo 'sleep 10' >> /usr/local/bin/startup.sh && \
    echo '/usr/local/bin/setup-topics.sh' >> /usr/local/bin/startup.sh && \
    echo 'wait $!' >> /usr/local/bin/startup.sh && \
    chmod +x /usr/local/bin/startup.sh

# Override the entrypoint
ENTRYPOINT ["/usr/local/bin/startup.sh"]