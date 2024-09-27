#!/bin/bash

# Check if the container name is provided as an argument
# If no container name is provided, print an error message and exit
if [ -z "$1" ]; then
    echo "Error: Container name not provided."
    exit 1
fi

# Check if the action (create/delete) is provided as an argument
# If no action is provided, print an error message and exit
if [ -z "$2" ]; then
    echo "Error: Action (create/delete) not provided."
    exit 1
fi

# Assigning Kafka container name and action from the provided arguments
KAFKA_CONTAINER="$1"
ACTION="$2"

# Kafka broker configuration
BROKER="localhost:9092"
TIMEOUT=30  # Maximum wait time for Kafka to be ready (in seconds)
SLEEP_INTERVAL=2  # Time to wait between each Kafka readiness check (in seconds)

# Function to check if the Kafka container is running
is_container_running() {
    local status
    # Get the container's status using Docker inspect
    status=$(docker inspect -f '{{.State.Status}}' "$KAFKA_CONTAINER" 2>/dev/null)

    # Return true if the container is running, otherwise false
    if [ "$status" == "running" ]; then
        return 0  # Container is running
    else
        return 1  # Container is not running
    fi
}

# Function to check if Kafka is ready by attempting to list topics
is_kafka_ready() {
    # Try to list topics to verify if Kafka is responsive
    if docker exec "$KAFKA_CONTAINER" kafka-topics.sh --bootstrap-server "$BROKER" --list >/dev/null 2>&1; then
        return 0  # Kafka is ready
    else
        return 1  # Kafka is not ready
    fi
}

# Function to wait for Kafka to be ready before proceeding
wait_for_kafka() {
    local elapsed=0
    # Wait until Kafka is ready or until the timeout is reached
    while [ "$elapsed" -lt "$TIMEOUT" ]; do
        if is_kafka_ready; then
            echo "Kafka is ready. Proceeding to create topics..."
            return 0  # Kafka is ready
        else
            echo "Waiting for Kafka to be ready... (elapsed time: $elapsed seconds)"
            sleep "$SLEEP_INTERVAL"
            elapsed=$((elapsed + SLEEP_INTERVAL))
        fi
    done
    # If Kafka doesn't become ready in time, exit with an error
    echo "Error: Timed out waiting for Kafka to be ready."
    exit 1
}

# Function to delete a topic if it exists
delete_topic_if_exists() {
    local topic_name="$1"

    # Check if the topic exists
    if docker exec "$KAFKA_CONTAINER" kafka-topics.sh --bootstrap-server "$BROKER" --list | grep -w "$topic_name"; then
        echo "Deleting topic: $topic_name"
        # Attempt to delete the topic
        if docker exec "$KAFKA_CONTAINER" kafka-topics.sh --bootstrap-server "$BROKER" --delete --topic "$topic_name"; then
            echo "Topic '$topic_name' deleted."
        else
            echo "Error: Failed to delete topic '$topic_name'. Exiting..."
            exit 1  # Exit if the topic deletion fails
        fi
    else
        echo "Error: Topic '$topic_name' does not exist. Exiting..."
        exit 1  # Exit if the topic doesn't exist
    fi
}

# Function to create a topic if it doesn't exist
create_topic_if_not_exists() {
    local topic_name="$1"
    local partitions="$2"
    local replication_factor="$3"

    # Check if the topic already exists
    if ! docker exec "$KAFKA_CONTAINER" kafka-topics.sh --bootstrap-server "$BROKER" --list | grep -w "$topic_name"; then
        echo "Creating topic: $topic_name"
        # Attempt to create the topic with specified partitions and replication factor
        if docker exec "$KAFKA_CONTAINER" kafka-topics.sh --bootstrap-server "$BROKER" --create \
            --topic "$topic_name" --partitions "$partitions" --replication-factor "$replication_factor"; then
            echo "Topic '$topic_name' created with $partitions partitions and replication factor of $replication_factor."
        else
            echo "Error: Failed to create topic '$topic_name'. Exiting..."
            exit 1  # Exit if the topic creation fails
        fi
    else
        echo "Topic '$topic_name' already exists."  # Topic already exists, no need to create
    fi
}

# Function to handle topics based on the provided action (create or delete)
process_topics() {
    local action="$1"

    case "$action" in
        # If the action is 'create', create the following topics if they don't exist
        "create")
            create_topic_if_not_exists "video" 10 1
            create_topic_if_not_exists "videoResponse" 10 1
            create_topic_if_not_exists "videoResolutions" 10 1
            create_topic_if_not_exists "videoResolutionsResponse" 10 1
            create_topic_if_not_exists "image" 10 1
            create_topic_if_not_exists "imageResponse" 10 1
            create_topic_if_not_exists "audio" 10 1
            create_topic_if_not_exists "audioResponse" 10 1
            create_topic_if_not_exists "deleteFile" 10 1
            ;;
        # If the action is 'delete', delete the specified topics if they exist
        "delete")
            delete_topic_if_exists "video"
            delete_topic_if_exists "videoResponse"
            delete_topic_if_exists "videoResolutions"
            delete_topic_if_exists "videoResolutionsResponse"
            delete_topic_if_exists "image"
            delete_topic_if_exists "imageResponse"
            delete_topic_if_exists "audio"
            delete_topic_if_exists "audioResponse"
            ;;
        # For any other action, print an error and exit
        *)
            echo "Error: Invalid action '$action'. Use 'create' or 'delete'."
            exit 1
            ;;
    esac
}

# Main execution flow

# Check if the Kafka container is running
if is_container_running; then
    echo "Kafka container '$KAFKA_CONTAINER' is running. Waiting for Kafka to be ready..."
    wait_for_kafka  # Wait until Kafka is ready
    process_topics "$ACTION"  # Process topics based on the action (create/delete)
else
    echo "Error: Kafka container '$KAFKA_CONTAINER' is not running. Exiting..."
    exit 1  # Exit if the container is not running
fi
