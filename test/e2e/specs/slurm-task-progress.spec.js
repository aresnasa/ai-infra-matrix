/**
 * SLURM Cluster Expansion Progress Bug Test
 * 
 * Tests the task progress bar issue showing 6667% instead of correct percentage
 * 
 * Issue: Progress bar shows incorrect percentage (6667%) for Minion deployment tasks
 */

const { test, expect } = require('@playwright/test');

const SLURM_PAGE_URL = 'http://192.168.0.200:8080/slurm';

test.describe('SLURM Task Progress Bar Tests', () => {
  
  test('should verify task progress bar displays correctly', async ({ page }) => {
    console.log('🔍 Testing SLURM task progress bar...');
    
    await page.goto(SLURM_PAGE_URL);
    await page.waitForLoadState('networkidle');

    // Take initial screenshot
    await page.screenshot({ 
      path: 'test-screenshots/slurm-tasks-initial.png',
      fullPage: true 
    });
    console.log('  📸 Initial screenshot saved');

    // Look for task items
    const taskItems = page.locator('[class*="task"], [class*="Task"], .list-item, .card');
    const taskCount = await taskItems.count();
    console.log(`  📋 Found ${taskCount} potential task items`);

    // Look for progress indicators
    const progressElements = page.locator('text=/\\d+%/');
    const progressCount = await progressElements.count();
    console.log(`  📊 Found ${progressCount} progress indicators`);

    if (progressCount > 0) {
      for (let i = 0; i < progressCount; i++) {
        const progressText = await progressElements.nth(i).textContent();
        const percentage = parseInt(progressText.match(/(\d+)%/)?.[1] || '0');
        
        console.log(`  Progress ${i + 1}: ${progressText}`);
        
        // Check for invalid progress (should be 0-100)
        if (percentage > 100) {
          console.error(`  ❌ INVALID PROGRESS: ${percentage}% (should be 0-100)`);
          
          // Take screenshot of the problematic element
          await progressElements.nth(i).screenshot({
            path: `test-screenshots/invalid-progress-${percentage}.png`
          });
        } else if (percentage === 100) {
          console.log(`  ✓ Valid: ${percentage}% (completed)`);
        } else {
          console.log(`  ✓ Valid: ${percentage}% (in progress)`);
        }
      }
    }

    // Look for the specific 6667% issue
    const invalidProgress = page.locator('text=/6667%/');
    if (await invalidProgress.isVisible()) {
      console.error('  ❌ Found 6667% progress - BUG CONFIRMED!');
      
      // Get the parent element to see context
      const parentElement = invalidProgress.locator('..');
      const parentText = await parentElement.textContent();
      console.log(`  Context: ${parentText}`);
      
      // Take detailed screenshot
      await parentElement.screenshot({
        path: 'test-screenshots/bug-6667-percent.png'
      });
    }

    // Check for "部署Minion到test-ssh03" task
    const minionTask = page.locator('text=/部署Minion|test-ssh03/').first();
    if (await minionTask.isVisible()) {
      console.log('  ✓ Found Minion deployment task');
      const taskText = await minionTask.textContent();
      console.log(`  Task: ${taskText}`);
    }
  });

  test('should check API response for task progress data', async ({ request }) => {
    console.log('🌐 Checking API for task progress data...');
    
    // Try common API endpoints for task status
    const endpoints = [
      '/api/slurm/tasks',
      '/api/tasks',
      '/api/saltstack/tasks',
      '/api/slurm/expansion/tasks',
      '/api/deployment/tasks'
    ];

    for (const endpoint of endpoints) {
      try {
        const response = await request.get(`http://192.168.0.200:8080${endpoint}`);
        if (response.ok()) {
          const data = await response.json();
          console.log(`  ✓ ${endpoint}:`);
          console.log(JSON.stringify(data, null, 2));
          
          // Check for progress values in response
          const dataStr = JSON.stringify(data);
          if (dataStr.includes('6667') || dataStr.includes('progress')) {
            console.log('  📊 Found progress data in response');
          }
        }
      } catch (error) {
        // Endpoint doesn't exist, continue
      }
    }
  });

  test('should inspect page source for progress calculation', async ({ page }) => {
    console.log('🔬 Inspecting page source for progress logic...');
    
    await page.goto(SLURM_PAGE_URL);
    await page.waitForLoadState('networkidle');

    // Get all script tags
    const scripts = await page.locator('script').evaluateAll(elements => 
      elements.map(el => el.textContent || '')
    );

    // Search for progress calculation logic
    for (const script of scripts) {
      if (script.includes('progress') || script.includes('percent')) {
        // Look for calculation patterns
        const progressPatterns = [
          /progress\s*=\s*([^;]+)/gi,
          /percent\s*=\s*([^;]+)/gi,
          /\*\s*100/gi,
          /\/\s*\d+\s*\*\s*100/gi
        ];

        for (const pattern of progressPatterns) {
          const matches = script.match(pattern);
          if (matches) {
            console.log('  📝 Found progress calculation:');
            matches.forEach(match => console.log(`    ${match}`));
          }
        }
      }
    }
  });

  test.afterAll(async () => {
    console.log('✅ Task progress tests completed');
    console.log('📁 Screenshots saved to test-screenshots/');
  });
});
