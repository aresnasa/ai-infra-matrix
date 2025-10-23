const { test, expect } = require('@playwright/test');
const { Client } = require('pg');

test.describe('Nightingale Database Initialization Tests', () => {
  let pgClient;

  test.beforeAll(async () => {
    // Connect to Nightingale database
    pgClient = new Client({
      host: process.env.DB_HOST || '192.168.18.114',
      port: process.env.DB_PORT || 5432,
      database: process.env.NIGHTINGALE_DB_NAME || 'nightingale',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || 'postgres123',
    });

    try {
      await pgClient.connect();
      console.log('✓ Connected to Nightingale database');
    } catch (error) {
      console.error('❌ Failed to connect to database:', error.message);
      throw error;
    }
  });

  test.afterAll(async () => {
    if (pgClient) {
      await pgClient.end();
    }
  });

  test('check if Nightingale database exists', async () => {
    console.log('\n=== Checking Nightingale Database ===\n');

    const result = await pgClient.query(`
      SELECT datname FROM pg_database WHERE datname = 'nightingale'
    `);

    console.log('Database exists:', result.rows.length > 0 ? '✅ YES' : '❌ NO');
    expect(result.rows.length).toBeGreaterThan(0);
  });

  test('check if required tables exist', async () => {
    console.log('\n=== Checking Required Tables ===\n');

    const requiredTables = [
      'users',
      'user_group',
      'user_group_member',
      'role',
      'target',
      'busi_group',
      'busi_group_member'
    ];

    for (const tableName of requiredTables) {
      const result = await pgClient.query(`
        SELECT EXISTS (
          SELECT FROM information_schema.tables 
          WHERE table_schema = 'public' 
          AND table_name = $1
        )
      `, [tableName]);

      const exists = result.rows[0].exists;
      console.log(`  Table '${tableName}':`, exists ? '✅ EXISTS' : '❌ MISSING');
      expect(exists).toBe(true);
    }
  });

  test('check admin user configuration', async () => {
    console.log('\n=== Checking Admin User ===\n');

    // Check if admin user exists
    const userResult = await pgClient.query(`
      SELECT id, username, nickname, email, roles, password, create_by, update_by
      FROM users 
      WHERE username = 'admin'
    `);

    console.log('Admin user exists:', userResult.rows.length > 0 ? '✅ YES' : '❌ NO');

    if (userResult.rows.length === 0) {
      console.log('❌ Admin user not found!');
      console.log('   Expected: username = "admin"');
      console.log('   This should be synced from main system during backend-init');
      
      // Check if there's a 'root' user instead
      const rootResult = await pgClient.query(`
        SELECT id, username, nickname, email, roles 
        FROM users 
        WHERE username = 'root'
      `);
      
      if (rootResult.rows.length > 0) {
        console.log('\n⚠️  Found "root" user instead:');
        console.log('   Username:', rootResult.rows[0].username);
        console.log('   Nickname:', rootResult.rows[0].nickname);
        console.log('   Email:', rootResult.rows[0].email);
        console.log('   Roles:', rootResult.rows[0].roles);
      }

      // List all users
      const allUsers = await pgClient.query('SELECT id, username, nickname, roles FROM users');
      console.log('\n📋 All users in database:');
      allUsers.rows.forEach(user => {
        console.log(`   - ${user.username} (${user.nickname}) - Roles: ${user.roles}`);
      });

      throw new Error('Admin user not found in Nightingale database');
    }

    const admin = userResult.rows[0];
    console.log('\n✅ Admin User Details:');
    console.log('   ID:', admin.id);
    console.log('   Username:', admin.username);
    console.log('   Nickname:', admin.nickname);
    console.log('   Email:', admin.email);
    console.log('   Roles:', admin.roles);
    console.log('   Password Hash:', admin.password ? '✓ SET' : '✗ NOT SET');
    console.log('   Created By:', admin.create_by);
    console.log('   Updated By:', admin.update_by);

    // Verify admin has Admin role
    expect(admin.roles).toContain('Admin');
    console.log('\n✓ Admin has Admin role');

    // Verify password is set
    expect(admin.password).toBeTruthy();
    console.log('✓ Admin password is configured');
  });

  test('check admin user group', async () => {
    console.log('\n=== Checking Admin User Group ===\n');

    // Check if admin-group exists
    const groupResult = await pgClient.query(`
      SELECT id, name, note, create_by 
      FROM user_group 
      WHERE name = 'admin-group'
    `);

    console.log('Admin group exists:', groupResult.rows.length > 0 ? '✅ YES' : '❌ NO');

    if (groupResult.rows.length === 0) {
      console.log('❌ Admin group not found!');
      
      // List all groups
      const allGroups = await pgClient.query('SELECT id, name, note FROM user_group');
      console.log('\n📋 All user groups:');
      if (allGroups.rows.length === 0) {
        console.log('   (no groups found)');
      } else {
        allGroups.rows.forEach(group => {
          console.log(`   - ${group.name}: ${group.note}`);
        });
      }
      return;
    }

    const group = groupResult.rows[0];
    console.log('   Group ID:', group.id);
    console.log('   Group Name:', group.name);
    console.log('   Note:', group.note);
    console.log('   Created By:', group.create_by);

    // Check if admin is member of admin-group
    const adminUser = await pgClient.query(`SELECT id FROM users WHERE username = 'admin'`);
    
    if (adminUser.rows.length > 0) {
      const memberResult = await pgClient.query(`
        SELECT * FROM user_group_member 
        WHERE group_id = $1 AND user_id = $2
      `, [group.id, adminUser.rows[0].id]);

      console.log('\nAdmin is member of admin-group:', memberResult.rows.length > 0 ? '✅ YES' : '❌ NO');
      expect(memberResult.rows.length).toBeGreaterThan(0);
    }
  });

  test('check roles configuration', async () => {
    console.log('\n=== Checking Roles ===\n');

    const requiredRoles = ['Admin', 'Standard', 'Guest'];

    for (const roleName of requiredRoles) {
      const result = await pgClient.query(`
        SELECT id, name, note 
        FROM role 
        WHERE name = $1
      `, [roleName]);

      const exists = result.rows.length > 0;
      console.log(`  Role '${roleName}':`, exists ? '✅ EXISTS' : '❌ MISSING');
      
      if (exists) {
        console.log(`    Note: ${result.rows[0].note}`);
      }

      expect(exists).toBe(true);
    }
  });

  test('check default business group', async () => {
    console.log('\n=== Checking Default Business Group ===\n');

    const groupResult = await pgClient.query(`
      SELECT id, name, create_by 
      FROM busi_group 
      WHERE name = 'Default Group'
    `);

    console.log('Default business group exists:', groupResult.rows.length > 0 ? '✅ YES' : '❌ NO');

    if (groupResult.rows.length === 0) {
      console.log('❌ Default business group not found!');
      
      // List all business groups
      const allGroups = await pgClient.query('SELECT id, name FROM busi_group');
      console.log('\n📋 All business groups:');
      if (allGroups.rows.length === 0) {
        console.log('   (no business groups found)');
      } else {
        allGroups.rows.forEach(group => {
          console.log(`   - ${group.name}`);
        });
      }
      return;
    }

    const group = groupResult.rows[0];
    console.log('   Group ID:', group.id);
    console.log('   Group Name:', group.name);
    console.log('   Created By:', group.create_by);

    // Check if admin-group is linked to business group
    const adminGroup = await pgClient.query(`SELECT id FROM user_group WHERE name = 'admin-group'`);
    
    if (adminGroup.rows.length > 0) {
      const memberResult = await pgClient.query(`
        SELECT perm_flag 
        FROM busi_group_member 
        WHERE busi_group_id = $1 AND user_group_id = $2
      `, [group.id, adminGroup.rows[0].id]);

      if (memberResult.rows.length > 0) {
        console.log('\nAdmin group linked to business group: ✅ YES');
        console.log('   Permission:', memberResult.rows[0].perm_flag);
        expect(memberResult.rows[0].perm_flag).toBe('rw');
      } else {
        console.log('\nAdmin group linked to business group: ❌ NO');
      }
    }
  });

  test('check main system admin for comparison', async () => {
    console.log('\n=== Checking Main System Admin (for comparison) ===\n');

    // Connect to main database
    const mainClient = new Client({
      host: process.env.DB_HOST || '192.168.18.114',
      port: process.env.DB_PORT || 5432,
      database: process.env.DB_NAME || 'ai_infra',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || 'postgres123',
    });

    try {
      await mainClient.connect();
      
      const result = await mainClient.query(`
        SELECT id, username, email, name, password 
        FROM users 
        WHERE username = 'admin'
      `);

      if (result.rows.length > 0) {
        const mainAdmin = result.rows[0];
        console.log('Main System Admin:');
        console.log('   Username:', mainAdmin.username);
        console.log('   Name:', mainAdmin.name);
        console.log('   Email:', mainAdmin.email);
        console.log('   Password Hash:', mainAdmin.password ? mainAdmin.password.substring(0, 20) + '...' : 'NOT SET');

        // Compare with Nightingale admin
        const nightingaleAdmin = await pgClient.query(`
          SELECT username, email, nickname, password 
          FROM users 
          WHERE username = 'admin'
        `);

        if (nightingaleAdmin.rows.length > 0) {
          const ngAdmin = nightingaleAdmin.rows[0];
          console.log('\nNightingale Admin:');
          console.log('   Username:', ngAdmin.username);
          console.log('   Nickname:', ngAdmin.nickname);
          console.log('   Email:', ngAdmin.email);
          console.log('   Password Hash:', ngAdmin.password ? ngAdmin.password.substring(0, 20) + '...' : 'NOT SET');

          console.log('\n📊 Comparison:');
          console.log('   Username Match:', mainAdmin.username === ngAdmin.username ? '✅' : '❌');
          console.log('   Email Match:', mainAdmin.email === ngAdmin.email ? '✅' : '❌');
          console.log('   Password Match:', mainAdmin.password === ngAdmin.password ? '✅' : '❌');
          
          if (mainAdmin.password !== ngAdmin.password) {
            console.log('\n⚠️  Password hashes do NOT match!');
            console.log('   This means password sync is not working correctly.');
          }
        } else {
          console.log('\n❌ Admin user not found in Nightingale database!');
        }
      } else {
        console.log('❌ Admin user not found in main system!');
      }

      await mainClient.end();
    } catch (error) {
      console.error('Error checking main system:', error.message);
      await mainClient.end();
    }
  });

  test('diagnose initialization issues', async () => {
    console.log('\n=== Diagnosis Summary ===\n');

    const checks = [];

    // Check 1: Tables exist
    const tablesResult = await pgClient.query(`
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public'
      ORDER BY table_name
    `);
    checks.push({
      name: 'Database tables created',
      status: tablesResult.rows.length > 0,
      detail: `${tablesResult.rows.length} tables found`
    });

    // Check 2: Admin user exists
    const adminResult = await pgClient.query(`SELECT COUNT(*) as count FROM users WHERE username = 'admin'`);
    checks.push({
      name: 'Admin user exists',
      status: parseInt(adminResult.rows[0].count) > 0,
      detail: `Count: ${adminResult.rows[0].count}`
    });

    // Check 3: Roles exist
    const rolesResult = await pgClient.query(`SELECT COUNT(*) as count FROM role`);
    checks.push({
      name: 'Roles configured',
      status: parseInt(rolesResult.rows[0].count) >= 3,
      detail: `${rolesResult.rows[0].count} roles found (expected 3+)`
    });

    // Check 4: User groups exist
    const groupsResult = await pgClient.query(`SELECT COUNT(*) as count FROM user_group`);
    checks.push({
      name: 'User groups created',
      status: parseInt(groupsResult.rows[0].count) > 0,
      detail: `${groupsResult.rows[0].count} groups found`
    });

    // Check 5: Business groups exist
    const busiGroupsResult = await pgClient.query(`SELECT COUNT(*) as count FROM busi_group`);
    checks.push({
      name: 'Business groups created',
      status: parseInt(busiGroupsResult.rows[0].count) > 0,
      detail: `${busiGroupsResult.rows[0].count} business groups found`
    });

    console.log('Initialization Status:\n');
    checks.forEach((check, index) => {
      console.log(`${index + 1}. ${check.name}: ${check.status ? '✅' : '❌'}`);
      console.log(`   ${check.detail}`);
    });

    const allPassed = checks.every(check => check.status);
    
    if (!allPassed) {
      console.log('\n⚠️  INITIALIZATION INCOMPLETE!\n');
      console.log('Recommended Actions:');
      console.log('1. Run backend-init:');
      console.log('   docker-compose run --rm backend-init');
      console.log('');
      console.log('2. Check backend-init logs:');
      console.log('   docker-compose logs backend-init');
      console.log('');
      console.log('3. Verify database connection settings in .env');
    } else {
      console.log('\n✅ All initialization checks passed!');
    }
  });
});
