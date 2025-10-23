const { test, expect } = require('@playwright/test');

test.describe('Nightingale Initialization Verification', () => {
  test('verify Nightingale database creation in backend-init', async () => {
    const fs = require('fs');
    const path = require('path');
    
    const filePath = path.join(__dirname, '../../../src/backend/cmd/init/main.go');
    const content = fs.readFileSync(filePath, 'utf-8');

    console.log('\n=== Nightingale Database Initialization Check ===\n');

    // Check if createNightingaleDatabase function exists
    const hasCreateFunction = content.includes('createNightingaleDatabase');
    console.log('1. Has createNightingaleDatabase function:', hasCreateFunction ? '✅ YES' : '❌ NO');
    expect(hasCreateFunction).toBe(true);

    // Check if initializeNightingaleAdmin function exists
    const hasInitAdminFunction = content.includes('initializeNightingaleAdmin');
    console.log('2. Has initializeNightingaleAdmin function:', hasInitAdminFunction ? '✅ YES' : '❌ NO');
    expect(hasInitAdminFunction).toBe(true);

    // Check if the function is called in main
    const isCalledInMain = content.includes('if err := createNightingaleDatabase(cfg)');
    console.log('3. Function called in main():', isCalledInMain ? '✅ YES' : '❌ NO');
    expect(isCalledInMain).toBe(true);

    // Check for admin credentials setup
    const hasAdminCredentials = content.includes('Username: root') && content.includes('Password: root.2020');
    console.log('4. Admin credentials configured:', hasAdminCredentials ? '✅ YES' : '❌ NO');
    expect(hasAdminCredentials).toBe(true);

    console.log('\n✅ All Nightingale initialization checks passed!\n');
  });

  test('verify Nightingale database name matches config', async () => {
    const fs = require('fs');
    const path = require('path');

    console.log('\n=== Nightingale Database Name Verification ===\n');

    // Check backend-init code
    const initPath = path.join(__dirname, '../../../src/backend/cmd/init/main.go');
    const initContent = fs.readFileSync(initPath, 'utf-8');
    
    const hasNightingaleDB = initContent.includes('NIGHTINGALE_DB_NAME') && 
                             initContent.includes('"nightingale"');
    console.log('1. Backend-init uses "nightingale" database:', hasNightingaleDB ? '✅ YES' : '❌ NO');
    expect(hasNightingaleDB).toBe(true);

    // Check Nightingale config
    const configPath = path.join(__dirname, '../../../src/nightingale/etc/config.toml');
    const configContent = fs.readFileSync(configPath, 'utf-8');
    
    const configHasNightingaleDB = configContent.includes('dbname=nightingale');
    console.log('2. Nightingale config.toml uses "nightingale" database:', configHasNightingaleDB ? '✅ YES' : '❌ NO');
    expect(configHasNightingaleDB).toBe(true);

    console.log('\n✅ Database name configuration is consistent!\n');
  });

  test('print deployment instructions', async () => {
    console.log('\n');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('  Nightingale 初始化配置完成');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('');
    console.log('✅ 已完成的修改:');
    console.log('  1. backend-init 中添加了 createNightingaleDatabase() 函数');
    console.log('  2. backend-init 中添加了 initializeNightingaleAdmin() 函数');
    console.log('  3. 自动创建 nightingale 数据库');
    console.log('  4. 自动创建 root 管理员账号');
    console.log('');
    console.log('📦 重新构建和部署:');
    console.log('  # 重新构建 backend-init');
    console.log('  docker-compose build backend-init');
    console.log('');
    console.log('  # 重新运行初始化（会自动停止并重新创建）');
    console.log('  docker-compose up backend-init');
    console.log('');
    console.log('  # 或者重启整个堆栈');
    console.log('  docker-compose down');
    console.log('  docker-compose up -d');
    console.log('');
    console.log('🔐 Nightingale 默认管理员账号:');
    console.log('  用户名: root');
    console.log('  密码: root.2020');
    console.log('');
    console.log('🌐 访问 Nightingale:');
    console.log('  直接访问: http://192.168.18.114:17000');
    console.log('  通过代理: http://192.168.18.114:8080/monitoring');
    console.log('');
    console.log('⚠️ 注意事项:');
    console.log('  1. 首次登录后请立即修改默认密码');
    console.log('  2. 确保 .env 中配置了正确的 POSTGRES_PASSWORD');
    console.log('  3. Nightingale 需要等待数据库初始化完成后才能正常使用');
    console.log('  4. 如果 users 表不存在，Nightingale 会在启动时自动创建');
    console.log('');
    console.log('🔍 检查初始化日志:');
    console.log('  docker-compose logs backend-init | grep -i nightingale');
    console.log('');
    console.log('🔧 故障排查:');
    console.log('  # 检查数据库是否创建');
    console.log('  docker exec ai-infra-postgres psql -U postgres -l');
    console.log('');
    console.log('  # 检查 Nightingale 容器状态');
    console.log('  docker-compose ps nightingale');
    console.log('');
    console.log('  # 查看 Nightingale 日志');
    console.log('  docker-compose logs nightingale');
    console.log('');
    console.log('═══════════════════════════════════════════════════════════');
    console.log('\n');
  });
});
