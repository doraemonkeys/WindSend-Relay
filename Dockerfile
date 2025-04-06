# Stage 1: Build the application
# FROM golang:1.21-alpine AS builder
FROM golang:alpine AS builder

# Set necessary environment variables
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set the working directory inside the container
WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies. This leverages Docker's layer caching.
# Dependencies are only re-downloaded if go.mod or go.sum change.
RUN go mod download

# Copy the entire source code
COPY . .

# Get version information (allow overriding via build args)
ARG BUILD_HASH="unknown"
ARG BUILD_TIME="unknown"

RUN echo "BUILD_HASH: ${BUILD_HASH}"
RUN echo "BUILD_TIME: ${BUILD_TIME}"

# Build the application statically linked
# Inject version information using ldflags
# The path for version variables must match your package structure
RUN go build \
    -ldflags="-w -s \
    -X 'github.com/doraemonkeys/WindSend-Relay/version.BuildHash=${BUILD_HASH}' \
    -X 'github.com/doraemonkeys/WindSend-Relay/version.BuildTime=${BUILD_TIME}'" \
    -o /windsend-relay main.go

# Stage 2: Create the final minimal image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /windsend-relay /app/windsend-relay

# Copy any essential non-code assets if needed (e.g., a default config)
# COPY config.json.example /app/config.json

# Expose the default port the application listens on
EXPOSE 16779

# Set default environment variables (can be overridden at runtime)
# Consistent with your code's defaults or common container practice
ENV WS_LISTEN_ADDR="0.0.0.0:16779"

# Command to run the application
# Use --use-env as the default for containerized environments
ENTRYPOINT ["/app/windsend-relay"]
CMD ["--use-env"]