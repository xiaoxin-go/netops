
# 网络自动化平台
对接jira工单系统，实现网络工单自动实施。 提供设备配置备份，白名单策略生成与推送，策略查询等功能。
- 开发语言: go 1.22
- 后端框架: gin
- 数据库: mysql8
- 前端: vue3+antd
- 权限: casbin

## 前端地址
https://github.com/xiaoxin-go/netops-web

## 演示地址
http://netops.xiaoxin-go.xyz guest/guest@123456

## 目录结构
```markdown
├─api
│  ├─admin
│  │  ├─device
│  │  ├─jira
│  │  └─platform
│  ├─auth
│  ├─device
│  │  ├─backup
│  │  ├─firewall
│  │  └─nlb
│  ├─policy
│  │  ├─firewall
│  │  ├─firewall_nat
│  │  └─nlb
│  ├─system
│  │  ├─log
│  │  ├─role
│  │  └─user
│  ├─task
│  ├─task_info
│  └─tools
│      ├─invalid_policy_task
│      └─public_whitelist
├─bin
├─conf
├─database
├─dist
│  ├─css
│  ├─img
│  └─js
├─docs
├─grpc_client
│  ├─net_api
│  └─protobuf
│      └─net_api
├─libs
├─logs
├─model
├─pkg
│  ├─auth
│  ├─device
│  ├─parse
│  ├─policy
│  ├─subnet
│  ├─task
│  └─tools
├─routers
└─utils

```

# 安装
### 下载
```shell
git clone git@github.com:xiaoxin-go/netops.git
```

### 打包成linux可执行文件
#### windows
```shell
./build-win.bat
```
#### mac
```shell
./build-mac.bat
```

#### 初始化sql
```shell
# 创建mysql数据库 netops_devops
./docs/init_table.sql
./docs/init_auth.sql
```

#### 运行
```shell
chmod +x ./bin/netops
./bin/netops
```