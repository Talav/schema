# Start from the same base image
FROM qmcgaw/godevcontainer:debian

# Install Python 3 + pip (Debian way – very small footprint)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        python3 \
        python3-pip \
        python3-venv && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Create a virtual environment
RUN python3 -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Install MkDocs + Material theme + common useful plugins
# Pin versions if you want reproducibility
RUN pip install --no-cache-dir \
    mkdocs \
    mkdocs-material \
    mkdocs-gen-files \
    mkdocs-literate-nav \
    mkdocs-git-revision-date-localized-plugin \
    pymdown-extensions \
    mkdocs-macros-plugin \
    mkdocs-glightbox \
    mkdocs-awesome-pages-plugin

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

