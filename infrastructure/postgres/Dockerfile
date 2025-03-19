FROM postgres:14

ARG CONFIG_FILE=./postgres.conf

# Copy the custom configuration file
COPY ${CONFIG_FILE} /etc/postgresql/custom-postgres.conf

# Create an initialization script to apply the custom configuration
RUN echo '#!/bin/bash \n\
cat /etc/postgresql/custom-postgres.conf >> /var/lib/postgresql/data/postgresql.conf' > /docker-entrypoint-initdb.d/apply-custom-config.sh && \
chmod +x /docker-entrypoint-initdb.d/apply-custom-config.sh

# Set environment variables for postgres
ENV POSTGRES_HOST_AUTH_METHOD=trust

# Standard postgres ports
EXPOSE 5432

# Default CMD from the postgres image will be used