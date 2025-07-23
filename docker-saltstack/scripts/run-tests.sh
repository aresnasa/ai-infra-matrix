#!/bin/bash
set -e

echo "Starting Salt Infrastructure Tests..."

# Configuration
MASTER_HOST="salt-master"
MASTER_PORT="4506"
EXPECTED_MINIONS=("minion-1" "minion-2" "minion-3")
MAX_WAIT_TIME=300
POLL_INTERVAL=10

# Test functions
test_master_connectivity() {
    echo "Testing Salt Master connectivity..."
    if ! nc -z $MASTER_HOST $MASTER_PORT; then
        echo "ERROR: Cannot connect to Salt Master at $MASTER_HOST:$MASTER_PORT"
        return 1
    fi
    echo "✓ Salt Master is reachable"
}

test_minion_acceptance() {
    echo "Testing minion key acceptance..."
    
    local start_time=$(date +%s)
    local all_accepted=false
    
    while [ $all_accepted == false ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $MAX_WAIT_TIME ]; then
            echo "ERROR: Timeout waiting for minions to be accepted"
            return 1
        fi
        
        # Check if all expected minions are accepted
        local accepted_count=0
        for minion in "${EXPECTED_MINIONS[@]}"; do
            if docker exec salt-master salt-key -l accepted | grep -q "^$minion$"; then
                ((accepted_count++))
            fi
        done
        
        if [ $accepted_count -eq ${#EXPECTED_MINIONS[@]} ]; then
            all_accepted=true
            echo "✓ All minions accepted: ${EXPECTED_MINIONS[*]}"
        else
            echo "Waiting for minions to be accepted... ($accepted_count/${#EXPECTED_MINIONS[@]})"
            sleep $POLL_INTERVAL
        fi
    done
}

test_minion_connectivity() {
    echo "Testing minion connectivity..."
    
    for minion in "${EXPECTED_MINIONS[@]}"; do
        echo "Testing connectivity to $minion..."
        if docker exec salt-master salt "$minion" test.ping --timeout=30 | grep -q "True"; then
            echo "✓ $minion is responding to ping"
        else
            echo "ERROR: $minion is not responding to ping"
            return 1
        fi
    done
}

test_state_application() {
    echo "Testing state application..."
    
    # Apply highstate to all minions
    echo "Applying highstate to all minions..."
    if docker exec salt-master salt '*' state.apply --timeout=60; then
        echo "✓ State application completed"
    else
        echo "ERROR: State application failed"
        return 1
    fi
}

test_configuration_consistency() {
    echo "Testing configuration consistency..."
    
    # Check if test files exist on all minions
    for minion in "${EXPECTED_MINIONS[@]}"; do
        echo "Checking configuration on $minion..."
        if docker exec salt-master salt "$minion" cmd.run "test -f /tmp/salt-test.txt" --timeout=30 | grep -q "True"; then
            echo "✓ $minion has correct base configuration"
        else
            echo "ERROR: $minion missing base configuration"
            return 1
        fi
    done
}

test_grains_consistency() {
    echo "Testing grains consistency..."
    
    # Check grains for each minion
    local expected_grains=(
        "minion-1:frontend"
        "minion-2:backend" 
        "minion-3:database"
    )
    
    for grain_check in "${expected_grains[@]}"; do
        local minion=$(echo $grain_check | cut -d: -f1)
        local expected_role=$(echo $grain_check | cut -d: -f2)
        
        echo "Checking grains on $minion..."
        if docker exec salt-master salt "$minion" grains.get roles --timeout=30 | grep -q "$expected_role"; then
            echo "✓ $minion has correct role: $expected_role"
        else
            echo "ERROR: $minion missing expected role: $expected_role"
            return 1
        fi
    done
}

test_pillar_data() {
    echo "Testing pillar data..."
    
    for minion in "${EXPECTED_MINIONS[@]}"; do
        echo "Checking pillar data on $minion..."
        if docker exec salt-master salt "$minion" pillar.get common:environment --timeout=30 | grep -q "docker"; then
            echo "✓ $minion has correct pillar data"
        else
            echo "ERROR: $minion missing pillar data"
            return 1
        fi
    done
}

generate_test_report() {
    echo "Generating test report..."
    
    local report_file="/tests/test-report-$(date +%Y%m%d-%H%M%S).txt"
    
    {
        echo "Salt Infrastructure Test Report"
        echo "=============================="
        echo "Test Date: $(date)"
        echo "Expected Minions: ${EXPECTED_MINIONS[*]}"
        echo ""
        
        echo "Accepted Keys:"
        docker exec salt-master salt-key -l accepted
        echo ""
        
        echo "Minion Status:"
        docker exec salt-master salt-run manage.status
        echo ""
        
        echo "Grains Summary:"
        for minion in "${EXPECTED_MINIONS[@]}"; do
            echo "=== $minion ==="
            docker exec salt-master salt "$minion" grains.items --timeout=30 2>/dev/null || echo "Failed to get grains"
            echo ""
        done
        
    } > "$report_file"
    
    echo "Test report saved to: $report_file"
}

# Main test execution
main() {
    echo "Salt Infrastructure Automated Test Suite"
    echo "========================================"
    
    local start_time=$(date +%s)
    local failed_tests=0
    
    # Run tests
    test_master_connectivity || ((failed_tests++))
    test_minion_acceptance || ((failed_tests++))
    test_minion_connectivity || ((failed_tests++))
    test_state_application || ((failed_tests++))
    test_configuration_consistency || ((failed_tests++))
    test_grains_consistency || ((failed_tests++))
    test_pillar_data || ((failed_tests++))
    
    # Generate report
    generate_test_report
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "========================================"
    echo "Test Summary:"
    echo "Duration: ${duration}s"
    echo "Failed Tests: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        echo "✓ All tests passed!"
        exit 0
    else
        echo "✗ Some tests failed!"
        exit 1
    fi
}

# Run main function
main "$@"
