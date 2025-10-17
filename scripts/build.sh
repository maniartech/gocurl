#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$SCRIPT_DIR/../cmd/gocurl"

# Define variables
APP_NAME="gocurl"
BIN_DIR="$SCRIPT_DIR/../bin"



# Create the bin directory if it doesn't exist
mkdir -p "$BIN_DIR"

echo "Building $APP_NAME for local system..."




# Build the executable and place it in the bin directory
# The `.` argument tells Go to build the package in the current directory, which is the source directory.
go build -o "$BIN_DIR/$APP_NAME" "$SOURCE_DIR"

echo "Build complete. Executable is in $BIN_DIR/$APP_NAME"
