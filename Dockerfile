# Use Go 1.25 Alpine as the base image for building the application
FROM golang:1.25-alpine AS builder

# Define the base directory for the application as an environment variable
ENV SERVER_DIR=/openim-server

# Set the working directory inside the container based on the environment variable
WORKDIR $SERVER_DIR

# Set the Go proxy to improve dependency resolution speed
# ENV GOPROXY=https://goproxy.io,direct

# Reuse the host's already-populated Go module cache so this step needs no network access.
# Kept before the source copy so this large layer stays cached across code changes
COPY --from=modcache . /go/pkg/mod

COPY go.mod go.sum ./

RUN go mod download

# Install Mage to use for building the application
RUN go install github.com/magefile/mage@v1.15.0

# Copy all files from the current directory into the container
COPY . .

# Optionally build your application if needed
# The Go build cache lives in a cache mount so it is reused between builds and stays out of the image layers
RUN --mount=type=cache,target=/root/.cache/go-build mage build

# Using Alpine Linux with Go environment for the final image
FROM golang:1.25-alpine

# Install necessary packages, such as bash
RUN apk add --no-cache bash

# Set the environment and work directory
ENV SERVER_DIR=/openim-server
WORKDIR $SERVER_DIR


# Copy the compiled binaries and mage from the builder image to the final image
COPY --from=builder $SERVER_DIR/_output $SERVER_DIR/_output
COPY --from=builder $SERVER_DIR/config $SERVER_DIR/config
COPY --from=builder /go/bin/mage /usr/local/bin/mage
COPY --from=builder $SERVER_DIR/magefile_windows.go $SERVER_DIR/
COPY --from=builder $SERVER_DIR/magefile_unix.go $SERVER_DIR/
COPY --from=builder $SERVER_DIR/magefile.go $SERVER_DIR/
COPY --from=builder $SERVER_DIR/start-config.yml $SERVER_DIR/
COPY --from=builder $SERVER_DIR/version $SERVER_DIR/version
COPY --from=builder $SERVER_DIR/go.mod $SERVER_DIR/
COPY --from=builder $SERVER_DIR/go.sum $SERVER_DIR/

# Reuse the module cache the builder stage already resolved (matches go.mod's
# github.com/openimsdk/gomake version) so `mage start`'s runtime compile needs no network
COPY --from=builder /go/pkg/mod /go/pkg/mod

# Set the command to run when the container starts
ENTRYPOINT ["sh", "-c", "mage start && tail -f /dev/null"]
