#!/bin/bash
set -e

# ============================================================
# Booking Rush - Undeploy Services
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
# Undeploy Functions
# ============================================================
undeploy_service() {
    local name=$1
    echo "Removing $name..."
    ssh_cmd "kubectl delete deployment $name -n $NAMESPACE" 2>/dev/null || print_warning "$name deployment not found"
    ssh_cmd "kubectl delete service $name -n $NAMESPACE" 2>/dev/null || print_warning "$name service not found"
    print_success "$name removed"
}

undeploy_all_services() {
    print_header "Removing All Services"

    undeploy_service "api-gateway"
    undeploy_service "auth-service"
    undeploy_service "ticket-service"
    undeploy_service "booking-service"
    undeploy_service "payment-service"
    undeploy_service "frontend-web"

    # Remove NodePort services
    ssh_cmd "kubectl delete service api-gateway-nodeport -n $NAMESPACE" 2>/dev/null || true
    ssh_cmd "kubectl delete service frontend-web-nodeport -n $NAMESPACE" 2>/dev/null || true

    print_success "All services removed"
}

undeploy_ingress() {
    print_header "Removing Ingress"
    ssh_cmd "kubectl delete ingress booking-rush-ingress -n $NAMESPACE" 2>/dev/null || print_warning "Ingress not found"
    print_success "Ingress removed"
}

undeploy_config() {
    print_header "Removing ConfigMap and Secrets"
    ssh_cmd "kubectl delete configmap booking-rush-config -n $NAMESPACE" 2>/dev/null || true
    ssh_cmd "kubectl delete secret booking-rush-secrets -n $NAMESPACE" 2>/dev/null || true
    ssh_cmd "kubectl delete secret ghcr-secret -n $NAMESPACE" 2>/dev/null || true
    print_success "Config removed"
}

undeploy_all() {
    print_header "Removing Everything"
    undeploy_ingress
    undeploy_all_services
    undeploy_config
    print_success "All removed (namespace kept for addons)"
}

show_status() {
    print_header "Current Status"

    echo "Pods:"
    ssh_cmd "kubectl get pods -n $NAMESPACE" || true
    echo ""
    echo "Services:"
    ssh_cmd "kubectl get svc -n $NAMESPACE" || true
    echo ""
    echo "Deployments:"
    ssh_cmd "kubectl get deployments -n $NAMESPACE" || true
}

# ============================================================
# Menu
# ============================================================
show_menu() {
    echo ""
    echo "Booking Rush - Undeploy Services"
    echo "================================="
    echo "Target: $SSH_USER@$HOST"
    echo "Namespace: $NAMESPACE"
    echo ""
    echo "1) Remove ALL services"
    echo "2) Remove api-gateway"
    echo "3) Remove auth-service"
    echo "4) Remove ticket-service"
    echo "5) Remove booking-service"
    echo "6) Remove payment-service"
    echo "7) Remove frontend-web"
    echo "8) Remove ingress"
    echo "9) Show status"
    echo "0) Exit"
    echo ""
    read -p "Select an option: " OPTION

    case $OPTION in
        1) undeploy_all ;;
        2) undeploy_service "api-gateway" ;;
        3) undeploy_service "auth-service" ;;
        4) undeploy_service "ticket-service" ;;
        5) undeploy_service "booking-service" ;;
        6) undeploy_service "payment-service" ;;
        7) undeploy_service "frontend-web" ;;
        8) undeploy_ingress ;;
        9) show_status ;;
        0) echo "Exiting..."; exit 0 ;;
        *) print_error "Invalid option"; show_menu ;;
    esac
}

# ============================================================
# Main
# ============================================================
if [ "$1" == "--all" ]; then
    undeploy_all
elif [ "$1" == "--services" ]; then
    undeploy_all_services
elif [ "$1" == "--ingress" ]; then
    undeploy_ingress
elif [ "$1" == "--config" ]; then
    undeploy_config
elif [ "$1" == "--status" ]; then
    show_status
elif [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Remove services from k3s"
    echo ""
    echo "Options:"
    echo "  --all        Remove everything (services, config, ingress)"
    echo "  --services   Remove services only"
    echo "  --ingress    Remove ingress only"
    echo "  --config     Remove configmap and secrets"
    echo "  --status     Show current status"
    echo "  --help       Show this help"
else
    show_menu
fi
