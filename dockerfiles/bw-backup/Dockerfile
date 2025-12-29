FROM node:20-bookworm-slim

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive \
    PATH="/usr/local/bin:${PATH}"

# Install dependencies and Bitwarden CLI via npm (works on all architectures)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* && \
    npm install -g @bitwarden/cli

# Create non-root user (or use existing node user if UID 1000 exists)
RUN id -u 1000 >/dev/null 2>&1 || useradd -m -u 1000 -s /bin/bash bwbackup && \
    mkdir -p /app && \
    chown -R 1000:1000 /app

# Copy backup script
COPY --chown=1000:1000 backup.sh /app/

RUN chmod +x /app/backup.sh

# Switch to non-root user
USER 1000
WORKDIR /app

# Setup tmpfs volumes for no-trace execution
VOLUME ["/tmp", "/var/tmp"]

# Set entrypoint to backup script
ENTRYPOINT ["/app/backup.sh"]
