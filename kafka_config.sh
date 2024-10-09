#!/bin/bash

# Define topics and their default partition values
declare -A topics_and_partitions=(
    ["video"]=100
    ["video-response"]=50
    ["video-resolutions"]=100
    ["video-resolutions-response"]=50
    ["image"]=100
    ["image-response"]=50
    ["audio"]=50
    ["audio-response"]=25
    ["delete-file"]=20
)

# Export the topics_and_partitions array for use in other scripts
export topics_and_partitions
