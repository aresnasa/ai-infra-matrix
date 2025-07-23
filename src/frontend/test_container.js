#!/usr/bin/env node

// 容器内测试脚本 - 通过API直接测试前端功能
const http = require('http');
const https = require('https');

class ContainerTester {
    constructor() {
        this.baseURL = 'http://localhost'; // 同容器内nginx地址
        this.apiURL = 'http://backend:8082'; // 后端API地址
        this.token = null;
    }

    // HTTP请求封装
    async makeRequest(url, options = {}) {
        return new Promise((resolve, reject) => {
            const urlObj = new URL(url);
            const isHttps = urlObj.protocol === 'https:';
            const client = isHttps ? https : http;
            
            const requestOptions = {
                hostname: urlObj.hostname,
                port: urlObj.port || (isHttps ? 443 : 80),
                path: urlObj.pathname + urlObj.search,
                method: options.method || 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'User-Agent': 'Container-Test-Agent',
                    ...options.headers
                }
            };

            const req = client.request(requestOptions, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    try {
                        const response = {
                            status: res.statusCode,
                            headers: res.headers,
                            data: data,
                            json: () => JSON.parse(data)
                        };
                        resolve(response);
                    } catch (e) {
                        resolve({
                            status: res.statusCode,
                            headers: res.headers,
                            data: data,
                            json: () => null
                        });
                    }
                });
            });

            req.on('error', reject);

            if (options.body) {
                req.write(typeof options.body === 'string' ? options.body : JSON.stringify(options.body));
            }

            req.end();
        });
    }

    // 测试前端页面是否可访问
    async testFrontendAccess() {
        console.log('📱 测试前端页面访问...');
        try {
            const response = await this.makeRequest(this.baseURL);
            if (response.status === 200) {
                console.log('✅ 前端页面可正常访问');
                return true;
            } else {
                console.log(`❌ 前端页面访问失败: ${response.status}`);
                return false;
            }
        } catch (error) {
            console.log(`❌ 前端页面访问错误: ${error.message}`);
            return false;
        }
    }

    // 测试Kubernetes页面
    async testKubernetesPage() {
        console.log('🚢 测试Kubernetes管理页面...');
        try {
            const response = await this.makeRequest(`${this.baseURL}/kubernetes`);
            if (response.status === 200) {
                console.log('✅ Kubernetes页面可正常访问');
                // 检查页面内容是否包含预期的JavaScript文件
                const content = response.data;
                const hasReactApp = content.includes('react') || content.includes('webpack') || content.includes('main.');
                console.log(`📄 页面包含React应用: ${hasReactApp ? '是' : '否'}`);
                return true;
            } else {
                console.log(`❌ Kubernetes页面访问失败: ${response.status}`);
                return false;
            }
        } catch (error) {
            console.log(`❌ Kubernetes页面访问错误: ${error.message}`);
            return false;
        }
    }

    // 测试API登录
    async testLogin() {
        console.log('🔐 测试API登录...');
        try {
            const response = await this.makeRequest(`${this.apiURL}/api/auth/login`, {
                method: 'POST',
                body: {
                    username: 'admin',
                    password: 'admin123'
                }
            });

            if (response.status === 200) {
                const data = response.json();
                if (data && data.token) {
                    this.token = data.token;
                    console.log('✅ API登录成功，获取到token');
                    return true;
                } else {
                    console.log('❌ API登录响应格式错误');
                    console.log('响应数据:', response.data);
                    return false;
                }
            } else {
                console.log(`❌ API登录失败: ${response.status}`);
                console.log('响应数据:', response.data);
                return false;
            }
        } catch (error) {
            console.log(`❌ API登录错误: ${error.message}`);
            return false;
        }
    }

    // 测试集群API
    async testClustersAPI() {
        console.log('📊 测试集群数据API...');
        if (!this.token) {
            console.log('❌ 没有有效token，无法测试集群API');
            return false;
        }

        try {
            const response = await this.makeRequest(`${this.apiURL}/api/kubernetes/clusters`, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });

            if (response.status === 200) {
                const clusters = response.json();
                console.log(`✅ 集群API调用成功，返回 ${clusters.length} 个集群`);
                
                if (clusters.length > 0) {
                    const firstCluster = clusters[0];
                    console.log(`📋 第一个集群信息:`);
                    console.log(`   名称: ${firstCluster.name}`);
                    console.log(`   状态: ${firstCluster.status}`);
                    console.log(`   API服务器: ${firstCluster.api_server}`);
                    console.log(`   创建时间: ${firstCluster.created_at}`);
                    
                    // 检查是否有连接状态的集群
                    const connectedClusters = clusters.filter(c => c.status === 'connected');
                    console.log(`🔗 已连接集群数量: ${connectedClusters.length}`);
                    
                    return {
                        success: true,
                        totalClusters: clusters.length,
                        connectedClusters: connectedClusters.length,
                        firstClusterStatus: firstCluster.status
                    };
                } else {
                    console.log('⚠️ 没有找到集群数据');
                    return { success: true, totalClusters: 0 };
                }
            } else {
                console.log(`❌ 集群API调用失败: ${response.status}`);
                console.log('响应数据:', response.data);
                return false;
            }
        } catch (error) {
            console.log(`❌ 集群API调用错误: ${error.message}`);
            return false;
        }
    }

    // 测试连接测试API
    async testConnectionAPI() {
        console.log('🔍 测试连接测试API...');
        if (!this.token) {
            console.log('❌ 没有有效token，无法测试连接API');
            return false;
        }

        try {
            // 先获取第一个集群的ID
            const clustersResponse = await this.makeRequest(`${this.apiURL}/api/kubernetes/clusters`, {
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });

            if (clustersResponse.status !== 200) {
                console.log('❌ 无法获取集群列表进行连接测试');
                return false;
            }

            const clusters = clustersResponse.json();
            if (clusters.length === 0) {
                console.log('❌ 没有集群可进行连接测试');
                return false;
            }

            const clusterId = clusters[0].id;
            console.log(`🎯 对集群ID ${clusterId} 进行连接测试...`);

            const testResponse = await this.makeRequest(`${this.apiURL}/api/kubernetes/clusters/${clusterId}/test`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${this.token}`
                }
            });

            if (testResponse.status === 200) {
                const result = testResponse.json();
                console.log('✅ 连接测试API调用成功');
                console.log('测试结果:', JSON.stringify(result, null, 2));
                return true;
            } else {
                console.log(`❌ 连接测试失败: ${testResponse.status}`);
                console.log('响应数据:', testResponse.data);
                return false;
            }
        } catch (error) {
            console.log(`❌ 连接测试错误: ${error.message}`);
            return false;
        }
    }

    // 运行全部测试
    async runAllTests() {
        console.log('🚀 开始容器内端到端测试...\n');
        
        const results = {
            frontendAccess: false,
            kubernetesPage: false,
            apiLogin: false,
            clustersAPI: false,
            connectionTest: false
        };

        // 1. 测试前端访问
        results.frontendAccess = await this.testFrontendAccess();
        console.log('');

        // 2. 测试Kubernetes页面
        results.kubernetesPage = await this.testKubernetesPage();
        console.log('');

        // 3. 测试API登录
        results.apiLogin = await this.testLogin();
        console.log('');

        // 4. 测试集群API
        if (results.apiLogin) {
            results.clustersAPI = await this.testClustersAPI();
            console.log('');

            // 5. 测试连接测试功能
            if (results.clustersAPI) {
                results.connectionTest = await this.testConnectionAPI();
                console.log('');
            }
        }

        // 输出测试总结
        console.log('🏁 测试结果总结:');
        console.log('=====================================');
        console.log(`前端页面访问: ${results.frontendAccess ? '✅ 通过' : '❌ 失败'}`);
        console.log(`Kubernetes页面: ${results.kubernetesPage ? '✅ 通过' : '❌ 失败'}`);
        console.log(`API登录功能: ${results.apiLogin ? '✅ 通过' : '❌ 失败'}`);
        console.log(`集群数据API: ${results.clustersAPI ? '✅ 通过' : '❌ 失败'}`);
        console.log(`连接测试API: ${results.connectionTest ? '✅ 通过' : '❌ 失败'}`);
        
        const totalTests = Object.keys(results).length;
        const passedTests = Object.values(results).filter(Boolean).length;
        console.log(`\n总测试: ${totalTests}, 通过: ${passedTests}, 失败: ${totalTests - passedTests}`);
        
        const allPassed = passedTests === totalTests;
        console.log(`\n🎯 整体结果: ${allPassed ? '✅ 全部通过' : '❌ 部分失败'}`);
        
        return {
            success: allPassed,
            results,
            summary: {
                total: totalTests,
                passed: passedTests,
                failed: totalTests - passedTests
            }
        };
    }
}

// 运行测试
if (require.main === module) {
    const tester = new ContainerTester();
    tester.runAllTests().then(result => {
        process.exit(result.success ? 0 : 1);
    }).catch(error => {
        console.error('💥 测试异常:', error);
        process.exit(1);
    });
}

module.exports = ContainerTester;
