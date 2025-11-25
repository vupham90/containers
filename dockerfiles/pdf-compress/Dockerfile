FROM alpine:latest

# Install Ghostscript
RUN apk add --no-cache ghostscript

# Set working directory
WORKDIR /workspace

# Default entrypoint is gs (Ghostscript)
# The command will be passed from the Go binary
ENTRYPOINT ["gs"]

