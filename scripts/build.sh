#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Define variables
APP_NAME="gocurl"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE}")" && pwd)"
OUTPUT_DIR="$SCRIPT_DIR/../bin"
SOURCE_DIR="$SCRIPT_DIR/../cmd/gocurl"

# Clear the output directory
rm -rf "$OUTPUT_DIR"

# Define the target platforms (OS/ARCH combinations) for cross-compilation
# Note: For Apple Silicon (M1/M2/M3), use "darwin/arm64".
#       For Intel-based Macs, use "darwin/amd64".
CROSS_PLATFORMS=(
    "windows/amd64"
    "linux/amd64"
    "darwin/amd64"
    "linux/arm64"
    "darwin/arm64"
)

# Create the output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# --- 1. Build for the current platform first ---
echo "Building $APP_NAME for the current platform..."

# Get the current operating system and architecture
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

# Set the binary name with the .exe extension for Windows
if [ "$GOOS" = "windows" ]; then
    BINARY_NAME="$APP_NAME.exe"
else
    BINARY_NAME="$APP_NAME"
fi

# Build and place the executable directly in the output directory
go build -o "$OUTPUT_DIR/$BINARY_NAME" "$SOURCE_DIR"

echo "Build for current platform complete. Executable is at $OUTPUT_DIR/$BINARY_NAME"
echo "----------------------------------------------------"

# --- 2. Perform cross-compilation for other platforms ---
echo "Starting cross-compilation for additional platforms..."
echo "----------------------------------------------------"

for PLATFORM in "${CROSS_PLATFORMS[@]}"; do
    # Split the platform string into GOOS and GOARCH
    CROSS_GOOS=$(echo "$PLATFORM" | cut -d '/' -f 1)
    CROSS_GOARCH=$(echo "$PLATFORM" | cut -d '/' -f 2)

    # Determine the file extension based on the target OS
    BINARY_NAME="$APP_NAME-$CROSS_GOOS-$CROSS_GOARCH"
    if [ "$CROSS_GOOS" = "windows" ]; then
        BINARY_NAME+=".exe"
    fi

    echo "Building for $CROSS_GOOS/$CROSS_GOARCH..."

    # Compile the Go application with the specified environment variables
    env GOOS="$CROSS_GOOS" GOARCH="$CROSS_GOARCH" go build -o "$OUTPUT_DIR/$BINARY_NAME" "$SOURCE_DIR"

    echo "Finished building $BINARY_NAME"
done

echo "----------------------------------------------------"
echo "Cross-compilation complete. Binaries are in the '$OUTPUT_DIR' directory."
