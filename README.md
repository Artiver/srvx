# Srvx

多协议文件服务器，通过统一 CLI 入口同时支持 **HTTP、SFTP、FTP、NFS** 四种协议访问本地文件系统。

## 功能特性

- **四种协议**：HTTP / SFTP / FTP / NFS v3，每种对应一个子命令
- **统一认证**：全局用户名/密码参数，空用户时自动启用匿名访问
- **TLS 加密**：HTTP、SFTP、FTP 均支持 TLS（FTP 为 Explicit FTPS 模式）
- **文件上传**：HTTP 子命令支持 `--upload` 启用 PUT 上传与 DELETE 删除
- **安全防护**：常量时间密码比较（防时序攻击）、路径穿越防护、TLS 1.2+ 最低版本
- **自动密钥**：SFTP 未提供主机密钥时自动生成临时 RSA 2048 密钥
- **跨平台编译**：Makefile 一键交叉编译 Linux (amd64/arm64) 与 Windows (amd64)

## 安装

### 前置要求

- Go 1.26+

### 从源码构建

```bash
git clone https://github.com/Artiver/srvx.git
cd srvx
make
```

编译产物输出到 `bin/` 目录：

| 文件 | 平台 |
|------|------|
| `srvx_linux_amd64` | Linux x86_64 |
| `srvx_linux_arm64` | Linux ARM64 |
| `srvx_windows_amd64.exe` | Windows x86_64 |

也可单独编译某一平台：

```bash
make linux_amd64
make linux_arm64
make windows_amd64
```

### 直接运行

```bash
go run . <子命令> [参数]
```

## 使用方法

### 全局参数

所有子命令共享以下参数：

| 参数 | 缩写 | 说明 | 默认值 |
|------|------|------|--------|
| `--root` | `-r` | 共享根目录 | `.` |
| `--user` | `-u` | 用户名（空 = 匿名） | 空 |
| `--pass` | `-p` | 密码 | 空 |
| `--tls-cert` | | TLS 证书路径 | 空 |
| `--tls-key` | | TLS 私钥路径 | 空 |

### HTTP 文件服务器

默认端口 `8080`，支持 Basic Auth 与 TLS。

```bash
# 匿名访问
srvx http -r /path/to/share

# 带认证 + 上传/删除 + TLS
srvx http -r ./data -u admin -p secret --upload \
    --tls-cert cert.pem --tls-key key.pem

# 自定义端口
srvx http -r ./data -P 9000
```

| 参数 | 说明 |
|------|------|
| `-P, --port` | 监听端口（默认 8080） |
| `--upload` | 启用 PUT 上传与 DELETE 删除 |

启用 `--upload` 后支持的 HTTP 方法：

- `GET` / `HEAD` — 文件浏览与下载
- `PUT` — 上传文件（返回 201）
- `DELETE` — 删除文件或目录（返回 204）
- `OPTIONS` — CORS 预检

### SFTP 文件服务器

默认端口 `2022`，基于 SSH 协议。

```bash
# 自动生成临时主机密钥
srvx sftp -r ./data -u admin -p secret

# 指定主机密钥
srvx sftp -r ./data -u admin -p secret --host-key ~/.ssh/id_rsa

# 启用 TLS（TLS over SSH）
srvx sftp -r ./data -u admin -p secret \
    --tls-cert cert.pem --tls-key key.pem
```

| 参数 | 说明 |
|------|------|
| `-P, --port` | 监听端口（默认 2022） |
| `--host-key` | SSH 主机密钥路径（空则自动生成临时 RSA 2048） |

### FTP 文件服务器

默认端口 `2121`，支持 Explicit FTPS。

```bash
# 匿名 FTP
srvx ftp -r /path/to/share

# 带认证的 FTPS
srvx ftp -r ./data -u admin -p secret \
    --tls-cert cert.pem --tls-key key.pem
```

| 参数 | 说明 |
|------|------|
| `-P, --port` | 监听端口（默认 2121） |

### NFS v3 文件服务器

默认端口 `2049`，无认证，内置 LRU 缓存（1024 项）。

```bash
srvx nfs -r /path/to/share

# 自定义端口
srvx nfs -r /path/to/share -P 2049
```

| 参数 | 说明 |
|------|------|
| `-P, --port` | 监听端口（默认 2049） |

### 协议端口汇总

| 子命令 | 协议 | 默认端口 | 认证 | TLS |
|--------|------|----------|------|-----|
| `http` | HTTP | 8080 | Basic Auth | 支持 |
| `sftp` | SFTP | 2022 | SSH 密码 | 支持 |
| `ftp` | FTP | 2121 | 用户密码 | 支持（FTPS） |
| `nfs` | NFS v3 | 2049 | 无 | 不支持 |

## 项目结构

```
srvx/
├── main.go                 # 程序入口
├── cmd/                    # CLI 命令层（cobra）
│   ├── root.go             # 根命令 + 全局参数
│   ├── http.go             # http 子命令
│   ├── sftp.go             # sftp 子命令
│   ├── ftp.go              # ftp 子命令
│   └── nfs.go              # nfs 子命令
├── internal/               # 内部实现
│   ├── auth/               # 认证存储（常量时间比较）
│   ├── httpfs/             # HTTP 文件服务器
│   ├── sftp/               # SFTP 服务器
│   ├── ftp/                # FTP 服务器
│   └── nfs/                # NFS v3 服务器
├── Makefile                # 跨平台构建脚本
├── go.mod                  # Go 模块定义
└── bin/                    # 编译产物（gitignore）
```

## 技术栈

| 组件 | 依赖 |
|------|------|
| CLI 框架 | [spf13/cobra](https://github.com/spf13/cobra) |
| SFTP / SSH | [pkg/sftp](https://github.com/pkg/sftp) + [golang.org/x/crypto/ssh](https://pkg.go.dev/golang.org/x/crypto/ssh) |
| FTP | [goftp/server](https://github.com/goftp/server) |
| NFS v3 | [willscott/go-nfs](https://github.com/willscott/go-nfs) |
| 文件系统抽象 | [go-git/go-billy](https://github.com/go-git/go-billy) |

## 安全说明

- **密码比较**：使用 `crypto/subtle.ConstantTimeCompare` 做常量时间比较，防止时序攻击
- **路径穿越**：HTTP 服务器通过 `safePath()` 确保解析后路径仍在根目录之内
- **TLS 版本**：强制最低 TLS 1.2
- **SFTP 工作目录**：通过 `sftp.WithServerWorkingDirectory` 限定工作目录范围
- **匿名访问**：未设置用户名时自动允许匿名访问

## 许可证

MIT License