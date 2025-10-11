1. 现在我需要在slurm管理页面进行扩展，
2. 需要初始化slurm集群，这里已经有了slurm的软件包构建程序，一个字页能够初始化安装slurm节点，
3. 创建一个src/apphub，部署一个nginx作为web服务，将所有编译好的slurm包放到这个apphub中，创建一个全局源管理容器，支持各类常用工具的deb/rpm包的管理，
4. 这里可以使用go的ssh远程访问test-ssh-01-02-03的容器配置deb源到步骤3.中创建的deb源下载并安装slurm
5. 然后在所有操作都集成到slurm集群管理页面
6. slurm添加节点、执行任务以及其他都需要一个进度，这里需要一个进度显示的通用函数能够看到步骤的执行过程
7. 所有任务都需要能够支持查看进度，
8. 等待作业也是需要能看到进度，需要能够检查到相关slurm任务的情况和阶段
9. 现在为slurm增加一个任务页这个任务页是go程序远程执行安装slurm客户端的进度和其他任务进度的查看子页面，请实现这个功能，结合前端和后端
10. 现在读取整个项目，完成我之前描述的功能，现在期望的是将saltstack和slurm进行融合构建出一个快速部署和扩缩容slurm集群的平台，使用go的ssh库安装saltstack的客户端到test-ssh容器，然后使用saltstack控制节点的slurm程序安装，请继续。按照我的需求设计backend程序和frontend页面，做到输入过程记录到数据库中同时将ssh过程也记录下来避免黑箱，保证全程的透明方便排查错误。
11. 移除了foreign_key_checks，pgsql支持Foreign Key Constraint
12. 在slurm页面增加一个任务拦，所有提交的任务都能在那里显示
13. 需要实现提交任务自动安装saltstack的客户端，同时需要在任务页面中能够查看到，继续修复
14. http://localhost:8080/api/slurm/saltstack/jobs报错404，这里需要实现后端ssh的客户端和安装saltstack的代码然后暴露出api给到frontend调用，否则无法完成任务，请继续
15. 这里需要一个子页面能够管理不同的sbatch脚本模板，用户可以自行管理模板，实现这个目标
16. 需要修改前端能够解析多行输入扩容 SLURM 节点，节点配置，支持解析多行hostname或者ip输入test-ssh01，test-ssh02，test-ssh03，能够支持解析端口，这里端口如果不统一支持解析IP:port这种格式或者hostname:port这种格式，调整前端
17. root@test-ssh01:22\nroot@test-ssh02:22\nroot@test-ssh03:22使用这个输入，报错：{"duration":8004,"error":"SSH连接失败: dial tcp: lookup test-host on 127.0.0.11:53: server misbehaving","host":"test-host","output":"","success":false}，需要修复，这里期望的是能够解析这种多行输入
18. 同时还需要解析不输入端口，test-ssh01这种简单的hosts或者IP172.16.0.23类似这种的，同时前端需要能够校验用户输入的是否正确如果格式错误需要提前提示客户检查配置。
19. SSH连接测试失败: SSH连接失败: dial tcp: lookup test-ssh01 on 127.0.0.11:53: no such host报错换了，现在是前端无法解析这个了，这里应该是frontend填写表单，然后发送给backend执行初始化，请按照这个思路进行修改
20. 修复通过backend安装好minion后无法在saltstack集群中看到新节点的问题，这里需要交叉检查所有后端代码及saltstack集成的问题。无法连接到SaltStack请确认SaltStack服务正在运行且后端API可达。报错
21. 这里的go程序不能写死masterURL: "http://saltstack:8000",而是通过读取.env文件进行配置，这里需要增加.env.example和build.sh脚本来适配
22. 检查全局配置能够支持saltstack相关的服务部署，这里需要避免服务的端口冲突
23. 现在需要使用nginx构建一个apphub作为本项目的二进制包仓库（存放saltstack-minions客户端和slurm客户端，包括rpm和deb包）
24. 现在使用curl测试添加test-ssh01到slurm集群，需要通过go的ssh自动安装saltstack-minions客户端，保证salt的正常，可以通过go的ssh安装minions客户端和slurm客户端，然后记录相关的任务提交日志和任务详情到pgsql中，deb包的源头已经在slurm-deb:25.05.3容器镜像中，需要先将这个容器中的文件拷贝出来放到pkgs/slurm-deb中，然后使用nginx作为deb源/rpm源
25. 接下来需要调试安装slurm-client和saltstack服务保证能够正确安装saltstack客户端和通过golang的ssh的库安装saltstack，然后再用saltstack触发安装slurm的客户端
26. 现在需要将minio对象存储单独增加一个页面，就叫对象存储，这里不止需要minio，还可以接入其他对象存储，如果已经配置了minio，则需要一个子页面和slurm等同等的iframe子页面即可，这里还需要在管理中心中增加一个配置页面能够管理对象存储的配置
27. http://192.168.0.200:8080/object-storage/minio/1现在访问这里能够跳转到minio了，但是有个问题，需要输入minio的ak和sk这两个配置已经在env中申明了，需要在页面中自动配置上，保证无感跳转，调整下这个前端页面，同时，build-all这个函数需要自动适配内网环境和外网环境，保证一个主build-all函数就能在内网机器启动，同时支持在外网机器构建，这里需要读取build.sh的相关函数实现这个功能。
28. 现在使用playwright访问http://192.168.18.137:8080/slurm，然后点击添加节点，添加这三个节点test-ssh01，test-ssh02，test-ssh03，ssh端口22，密码rootpass123，然后1核1g内存1g磁盘，ubuntu22.04系统，然后提交并测试安装saltstack客户端然后使用playwright访问http://192.168.18.137:8080/saltstack进行集群状态检查，期望是能够正确的查询到saltstack的集群状态和节点状态，如果返回失败则需要修复项目中的相关代码。
29. 调整所有dockerfile中的时区为Asia/Shanghai，所有的基础镜像都需要安装lsof，然后在env环境文件中增加一个ntp时钟源的配置能够自定义的配置时钟源 ✅
30. 现在调整kubernetes的子模块，需要能够读取所有k8s对象，这里需要根据1.27.5的版本进行适配包括所有的CRD及其其它k8s对象，然后能够在kubernetes子页面中进行管理，如果client存在多个版本则都需要考虑，保证客户端能够兼容多种版本的k8s就好，这才是符合预期的 ✅
    - 后端: 添加 GetClusterVersion、GetEnhancedDiscovery、CRD 解析等功能
    - API: 新增 /version 和 /enhanced-discovery 端点
    - 前端: ResourceTree、ResourceList、ResourceDetails 组件
    - 路由: /kubernetes/resources 增强管理页面
    - 依赖: @monaco-editor/react, js-yaml
    - 兼容: client-go v0.33.1 支持 k8s 1.16-1.33+（包括 1.27.5）
    - 文档: docs/KUBERNETES_MULTI_VERSION_IMPLEMENTATION.md
    - 快速指南: docs/K8S_QUICK_START.md
31. 现在调整下build.sh脚本的检查规则，由于镜像可能存在拉取失败的情况，需要先拉取所有dockerfile中的依赖镜像，然后再构建，避免这类情况的发生 ✅
    - 新增 extract_base_images() 函数：解析 Dockerfile 中的 FROM 指令
    - 新增 prefetch_base_images() 函数：单个服务构建前预拉取依赖镜像
    - 新增 prefetch_all_base_images() 函数：批量构建前预拉取所有依赖镜像
    - 支持多种 FROM 格式：基本格式、多阶段构建、跨平台构建
    - 智能去重：自动跳过已存在的镜像
    - 错误处理：拉取失败不中断构建流程
    - 进度显示：详细的统计信息（新拉取/已存在/失败）
    - 集成到 build-service 和 build-all 流程
    - 文档: docs/BUILD_IMAGE_PREFETCH.md
32. 需要调整build.sh能够检查哪些镜像是正确构建了，哪些是不正确构建，build-all能够自动的过滤需要的构建镜像，而不是直接使用--no-cache全量构建，这样非常浪费时间 ✅
    - 新增 verify_image_build() 函数：验证镜像是否正确构建（检查存在性、大小、标签）
    - 新增 get_build_status() 函数：获取所有服务的构建状态（OK/MISSING/INVALID）
    - 新增 show_build_status() 函数：显示详细的构建状态报告和统计信息
    - 新增 get_services_to_build() 函数：智能过滤需要构建的服务列表
    - 新增 check-status 命令：检查所有服务的镜像构建状态
    - 修改 build_all_services()：添加步骤0（状态检查）和步骤5（结果验证）
    - 智能构建模式（默认）：只构建缺失或无效的镜像，跳过已成功构建的镜像
    - 强制重建模式（--force）：重新构建所有服务，使用 --no-cache 参数
    - 性能优化：节省50%-99%的构建时间（取决于场景）
    - 测试脚本: scripts/test-build-status.sh（7个测试用例）
    - 文档: docs/BUILD_SMART_STATUS_CHECK.md（功能说明）
    - 总结: docs/REQUIREMENT_32_SUMMARY.md（实施总结）
33. 这里需要改造一下build.sh脚本需要增加一个递增的构建ID，这里构建需要能够匹配到时间戳和文件变化以及镜像变化，如果镜像中的配置文件和本地配置文件无变化，则可以直接使用镜像，不需要重复--no-cache的构建，这样节省时间同时能够保证镜像符合预期
34. 读取apphub和slurm-build两个文件夹中的内容，将这两个容器的配置进行合并，只保留apphub即可，这个apphub作为所有构建工具的工具链镜像进行管理
35. 检查下载基础镜像的函数，这些依赖镜像也需要提前下载，使用build-all需要提前将所有本项目涉及到的镜像都下载一遍，如果配置了内部harbor源则需要对应的添加tag，如果没有配置内部harbor则直接代表在本地测试，这里修复一下，参考之前的构建脚本。
36. 使用playwright进行使用playwright-mcp进行测试，请继续，需要正确测试符合期望