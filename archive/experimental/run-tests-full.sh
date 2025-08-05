#!/bin/bash
set -e

echo "Running Complete Salt Infrastructure Tests..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}=====================================
$1
=====================================${NC}"
}

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if infrastructure is running
check_infrastructure() {
    print_header "Checking Infrastructure Status"
    
    if ! docker-compose ps | grep -q "Up"; then
        print_error "Salt infrastructure is not running. Please run ./start.sh first"
        exit 1
    fi
    
    print_status "Infrastructure is running"
}

# Run bash-based tests
run_bash_tests() {
    print_header "Running Bash-based Tests"
    
    if docker-compose run --rm test-runner; then
        print_status "✓ Bash tests passed"
        return 0
    else
        print_error "✗ Bash tests failed"
        return 1
    fi
}

# Run Python-based tests
run_python_tests() {
    print_header "Running Python-based Tests"
    
    if docker run --rm --network docker-saltstack_salt-network \
        -v "$(pwd)/tests:/tests" \
        -w /tests \
        python:3.11-alpine \
        sh -c "pip install pytest docker && python -m pytest test_salt_infrastructure.py -v"; then
        print_status "✓ Python tests passed"
        return 0
    else
        print_error "✗ Python tests failed"
        return 1
    fi
}

# Generate comprehensive report
generate_report() {
    print_header "Generating Comprehensive Report"
    
    local report_file="salt-infrastructure-report-$(date +%Y%m%d-%H%M%S).html"
    
    cat > "$report_file" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Salt Infrastructure Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 10px; border-radius: 5px; }
        .success { color: green; }
        .error { color: red; }
        .warning { color: orange; }
        .section { margin: 20px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        pre { background-color: #f8f8f8; padding: 10px; border-radius: 3px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Salt Infrastructure Test Report</h1>
        <p>Generated on: $(date)</p>
    </div>
    
    <div class="section">
        <h2>Container Status</h2>
        <pre>$(docker-compose ps)</pre>
    </div>
    
    <div class="section">
        <h2>Accepted Keys</h2>
        <pre>$(docker exec salt-master salt-key -l accepted 2>/dev/null || echo "Failed to get keys")</pre>
    </div>
    
    <div class="section">
        <h2>Minion Status</h2>
        <pre>$(docker exec salt-master salt-run manage.status 2>/dev/null || echo "Failed to get status")</pre>
    </div>
    
    <div class="section">
        <h2>Grains Information</h2>
        <h3>Minion-1</h3>
        <pre>$(docker exec salt-master salt minion-1 grains.items 2>/dev/null || echo "Failed to get grains")</pre>
        
        <h3>Minion-2</h3>
        <pre>$(docker exec salt-master salt minion-2 grains.items 2>/dev/null || echo "Failed to get grains")</pre>
        
        <h3>Minion-3</h3>
        <pre>$(docker exec salt-master salt minion-3 grains.items 2>/dev/null || echo "Failed to get grains")</pre>
    </div>
    
    <div class="section">
        <h2>Network Information</h2>
        <pre>$(docker network inspect docker-saltstack_salt-network 2>/dev/null || echo "Failed to get network info")</pre>
    </div>
    
    <div class="section">
        <h2>Volume Information</h2>
        <pre>$(docker volume ls | grep docker-saltstack || echo "No volumes found")</pre>
    </div>
    
</body>
</html>
EOF
    
    print_status "Report generated: $report_file"
}

# Performance test
run_performance_test() {
    print_header "Running Performance Tests"
    
    print_status "Testing command execution time..."
    
    local start_time=$(date +%s%N)
    docker exec salt-master salt '*' test.ping --timeout=30 > /dev/null 2>&1
    local end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))
    
    print_status "Ping test completed in ${duration}ms"
    
    if [ $duration -lt 5000 ]; then
        print_status "✓ Performance test passed (under 5 seconds)"
        return 0
    else
        print_warning "⚠ Performance test warning (over 5 seconds)"
        return 1
    fi
}

# Configuration drift test
test_configuration_drift() {
    print_header "Testing Configuration Drift Detection"
    
    print_status "Applying configuration and checking for drift..."
    
    # Apply state
    docker exec salt-master salt '*' state.apply --timeout=60 > /dev/null 2>&1
    
    # Test for changes (should be none if configuration is consistent)
    if docker exec salt-master salt '*' state.apply test=True --timeout=60 | grep -q "would be"; then
        print_warning "⚠ Configuration drift detected"
        return 1
    else
        print_status "✓ No configuration drift detected"
        return 0
    fi
}

# Main execution
main() {
    local test_start_time=$(date +%s)
    local failed_tests=0
    
    print_header "Salt Infrastructure Complete Test Suite"
    
    # Pre-flight checks
    check_infrastructure || exit 1
    
    # Run all test suites
    run_bash_tests || ((failed_tests++))
    run_python_tests || ((failed_tests++))
    run_performance_test || ((failed_tests++))
    test_configuration_drift || ((failed_tests++))
    
    # Generate report
    generate_report
    
    # Summary
    local test_end_time=$(date +%s)
    local duration=$((test_end_time - test_start_time))
    
    print_header "Test Summary"
    echo "Total Duration: ${duration}s"
    echo "Failed Test Suites: $failed_tests/4"
    
    if [ $failed_tests -eq 0 ]; then
        print_status "🎉 All test suites passed!"
        exit 0
    else
        print_error "❌ Some test suites failed!"
        exit 1
    fi
}

# Execute main function
main "$@"
