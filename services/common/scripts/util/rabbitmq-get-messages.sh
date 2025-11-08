#!/bin/bash

# Get messages from RabbitMQ queue using Management HTTP API
# Usage: ./rabbitmq-get-messages.sh [count]

COUNT=${1:-1}

curl -s -u guest:guest \
  -H "Content-Type: application/json" \
  -d "{\"count\":${COUNT},\"ackmode\":\"ack_requeue_false\",\"encoding\":\"auto\"}" \
  http://localhost:15672/api/queues/%2F/test-events-queue/get
