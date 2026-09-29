# Srvx

一个统一的多协议文件服务器，通过独立子命令启动 HTTP / HTTPS / FTP / FTPS / SFTP / NFS 服务，共享同一根目录。

内置默认 TLS 证书与 SSH 主机密钥，`https` / `ftps` / `sftp` 开箱即用，无需额外配置即可启用加密传输。

## 功能特性

- **多协议**：HTTP、HTTPS、FTP、显式 FTPS、SFTP、NFS v3
- **统一鉴权**：用户名/密码认证，留空则匿名访问（常量时间比较防时序攻击）
- **嵌入证书**：默认 TLS 证书与 ed25519/RSA 主机密钥编译进二进制，零配置启用加密
- **可写 HTTP**：可选开启 PUT 上传 / DELETE 删除，带路径穿越防护
- **单二进制**：纯 Go 实现，交叉编译 linux/arm64/windows

## 安装

需要 Go 1.26+。

```bash
go install srvx@latest
```

或从源码构建：

```bash
go build -o srvx .
```

交叉编译（生成到 `bin/`）：

```bash
make all              # linux_amd64 linux_arm64 windows_amd64
make linux_arm64      # 单一目标
```

## 使用方法

每个协议对应一个独立子命令，语义单一。通用选项：

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-r, --root` | 共享根目录 | `.` |
| `-u, --user` | 用户名（空 = 匿名） | 空 |
| `-p, --pass` | 密码 | 空 |
| `--tls-cert` | TLS 证书文件路径 | 空 |
| `--tls-key` | TLS 密钥文件路径 | 空 |

### HTTP

启动纯 HTTP 文件服务器（默认端口 8080）。

```bash
srvx http                          # 匿名只读
srvx http -P 9000 --upload         # 开启 PUT/DELETE
srvx http -u admin -p secret       # Basic 认证
```

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-P, --port` | 监听端口 | `8080` |
| `--upload` | 启用上传(PUT)与删除(DELETE) | `false` |

### HTTPS

启动 HTTPS 文件服务器（默认端口 8443）。未指定 `--tls-cert/--tls-key` 时自动使用嵌入证书。

```bash
srvx https                         # 嵌入证书
srvx https --tls-cert cert.pem --tls-key key.pem
```

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-P, --port` | 监听端口 | `8443` |
| `--upload` | 启用上传(PUT)与删除(DELETE) | `false` |

### FTP

启动 FTP 文件服务器（默认端口 2121）。

```bash
srvx ftp
srvx ftp -u user -p pass
```

### FTPS

启动显式 FTPS 文件服务器（默认端口 2121）。未指定证书时自动使用嵌入证书。

```bash
srvx ftps
srvx ftps --tls-cert cert.pem --tls-key key.pem
```

### SFTP

启动 SFTP 文件服务器（默认端口 2022）。未指定 `--key` 时使用嵌入主机密钥。

```bash
srvx sftp                          # 嵌入 ed25519 主机密钥
srvx sftp --type rsa               # 使用嵌入 RSA 主机密钥
srvx sftp --key ~/.ssh/id_ed25519  # 自定义主机密钥
srvx sftp --banner "Welcome to Srvx"
```

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-P, --port` | 监听端口 | `2022` |
| `--key` | SSH 主机私钥路径（空 = 嵌入默认密钥） | 空 |
| `--type` | 嵌入默认密钥类型：`ed25519` 或 `rsa` | `ed25519` |
| `--banner` | 认证前发送给客户端的 SSH banner | 空 |

### NFS

启动 NFS v3 文件服务器（默认端口 2049）。无鉴权，依赖 NFS 自身的 null-auth。

```bash
srvx nfs
srvx nfs -P 2049 -r /data
```

## 项目结构

```
srvx/
├── main.go              # 入口，注入嵌入证书/主机密钥
├── cmd/                 # CLI 子命令（cobra）
│   ├── root.go          # 根命令与通用选项
│   ├── http.go https.go
│   ├── ftp.go  ftps.go
│   ├── sftp.go
│   └── nfs.go
├── internal/
│   ├── auth/            # 用户存储与常量时间密码校验
│   ├── httpfs/          # HTTP/HTTPS 实现（含上传/删除）
│   ├── ftp/             # FTP/FTPS 实现
│   ├── sftp/            # SFTP over SSH 实现
│   └── nfs/             # NFS v3 实现
├── resources/           # go:embed 嵌入的默认证书与主机密钥
└── Makefile             # 交叉编译
```

## 许可证

MIT License © Artiver