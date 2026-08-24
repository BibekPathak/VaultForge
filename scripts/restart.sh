#!/usr/bin/env bash
set -euo pipefail

# VaultForge Graceful Restart
# Drains in-flight requests before restarting the API server
#
# Usage:
#   ./scripts/restart.sh           # Restart via systemd
#   ./scripts/restart.sh --docker  # Restart via docker-compose

MODE="${1:-systemd}"
SERVICE="vaultforge-api"

echo "=== VaultForge Graceful Restart ==="

if [ "$MODE" = "--docker" ]; then
    echo "Restarting via Docker Compose..."
    cd docker && docker compose restart vaultforge-api
    echo "Docker restart complete."
    exit 0
fi

echo "Mode: systemd"

# Check if service is running
if ! systemctl is-active --quiet "$SERVICE" 2>/dev/null; then
    echo "Service $SERVICE is not running. Starting..."
    sudo systemctl start "$SERVICE"
    echo "Service started."
    exit 0
fi

# Get current PID
OLD_PID=$(systemctl show -p MainPID --value "$SERVICE")
echo "Current PID: $OLD_PID"

# Send SIGTERM for graceful shutdown
echo "Sending SIGTERM for graceful shutdown..."
sudo systemctl kill -s SIGTERM "$SERVICE"

# Wait for process to exit (up to 30 seconds)
echo "Waiting for graceful shutdown..."
TIMEOUT=30
while [ $TIMEOUT -gt 0 ]; do
    if ! systemctl is-active --quiet "$SERVICE" 2>/dev/null; then
        echo "Service stopped gracefully."
        break
    fi
    sleep 1
    TIMEOUT=$((TIMEOUT - 1))
done

if [ $TIMEOUT -eq 0 ]; then
    echo "WARNING: Service did not stop within 30s. Force killing..."
    sudo systemctl kill -s SIGKILL "$SERVICE"
fi

# Wait for port to be free
echo "Waiting for port 8080 to be free..."
sleep 2

# Start the service
echo "Starting service..."
sudo systemctl start "$SERVICE"

# Wait for service to be ready
echo "Waiting for service to be ready..."
for i in $(seq 1 15); do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        echo "Service is ready!"
        break
    fi
    sleep 1
done

# Verify
NEW_PID=$(systemctl show -p MainPID --value "$SERVICE")
echo ""
echo "=== Restart Complete ==="
echo "New PID: $NEW_PID"
echo "Status: $(systemctl is-active $SERVICE)"
curl -s http://localhost:8080/health 2>/dev/null || echo "WARNING: Health check failed"
