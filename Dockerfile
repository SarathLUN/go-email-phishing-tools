# ---- Build Stage ----
# Use a Debian-based Go image (e.g., bookworm, which is Debian 12)
ARG GO_VERSION=1.23
FROM golang:${GO_VERSION} AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the source code into the container
COPY . .

# Build the Go app
# CGO_ENABLED=1 is needed for mattn/go-sqlite3.
# GOOS=linux GOARCH=amd64 are appropriate for most Docker environments.
RUN go build -v -o /app/ ./...

# ---- Runtime Stage ----
# Use a minimal Debian-based image for the runtime environment for glibc compatibility
FROM debian:bookworm-slim AS runtime
# For even smaller images, you could explore:
# FROM gcr.io/distroless/base-debian12 AS runtime
# Note: Distroless images have no shell, which is good for security but harder to debug.

# ----> ADD THIS SECTION TO ENSURE CA CERTIFICATES ARE PRESENT <----
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
# ----> END OF ADDED SECTION <----

# Set the Current Working Directory inside the container
WORKDIR /app

# Create a non-root user and group for better security
# Using -r for system user/group, --no-create-home as we don't need a home dir.
RUN groupadd -r appgroup && useradd --no-log-init -r -g appgroup appuser

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/email-phishing-tools /app/email-phishing-tools

# Copy configuration templates and migration files
COPY configs /app/configs
COPY db/migrations /app/db/migrations

# Create and set permissions for the data directory where SQLite DB will live
# This directory will be owned by appuser.
RUN mkdir -p /app/data && chown appuser:appgroup /app/data
# Set ownership for the rest of the app directory as well
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

# Expose port if the application runs a web server
# The actual port is defined by environment variables.
EXPOSE 8080

# Define environment variables (defaults)
# These can be overridden at runtime (docker run -e or via docker-compose)
ENV DB_PATH=/app/data/phishing_simulation.db
ENV TRACKER_HOST=0.0.0.0
ENV TRACKER_PORT=8080
ENV EMAIL_TEMPLATE_PATH=/app/configs/email_template.html
ENV REDIRECT_URL_AFTER_CLICK=https://www.google.com
# SMTP variables should be provided at runtime for security.

# Command to run the executable
# The application binary will be the entrypoint, and the command (serve, import, etc.)
# will be passed as arguments to `docker run` or in docker-compose.yml.
ENTRYPOINT ["/app/email-phishing-tools"]

# Default command if none is provided (optional, can be useful for `docker run <image_name>`)
CMD ["serve"]
