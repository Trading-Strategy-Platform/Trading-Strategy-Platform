#!/bin/bash
/etc/confluent/docker/run &
KAFKA_PID=\$!
echo "Kafka started with PID \"
sleep 15
echo "Ready to create topics"
/usr/local/bin/setup-topics.sh
wait \
