[测试环境 root@xjhpc-master ~]# for i in $(ls /etc/slurm/*.conf);do echo $i;cat $i;done
/etc/slurm/acct_gather.conf
###
# Slurm acct_gather configuration file
###
# Parameters for acct_gather_energy/impi plugin
#EnergyIPMIFrequency=10
#EnergyIPMICalcAdjustment=yes
#
# Parameters for acct_gather_profile/hdf5 plugin
#ProfileHDF5Dir=/app/slurm/profile_data
# Parameters for acct_gather_interconnect/ofed plugin
#InfinibandOFEDPort=1
/etc/slurm/cgroup.conf
###
#
# Slurm cgroup support configuration file
#
# See man slurm.conf and man cgroup.conf for further
# information on cgroup configuration parameters
#--
# 启用cgroup资源限制，可以防止用户实际使用的资源超过用户为该作业通过作业调度系统申请到的资源。
# 如不设定，注释掉在 /etc/slurm/slurm.conf中 ProctrackType=proctrack/cgroup 及 TaskPlugin=task/cgroup 参数。
# 如需要设定，打开/etc/slurm/slurm.conf中 ProctrackType=proctrack/cgroup 和 TaskPlugin=task/cgroup 参数，
# 还需要设定 /etc/slurm/cgroup.conf 文件内容类似如下：

# Slurm cgroup support configuration file
# See man slurm.conf and man cgroup.conf for further
# information on cgroup configuration parameters
#CgroupAutomount=yes
# Cgroup自动挂载。Slurm cgroup插件需要挂载有效且功能正常的cgroup子系统于 /sys/fs/cgroup/<subsystem_name>。
# 当启动时，插件检查该子系统是否可用。如不可用，该插件将启动失败，直到CgroupAutomount设置为yes。
# 在此情形侠，插件首先尝试挂载所需的子系统。
#CgroupMountpoint=/sys/fs/cgroup 
# 设置cgroup挂载点，该目录应该是可写的，可以含有每个子系统挂载的cgroups。默认在/sys/fs/cgroup。
# CgroupPlugin= 
# CgroupPlugin=[autodetect|cgroup/v1|cgroup/v2] 。
CgroupPlugin=cgroup/v2
# 设置与cgroup子系统交互采用的插件。其值可以为cgroup/v1（支持传统的cgroup v1接口）或autodetect（根据系统提供的cgroup版本自动选择）。
#默认为autodetect。

ConstrainCores=yes 
# 如设为yes，则容器允许将CPU核作为可分配资源子集，该项功能使用cpuset子系统。由于HWLOC 1.11.5版本中修复的错误，
# 除了task/cgroup外，可能还需要task/affinity插件才能正常运行。默认为no。
ConstrainDevices=yes 
# 如设为yes，则容器允许将基于GRES的设备作为可分配资源，这使用设备子系统。默认为no。
ConstrainRAMSpace=yes
# 如设为yes，则通过将内存软限制设置为分配的内存，并将硬限制设置为分配的内存AllowedRAMSpace来限制作业的内存使用。
# 默认值为no，在这种情况下，如ConstrainSwapSpace设为“yes”，则作业的内存限制将设置为其交换空间(SWAP)限制。
#注意：在使用ConstrainRAMSpace时，如果一个作业步中所有进程使用的总内存大于限制，那么内核将触发内存不足(Out Of Memory，OOM)事件，
# 将杀死作业步中的一个或多个进程。作业步状态将被标记为OOM，但作业步本身将继续运行，作业步中的其它进程也可能继续运行。
# MaxRAMPercent=98 #运行作业使用的最大内存百分比，将应用于slurm未显示分配内存的作业约束，（如slurm的选择插件为配置为管理内存分配），默认值100
# AllowedRANSpace=96 #运行作业/作业步使用最大Cgroup内存百分比，例如101.5，在分配的内存大小下设置cgroup软内存限制，然后再分配的内存（AllowedRAMSpace/100）处设置作业/作业步硬内限制，如果作业/作业步超出硬限制，则可能触发内存不足OOM事件（包括内存关闭），这些时间记录到内核日志环缓冲区（linux的dmesg）,设置AllowedRAMSpace超过100可能会导致内存不足00M事件产生，因为它运行作业/作业步分配配置给节点更多的内存，建议减少已配置的节点可以用内存，以避免系统内存不足OOM事件产生，将AllowedRANSpace设置为低于100将导致作业接收得内存少于分配的内存，软内存限制将设置为与硬内存限制相同的值，默认为100

#centos7和rhel差排查

ConstrainSwapSpace=yes
AllowedSwapSpace=0
#EnableControllers=yes

/etc/slurm/gres.conf
#AutoDetect=nvml
NodeName=xjhpc-164-1-ai Type=5090 Name=gpu File=/dev/nvidia[0-7]
NodeName=xjhpc-164-2-ai Type=5090 Name=gpu File=/dev/nvidia[0-7]
NodeName=xjhpc-164-100-ai Type=h100 Name=gpu File=/dev/nvidia[0-7]
NodeName=xjhpc-164-101-ai Type=h100 Name=gpu File=/dev/nvidia[0-7]
/etc/slurm/job_container.conf
AutoBasePath=true
BasePath=/home/slurm_tmp
/etc/slurm/mpi.conf
#PMIxCliTmpDirBase=<path>
#要让 PMIx 用于临时文件的目录。 默认为未设置。

#PMIxCollFence={mixed|tree|ring}
#定义用于收集节点间数据的围栏类型。 默认为未设置。另请参阅PMIxFenceBarrier。

PMIxDebug=1
#为 PMIx 插件启用调试日志记录。 默认值为 0。

#PMIxDirectConn={true|false}
#禁用直接启动任务。默认值为“true”。

#PMIxDirectConnEarly=true
#允许与父节点的早期连接。 默认为“false”。

PMIxDirectConnUCX=true
#允许 PMIx 使用 UCX 进行通信。 默认为“false”。

#PMIxDirectSameArch=true
#在PMIxDirectConn出现时启用其他通信优化 设置为 true，假设作业的所有节点都具有相同的体系结构。 默认为“false”。

PMIxEnv=/data/hpc/software/pmix/2.2.5;/data/hpc/software/pmix/3.2.4;/data/hpc/software/pmix/4.2.4
#要在作业环境中设置的环境变量的分号分隔列表 供 PMIx 使用。默认为未设置。

#PMIxFenceBarrier={true|false}
#定义是否屏蔽节点间通信以进行数据收集。 默认值为“false”。另请参阅PMIxCollFence。

#PMIxNetDevicesUCX=ens32
#用于通信的网络设备的类型。 默认为未设置。

#PMIxTimeout=<time>
#允许主机之间通信的最长时间（以秒为单位） 发生。默认为 10 秒。

#PMIxTlsUCX=<tl1>[,<tl2>...] 设置限制要使用的传输的 UCX_TLS 变量。被接受的 值在 UCX 文档中定义，可能因安装而异。 可以设置多个值，并且必须用逗号分隔。 如果未设置，UCX 将尝试使用所有可用的传输并选择最佳传输 根据其性能能力和规模。 默认为未设置。
/etc/slurm/nodes.conf
################################################
#                    NODES                     #
#NodeName Slurm用来指定节点的名称。通常这是“/bin/hostname -s”返回的字符串。或通过/etc/hosts或DNS与主机关联的任何有效域名。多个节点名可以用 逗号分隔(例如:"alpha,beta,gamma")，或使用一个简单的节点范围(例如“linux[000-100]”)。
#Boards 节点中的主板数量。当指定Boards时，应指定SocketsPerBoard、CoresPerSocket和ThreadsPerCore。默认值为1。
#CoresPerSocket 单个物理处理器Socket中的核心数(例如:“2”)。CoresPerSocket描述的是物理核，而不是每个Socket的逻辑处理器。
#CPUs 节点上逻辑处理器的数量(例如:“2”)。当希望只调度超线程节点上的核心时，这很有用。如果省略了CPUs，则其默认值将被设置为Boards、Sockets、CoresPerSocket和ThreadsPerCore的乘积。
#Features 与节点关联的某些特征。所需的特性可能包含一个数字组件，例如，表示处理器速度，缺省情况下，节点没有特性。
#Gres 通用资源规范的逗号分隔列表。格式为：“<name>[:<type>][:no_consume]:<number>[K|M|G]，默认情况下，节点没有通用资源。(例如“Gres=gpu:tesla:1,bandwidth:lustre:no_consume:4G”）。
#RealMemory 节点实际内存的大小，以megabytes为单位。(例如“2048”)。默认值为1。如果在SelectTypeParameters中将Memory设置为可消耗的资源。
#Reason 标识节点处于“DOWN”、“DRAINED”、“DRAINING”、“FAIL”或“FAILING”状态的原因。
#Sockets 节点上的物理处理器sockets/chips的数量(例如:“2”)。如果Sockets被省略，将从CPU、CoresPerSocket和ThreadsPerCore中进行推断。
#SocketsPerBoard 主板上的物理处理器sockets/chips的数量。Sockets和SocketsPerBoard是互斥的。默认值为1。
#State 节点状态。可接受的值为CLOUD、DOWN、DRAIN、FAIL、FAILING、FUTURE和UNKNOWN，默认值为UNKNOWN。
#ThreadsPerCore 单个物理核中的逻辑线程数(例如:“2”)。如果系统为每个核配置了多个线程，默认值为1。
#TmpDisk TmpFS中临时磁盘存储的总大小，以megabytes为单位。(例如“16384”)。TmpFS(表示“临时文件系统”)标识作业应该用于临时存储的位置。默认值为0 。
#procs是实际CPU个数
#State=UNKNOW #状态，是否启用，State可以为以下之一
        #CLOUD 在云上存在
        #DOWN 节点失效，不能分配给在作业
        #DRAIN 作业不能分配给作业
        #FAIL 节点即将失效，不能接受分配新作业
        #FAILING 节点即将失效，但上面又作业未完成，不能接受新的作业
        #FUTURE 节点为了将来使用，当slurm守护进程启动时间设置不存在，可以之后采用scontrol命令简单的改变其状态，而不需要重启slurmctld守护进 程，这些节点有效后，修改slurm.conf中的他们的state值。在他们被设置有效前。采用slurm看不到他们。也尝试与其联系。
#动态未来节点
        #slurm启动时如有-F参数，将管理到一个与slurmd -C命令显示配置（sockets、cores、threads）相同的配置的FUTURE节点，节点的NodeAddr和NodeHostname从slurmd守护进程自动获取，并且当被设置为FUTURE状态后自动清除，动态未来节点在重启时保持non-FUTURE状态，利用scontrol可以将其设置为FUTURE状态
        #若NodeName与slurmd的HostName映射未通过DNS更新，动态未来节点不知道在之间如果通讯，其原因在于NodeAddr和NodeHostName未在slurm.conf被 定义，而且扇出通讯（fanout communication）需要讲TreeWidth设置为一个较高的数字（如65533）来使其无效，做了DNS映射，这可以使用cloud_dns slurmctldParameter
        #UNKNOWN 节点状态未被定义，但将在节点上启动slurmd进程后设置为BUSY或者IDLE，该值为默认值
################################################
NodeName=xjhpc-162-16-ai CPUs=16 Boards=1 SocketsPerBoard=1 CoresPerSocket=16 ThreadsPerCore=1 RealMemory=32059 NodeAddr=10.112.162.16 State=UNKNOWN
NodeName=xjhpc-master CPUs=16 Boards=1 SocketsPerBoard=1 CoresPerSocket=16 ThreadsPerCore=1 RealMemory=32059 State=UNKNOWN
NodeName=xjhpc-backup CPUs=16 Boards=1 SocketsPerBoard=1 CoresPerSocket=16 ThreadsPerCore=1 RealMemory=32059 State=UNKNOWN
#NodeName=xjhpc-37-5-ai CPUs=128 Boards=1 SocketsPerBoard=2 CoresPerSocket=32 ThreadsPerCore=2 RealMemory=515615 NodeAddr=10.112.37.5 Gres=gpu:5090:8 State=UNKNOWN
#NodeName=xjhpc-8-22-ai CPUs=128 Boards=1 SocketsPerBoard=2 CoresPerSocket=32 ThreadsPerCore=2 RealMemory=515609 NodeAddr=10.112.8.22 Gres=gpu:5090:8 State=UNKNOWN
#NodeName=xjhpc-164-1-ai CPUs=128 Boards=1 SocketsPerBoard=2 CoresPerSocket=32 ThreadsPerCore=2 RealMemory=515615 NodeAddr=10.112.164.1 Gres=gpu:5090:8 State=UNKNOWN
#NodeName=xjhpc-164-2-ai CPUs=128 Boards=1 SocketsPerBoard=2 CoresPerSocket=32 ThreadsPerCore=2 RealMemory=515615 NodeAddr=10.112.164.2 Gres=gpu:5090:8 State=UNKNOWN
NodeName=xjhpc-164-100-ai CPUs=192 Boards=1 SocketsPerBoard=2 CoresPerSocket=48 ThreadsPerCore=2 RealMemory=2063853 NodeAddr=10.112.164.100 Gres=gpu:h100:8 State=UNKNOWN
NodeName=xjhpc-164-101-ai CPUs=192 Boards=1 SocketsPerBoard=2 CoresPerSocket=48 ThreadsPerCore=2 RealMemory=2063853 NodeAddr=10.112.164.101 Gres=gpu:h100:8 State=UNKNOWN
/etc/slurm/partitions.conf
################################################
#                  PARTITIONS                  #
#PartitionName     分区名
#Nodes             节点名
#Default=YES       如果没有给作业指明分区，则会分配到默认分区，值时YES或NO。
#MaxTime=60        作业运行的最大时间(分钟)限制，INFINITE为没有限制。
#DefMemPerNode     每个节点默认分配的内存大小，单位MB，默认值为0（无限制），DefMemPerCPU、DefMemPerGPU和DefMemPerNode是互斥的。
#DefMemPerCPU      每个CPU默认分配的内存大小，单位MB。
#MaxMemPerCPU      每个节点最大内存大小，单位MB。
#DefMemPerGPU      每个GPU默认分配的内存大小，单位MB。
#OverSubscribe     控制分区在每个资源上一次执行多个作业的能力。
#  EXCLUSIVE  独占节点；
#  FORCE[:X]  使分区中的所有资源(除了GRES)可用于超额订阅；
#  YES        分区中的所有资源(除了GRES)可用于共享;
#  NO         资源分配给单个作业；
#PreemptMode       用于抢占作业或启用此分区的gang调度的机制。
#  OFF        默认值，禁用job抢占和gang scheduling；
#  CANCEL     抢占的作业将被取消；
#  GANG       启用同一分区中作业的gang调度，并允许恢复暂停的作业；
#  REQUEUE    通过对作业重新排队或取消它们来抢占作业；
#  SUSPEND    被抢占的作业将被暂停，稍后Gang调度程序将恢复它们；
#PriorityJobFactor  priority/multifactor插件在计算作业优先级时使用的分区因子。
#prioritytier       队列调度优先级。
#State        分区可用性的状态。取值为UP、DOWN、DRAIN和INACTIVE。系统默认值为UP。
#AllocNodes   可以在分区中提交作业的节点列表(逗号分隔)，默认值为“ALL”。
#AllowAccounts 可以在分区中执行作业的帐户列表(逗号分隔)，默认值为“ALL”。
################################################


#PartitionName=5090-1 Nodes=hpc-37-[6-8]-ai State=UP AllowAccounts=ALL MaxTime=15-00:00:00 OverTimeLimit=60

PartitionName=login-cpu Nodes=xjhpc-162-16-ai Default=YES State=UP AllowAccounts=ALL
PartitionName=xj-h100-ib-1 Nodes=xjhpc-164-[100,101]-ai Default=YES State=UP AllowAccounts=ALL
/etc/slurm/plugstack.conf
################################################################################
# Stunnel plugin configuration file
#
# this plugin can be used to add ssh tunnel support in slurm jobs using ssh port
# forwarding capabilities
#
# The following configuration parameters are available (the character |
# replaces the space in compound options) :
#
# ssh_cmd: can be used to modify the ssh binary to use.
#                 default corresponds to ssh_cmd=ssh
# ssh_args: can be used to modify the ssh arguments to use.
#                 default corresponds to ssh_args=
# helpertask_args: can be used to add a trailing argument to the helper task
#               responsible for setting up the ssh tunnel
#               default corresponds to helpertask_args=
#                 an interesting value can be helpertask_args=2>/tmp/log to
#                 capture the stderr of the helper task
#
# Users can ask for tunnel support for both interactive (srun) and batch (sbatch)
# jobs using parameter --tunnel=<submit port:exec port[,submit port:host port]>
# where submit port is the port number on the submit host and the exec port is
# the port number on the exec host.  A comma separated list can be used to
# forward multiple ports.
#
#
#-------------------------------------------------------------------------------
#optional          stunnel.so
#-------------------------------------------------------------------------------
#
#required /usr/local/lib/slurm/spank_pyxis.so execute_entrypoint=0 runtime_path=/home/pyxis #container_scope=job #sbatch_support=1
#include /etc/slurm/plugstack.conf.d/*.conf
/etc/slurm/slurm.conf
# slurm.conf file. Please run configurator.html
# (in doc/html) to build a configuration file customized
# for your environment.
#
#
# slurm.conf file generated by configurator.html.
# Put this file on all nodes of your cluster.
# See the slurm.conf man page for more information.
#
################################################
#                   CONTROL                    #
################################################
ClusterName=xjhpc    #集群名称
SlurmctldHost=xjhpc-master    #管理服务节点名称,启动slurmctld进程的节点名。
SlurmctldPort=6817    #slurmctld服务端口，如不设置默认为6817端口
SlurmdPort=6818   #slurmd服务的端口，如不设置默认为6818端口
SlurmUser=slurm    #slurm的主用户，slurmctld启动是采用的用户名
SlurmctldParameters=enable_configless #启用无配置模式
SlurmctldHost=xjhpc-backup  # 冗余备份节点，可空着


MaxJobCount=10000 #默认值为10000,当前时间最大任务数
#MaxStepCount=40000
##MaxTasksPerNode=128
#默认MPI类型
MpiDefault=none
        #MPI-pmi2: 对支持PMI2的MPI实现
        #MPI-pmix: Exascale PMI实现
        #none: 对于大多数其他MPI,建议设置


################################################
#            LOGGING & OTHER PATHS             #
################################################
#slurmctld和slurmd守护进程可配置为采用不同级别的详细日志记录，debug从0-7详细记录。
SlurmctldDebug=debug #默认为info
SlurmctldLogFile=/var/log/slurm/slurmctld.log #如果是空白，这记录到syslog
SlurmdDebug=debug #默认为info
SlurmdLogFile=/var/log/slurm/slurmd.log #如果空白这记录在syslog,如名字中有字符串%h，则将%h将被替换为节点名。

#进程ID记录
SlurmctldPidFile=/var/run/slurmctld.pid
SlurmdPidFile=/var/run/slurmd.pid

#state preservation 状态保持
SlurmdSpoolDir=/var/spool/slurmd  #slurmd服务所偶需要的目录，为各节点私有目录，不得多个slurmd节点共享
StateSaveLocation=/var/spool/slurmctld #/var/spool/slurmctld  #存储slurmctld服务状态的目录，比如有备份控制节点，这需要所有的slurmctldHost节点都能共享读写该目录


################################################
#                  ACCOUNTING                  #
################################################
AccountingStorageEnforce=associations,limits,qos  #account存储数据的配置选项
AccountingStorageHost=xjhpc-master #mysql-master    #记账数据库存储节点
AccountingStoragePass=/var/run/munge/munge.socket.2    #munge认证文件，与slurmdbd.conf文件中的AuthInfo文件同名。
AccountingStoragePort=6819    #slurmdbd数据库服务监听端口，默认为6819

AccountingStorageType=accounting_storage/slurmdbd    #数据库记账服务 
AccountingStorageTRES=gres/gpu,gres/gpu:a800,gres/npu,gres/npu:910b3,cpu,mem,energy,node,billing,fs/disk,vmem,pages #记账信息
GresTypes=gpu,mps,bandwidth,dcu,npu # 设置GPU时需要
AccountingStoreFlags=job_comment,job_script,job_env  #以逗号(,)分割列表，选项是:
        #job_comment: 在数据库中存储作业说明域
        #job_script: 在数据库中存储脚本
        #job_env: 存储批处理作业的环境变量

AcctGatherEnergyType=acct_gather_energy/none   #作业消耗能源信息，none代表不采集
AcctGatherFilesystemType=acct_gather_filesystem/none
AcctGatherInterconnectType=acct_gather_interconnect/none
AcctGatherNodeFreq=0
AcctGatherProfileType=acct_gather_profile/none

#20231210 需要查询原因
#ExtSensorsType=ext_sensors/none
#ExtSensorsFreq=0

################################################
#               JOBS 作业记录                  #
################################################
JobCompHost=mysql-xjvip      #作业完成信息的数据库本节点
#JobCompLoc=http://10.99.80.148:9200/slurm/_doc     
JobCompLoc=slurm_jobcomp_db_xj    #数据库名称,
        #设定记录作业完成信息的文本位置（若JobCompType=filetxt）
        #,或将要运行的脚本（若JobCompType=script）,
        #或Elastucsearch服务器的URL（若JobComtype=elasticsearch）,
        #或数据库名字(JobCompType=jobcomp/mysql)
JobCompUser=slurm_slurm_rw    #作业完成信息数据库用户名
JobCompPass=9sqbch_fcfbhgzkuicjxpomtinogfsdQ    #slurm用户数据库密码
JobCompPort=3306    #数据库端口

#JobCompType=jobcomp/elasticsearch
JobCompType=jobcomp/mysql  #作业完成信息数据存储类型，采用mysql数据库
        #指定作业完成时采用的记录机制，默认为None，可以设置如下之一
        #Node: 不记录作业完成信息
        #Elasticaearch #将作业信息记录到Elasticsearch服务器
        #FileTxt: 将作业完成信息记录在一个纯文本中
        #Lua: 利用名为jobcomp.lua的文件记录作业完成信息
        #Script: 采用任意脚本对原始作业完成信息进行处理后记录
        #MySQL: 将完成的状态写入Mysql或者Mariadb数据库

#作业记账
JobCompParams=timeout=5,connect_timeout=5

JobAcctGatherFrequency=30 #设定轮寻间隔，以秒为单位，若为“-”,则禁止周期抽样
JobAcctGatherType=jobacct_gather/cgroup #slurm记录每个作业消耗的资源，JobAcctGatherType值可以为一下之一：
        #jobacct_gather/none: 不对作业记账
        #jobacct_gather/cgroup: 收集linux cgroup信息
        #jobacct_gather/linux: 收集linux进程表信息，推荐建议

################################################
#           SCHEDULING & ALLOCATION            #
################################################
#DefMemPerCPU=10240 #默认每颗CPU可以用内存，以MB为单位，0为不限制，如果将单个处理器分配给作业（SelectType=select/cons_res或SelectType=select/cons_tres）,通常会使用DefMemPerCPU
#MAXMenPerCPU=0 #最大每颗CPU可以用内存，以MB为单位，0为不限制，如果将单个处理器分配给作业（SelectType=select/cons_res或SelectType=select/cons_tres）,通常会使用DefMemPerCPU

#调度
SchedulerType=sched/backfill #要使用的调度程序类型，注意，slurmctld守护程序必须重新启动才能使调度类型更改生效（重新配置正在运行的守护程序对此参数无效），如果需要,可以使用scontrol命令手动更改作业优先级，可接受的类型为：
        #sched/backfill # 用于回填调度模块以增加默认FIFO调度，如这样做不会延迟任何较高优先级作业的预期启动时间，则回填调度将启动较低优先级 作业，回填调度的有效性取决于用户指定的作业时间限制，否则所有的作业将具有相同的时间限制，并且回填是不可能的，注意上面SchedulerParameters选项的文档，这是默认配置。
        #sched/builtin # 按优先级顺序启动作业的FIFO调度，如队列中任何作业无法调度，则不会调度该队列中优先级较低的作业，对于作业的一个例外队列限制（如时间限制）或关闭/耗尽节点而无法运行。在这种情况下可以启动较低优先级的作业，而不会影响较高优先级作业。
        #sched/hold # 如果/etc/slurm.hold文件存在，则暂停所有新提交的作业，否则使用内置的FIFO调度程序。

PriorityType=priority/multifactor
#当使用sched/builtin：按顺序依次调度，这种调度类型按照PriorityType的设置分为两种，
        #当设置priority/basic时是按照作业提交时间的顺序也就是FIFO调度作业，
        #当设置priority/multifactor时它按优先级顺序调度作业，而作业最终优先级会考虑多种因素
##PriorityFlags=DEPTH_OBLIVIOU
#
#
#SchedulerParameters=batch_sched_delay=5,defer,sched_min_interval=20,sched_interval=30,default_queue_depth=100,bf_max_job_test=100,bf_interval=30
#sched_interval 控制两次全量调度之间的间隔时间，默认值是60秒。设置为-1将禁用主调度循环。
#max_sched_time 主调度循环在退出之前最多执行多长时间(以秒为单位)。如果配置了一个值，请注意所有其他Slurm操作将在此时间段内被推迟。确保该值低于MessageTimeout的一半才会有效。主调度时间不宜过长，2-3秒合适
#default_queue_depth    调度作业队列的深度，当一个正在运行的作业完成或发生其他例行操作时，尝试调度的默认作业数(即队列深度)，此参数在slurm19版本和slurm20版本的使用有所区别，默认值为100个任务。由于这种情况经常发生，所以相对较小的数目通常是最好的。
#partition_job_depth    在Slurm的主调度逻辑中，从每个分区/队列尝试调度的默认作业数(即队列深度)，同上如果太小导致后面其他用户作业调度不上， 设置太大导致只调度一个分区其他分区饿死，此参数的处理代码slurm19版本和slurm20版本也有所区别，19版本当分区调度作业数量到达限制之后不会改变作 业状态，但是20版本会改变作业状态
#sched_min_interval     主调度循环执行和测试排队作业的频率，单位为微秒。调度程序在每次可能启动作业的事件(例如作业提交、作业终止等)发生时都 以有限的方式运行。如果这些事件以很高的频率发生，调度器可以非常频繁地运行，如果不使用此选项，则会消耗大量资源。此选项指定从一个调度周期结束 到下一个调度周期开始之间的最短时间。值为0将禁用调度逻辑间隔的节流。缺省值是2微秒。
#assoc_limit_stop       如果设置，作业由于关联限制而无法启动，那么将作业所在分区加入分区黑名单，不要尝试在该分区中启动任何低优先级的作业。
#defer  如果设置则不要尝试在作业提交或有作业运行完成时单独调度作业。对于高吞吐量计算非常有用。
#bf_min_age_reserve     回填调度预留时间参数，当高优先级作业因空闲资源未成功运行，且排队的时间不满足bf_min_age_reserve回填预留的条件，则主 调度不会将该作业所在分区加入分区黑名单
#bf_min_prio_reserve    回填调度预留优先级参数，当高优先级作业因空闲资源未成功运行，同时作业优先级不满足bf_min_prio_reserve回填预留的条件，则主调度不会将该作业所在分区加入分区黑名单
#batch_sched_delay      批处理作业的调度可以延迟多长时间，单位为秒。这在高吞吐量的环境中非常有用，在这种环境中，批处理作业以非常高的速率提 交(即使用sbatch命令)，并且希望减少在提交时调度每个作业的开销。缺省值为3秒。
#sched_max_job_start    主调度逻辑在任何一次执行中启动的作业的最大数量。默认值为0，没有限制。

#SchedulerTimeSlice=300 #当Gang调度启用时的时间长度，以秒为单位

TaskPlugin=task/affinity,task/cgroup #设定任务启动插件，可被用于提供节点内的资源管理（如绑定任务到特定的处理器），TaskPlugin值可设置为:
        #task/affinity: CPU亲和支持（man srun查看其中--cpu-bind、--mem-bind和-E选项）
        #task/cgroup: 强制采用Linux控制组Cgroup分配资源（man slurm.conf查看帮助）
        #task/none: 无任务启动动作

#资源选择，定义作业资源（节点）选择的算法
SelectType=select/cons_tres
        #select/cons_tree: 单个的CPU核、内存、GPU以及其他可以追踪资源作为可消费资源（消费及分配），建议设置
        #select/cons_res: 单个CPU核和内存作为可消费资源
        #select/cray_aries: 对于Cray系统
        #select/linear: 基于主机的作为可消费资源，不管理单个CPU等分配

#资源选择类型参数，当SelectType=select/linear时仅支持CR_ONE_TASK_PER_CORE和CR_Memory;当SelectType=select/cons_res、SeletcType=select/cray_aries和SelectType=select/cons_tree时，默认采用CR_Core_Memory
SelectTypeParameters=CR_Core
        #CR_CPU: CPU核数作为可消费资源
        #CR_Socket: 整颗CPU作为可消费资源
        #CR_Core: CPU核作为可消费资源，默认
        #CR_Memory: 内存作为可消费资源，CR_Memory假定MaxShare大于等于1
        #CR_CPU_Memory: CPU和内存作为可消费资源
        #CR_Socket_Memory: 整颗CPU和内存作为消费资源
        #CR_Core_Memory: CPU核和内存作为可消费资源

################################################
##                   TOPOLOGY                   #
#################################################
TopologyPlugin=topology/none

################################################
#                    TIMERS                    #
################################################
BatchStartTimeout=100
CompleteWait=0
EpilogMsgTime=2000
InactiveLimit=0 #潜伏期控制器等待srun命令响应多少秒后，将在考虑过滤作业或作业步骤不活动并终止它之前，0表示无限长等待。
KillWait=30
MinJobAge=300 #在作业到达其时间限制前等待多少秒后在发送SIGKILL信号之前发送TERM信号以优雅终止
SlurmctldTimeout=60 #设定备份控制器在控制器等待多少秒后成为激活的控制器
SlurmdTimeout=300 #slurm控制器等待slurmd未响应请求多少秒后将该节点设置为DOWN
WaitTime=0 #在一个作业步的第一个任务结束后等待多少秒后结束所有其他任务。0表示无限长等待
MessageTimeout=30
TCPTimeout=10


################################################
##                    POWER                     #
#################################################
ResumeRate=300
ResumeTimeout=120
SuspendRate=60
SuspendTime=NONE
SuspendTimeout=60
#
################################################
#                    DEBUG                     #
################################################
DebugFlags=NO_CONF_HASH

################################################
#               PROCESS TRACKING               #
################################################
#进程追踪，定义用于确定特定的作业所应对的进程算法，他使用信号、杀死和记账与作业相关的进程
ProctrackType=proctrack/cgroup
        # proctrack/cgroup: 使用Linux中的cgroup 来约束和跟踪进程,需要设定/etc/slurm/cgroup.conf文件
        # proctrack/linuxproc: 采用父进程的IP记录，进程可脱离slurm控制
        # proctrack/pgid: 采用Unix进程组ID，进程如改变了其进程组ID则可以脱离slurm控制
        # Cray XC: 采用Cray XC专有进程追踪

#https://weikezhijia.feishu.cn/docx/Mf16dYiZdo6UgLxubf5cm3NinFb
#插件读取job_container.conf文件以查找配置设置。基于它们，它为作业构建了一个专用（或可选的共享）文件系统命名空间，并在其中装载一个目录列表（默认为/tmp和/dev/shm）

JobContainerType=job_container/tmpfs
PrologFlags=Contain

################################################
#             RESOURCE CONFINEMENT             #
################################################
#任务启动

#TaskPluginParam=threads

#Prolog and Epilog: 前处理及后处理
#Prolog/Epilog: 完整地绝对路径，在用户作业开始前（Prolog）或结束后（Epilog)在其每个运行节点上都采用root用户执行，可用于初始化某些参数，清理作业运行后删除文件等
#Prolog=/data/hpc/scripts/prolog.sh #作业运行前执行的文件，采用root账户执行
#Epilog=/data/hpc/scripts/epilog.sh #作业运行后执行的文件，采用root账户执行

#SrunProlog/Epilog: 完整地绝对路径，在用户作业开始前（Prolog）或结束后（Epilog)在其每个运行节点上都采用srun运行的用户执行，这些参数可以被srun的--prolog和--epilog选项覆盖
#SrunProlog=/data/hpc/scripts/srunprolog.sh #在srun作业开始运行前需要执行的文件，采用运行srun命令的用户执行。
#SrunEpilog=/data/hpc/scripts/srunepilog.sh #在srun作业运行结束后需要执行的文件，采用运行srun命令的用户执行

#TaskProlog/Epilog: 完整地绝对路径，在用户作业开始前（Prolog）或结束后（Epilog)在其每个运行节点上都采用运行作业的用户身份执行
#TaskProlog=/data/hpc/scripts/taskprolog.sh #在srun作业开始运行前需要执行的文件，采用运行业的用户身份执行
#TaskEpilog=/data/hpc/scripts/taskepilog.sh #在srun作业运行结束后需要执行的文件，采用运行业的用户身份执行
#
#顺序：
        #1、pre_launch_priv(): TaskPlugin内部函数
        #2、pre_launch(): TaskPlugin内部函数
        #3、TaskProlog: slurm.conf中定义的系统范围每个任务
        #4、User prolog: 作业步指定的，采用srun命令的--task-prolog参数或SLURM_TASK_PROLOG环境变量指定
        #5、Task:作业步任务中执行
        #6、User epilog: 作业步指定的，采用srun命令的--task-epilog参数或SLURM_TASK_EPILOG环境变量指定
        #7、TaskEpilog: slurm.conf中定义的系统范围每个任务
        #8、post_term(): TaskPlugin内部函数
##############################################
#                    PRIORITY                  #
################################################
#PrioritySiteFactorPlugin=
#设定的衰减速率
PriorityDecayHalfLife=7-00:00:00
PriorityCalcPeriod=00:05:00
PriorityFavorSmall=No
#PriorityMaxAge=7-00:00:0
PriorityUsageResetPeriod=NONE
PriorityWeightAge=0
PriorityWeightAssoc=0
PriorityWeightFairShare=0
PriorityWeightJobSize=0
PriorityWeightPartition=0
PriorityWeightQOS=1000

#--------token------
AuthAltTypes=auth/jwt
AuthAltParameters=jwt_key=/var/spool/slurmd/statesave/jwt_hs256.key

################################################
#                    OTHER                     #
################################################

AllowSpecResourcesUsage=No
#CoreSpecPlugin=core_spec/none
CpuFreqGovernors=Performance,OnDemand,UserSpace
CredType=cred/munge
EioTimeout=120
EnforcePartLimits=NO
#FirstJobId=2
FirstJobId=30000001
MaxJobId=60000000
JobFileAppend=0
JobRequeue=1
#MailProg=/bin/mail
#MailProg=/usr/bin/mutt
#MailProg=/usr/bin/slurm-spool-mail
MaxArraySize=1001
MaxDBDMsgs=24248
#MaxJobId=67043328
MaxMemPerNode=UNLIMITED
MaxStepCount=40000
MaxTasksPerNode=512
MCSPlugin=mcs/none
ReturnToService=2 #设定为DOWN（失去响应）状态节点如何恢复服务，默认为0
        #0: 节点状态保持DOWN状态，只有当管理员明确使其恢复服务时才恢复。
        #1: 仅当由于无响应而将DOWN节点设置为DOWN状态时，才可以当有效配置注册后使用DOWN节点恢复服务，如节点由于任何其他原因（如内存不足，意 外重启等）被设置为DOWN，其状态将不会自动更改，当节点内存GRES、GPU计算等于或大于slurm.conf中的配置时，改节点注册为有效配置
        #2: 使用有效配置注册后，DOWN节点将可供使用，该节点可能因任何原因被设置为DOWN状态，当节点内存,GREP，CPU计数等于或大于slurm.conf中的 配置值，改节点才注册为有效配置
RoutePlugin=route/default
TmpFS=/home/slurm_tmp
TrackWCKey=no
TreeWidth=50
UsePAM=0
SwitchType=switch/none
UnkillableStepTimeout=150
VSizeFactor=0

################################################
#              QOS                             #
################################################
PreemptMode=OFF

#PreemptType 设定为 preempt/qos 时，一个排队中的作业的QOS将被用于决定其是否可以抢占一个运行中的作业,默认为 preempt/none
PreemptType=preempt/none
##PreemptExemptTime 参数设定了在作业将抢占之前最小运行时间
PreemptExemptTime=00:00:00
#
SlurmSchedLogLevel=0
#

#Fairshare =  
##用于决定公平共享优先级的整数。本质上，这是对上述系统针对该关联和其子项的请求总数。也可以使用字符串”parent”，当被用户使用时，意味着针对公 平共享采用其父关联。如在该账户设置 Fairshare=parent ，该账户的子成员将被有效地利用它们的第一个不是 Fairshare=parent 的父母重新支付公平共享 计算。限制保持不变，仅影响其公平共享值￼
#MaxJobs =  
##在给定的任意时间，针对该关联能同时运行的作业总数。如该达到限制，新作业将处于排队状态，只有当该关联的作业有结束时才能运行。
#MaxJobsAccrue =  
##在给定的任意时间，能从关联允许的排队中作业累计年龄优先级的最大数。当该达到限制，新作业将处于排队中状态，但不累计年龄优先级，直到有关联的 作业从排队状态有退出排队。该限制不决定作业是否能运行，它只限制优先级的年龄因子。
#MaxSubmitJobs =
##在给定的任意时间，能从该关联提交到系统中的作业最大数。如达到该限制，新提交申请将被拒绝，直到存在该关联的作业退出。
#QOS =
##以逗号（,）分隔的能运行的QOS列表。

#支持的QOS特定限制
#MaxJobsAccruePerAccount=
#在任意时间，一个账户（或子账户）能从关联允许的排队中作业累计年龄优先级的最大数。该限制不决定作业是否能运行，它只限制优先级的年龄因子。
#MaxJobsAccruePerUser=
#在任意时间，一个用户能从关联允许的排队中作业累计年龄优先级的最大数。该限制不决定作业是否能运行，它只限制优先级的年龄因子。
#MaxJobsPerAccount=
#一个账户（或子账户）能允许的最大同时运行作业数。
#MaxJobsPerUser=1 
#每个用户能同时运行的最大作业数。
#MaxSubmitJobsPerAccount= 
#每个账户（或子账户）能同时运行和排队等待运行的最大作业数。
#MaxSubmitJobsPerUser= 
#每个用户能同时运行和排队等待运行的最大作业数。
#MaxTRESPerAccount= 
#每个账户能同时分的配最大TRES数。
#MaxTRESPerUse= 
#每个用户能同时分配的最大TRES数。
#MinTRESPerJob= 
#每个作业能申请的最小TRES尺寸。
#Epilog=/etc/slurm/epilog.d/*
#Prolog=/etc/slurm/prolog.d/*
Epilog=epilog.py
Prolog=prolog.py


include nodes.conf
include partitions.conf 
/etc/slurm/slurmdbd.conf
#
# slurmdbd.conf file.
#
# See the slurmdbd.conf man page for more information.
#
# Authentication info
AuthType=auth/munge     #认证方式，该处采用munge进行认证
AuthInfo=/var/run/munge/munge.socket.2     #为了与slurmctld控制节点通信的其它认证信息
#
#AuthAltTypes=auth/jwt
#AuthAltParameters=jwt_key=/var/spool/slurm/statesave/jwt_hs256.key

# slurmDBD info
DbdAddr=10.112.162.14      #数据库服务端IP-非数据库地址
DbdHost=xjhpc-master    #数据服务hostname
DbdBackupHost=xjhpc-backup #数据库冗余节点
#DbdPort=7031   #数据服务默认端口7031
SlurmUser=slurm     #用户数据库操作的用户
#MessageTimeout=60 #允许以秒为单位在完成往返通讯的时间，默认为10秒
DebugLevel=info     #调试信息级别，quiet：无调试信息；fatal：仅严重错误信息；error：仅错误信息； info：错误与通常信息；verbose：错误和详细 信息；debug：错误、详细和调试信息；debug2：错误、详细和更多调试信息；debug3：错误、详细和甚至更多调试信息；debug4：错误、详细和甚至更多调试信息；debug5：错误、详细和甚至更多调试信息。debug数字越大，信息越详细
#DefaultQOS=normal #默认QOS
LogFile=/var/log/slurm/slurmdbd.log     #slurmdbd守护进程日志文件绝对路径 
PidFile=/var/run/slurmdbd/slurmdbd.pid     #slurmdbd守护进程存储进程号文件绝对路径
#
#PrivateData=accounts,user,usage,jobs #对普通用户隐藏的数据，默认所有信息对所有用户开放，slurmUser/root和AdminLevel=Admin用户可以查看所有信息，多少值可以采用逗号（，）分割
        #accounts 阻止用户查看账户信息，除非该用户是他们的协调人
        #events 阻止用户查看时间信息，除非该用户具有操作员或者更高级别身份
        #jobs 阻止普通用户查看其他用户作业信息，如非该用户是使用sacct时运行的作业账户的协调员
        #reservations 限制具有操作员权限及以上身份的用户获取预留资源信息
        #usage: 阻止用户查看其他用户利用率，适用于sreport命令
        #users: 阻止用户查看除了自己以外的任何用户信息，使得用户智能看到他们处理的关联，协调人可以看到他们作为协调人的账户中所有用户的关联 ，但只有在列出用户的时候才能看到自己
#TrackWCKey=yes #工作负载特征键，用于设置Workload Characterization key的显示和跟踪，必须设置为跟踪wckey的使用，这必须设置为从WCKeys生产汇总使用表。注意：如果在此设置TrackWCKey而不是在你的各种slurm.conf文件中，则所有的作业都将归于因于他们的默认WCKey
#
# Database info
StorageType=accounting_storage/mysql     #数据存储类型
StorageHost=mysql-xjvip
StoragePass=9sqbch_fcfbhgzkuicjxpomtinogfsdQ     #存储数据库密码
StoragePort=3306   #数据库端口
StorageUser=slurm_slurm_rw     #存储数据库用户名
StorageLoc=slurm_acct_db_xj     #数据库名称