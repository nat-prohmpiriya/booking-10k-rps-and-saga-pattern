#!/bin/bash

# ============================================================
# Booking Rush - Check Add-ons Status
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
HOST="${HOST:-5.75.233.23}"
SSH_USER="${SSH_USER:-root}"
NAMESPACE="booking-rush"

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no "$SSH_USER@$HOST" "$@" 2>/dev/null
}

check_addon() {
    local name=$1
    local release=$2
    local pod_label=$3

    # Check helm release
    if ssh_cmd "helm status $release -n $NAMESPACE" &>/dev/null; then
        # Check pod status
        local pod_status=$(ssh_cmd "kubectl get pods -n $NAMESPACE -l $pod_label -o jsonpath='{.items[0].status.phase}'" 2>/dev/null)
        local ready=$(ssh_cmd "kubectl get pods -n $NAMESPACE -l $pod_label -o jsonpath='{.items[0].status.containerStatuses[0].ready}'" 2>/dev/null)

        if [ "$pod_status" == "Running" ] && [ "$ready" == "true" ]; then
            echo -e "${GREEN}✓${NC} $name: ${GREEN}Running${NC}"
            return 0
        elif [ "$pod_status" == "Running" ]; then
            echo -e "${YELLOW}⏳${NC} $name: ${YELLOW}Starting${NC}"
            return 1
        else
            echo -e "${RED}✗${NC} $name: ${RED}$pod_status${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗${NC} $name: ${RED}Not Installed${NC}"
        return 1
    fi
}

echo ""
echo -e "${BLUE}============================================================${NC}"
echo -e "${BLUE}Booking Rush - Add-ons Status${NC}"
echo -e "${BLUE}============================================================${NC}"
echo ""
echo "Target: $SSH_USER@$HOST"
echo "Namespace: $NAMESPACE"
echo ""
echo -e "${BLUE}--- Add-ons ---${NC}"
echo ""

check_addon "PostgreSQL" "booking-rush-pg" "app.kubernetes.io/name=postgresql"
check_addon "Redis" "booking-rush-redis" "app.kubernetes.io/name=redis"
check_addon "MongoDB" "booking-rush-mongodb" "app.kubernetes.io/name=mongodb"
check_addon "Redpanda" "booking-rush-redpanda" "app.kubernetes.io/name=redpanda"
check_addon "Node Exporter" "booking-rush-node-exporter" "app.kubernetes.io/name=prometheus-node-exporter"

echo ""
echo -e "${BLUE}--- Connection Info ---${NC}"
echo ""

# Show connection info only for running services
if ssh_cmd "helm status booking-rush-pg -n $NAMESPACE" &>/dev/null; then
    echo "PostgreSQL:"
    echo "  Host: booking-rush-pg-postgresql.$NAMESPACE.svc.cluster.local:5432"
fi

if ssh_cmd "helm status booking-rush-redis -n $NAMESPACE" &>/dev/null; then
    echo "Redis:"
    echo "  Host: booking-rush-redis-master.$NAMESPACE.svc.cluster.local:6379"
fi

if ssh_cmd "helm status booking-rush-mongodb -n $NAMESPACE" &>/dev/null; then
    echo "MongoDB:"
    echo "  Host: booking-rush-mongodb.$NAMESPACE.svc.cluster.local:27017"
fi

if ssh_cmd "helm status booking-rush-redpanda -n $NAMESPACE" &>/dev/null; then
    echo "Redpanda:"
    echo "  Host: booking-rush-redpanda.$NAMESPACE.svc.cluster.local:9092"
fi

if ssh_cmd "helm status booking-rush-node-exporter -n $NAMESPACE" &>/dev/null; then
    echo "Node Exporter:"
    echo "  Metrics: booking-rush-node-exporter.$NAMESPACE.svc.cluster.local:9100/metrics"
fi

echo ""
