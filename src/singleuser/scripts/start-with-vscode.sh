#!/bin/bash
# Start script for JupyterHub single-user container with VS Code Server support
# This script can start both Jupyter and VS Code Server simultaneously

set -e

# Configuration
WORKSPACE_DIR=${WORKSPACE_DIR:-/home/jovyan/work}
CODE_SERVER_PORT=${CODE_SERVER_PORT:-8443}
CODE_SERVER_BIND_ADDR=${CODE_SERVER_BIND_ADDR:-0.0.0.0:${CODE_SERVER_PORT}}
START_CODE_SERVER=${START_CODE_SERVER:-false}

echo "================================================"
echo "🚀 AI Infra Matrix - Singleuser Container"
echo "================================================"
echo "📁 Workspace: ${WORKSPACE_DIR}"
echo "🖥️  VS Code Server: ${START_CODE_SERVER}"
echo "================================================"

# Ensure workspace directory exists
mkdir -p "${WORKSPACE_DIR}"

# Function to start code-server
start_code_server() {
    echo "🔧 Starting code-server on ${CODE_SERVER_BIND_ADDR}..."
    
    # Export environment variables for code-server
    export CODE_SERVER_BIND_ADDR="${CODE_SERVER_BIND_ADDR}"
    
    # Start code-server in background
    code-server \
        --bind-addr "${CODE_SERVER_BIND_ADDR}" \
        --auth none \
        --disable-telemetry \
        --disable-update-check \
        "${WORKSPACE_DIR}" &
    
    CODE_SERVER_PID=$!
    echo "✅ code-server started (PID: ${CODE_SERVER_PID})"
}

# Function to check if code-server should be started
should_start_code_server() {
    if [ "${START_CODE_SERVER}" = "true" ] || [ "${START_CODE_SERVER}" = "1" ]; then
        return 0
    fi
    return 1
}

# Start code-server if enabled
if should_start_code_server; then
    start_code_server
fi

# If running as JupyterHub single-user
if [ -n "${JUPYTERHUB_API_TOKEN}" ]; then
    echo "🔌 Running in JupyterHub environment"
    # Execute the default start command
    exec start-singleuser.sh "$@"
else
    echo "🔌 Running in standalone mode"
    # Wait for code-server or run Jupyter standalone
    if should_start_code_server; then
        # If code-server is started, keep the container running
        wait
    else
        exec start.sh jupyter lab "$@"
    fi
fi
