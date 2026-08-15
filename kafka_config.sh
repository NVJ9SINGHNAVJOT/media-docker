#!/bin/bash

# Define topics and their default partition values
# INFO: All struct types are defined in the topics package in ./topics
#
# NOTE: The "document" and "other" categories have no topics. They are stored and
# served exactly as uploaded, so the server completes them itself with no
# consumer involved.
#
# CAUTION: A topic's partition count is the ceiling on useful parallelism for the
# service that consumes it. The sum of KAFKA_WORKERS across every instance of
# that service must not exceed it.
topics_and_partitions=(
    "video:10"
    "video-resolutions:10"
    "image:10"
    "audio:10"
    "delete-file:5"
    "failed-letter-queue:5"
)

# Export the topics_and_partitions array for use in other scripts
export topics_and_partitions
