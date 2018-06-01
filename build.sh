#!/bin/bash

# Build the app for linux first
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -a -tags netgo -ldflags '-w' .

# Build the image
docker build -t wumuxian/aws-slack-bot:latest .

# Clear up
images=$(docker images -q --filter "dangling=true")
echo $images
if [ "$images" != "" ]; then
docker rmi $images
fi
