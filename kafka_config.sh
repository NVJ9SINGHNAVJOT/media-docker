#!/bin/bash

# Define topics and their default partition values
declare -A topics_and_partitions=(
    ["video"]=100
    ["videoResponse"]=50
    ["videoResolutions"]=100
    ["videoResolutionsResponse"]=50
    ["image"]=100
    ["imageResponse"]=50
    ["audio"]=50
    ["audioResponse"]=25
    ["deleteFile"]=20
)

# Export the topics_and_partitions array for use in other scripts
export topics_and_partitions
