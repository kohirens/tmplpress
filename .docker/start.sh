#!/bin/sh

# We'll use this script to manage starting and stopping this container gracefully.
# It only takes up about 00.01 CPU % allotted to the container, you can verify
# by running `docker stats` after you start a container that uses this as
# as the CMD.

set -e

shutd () {
    printf "%s" "Shutting down the container gracefully..."

    # You can run clean commands here!
    last_signal="15"
}

trap 'shutd' TERM

echo "Starting up..."

# Run non-blocking commands here
go mod tidy
go mod vendor

echo "Ready!"

last_signal=""

# Run until the TERM signal to stop is received.
while [ "${last_signal}" = "15" ]; do
    sleep 1
done

echo "done"
