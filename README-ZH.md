<h3 align="center"> 中文 | <a href='https://github.com/doraemonkeys/WindSend-Relay'>English</a></h3>


# WindSend-Relay

[![Go Report Card](https://goreportcard.com/badge/github.com/doraemonkeys/WindSend-Relay)](https://goreportcard.com/report/github.com/doraemonkeys/WindSend-Relay)
[![LICENSE](https://img.shields.io/github/license/doraemonkeys/WindSend-Relay)](https://github.com/doraemonkeys/WindSend-Relay/blob/main/LICENSE)

WindSend-Relay 是 [WindSend](https://github.com/doraemonkeys/WindSend) 中继服务器的 Go 语言实现，WindSend 使用 TLS 证书认证并对中转流量进行加密，即使使用第三方中继服务器，也能保证数据安全。

## 安装

您可以通过预编译的二进制文件、从源代码构建或使用 Docker 来运行 WindSend-Relay。

**方式一：预编译二进制文件**

请访问 [Releases](https://github.com/doraemonkeys/WindSend-Relay/releases) 页面查找适用于您操作系统的预编译二进制文件。下载相应的压缩包，解压并运行可执行文件。通常，您应该下载 **Linux** 版本以在大多数 Linux 发行版上运行。

**方式二：使用 Docker**

```bash
docker pull doraemonkey/windsend-relay:latest
```

**方式三：从源代码构建**

1.  **克隆仓库：**

    ```bash
    git clone https://github.com/doraemonkeys/WindSend-Relay.git
    cd WindSend-Relay
    ```

2.  **构建应用：**

    ```bash
    # 需要 Go 1.24+
    go build -o windsend-relay
    ```

3.  **运行可执行文件：**

    ```bash
    ./windsend-relay [flags]
    ```

## 使用示例

**使用默认设置运行 (监听 `0.0.0.0:16779`, 无身份验证):**

```bash
./windsend-relay -max-conn=50
# 注意：SecretInfo 和 IDWhitelist 无法通过简单的命令行标志设置。请使用 JSON 或环境变量进行配置。
```

**在不同端口上运行并启用身份验证 (使用环境变量):**

```bash
# 监听端口 19999
export WS_LISTEN_ADDR="0.0.0.0:19999"
# 启用身份验证
export WS_ENABLE_AUTH="true"
# 设置第一个密钥的 secret key
export WS_SECRET_0_KEY="your_secret_key_0"
# 设置第一个密钥的最大连接数
export WS_SECRET_0_MAX_CONN="5"
# 使用环境变量运行中继服务器
./windsend-relay -use-env
```

**使用 JSON 配置文件运行:**

```bash
./windsend-relay -config /path/to/your/config.json
```

```json
{
  "listen_addr": "0.0.0.0:16779",
  "max_conn": 100,
  "secret_info": [
    {
      "secret_key": "your_secret_key_0",
      "max_conn": 5
    },
    {
      "secret_key": "your_secret_key_1",
      "max_conn": 10
    }
  ],
  "enable_auth": true
}
```

**使用 Docker 默认设置运行 (无身份验证)**

```bash
docker run -d \
  --name ws-relay \
  -p 16779:16779 \
  -e WS_MAX_CONN="100" \
  doraemonkey/windsend-relay:latest
```

**使用 Docker Compose 运行**

```yaml
services:
  windsend-relay:
    image: doraemonkey/windsend-relay:latest
    container_name: windsend-relay-app
    restart: unless-stopped
    ports:
      - "16779:16779"
    environment:
      # --- 基本配置 ---
      WS_MAX_CONN: "100"             # 全局最大连接数 (根据需要调整)
      WS_ENABLE_AUTH: "true"         # 设置为 "false" 禁用身份验证

      # --- 身份验证与白名单 ---
      # 根据您的需求配置 SecretInfo 或 IDWhitelist 其中之一。

      # 示例：使用 Secret Key (WS_SECRET 前缀)
      # 为多个密钥添加更多 WS_SECRET_<索引>_* 变量。索引从 0 开始。
      WS_SECRET_0_KEY: "YOUR_VERY_SECRET_KEY_HERE"  # !!! 重要：请修改此密钥 !!!
      WS_SECRET_0_MAX_CONN: "10"                    # 此特定密钥允许的最大连接数

      # 第二个密钥示例:
      # WS_SECRET_1_KEY: "ANOTHER_SECRET_KEY"
      # WS_SECRET_1_MAX_CONN: "5"

      # 示例：使用 ID 白名单 (WS_ID_WHITELIST 前缀)
      # 如果使用白名单，通常需要设置 WS_ENABLE_AUTH="false"，除非您希望同时使用密钥和白名单
      # WS_ID_WHITELIST_0: "allowed_client_id_1"
      # WS_ID_WHITELIST_1: "allowed_client_id_2"

    # 可选：如果您需要在容器外部持久化日志，请添加卷
    # volumes:
    #   - ./logs:/app/logs
    #   - ./config.json:/app/config.json

    # 可选：使用配置文件代替环境变量
    # command: ["-config", "/app/config.json"]
```

**获取版本信息:**

```bash
./windsend-relay -version
# 或者使用 Docker
docker run --rm doraemonkey/windsend-relay:latest -version
```

## 配置

WindSend-Relay 可以通过以下三种方式进行配置，优先级顺序如下：

1.  **命令行标志:** 最高优先级。如果指定了 `-config` 或 `-use-env`，则除 `-version` 外的其他标志将被忽略。
2.  **环境变量:** 如果传递了 `-use-env` 标志，或者在运行默认的 Docker 入口点时使用。
3.  **JSON 配置文件:** 如果传递了带有文件路径的 `-config` 标志时使用。
4.  **默认值:** 最低优先级，在没有为特定选项提供其他配置时使用。

### 配置选项

| 参数         | JSON 键        | 标志           | 环境变量                                      | 类型           | 默认值          | 描述                                                         |
| :----------- | :------------- | :------------- | :-------------------------------------------- | :------------- | :-------------- | :----------------------------------------------------------- |
| 监听地址     | `listen_addr`  | `-listen-addr` | `WS_LISTEN_ADDR`                              | `string`       | `0.0.0.0:16779` | 中继服务器监听的 IP 地址和端口。                             |
| 最大连接数   | `max_conn`     | `-max-conn`    | `WS_MAX_CONN`                                 | `int`          | `100`           | 允许的全局最大并发客户端连接数。                             |
| ID 白名单    | `id_whitelist` | *N/A*          | `WS_ID_WHITELIST_<n>`                         | `[]string`     | `[]`            | 允许连接的客户端 ID 列表。如果为空或省略，则允许所有 ID (需通过身份验证)。 |
| 密钥信息     | `secret_info`  | *N/A*          | `WS_SECRET_<n>_KEY`, `WS_SECRET_<n>_MAX_CONN` | `[]SecretInfo` | `[]`            | 用于身份验证的密钥列表及其关联的连接限制。详见下文。         |
| 启用身份验证 | `enable_auth`  | *N/A*          | `WS_ENABLE_AUTH`                              | `bool`         | `false`         | 如果为 `true`，客户端必须使用 `Secret Info` 中的有效密钥进行身份验证。 |
| 配置文件路径 | *N/A*          | `-config`      | *N/A*                                         | `string`       | `""`            | JSON 配置文件的路径。如果设置，除 `-version` 外的其他标志将被忽略。 |
| 使用环境变量 | *N/A*          | `-use-env`     | *N/A*                                         | `bool`         | `false`         | 如果为 `true`，则从环境变量读取配置。除 `-version` 外的其他标志将被忽略。 |
| 显示版本     | *N/A*          | `-version`     | *N/A*                                         | `bool`         | `false`         | 打印版本信息并退出。  
| 日志级别     | `log_level`    | `-log-level`   | `WS_LOG_LEVEL`                                | `string`       | `INFO`          | 日志级别。有效值：`DEBUG`, `INFO`, `WARN`, `ERROR`, `DPANIC`, `PANIC`, `FATAL`。 |

**注意：** 在 `v0.1.0` 及之后版本中，命令行标志 `-enable-auth` 已被移除，请使用环境变量 `WS_ENABLE_AUTH` 或 JSON 配置 `enable_auth` 来控制。

**关于环境变量用于 Slice 类型的说明：**

*   **`WS_ID_WHITELIST_<n>`:** 对于 ID 白名单，请使用从 0 开始的索引变量。示例：
    ```bash
    export WS_ID_WHITELIST_0="client_id_1"
    export WS_ID_WHITELIST_1="client_id_2"
    ```
*   **`WS_SECRET_<n>_KEY` / `WS_SECRET_<n>_MAX_CONN`:** 对于 Secret Info 切片，请为每个结构体字段使用索引变量。示例：
    ```bash
    # 密钥 0
    export WS_SECRET_0_KEY="mysecret1"
    export WS_SECRET_0_MAX_CONN="5"
    # 密钥 1
    export WS_SECRET_1_KEY="mysecret2"
    export WS_SECRET_1_MAX_CONN="10"
    ```

## 贡献

欢迎贡献！请随时在 [GitHub 仓库](https://github.com/doraemonkeys/WindSend-Relay) 提交拉取请求 (Pull Request) 或开启问题 (Issue)。

