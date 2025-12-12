#!/bin/bash
set -e

# ============================================================
# Booking Rush - Install k3s
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
# Install k3s
# ============================================================
install_k3s() {
    print_header "Installing k3s on $HOST"

    # Check if k3s is already installed
    if ssh_cmd "command -v k3s" &>/dev/null; then
        print_warning "k3s is already installed"
        ssh_cmd "k3s --version"
        return 0
    fi

    echo "Installing k3s..."
    ssh_cmd "curl -sfL https://get.k3s.io | sh -s -"

    # Wait for k3s to be ready
    echo "Waiting for k3s to be ready..."
    sleep 10

    # Check k3s status
    if ssh_cmd "systemctl is-active k3s" &>/dev/null; then
        print_success "k3s installed and running"
        ssh_cmd "k3s --version"
    else
        print_error "k3s installation failed"
        return 1
    fi
}

# ============================================================
# Configure kubectl
# ============================================================
configure_kubectl() {
    print_header "Configuring kubectl"

    # Get kubeconfig
    ssh_cmd "cat /etc/rancher/k3s/k3s.yaml" > /tmp/k3s-kubeconfig.yaml

    # Replace localhost with server IP
    sed -i.bak "s/127.0.0.1/$HOST/g" /tmp/k3s-kubeconfig.yaml

    echo ""
    echo "Kubeconfig saved to: /tmp/k3s-kubeconfig.yaml"
    echo ""
    echo "To use kubectl from local machine:"
    echo "  export KUBECONFIG=/tmp/k3s-kubeconfig.yaml"
    echo "  kubectl get nodes"
    echo ""
    print_success "kubectl configured"
}

# ============================================================
# Verify Installation
# ============================================================
verify_installation() {
    print_header "Verifying Installation"

    echo "Nodes:"
    ssh_cmd "kubectl get nodes"
    echo ""
    echo "System pods:"
    ssh_cmd "kubectl get pods -n kube-system"
    echo ""
    print_success "k3s is ready"
}

# ============================================================
# Main
# ============================================================
print_header "Booking Rush - k3s Installation"
echo "Target: $SSH_USER@$HOST"
echo ""

if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Install k3s on remote server"
    echo ""
    echo "Options:"
    echo "  --uninstall    Uninstall k3s"
    echo "  --status       Show k3s status"
    echo "  --help         Show this help"
    exit 0
fi

if [ "$1" == "--uninstall" ]; then
    print_header "Uninstalling k3s"
    ssh_cmd "k3s-uninstall.sh" || print_warning "k3s not installed"
    print_success "k3s uninstalled"
    exit 0
fi

if [ "$1" == "--status" ]; then
    print_header "k3s Status"
    ssh_cmd "systemctl status k3s" || print_warning "k3s not running"
    echo ""
    ssh_cmd "kubectl get nodes" || true
    exit 0
fi

install_k3s
configure_kubectl
verify_installation

echo ""
print_success "k3s installation complete!"
echo ""
echo "Next steps:"
echo "  1. Run ./02-install-addon.sh --all  # Install databases"
echo "  2. Run ./05-deploy-services.sh      # Deploy services"
echo ""
