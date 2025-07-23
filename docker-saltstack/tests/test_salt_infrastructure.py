import pytest
import subprocess
import time
import json

class TestSaltInfrastructure:
    """Python test suite for Salt infrastructure"""
    
    def test_master_is_running(self):
        """Test that Salt master container is running"""
        result = subprocess.run(['docker', 'ps', '--filter', 'name=salt-master', '--format', '{{.Status}}'], 
                               capture_output=True, text=True)
        assert 'Up' in result.stdout, "Salt master container is not running"
    
    def test_minions_are_running(self):
        """Test that all minion containers are running"""
        expected_minions = ['salt-minion-1', 'salt-minion-2', 'salt-minion-3']
        
        for minion in expected_minions:
            result = subprocess.run(['docker', 'ps', '--filter', f'name={minion}', '--format', '{{.Status}}'], 
                                   capture_output=True, text=True)
            assert 'Up' in result.stdout, f"Minion {minion} is not running"
    
    def test_minion_keys_accepted(self):
        """Test that all minion keys are accepted"""
        result = subprocess.run(['docker', 'exec', 'salt-master', 'salt-key', '-l', 'accepted'], 
                               capture_output=True, text=True)
        
        expected_minions = ['minion-1', 'minion-2', 'minion-3']
        for minion in expected_minions:
            assert minion in result.stdout, f"Minion {minion} key not accepted"
    
    def test_minion_ping(self):
        """Test that all minions respond to ping"""
        expected_minions = ['minion-1', 'minion-2', 'minion-3']
        
        for minion in expected_minions:
            result = subprocess.run(['docker', 'exec', 'salt-master', 'salt', minion, 'test.ping', '--timeout=30'], 
                                   capture_output=True, text=True)
            assert 'True' in result.stdout, f"Minion {minion} not responding to ping"
    
    def test_grains_configuration(self):
        """Test that grains are configured correctly"""
        expected_configs = {
            'minion-1': 'frontend',
            'minion-2': 'backend',
            'minion-3': 'database'
        }
        
        for minion, expected_role in expected_configs.items():
            result = subprocess.run(['docker', 'exec', 'salt-master', 'salt', minion, 'grains.get', 'roles', '--timeout=30'], 
                                   capture_output=True, text=True)
            assert expected_role in result.stdout, f"Minion {minion} missing expected role {expected_role}"
    
    def test_pillar_data(self):
        """Test that pillar data is accessible"""
        result = subprocess.run(['docker', 'exec', 'salt-master', 'salt', '*', 'pillar.get', 'common:environment', '--timeout=30'], 
                               capture_output=True, text=True)
        assert 'docker' in result.stdout, "Pillar data not configured correctly"
    
    def test_state_files_exist(self):
        """Test that state files are applied correctly"""
        result = subprocess.run(['docker', 'exec', 'salt-master', 'salt', '*', 'cmd.run', 'test -f /tmp/salt-test.txt', '--timeout=30'], 
                               capture_output=True, text=True)
        
        # All minions should have the test file
        minion_count = result.stdout.count('True')
        assert minion_count == 3, f"Expected 3 minions to have test file, found {minion_count}"
    
    def test_highstate_application(self):
        """Test that highstate can be applied successfully"""
        result = subprocess.run(['docker', 'exec', 'salt-master', 'salt', '*', 'state.apply', '--timeout=120'], 
                               capture_output=True, text=True)
        
        # Check for successful state application
        assert 'Failed:' not in result.stdout or result.stdout.count('Failed:    0') == 3, "State application failed"
        
if __name__ == '__main__':
    pytest.main([__file__, '-v', '--tb=short'])
