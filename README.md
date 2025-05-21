# SSHW - SSH 客户端包装器

SSHW 是一个强大的 SSH 客户端包装器，它提供了更便捷的 SSH 连接管理功能。通过简单的配置文件，你可以轻松管理多个 SSH 连接，支持加密存储敏感信息，并提供丰富的连接选项。

## 特别说明

本项目是基于 `https://github.com/yinheli/sshw` 二次开发，特别感谢原作者的努力。

## 功能特点

- 支持多级服务器配置
- 支持服务器分组和别名
- 支持密码和密钥认证
- 支持配置文件加密（使用 AES-256-GCM）
- 支持主密码保护（使用系统 keyring 或本地文件）
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

## 安全特性

### 主密码管理

SSHW 使用主密码来保护配置文件中的敏感信息。主密码的哈希值会优先存储在系统的密钥环中（如 Linux 的 `gnome-keyring`、macOS 的 Keychain、Windows 的 Credential Manager），如果系统密钥环不可用，则会存储在本地文件 `.sshw-master` 中。

#### 设置主密码

首次使用加密功能时，系统会提示你设置主密码。你也可以随时使用以下命令设置或更改主密码：

```bash
# 设置主密码
sshw -set-master-password

# 更改主密码
sshw -change-master-password

# 移除主密码
sshw -remove-master-password
```

> **注意**：
> - 主密码是保护你所有敏感信息的关键，请选择一个强密码并妥善保管。
> - 如果使用本地文件存储（`.sshw-master`），请确保该文件的安全。
> - 移除主密码后，配置文件仍然保持加密状态，需要重新设置主密码或解密才能访问。

### 配置文件加密

SSHW 支持对配置文件中的敏感信息（如密码和密钥密码）进行加密存储。加密使用 AES-256-GCM 算法，密钥通过 PBKDF2 从主密码派生。

#### 加密操作

```bash
# 加密配置文件（仅加密未加密的条目）
sshw -encrypt

# 解密配置文件
sshw -decrypt

# 检查配置文件加密状态
sshw -check
```

> **注意**：
> - 加密操作只会加密未加密的条目，如果所有条目都已加密，会提示用户。
> - 建议定期更改主密码以提高安全性。
> - 如果使用本地文件存储主密码，请确保 `.sshw-master` 文件的安全。

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
| alias | 服务器别名 | 否 | - |
| host | 服务器地址 | 是 | - |
| user | 用户名 | 否 | 当前系统用户 |
| port | 端口号 | 否 | 22 |
| password | 密码 | 否 | - |
| keypath | 密钥文件路径 | 否 | - |
| passphrase | 密钥密码 | 否 | - |
| is_encrypted | 是否已加密（内部字段） | 否 | false |
| children | 子服务器列表 | 否 | - |
| jump | 跳板机配置 | 否 | - |
| mask_host | 是否掩码主机名 | 否 | false |
| show_host | 是否显示主机名 | 否 | true |
| enable_login_marker | 是否启用登录标记 | 否 | false |
| callback-shells | 回调命令列表 | 否 | - |

## 高级功能

### 显示与掩码设置

SSHW 支持对主机信息进行掩码显示，以增加安全性。可以通过以下配置项控制：

```yaml
- name: "服务器"
  host: "example.com"
  show_host: true    # 是否显示主机信息
  mask_host: true    # 是否掩码主机名
```

掩码规则：
- IP 地址（如 `206.237.7.72`）会被掩码为 `206.237.*.72`
- 域名（如 `example.com`）会被掩码为 `e*e.com`

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

## 命令行选项

SSHW 提供以下命令行选项：

| 选项 | 说明 | 示例 |
|------|------|------|
| `-config` | 指定配置文件路径 | `sshw -config ~/my-config.yml` |
| `-set-master-password` | 设置主密码 | `sshw -set-master-password` |
| `-change-master-password` | 更改主密码 | `sshw -change-master-password` |
| `-remove-master-password` | 移除主密码 | `sshw -remove-master-password` |
| `-encrypt` | 加密配置文件中的敏感信息 | `sshw -encrypt` |
| `-decrypt` | 解密配置文件中的敏感信息 | `sshw -decrypt` |
| `-check` | 检查配置文件加密状态 | `sshw -check` |
| `-version` | 显示版本信息 | `sshw -version` |
| `-help` | 显示帮助信息 | `sshw -help` |
| `-s` | 显示系统 SSH 配置文件（~/.ssh/config）中的服务器列表 | `sshw -s` |
| `-S` | 显示配置文件中定义的服务器列表 | `sshw -S` |

> **注意**：
> - `-version` 和 `-help` 是标准命令行选项，用于显示版本信息和帮助信息
> - `-s` 和 `-S` 的区别：
>   - `-s` 显示系统 SSH 配置文件（~/.ssh/config）中的服务器列表
>   - `-S` 显示 SSHW 配置文件（~/.sshw.yml 等）中定义的服务器列表
> - 使用 `-help` 可以查看所有可用的命令行选项

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

4. 加密相关
   - 如果无法访问系统密钥环，主密码将存储在 `.sshw-master` 文件中
   - 确保 `.sshw-master` 文件的安全（建议权限 600）
   - 如果忘记主密码，需要重新设置主密码或解密配置文件

## 注意事项

1. 配置文件权限
   - 建议将配置文件权限设置为 600
   - 避免将配置文件提交到版本控制系统

2. 密码安全
   - 建议使用密钥认证
   - 如果使用密码，建议加密配置文件
   - 定期更改主密码

3. 跳板机配置
   - 确保跳板机配置正确
   - 注意跳板机的连接顺序

## 致谢
- [sshw](https://github.com/yinheli/sshw)
- [go-prompt](https://github.com/c-bata/go-prompt)
- [ssh_config](https://github.com/kevinburke/ssh_config)
- [homedir](https://github.com/atrox/homedir)

## 许可证

MIT License