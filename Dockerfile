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
# RELEASE=true builds with -trimpath -ldflags "-s -w" (no debug info, much smaller binaries).
# The magefile is compiled into a standalone binary as well, so the final image needs neither
# the Go toolchain nor the module cache to run `mage start` / `mage check`.
RUN --mount=type=cache,target=/root/.cache/go-build RELEASE=true mage build && mage -compile /openim-server/mage-static

# Plain Alpine for the final image: binaries and the compiled magefile only
FROM alpine:3.22

# Install necessary packages, such as bash
RUN apk add --no-cache bash ca-certificates tzdata

# Set the environment and work directory
ENV SERVER_DIR=/openim-server
WORKDIR $SERVER_DIR

# Copy the compiled binaries and mage from the builder image to the final image.
# The compiled magefile is installed as `mage`, so `mage start` and the compose healthcheck
# `mage check` keep working.
COPY --from=builder $SERVER_DIR/_output $SERVER_DIR/_output
COPY --from=builder $SERVER_DIR/config $SERVER_DIR/config
COPY --from=builder $SERVER_DIR/mage-static /usr/local/bin/mage
COPY --from=builder $SERVER_DIR/start-config.yml $SERVER_DIR/

# Set the command to run when the container starts
ENTRYPOINT ["sh", "-c", "mage start && tail -f /dev/null"]
