#!/bin/bash

# Set up Docker buildx for multi-architecture builds
docker buildx create --name multiarch-builder --use || docker buildx use multiarch-builder
docker buildx inspect --bootstrap

# Build and push multi-architecture images
docker buildx build --platform linux/amd64,linux/arm64 \
  -t containerman17/local_agg:latest \
  --push \
  .

# For local testing with specific architecture (optional)
# docker run -it --platform linux/amd64 containerman17/local_agg
# docker run -it --platform linux/arm64 containerman17/local_agg

