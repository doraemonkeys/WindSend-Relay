<h3 align="center"> 中文 | <a href='https://github.com/doraemonkeys/WindSend-Relay'>English</a></h3>


# WindSend-中继服务器

[![Go Report Card](https://goreportcard.com/badge/github.com/doraemonkeys/WindSend-Relay/server)](https://goreportcard.com/report/github.com/doraemonkeys/WindSend-Relay/server)
[![LICENSE](https://img.shields.io/github/license/doraemonkeys/WindSend-Relay)](https://github.com/doraemonkeys/WindSend-Relay/blob/main/LICENSE)



WindSend-Relay 是 [WindSend](https://github.com/doraemonkeys/WindSend) 中继服务器的 Go 语言实现。WindSend 使用 TLS 证书进行身份验证并加密中继流量，即使在使用第三方中继服务器时也能确保数据安全。


## 使用 Docker 快速开始

1. 拉取最新镜像

   ```bash
   docker pull doraemonkey/windsend-relay:latest
   ```

2. 显示版本信息

   ```bash
   docker run --rm doraemonkey/windsend-relay:latest -version
   ```

3. 运行容器

   ```bash
   docker run -d \
   --name ws-relay \
   -p 16779:16779 \
   -p 16780:16780 \
   -e WS_MAX_CONN="100" \
   doraemonkey/windsend-relay:latest
   # 后台管理密码将被生成并显示在 Docker 日志中。
   ```

> 16779 端口用于中继流量（协议：TCP），16780 端口用于后台页面（协议：HTTP）。


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
    cd WindSend-Relay/server
    ```


2.  **构建应用：**

    ```bash
    # 需要 Go 1.24+
    cd WindSend-Relay/server
    go build -o windsend-relay
    
    cd ../relay_admin
    npm install
    npm run build
    ```


## 使用示例

**使用默认设置运行（监听 `0.0.0.0:16779`，无认证，管理后台监听 `0.0.0.0:16780`，使用默认用户名/生成的密码）：**

```bash
./windsend-relay -max-conn=50
# 注意：SecretInfo 和 IDWhitelist 不能通过简单的命令行标志设置。请使用 JSON 或环境变量。
# 管理后台密码将在首次运行时生成并打印到日志中。
```

**在不同端口上运行并启用身份验证（使用环境变量）：**

```bash
# 中继服务监听端口 19999
export WS_LISTEN_ADDR="0.0.0.0:19999"
# 启用身份验证
export WS_ENABLE_AUTH="true"
# 设置第一个密钥的密钥值
export WS_SECRET_0_KEY="your_secret_key_0"
# 设置第一个密钥的最大连接数
export WS_SECRET_0_MAX_CONN="5"
# 配置管理后台
export WS_ADMIN_ADDR="0.0.0.0:19998"
export WS_ADMIN_USER="myadmin"
export WS_ADMIN_PASSWORD="a_very_secure_password_at_least_12_chars" # 至少12个字符的安全密码
# 使用环境变量运行中继服务
./windsend-relay -use-env
```

**使用 JSON 配置文件运行：**

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
  "enable_auth": true,
  "log_level": "INFO",
  "admin_config": {
    "user": "admin",
    "password": "a_very_secure_password_at_least_12_chars",
    "addr": "0.0.0.0:16780"
  }
}
```

**使用 Docker 默认设置运行（无认证）**

```bash
docker run -d \
  --name ws-relay \
  -p 16779:16779 \
  -p 16780:16780 \
  -e WS_MAX_CONN="100" \
  doraemonkey/windsend-relay:latest
# 管理后台密码将被生成并显示在 Docker 日志中。
```
**使用 Docker Compose 运行**

```yaml
services:
  windsend-relay:
    image: doraemonkey/windsend-relay:latest
    container_name: windsend-relay-app
    restart: unless-stopped
    ports:
      - "16779:16779" # 中继端口（协议：TCP）
      - "16780:16780" # 后台 UI 端口（协议：HTTP）
    environment:
      # --- 基本中继配置 ---
      WS_LISTEN_ADDR: "0.0.0.0:16779"
      WS_MAX_CONN: "100"             # 全局最大连接数（按需调整）
      WS_LOG_LEVEL: "INFO"

      # --- 认证 ---
      WS_ENABLE_AUTH: "true"         # 设置为 "false" 禁用身份验证
      # 根据您的需求配置 SecretInfo 。

      # 示例：使用密钥（WS_SECRET 前缀）
      # 为多个密钥添加更多 WS_SECRET_<index>_* 变量。索引从 0 开始。
      WS_SECRET_0_KEY: "YOUR_VERY_SECRET_KEY_HERE"  # !!! 重要：请修改此项 !!!
      WS_SECRET_0_MAX_CONN: "10"                    # 此特定密钥允许的最大连接数

      # 第二个密钥示例：
      # WS_SECRET_1_KEY: "ANOTHER_SECRET_KEY"
      # WS_SECRET_1_MAX_CONN: "5"

      # --- 管理后台 Web 界面配置 ---
      WS_ADMIN_ADDR: "0.0.0.0:16780" # 管理后台 UI 的地址
      WS_ADMIN_USER: "admin"         # 管理后台用户名
      WS_ADMIN_PASSWORD: "" # !!! 请修改此项 !!! 如果留空，首次启动时会生成一个随机密码并输出（至少12字符）。

    volumes:
      - ./logs:/app/logs
      - ./data:/app/data # 包含 relay.db
      #- ./config.json:/app/config.json # 可选：使用配置文件代替环境变量

    # 可选：使用配置文件代替环境变量
    # command: ["-config", "/app/config.json"]
```


**获取版本信息：**

```bash
./windsend-relay -version
# 或者使用 Docker
docker run --rm doraemonkey/windsend-relay:latest -version
```



## 配置

WindSend-Relay 可以通过以下三种方式进行配置，优先级顺序如下：

1.  **命令行标志：** 最高优先级。如果指定了 `-config` 或 `-use-env`，则忽略其他标志（`-version` 除外）。
2.  **环境变量：** 如果传递了 `-use-env` 标志或运行默认的 Docker 入口点，则使用环境变量。如果同时使用了 `-config` 和 `-use-env`（尽管通常只选择一种方法），环境变量会覆盖 JSON 文件中的设置。
3.  **JSON 配置文件：** 如果通过 `-config` 标志传递了文件路径，则使用此文件。
4.  **默认值：** 最低优先级，如果没有为特定选项提供其他配置，则使用默认值。

### 配置选项

| 参数                 | JSON 键              | 标志           | 环境变量                          | 类型           | 默认值                                | 描述                                                                                                                 |
| :------------------- | :-------------------- | :------------- | :-------------------------------------------- | :------------- | :------------------------------------ | :------------------------------------------------------------------------------------------------------------------- |
| 监听地址             | `listen_addr`         | `-listen-addr` | `WS_LISTEN_ADDR`                              | `string`       | `0.0.0.0:16779`                       | 中继服务器监听的 IP 地址和端口。                                                                                     |
| 最大连接数           | `max_conn`            | `-max-conn`    | `WS_MAX_CONN`                                 | `int`          | `100`                                 | 允许的全局最大并发客户端连接数。                                                                                       |
| ID 白名单            | `id_whitelist`        | *N/A*          | `WS_ID_WHITELIST`                            | `[]string`     | `[]`                                  | 允许连接的客户端 ID 列表。如果为空或省略，则允许所有 ID（需通过认证）。从 0 开始索引。                               |
| 密钥信息             | `secret_info`         | *N/A*          | `WS_SECRET_<n>_KEY`, `WS_SECRET_<n>_MAX_CONN` | `[]SecretInfo` | `[]`                                  | 用于身份验证的密钥及其关联连接限制的列表。详见下文。从 0 开始索引。                                                 |
| 启用认证             | `enable_auth`         | *N/A*          | `WS_ENABLE_AUTH`                              | `bool`         | `false`                               | 如果为 `true`，客户端必须使用 `Secret Info` 中的有效密钥进行身份验证。                                                   |
| 日志级别             | `log_level`           | `-log-level`   | `WS_LOG_LEVEL`                                | `string`       | `INFO`                                | 日志级别。有效值：`DEBUG`, `INFO`, `WARN`, `ERROR`, `DPANIC`, `PANIC`, `FATAL`。                                      |
| 管理员用户名         | `admin_config.user`   | *N/A*          | `WS_ADMIN_USER`                               | `string`       | `admin`                               | 管理后台 Web 界面的用户名。                                                                                          |
| 管理员密码           | `admin_config.password`| *N/A*          | `WS_ADMIN_PASSWORD`                           | `string`       | *(生成的12位ASCII字符串)*             | 管理后台 Web 界面的密码。如果为空，则在启动时生成一个 12 位的随机 ASCII 密码并记录在日志中。如果设置，则必须至少包含 12 个字符。 |
| 管理后台监听地址     | `admin_config.addr`   | *N/A*          | `WS_ADMIN_ADDR`                               | `string`       | `0.0.0.0:16780`                       | 管理后台 Web 界面监听的 IP 地址和端口。                                                                              |
| 配置文件             | *N/A*                 | `-config`      | *N/A*                                         | `string`       | `""`                                  | JSON 配置文件的路径。如果设置，则忽略其他标志（`-version` 除外）。                                                      |
| 使用环境变量         | *N/A*                 | `-use-env`     | *N/A*                                         | `bool`         | `false`                               | 如果为 `true`，则从环境变量读取配置。忽略其他标志（`-version` 除外）。                                                   |
| 显示版本             | *N/A*                 | `-version`     | *N/A*                                         | `bool`         | `false`                               | 打印版本信息并退出。                                                                                               |

**注意：** 在 `v0.1.0` 及更高版本中，命令行标志 `-enable-auth` 已被移除。请使用环境变量 `WS_ENABLE_AUTH` 或 JSON 配置 `enable_auth` 来控制它。

**关于用于切片和嵌套结构的环境变量的说明：**

*   **`WS_ID_WHITELIST`:** 示例：
    ```bash
    export WS_ID_WHITELIST="client_id_1,client_id_2"
    ```
*   **`WS_SECRET_<n>_KEY` / `WS_SECRET_<n>_MAX_CONN`:** 对于 Secret Info 切片，请为每个结构字段使用索引变量。示例：
    ```bash
    # 密钥 0
    export WS_SECRET_0_KEY="mysecret1"
    export WS_SECRET_0_MAX_CONN="5"
    # 密钥 1
    export WS_SECRET_1_KEY="mysecret2"
    export WS_SECRET_1_MAX_CONN="10"
    ```
*   **`WS_ADMIN_*`**: 对于 Admin 配置，请使用前缀 `WS_ADMIN_` 后跟大写的字段名称（`USER`, `PASSWORD`, `ADDR`）。示例：
    ```bash
    export WS_ADMIN_USER="myadmin"
    export WS_ADMIN_PASSWORD="a_very_secure_password_at_least_12_chars" # 至少12字符的安全密码
    export WS_ADMIN_ADDR="0.0.0.0:19998"
    ```


## 贡献

欢迎贡献！请随时在 [GitHub 仓库](https://github.com/doraemonkeys/WindSend-Relay) 提交拉取请求 (Pull Request) 或开启问题 (Issue)。

