#!/bin/bash

source ./logging.sh

# Create the media-docker-proxy network (internal)
create_media_docker_proxy_network() {
  # Check if the network already exists
  if docker network ls | grep -w "media-docker-proxy" > /dev/null 2>&1; then
    loginf "Network 'media-docker-proxy' already exists."
  else
    loginf "Creating internal network 'media-docker-proxy'..."
    if docker network create --driver bridge --internal "media-docker-proxy"; then
      logsuccess "'media-docker-proxy' network created."
    else
      logerr "Error: Failed to create network 'media-docker-proxy'."
      exit 1
    fi
    
  fi
}

create_media_docker_proxy_network
