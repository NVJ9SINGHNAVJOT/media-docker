#!/bin/bash

# Source the file containing the container status and Kafka topic management functions
source ./docker_container_status.sh
source ./manage_kafka_topics.sh

# Check if the container name is provided as argument
if [ -z "$1" ] ; then
    echo "Usage: $0 <container_name>"
    exit 1
fi

KAFKA_CONTAINER="$1"   # Local variable for container name

# Check if Kafka container is running
if ! is_container_running "$KAFKA_CONTAINER"; then
    echo "Error: Kafka container '$KAFKA_CONTAINER' is not running."
    exit 1
fi

# Wait for Kafka to be ready
wait_for_kafka_ready "$KAFKA_CONTAINER"

# Declare an array of topics
topics=(
    "video"
    "videoResponse"
    "videoResolutions"
    "videoResolutionsResponse"
    "image"
    "imageResponse"
    "audio"
    "audioResponse"
    "deleteFile"
)

# Loop through each topic in the array and delete it if it exists
for topic in "${topics[@]}"; do
    delete_topic_if_exists "$KAFKA_CONTAINER" "$topic"
done

echo "Successfully deleted topics"