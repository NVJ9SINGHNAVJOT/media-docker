#!/bin/bash

# Define topics and their default partition values
declare -A topics_and_partitions=(
    ["video"]=100
    ["video-resolutions"]=100
    ["image"]=100
    ["audio"]=50
    ["delete-file"]=20
    ["media-docker-files-response"]=100
)

# Export the topics_and_partitions array for use in other scripts
export topics_and_partitions
