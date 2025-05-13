#!/bin/sh
docker buildx build --push --platform linux/arm/v7,linux/arm64/v8,linux/amd64 --tag jgulick48/rv-homekit:release-arm-0.0.12-rc$1 .