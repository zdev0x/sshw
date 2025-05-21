# SSHW - SSH 客户端包装器

SSHW 是一个强大的 SSH 客户端包装器，它提供了更便捷的 SSH 连接管理功能。通过简单的配置文件，你可以轻松管理多个 SSH 连接，支持加密存储敏感信息，并提供丰富的连接选项。

## 特别说明

本项目是基于 `https://github.com/yinheli/sshw` 二次开发，特别感谢原作者的努力。

## 功能特点

- 支持多级服务器配置
- 支持服务器分组和别名
- 支持密码和密钥认证
- 支持配置文件加密
- 支持配置文件格式：YAML 和 JSON
- 支持 SSH 跳板机
- 支持自定义连接参数
- 支持登录标记
- 支持回调命令
- 支持主机名掩码

## 系统要求

- Go 1.16 或更高版本
- 支持的操作系统：Linux、macOS、Windows

## 安装

```bash
go install github.com/zdev0x/sshw@latest
```

## 快速开始

1. 创建配置文件：

```yaml
# ~/.sshw.yml 或 ~/.sshw.json
- name: "开发服务器"
  host: "dev.example.com"
  user: "dev"
  port: 22
  password: "your-password"  # 可选，支持加密存储
  keypath: "~/.ssh/id_rsa"   # 可选
  passphrase: "key-passphrase" # 可选，支持加密存储
```

2. 启动程序：

```bash
sshw
```

3. 使用方向键选择服务器，回车连接

## 配置管理

### 配置文件

SSHW 支持 YAML 和 JSON 两种格式的配置文件，你可以选择任意一种格式：

1. YAML 格式（推荐）：
```yaml
- name: "服务器1"
  host: "server1.example.com"
  user: "user1"
  port: 22
  password: "password1"
  children:
    - name: "子服务器1"
      host: "sub1.example.com"
      user: "user2"
```

2. JSON 格式：
```json
[
  {
    "name": "服务器1",
    "host": "server1.example.com",
    "user": "user1",
    "port": 22,
    "password": "password1",
    "children": [
      {
        "name": "子服务器1",
        "host": "sub1.example.com",
        "user": "user2"
      }
    ]
  }
]
```

### 配置文件位置

SSHW 会按以下顺序查找配置文件：

1. 命令行指定的配置文件（使用 `-config` 参数）
2. `~/.sshw`
3. `~/.sshw.yml`
4. `~/.sshw.yaml`
5. `~/.sshw.json`
6. `./.sshw`
7. `./.sshw.yml`
8. `./.sshw.yaml`
9. `./.sshw.json`

### 配置项说明

| 配置项 | 说明 | 必填 | 默认值 |
|--------|------|------|--------|
| name | 服务器名称 | 是 | - |
| host | 服务器地址 | 是 | - |
| user | 用户名 | 否 | 当前系统用户 |
| port | 端口号 | 否 | 22 |
| password | 密码 | 否 | - |
| keypath | 密钥文件路径 | 否 | - |
| passphrase | 密钥密码 | 否 | - |
| is_encrypted | 是否已加密 | 否  | - |
| children | 子服务器列表 | 否 | - |
| jump | 跳板机配置 | 否 | - |
| mask_host | 是否掩码主机名 | 否 | false |
| show_host | 是否显示主机名 | 否 | false |
| enable_login_marker | 是否启用登录标记 | 否 | false |
| callback-shells | 回调命令列表 | 否 | - |

## 高级功能

### 配置文件加密

1. 加密配置文件：

```bash
sshw -encrypt
```

2. 解密配置文件：

```bash
sshw -decrypt
```

3. 检查配置文件加密状态：

```bash
sshw -check
```

### 指定配置文件

```bash
# 指定配置文件
sshw -config /path/to/config.yml
# 或
sshw -config /path/to/config.json
```

### 跳板机配置

```yaml
- name: "目标服务器"
  host: "target.example.com"
  user: "target_user"
  jump:
    - name: "跳板机1"
      host: "jump1.example.com"
      user: "jump_user1"
    - name: "跳板机2"
      host: "jump2.example.com"
      user: "jump_user2"
```

### 回调命令

```yaml
- name: "服务器"
  host: "example.com"
  callback-shells:
    - cmd: "echo 'Hello'"
      delay: 1s
    - cmd: "pwd"
      delay: 2s
```

## 常见问题

1. 配置文件格式错误
   - 确保配置文件格式正确（YAML 或 JSON）
   - 检查缩进和空格
   - 使用在线 YAML/JSON 验证工具验证

2. 连接失败
   - 检查服务器地址和端口是否正确
   - 确认用户名和密码/密钥是否正确
   - 检查网络连接是否正常

3. 权限问题
   - 确保配置文件权限正确（建议 600）
   - 检查密钥文件权限（建议 600）

## 注意事项

1. 配置文件权限
   - 建议将配置文件权限设置为 600
   - 避免将配置文件提交到版本控制系统

2. 密码安全
   - 建议使用密钥认证
   - 如果使用密码，建议加密配置文件

3. 跳板机配置
   - 确保跳板机配置正确
   - 注意跳板机的连接顺序

## 致谢

- [go-prompt](https://github.com/c-bata/go-prompt)
- [ssh_config](https://github.com/kevinburke/ssh_config)
- [homedir](https://github.com/atrox/homedir)

## 许可证

MIT License