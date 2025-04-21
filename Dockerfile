# Stage 1: Build the Go application backend
# FROM golang:1.21-alpine AS builder
FROM golang:alpine AS builder

# Set Go environment variables
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app/server

# Copy Go module files first for layer caching
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy the rest of the backend source code
COPY server/. .

# Get version information (allow overriding via build args)
ARG BUILD_HASH="unknown"
ARG BUILD_TIME="unknown"
RUN echo "BUILD_HASH: ${BUILD_HASH}"
RUN echo "BUILD_TIME: ${BUILD_TIME}"

# Build the Go application statically linked
# Inject version information using ldflags
RUN go build \
    -ldflags="-w -s \
    -X 'github.com/doraemonkeys/WindSend-Relay/server/version.BuildHash=${BUILD_HASH}' \
    -X 'github.com/doraemonkeys/WindSend-Relay/server/version.BuildTime=${BUILD_TIME}'" \
    -o /windsend-relay main.go

# Stage 2: Build the Admin UI frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app/relay_admin

# Copy package files first for layer caching
COPY relay_admin/package.json relay_admin/package-lock.json ./
# Use npm ci for potentially faster and more reliable installs in CI/build environments
RUN npm ci

# Copy the rest of the frontend source code
COPY relay_admin/. .

# Build the frontend application
# The output should be in the 'dist' directory inside WORKDIR (/app/relay_admin/dist)
RUN npm run build

# Stage 3: Create the final minimal image
FROM alpine:latest

WORKDIR /app

# Copy the static Go binary from the builder stage
COPY --from=builder /windsend-relay /app/windsend-relay

# Copy the built frontend static assets from the frontend-builder stage
# The Go application needs to be configured to serve files from this 'web' directory.
# Adjust '/app/static/web' if your Go backend serves from a different path (e.g., '/app/dist').
COPY --from=frontend-builder /app/relay_admin/dist /app/static/web

# Expose the default relay port and the default admin UI port
EXPOSE 16779
EXPOSE 16780

# Set default environment variables
ENV WS_LISTEN_ADDR="0.0.0.0:16779"
ENV WS_ADMIN_ADDR="0.0.0.0:16780"
# Note: Admin user/password defaults are handled by the application logic
# (user 'admin', generated password if WS_ADMIN_PASSWORD is not set)

# Command to run the application
# Use --use-env as the default for containerized environments
ENTRYPOINT ["/app/windsend-relay"]
CMD ["--use-env"]