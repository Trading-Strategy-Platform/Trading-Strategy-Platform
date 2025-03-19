#!/bin/bash

# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
until kafka-topics.sh --bootstrap-server kafka:9092 --list > /dev/null 2>&1; do
    sleep 5
    echo "Waiting for Kafka..."
done

echo "Creating Kafka topics..."

# Create topics with specified partitions and replication factor
# Using replication factor of 1 for local development
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic strategy-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic user-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic user-notifications --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic backtest-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic backtest-completions --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic marketplace-events --partitions 3 --replication-factor 1

echo "List of topics:"
kafka-topics.sh --bootstrap-server kafka:9092 --list

echo "Topic creation completed!"