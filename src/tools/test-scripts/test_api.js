const axios = require('axios');

const API_BASE = 'http://localhost:8082/api';

async function testLogin() {
  try {
    console.log('Testing login...');
    const response = await axios.post(`${API_BASE}/auth/login`, {
      username: 'admin',
      password: 'admin123'
    });
    
    console.log('Login successful:', response.status);
    const token = response.data.token;
    console.log('Token received:', token ? 'Yes' : 'No');
    
    // Test clusters API with token
    console.log('\nTesting clusters API...');
    const clustersResponse = await axios.get(`${API_BASE}/kubernetes/clusters`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    });
    
    console.log('Clusters API response status:', clustersResponse.status);
    console.log('Clusters data:', JSON.stringify(clustersResponse.data, null, 2));
    
  } catch (error) {
    console.error('Error:', error.response ? error.response.data : error.message);
  }
}

testLogin();
