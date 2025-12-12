#!/bin/bash
set -e

# ============================================================
# Booking Rush - Uninstall Add-ons
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

print_header() {
    echo -e "\n${BLUE}============================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================================${NC}\n"
}

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no "$SSH_USER@$HOST" "$@"
}

# ============================================================
# Uninstall Functions
# ============================================================
uninstall_postgresql() {
    print_header "Uninstalling PostgreSQL"
    ssh_cmd "helm uninstall booking-rush-pg -n $NAMESPACE" || print_warning "PostgreSQL not found"
    ssh_cmd "kubectl delete pvc -l app.kubernetes.io/name=postgresql -n $NAMESPACE" || true
    print_success "PostgreSQL uninstalled"
}

uninstall_redis() {
    print_header "Uninstalling Redis"
    ssh_cmd "helm uninstall booking-rush-redis -n $NAMESPACE" || print_warning "Redis not found"
    ssh_cmd "kubectl delete pvc -l app.kubernetes.io/name=redis -n $NAMESPACE" || true
    print_success "Redis uninstalled"
}

uninstall_mongodb() {
    print_header "Uninstalling MongoDB"
    ssh_cmd "helm uninstall booking-rush-mongodb -n $NAMESPACE" || print_warning "MongoDB not found"
    ssh_cmd "kubectl delete pvc -l app.kubernetes.io/name=mongodb -n $NAMESPACE" || true
    print_success "MongoDB uninstalled"
}

uninstall_redpanda() {
    print_header "Uninstalling Redpanda"
    ssh_cmd "helm uninstall booking-rush-redpanda -n $NAMESPACE" || print_warning "Redpanda not found"
    ssh_cmd "kubectl delete pvc -l app.kubernetes.io/name=redpanda -n $NAMESPACE" || true
    print_success "Redpanda uninstalled"
}

uninstall_node_exporter() {
    print_header "Uninstalling Node Exporter"
    ssh_cmd "helm uninstall booking-rush-node-exporter -n $NAMESPACE" || print_warning "Node Exporter not found"
    print_success "Node Exporter uninstalled"
}

uninstall_all() {
    print_header "Uninstalling ALL Add-ons"
    uninstall_postgresql
    uninstall_redis
    uninstall_mongodb
    uninstall_redpanda
    uninstall_node_exporter
    print_success "All add-ons uninstalled"
}

show_status() {
    print_header "Current Status"
    echo "Helm releases:"
    ssh_cmd "helm list -n $NAMESPACE" || true
    echo ""
    echo "Pods:"
    ssh_cmd "kubectl get pods -n $NAMESPACE" || true
    echo ""
    echo "PVCs:"
    ssh_cmd "kubectl get pvc -n $NAMESPACE" || true
}

# ============================================================
# Menu
# ============================================================
show_menu() {
    echo ""
    echo "Booking Rush - Uninstall Add-ons"
    echo "================================="
    echo "Target: $SSH_USER@$HOST"
    echo "Namespace: $NAMESPACE"
    echo ""
    echo "1) Uninstall ALL"
    echo "2) Uninstall PostgreSQL"
    echo "3) Uninstall Redis"
    echo "4) Uninstall MongoDB"
    echo "5) Uninstall Redpanda"
    echo "6) Uninstall Node Exporter"
    echo "7) Show status"
    echo "0) Exit"
    echo ""
    read -p "Select an option: " OPTION

    case $OPTION in
        1) uninstall_all ;;
        2) uninstall_postgresql ;;
        3) uninstall_redis ;;
        4) uninstall_mongodb ;;
        5) uninstall_redpanda ;;
        6) uninstall_node_exporter ;;
        7) show_status ;;
        0) echo "Exiting..."; exit 0 ;;
        *) print_error "Invalid option"; show_menu ;;
    esac
}

# ============================================================
# Main
# ============================================================
if [ "$1" == "--all" ]; then
    uninstall_all
elif [ "$1" == "--pg" ]; then
    uninstall_postgresql
elif [ "$1" == "--redis" ]; then
    uninstall_redis
elif [ "$1" == "--mongodb" ]; then
    uninstall_mongodb
elif [ "$1" == "--redpanda" ]; then
    uninstall_redpanda
elif [ "$1" == "--node-exporter" ]; then
    uninstall_node_exporter
elif [ "$1" == "--status" ]; then
    show_status
elif [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Uninstall add-ons from k3s"
    echo ""
    echo "Options:"
    echo "  --all            Uninstall all"
    echo "  --pg             Uninstall PostgreSQL"
    echo "  --redis          Uninstall Redis"
    echo "  --mongodb        Uninstall MongoDB"
    echo "  --redpanda       Uninstall Redpanda"
    echo "  --node-exporter  Uninstall Node Exporter"
    echo "  --status         Show current status"
    echo "  --help           Show this help"
else
    show_menu
fi
