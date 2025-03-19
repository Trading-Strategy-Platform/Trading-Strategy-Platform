FROM timescale/timescaledb:latest-pg14

ARG CONFIG_FILE=./timescaledb.conf

# Copy the configuration file
COPY ${CONFIG_FILE} /etc/postgresql/timescaledb.conf

# Create init script to include our configuration
RUN echo '#!/bin/bash \n\
cat /etc/postgresql/timescaledb.conf >> /var/lib/postgresql/data/postgresql.conf' > /docker-entrypoint-initdb.d/update-config.sh && \
chmod +x /docker-entrypoint-initdb.d/update-config.sh