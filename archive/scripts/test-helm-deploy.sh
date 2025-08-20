#!/bin/bash

# Helm Chart Test Deployment Script
# This script tests the ai-infra-matrix Helm chart deployment

set -e

# Configuration
CHART_PATH="./helm/ai-infra-matrix"
RELEASE_NAME="ai-infra-test"
NAMESPACE="ai-infra-matrix"
TIMEOUT="10m"

echo "🚀 Starting Helm Chart Test Deployment..."
echo "Chart Path: $CHART_PATH"
echo "Release Name: $RELEASE_NAME"
echo "Namespace: $NAMESPACE"
echo ""

# Function to print section headers
print_section() {
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

# Clean up function
cleanup() {
    print_section "🧹 Cleaning up existing deployment"
    helm uninstall $RELEASE_NAME -n $NAMESPACE --ignore-not-found || true
    kubectl delete namespace $NAMESPACE --ignore-not-found || true
    echo "Cleanup completed"
    echo ""
}

# Validate Helm chart
validate_chart() {
    print_section "✅ Validating Helm Chart"
    
    # Check if chart directory exists
    if [ ! -d "$CHART_PATH" ]; then
        echo "❌ Chart directory not found: $CHART_PATH"
        exit 1
    fi
    
    # Validate chart syntax
    echo "Linting Helm chart..."
    helm lint $CHART_PATH
    
    # Check dependencies
    echo "Updating chart dependencies..."
    helm dependency update $CHART_PATH
    
    echo "✅ Chart validation completed"
    echo ""
}

# Deploy function
deploy() {
    print_section "🚀 Deploying Helm Chart"
    
    # Create namespace
    echo "Creating namespace: $NAMESPACE"
    kubectl create namespace $NAMESPACE || true
    
    # Install chart
    echo "Installing Helm chart..."
    helm install $RELEASE_NAME $CHART_PATH \
        --namespace $NAMESPACE \
        --timeout $TIMEOUT \
        --wait \
        --debug
    
    echo "✅ Deployment completed"
    echo ""
}

# Verify deployment
verify_deployment() {
    print_section "🔍 Verifying Deployment"
    
    # Check all pods
    echo "Checking pod status..."
    kubectl get pods -n $NAMESPACE
    echo ""
    
    # Check services
    echo "Checking services..."
    kubectl get services -n $NAMESPACE
    echo ""
    
    # Check persistent volumes
    echo "Checking persistent volumes..."
    kubectl get pvc -n $NAMESPACE
    echo ""
    
    # Check configmaps
    echo "Checking configmaps..."
    kubectl get configmaps -n $NAMESPACE
    echo ""
    
    # Check secrets
    echo "Checking secrets..."
    kubectl get secrets -n $NAMESPACE
    echo ""
    
    # Wait for pods to be ready
    echo "Waiting for pods to be ready..."
    kubectl wait --for=condition=Ready pods --all -n $NAMESPACE --timeout=300s || true
    
    echo "✅ Verification completed"
    echo ""
}

# Port forward function
setup_port_forward() {
    print_section "🌐 Setting up Port Forwarding"
    
    # Get nginx service
    NGINX_SERVICE=$(kubectl get service -n $NAMESPACE -l app.kubernetes.io/component=nginx -o jsonpath='{.items[0].metadata.name}')
    
    if [ -n "$NGINX_SERVICE" ]; then
        echo "Setting up port forward for nginx service: $NGINX_SERVICE"
        echo "Access the application at: http://localhost:8080"
        echo ""
        echo "To access the application, run:"
        echo "kubectl port-forward service/$NGINX_SERVICE -n $NAMESPACE 8080:80"
        echo ""
        echo "Press Ctrl+C to stop port forwarding"
        
        # Uncomment the next line to automatically start port forwarding
        # kubectl port-forward service/$NGINX_SERVICE -n $NAMESPACE 8080:80
    else
        echo "❌ Nginx service not found"
    fi
}

# Main execution
main() {
    case "${1:-deploy}" in
        "clean")
            cleanup
            ;;
        "validate")
            validate_chart
            ;;
        "deploy")
            validate_chart
            deploy
            verify_deployment
            setup_port_forward
            ;;
        "verify")
            verify_deployment
            ;;
        "port-forward")
            setup_port_forward
            ;;
        "redeploy")
            cleanup
            sleep 5
            validate_chart
            deploy
            verify_deployment
            setup_port_forward
            ;;
        *)
            echo "Usage: $0 {clean|validate|deploy|verify|port-forward|redeploy}"
            echo ""
            echo "Commands:"
            echo "  clean       - Remove existing deployment"
            echo "  validate    - Validate Helm chart only"
            echo "  deploy      - Full deployment process"
            echo "  verify      - Verify existing deployment"
            echo "  port-forward - Setup port forwarding"
            echo "  redeploy    - Clean and redeploy"
            exit 1
            ;;
    esac
}

# Check prerequisites
check_prerequisites() {
    print_section "🔧 Checking Prerequisites"
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        echo "❌ kubectl is not installed or not in PATH"
        exit 1
    fi
    
    # Check if helm is available
    if ! command -v helm &> /dev/null; then
        echo "❌ helm is not installed or not in PATH"
        exit 1
    fi
    
    # Check kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        echo "❌ Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    echo "✅ Prerequisites check passed"
    echo ""
}

# Run prerequisites check
check_prerequisites

# Execute main function with arguments
main "$@"
