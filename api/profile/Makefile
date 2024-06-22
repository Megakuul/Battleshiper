# Define the binary name
BINARY_NAME=bootstrap

# Define the source file
SRC=main.go

# Phony targets
.PHONY: all build clean package
# Default target
all: build

# Build the Go binary
build: build-ProfileFunc

# Custom build target for SAM CLI
build-ProfileFunc:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o $(BINARY_NAME) $(SRC)
	cp -r $(BINARY_NAME) $(ARTIFACTS_DIR)