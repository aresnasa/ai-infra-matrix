const { test, expect } = require('@playwright/test');

test.describe('Navigation Config API Tests', () => {
  let authCookie;

  test.beforeAll(async ({ browser }) => {
    // Login to get authentication
    const context = await browser.newContext();
    const page = await context.newPage();

    await page.goto('http://192.168.0.200:8080/login');
    await page.fill('input[name="username"]', 'admin');
    await page.fill('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    
    // Wait for redirect after login
    await page.waitForURL('http://192.168.0.200:8080/', { timeout: 5000 });
    
    // Get authentication cookies
    const cookies = await context.cookies();
    const sessionCookie = cookies.find(c => c.name === 'token' || c.name === 'session');
    if (sessionCookie) {
      authCookie = `${sessionCookie.name}=${sessionCookie.value}`;
    }

    await context.close();
  });

  test('should return 200 from /api/navigation/config', async ({ request }) => {
    const response = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });

    console.log('Navigation config response status:', response.status());
    console.log('Navigation config response body:', await response.text());

    expect(response.status()).toBe(200);
    
    const data = await response.json();
    expect(data).toHaveProperty('status');
    expect(data.status).toBe('success');
  });

  test('should return navigation items in config', async ({ request }) => {
    const response = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });

    expect(response.status()).toBe(200);
    
    const data = await response.json();
    expect(data).toHaveProperty('data');
    expect(data.data).toHaveProperty('items');
    expect(Array.isArray(data.data.items)).toBe(true);
    
    console.log('Navigation items count:', data.data.items.length);
    console.log('Navigation items:', JSON.stringify(data.data.items, null, 2));
  });

  test('should include monitoring in navigation items', async ({ request }) => {
    const response = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });

    expect(response.status()).toBe(200);
    
    const data = await response.json();
    const items = data.data.items;
    
    const monitoringItem = items.find(item => item.path === '/monitoring');
    expect(monitoringItem).toBeDefined();
    expect(monitoringItem.name).toBe('监控仪表板');
    expect(monitoringItem.icon).toBe('monitor');
    
    console.log('Monitoring item found:', JSON.stringify(monitoringItem, null, 2));
  });

  test('should return default navigation config', async ({ request }) => {
    const response = await request.get('http://192.168.0.200:8080/api/navigation/default', {
      headers: {
        'Cookie': authCookie
      }
    });

    console.log('Default navigation response status:', response.status());
    expect(response.status()).toBe(200);
    
    const data = await response.json();
    expect(data).toHaveProperty('status');
    expect(data.status).toBe('success');
    expect(data.data).toHaveProperty('items');
    expect(Array.isArray(data.data.items)).toBe(true);
    
    console.log('Default navigation items count:', data.data.items.length);
  });

  test('should be able to save custom navigation config', async ({ request }) => {
    // First get current config
    const getResponse = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });
    
    const currentData = await getResponse.json();
    const items = currentData.data.items;
    
    // Modify order - move monitoring to first position
    const monitoringIndex = items.findIndex(item => item.path === '/monitoring');
    if (monitoringIndex > 0) {
      const monitoring = items.splice(monitoringIndex, 1)[0];
      items.unshift(monitoring);
    }
    
    // Save modified config
    const saveResponse = await request.post('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie,
        'Content-Type': 'application/json'
      },
      data: {
        items: items
      }
    });

    console.log('Save navigation response status:', saveResponse.status());
    console.log('Save navigation response body:', await saveResponse.text());
    
    expect(saveResponse.status()).toBe(200);
    
    const saveData = await saveResponse.json();
    expect(saveData.status).toBe('success');
    
    // Verify saved config
    const verifyResponse = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });
    
    const verifyData = await verifyResponse.json();
    const firstItem = verifyData.data.items[0];
    
    console.log('First item after save:', JSON.stringify(firstItem, null, 2));
    expect(firstItem.path).toBe('/monitoring');
  });

  test('should be able to reset navigation config', async ({ request }) => {
    // Reset to default
    const resetResponse = await request.delete('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });

    console.log('Reset navigation response status:', resetResponse.status());
    expect(resetResponse.status()).toBe(200);
    
    const resetData = await resetResponse.json();
    expect(resetData.status).toBe('success');
    
    // Verify config is reset to default
    const verifyResponse = await request.get('http://192.168.0.200:8080/api/navigation/config', {
      headers: {
        'Cookie': authCookie
      }
    });
    
    const verifyData = await verifyResponse.json();
    console.log('Navigation items after reset:', verifyData.data.items.length);
    
    expect(verifyData.data.items.length).toBeGreaterThan(0);
  });
});
