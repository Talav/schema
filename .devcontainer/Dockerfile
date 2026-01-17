# Start from the same base image
FROM qmcgaw/godevcontainer:debian

# Install Go development tools (build-time, cached in image)
# These don't need the mounted workspace, so install them at build time
RUN go install golang.org/x/tools/gopls@latest && \
    go install github.com/cweill/gotests/...@latest && \ 
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2 && \
    go install mvdan.cc/gofumpt@latest && \
    go install golang.org/x/tools/cmd/goimports@latest

# Copy the entrypoint script into the image
COPY .devcontainer/entrypoint.sh /usr/local/bin/entrypoint.sh

# Make the script executable
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the entrypoint (runs every time container starts)
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Default command (can be overridden by docker-compose.yml)
CMD ["sleep", "infinity"]

