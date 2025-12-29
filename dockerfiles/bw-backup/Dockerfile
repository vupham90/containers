FROM debian:bookworm-slim

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive \
    BW_CLI_VERSION=2024.9.0 \
    PATH="/usr/local/bin:${PATH}"

# Install dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    curl \
    wget \
    unzip \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Bitwarden CLI
RUN wget "https://github.com/bitwarden/clients/releases/download/cli-v${BW_CLI_VERSION}/bw-linux-${BW_CLI_VERSION}.zip" -O bw.zip && \
    unzip bw.zip && \
    chmod +x bw && \
    mv bw /usr/local/bin/ && \
    rm bw.zip

# Create non-root user
RUN useradd -m -u 1000 -s /bin/bash bwbackup && \
    mkdir -p /app && \
    chown -R bwbackup:bwbackup /app

# Copy backup script
COPY --chown=bwbackup:bwbackup backup.sh /app/

RUN chmod +x /app/backup.sh

# Switch to non-root user
USER bwbackup
WORKDIR /app

# Setup tmpfs volumes for no-trace execution
VOLUME ["/tmp", "/var/tmp"]

# Set entrypoint to backup script
ENTRYPOINT ["/app/backup.sh"]
