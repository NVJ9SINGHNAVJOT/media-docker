#!/bin/bash

# Define topics and their default partition values
# INFO: All struct types are defined in the topics package in ./topics
declare -A topics_and_partitions=(
    ["video"]=100
    ["video-resolutions"]=100
    ["image"]=100
    ["audio"]=50
    ["delete-file"]=20
    ["media-docker-files-response"]=50
    ["failed-letter-queue"]=10
)

# Export the topics_and_partitions array for use in other scripts
export topics_and_partitions
