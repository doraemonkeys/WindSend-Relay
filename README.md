<h3 align="center"> English | <a href='README-ZH.md'>简体中文</a></h3>


# WindSend-Relay

[![Go Report Card](https://goreportcard.com/badge/github.com/doraemonkeys/WindSend-Relay/server)](https://goreportcard.com/report/github.com/doraemonkeys/WindSend-Relay/server)
[![LICENSE](https://img.shields.io/github/license/doraemonkeys/WindSend-Relay)](https://github.com/doraemonkeys/WindSend-Relay/blob/main/LICENSE)



WindSend-Relay is a Go implementation of the [WindSend](https://github.com/doraemonkeys/WindSend) relay server. WindSend uses TLS certificates for authentication and encrypts the relay traffic, ensuring data security even when using third-party relay servers.

## Quick Start with Docker

1. Pull the latest image

   ```bash
   docker pull doraemonkey/windsend-relay:latest
   ```

2. Show version information

   ```bash
   docker run --rm doraemonkey/windsend-relay:latest -version
   ```

3. Run the container

   ```bash
   docker run -d \
   --name ws-relay \
   -p 16779:16779 \
   -p 16780:16780 \
   -e WS_MAX_CONN="100" \
   doraemonkey/windsend-relay:latest
   # Admin password will be generated and shown in docker logs.
   ```

> Port 16779 is used for relay traffic (protocol: TCP), and port 16780 is used for the web interface (protocol: HTTP).


## Installation

You can run WindSend-Relay using pre-built binaries, by building from source, or using Docker.



**Option 1: Pre-built Binaries**

Check the [Releases](https://github.com/doraemonkeys/WindSend-Relay/releases) page for pre-compiled binaries for your operating system. Download the appropriate archive, extract it, and run the executable. Usually, you should download **Linux** version to run on most Linux distributions.



**Option 2: Using Docker**

```bash
docker pull doraemonkey/windsend-relay:latest
```



**Option 3: Build from Source**

1. **Clone the repository:**

   ```bash
   git clone https://github.com/doraemonkeys/WindSend-Relay.git
   ```


2. **Build the application:**

   ```bash
   # Go 1.24+
   cd WindSend-Relay/server
   go build -o windsend-relay
   
   cd ../relay_admin
   npm install
   npm run build
   ```


## Usage Examples

**Run with defaults (listening on `0.0.0.0:16779`, no auth, admin on `0.0.0.0:16780` with default user/generated password):**

```bash
./windsend-relay -max-conn=50
# Note: SecretInfo and IDWhitelist cannot be set via simple flags. Use JSON or Env Vars for those.
# Admin password will be generated and printed to the log on first run.
```

**Run on a different port with authentication enabled (using env vars):**

```bash
# Listen on port 19999 for relay
export WS_LISTEN_ADDR="0.0.0.0:19999"
# Enable authentication
export WS_ENABLE_AUTH="true"
# Set secret key for the first secret
export WS_SECRET_0_KEY="your_secret_key_0"
# Set max connections for the first secret
export WS_SECRET_0_MAX_CONN="5"
# Configure admin interface
export WS_ADMIN_ADDR="0.0.0.0:19998"
export WS_ADMIN_USER="myadmin"
export WS_ADMIN_PASSWORD="a_very_secure_password_at_least_12_chars"
# Run the relay using environment variables
./windsend-relay -use-env
```

**Run using a JSON config file:**

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

**Run with docker defaults (no auth)**

```bash
docker run -d \
  --name ws-relay \
  -p 16779:16779 \
  -p 16780:16780 \
  -e WS_MAX_CONN="100" \
  doraemonkey/windsend-relay:latest
# Admin password will be generated and shown in docker logs.
```
**Run with Docker Compose**

```yaml
services:
  windsend-relay:
    image: doraemonkey/windsend-relay:latest
    container_name: windsend-relay-app
    restart: unless-stopped
    ports:
      - "16779:16779" # Relay traffic port(protocol: TCP)
      - "16780:16780" # Web Interface port(protocol: HTTP)
    environment:
      # --- Basic Relay Configuration ---
      WS_LISTEN_ADDR: "0.0.0.0:16779"
      WS_MAX_CONN: "100"             # Overall max connections (adjust as needed)
      WS_LOG_LEVEL: "INFO"

      # --- Authentication ---
      WS_ENABLE_AUTH: "true"         # Set to "false" to disable authentication
      # Configure SecretInfo.

      # Example: Using Secret Keys (WS_SECRET prefix)
      # Add more WS_SECRET_<index>_* variables for multiple secrets. Index starts at 0.
      WS_SECRET_0_KEY: "YOUR_VERY_SECRET_KEY_HERE"  # !!! IMPORTANT: CHANGE THIS !!!
      WS_SECRET_0_MAX_CONN: "10"                    # Max connections allowed for this specific key

      # Example for a second secret key:
      # WS_SECRET_1_KEY: "ANOTHER_SECRET_KEY"
      # WS_SECRET_1_MAX_CONN: "5"

      # --- Admin Web Interface Configuration ---
      WS_ADMIN_ADDR: "0.0.0.0:16780" # Address for the Admin UI
      WS_ADMIN_USER: "admin"         # Admin username
      WS_ADMIN_PASSWORD: "YOUR_SECURE_ADMIN_PASSWORD_12_CHARS_MIN" # !!! CHANGE THIS !!! If left empty or omitted, a random password is generated and printed to the container logs on the first start.

    volumes:
      - ./logs:/app/logs
      - ./data:/app/data # relay.db
      #- ./config.json:/app/config.json # Optional: use config file instead of env vars

    # Optional: use config file instead of env vars
    # command: ["-config", "/app/config.json"]
```


**Get Version Information:**

```bash
./windsend-relay -version
# Or with Docker
docker run --rm doraemonkey/windsend-relay:latest -version
```



## Configuration

WindSend-Relay can be configured using three methods, with the following order of precedence:

1.  **Command-line Flags:** Highest precedence. If `-config` or `-use-env` is specified, other flags (except `-version`) are ignored.
2.  **Environment Variables:** Used if the `-use-env` flag is passed or if running the default Docker entrypoint. Environment variables override JSON file settings if both `-config` and `-use-env` are somehow used (though typically only one method is chosen).
3.  **JSON Configuration File:** Used if the `-config` flag is passed with a file path.
4.  **Default Values:** Lowest precedence, used if no other configuration is provided for a specific option.

### Configuration Options

| Parameter            | JSON Key              | Flag           | Environment Variable                          | Type           | Default                               | Description                                                                                                          |
| :------------------- | :-------------------- | :------------- | :-------------------------------------------- | :------------- | :------------------------------------ | :------------------------------------------------------------------------------------------------------------------- |
| Listen Address       | `listen_addr`         | `-listen-addr` | `WS_LISTEN_ADDR`                              | `string`       | `0.0.0.0:16779`                       | IP address and port for the relay server to listen on.                                                              |
| Max Connections      | `max_conn`            | `-max-conn`    | `WS_MAX_CONN`                                 | `int`          | `100`                                 | Global maximum number of concurrent client connections allowed.                                                        |
| ID Whitelist         | `id_whitelist`        | *N/A*          | `WS_ID_WHITELIST_<n>`                         | `[]string`     | `[]`                                  | List of client IDs allowed to connect. If empty or omitted, all IDs are allowed (subject to auth). Indexed from 0.    |
| Secret Info          | `secret_info`         | *N/A*          | `WS_SECRET_<n>_KEY`, `WS_SECRET_<n>_MAX_CONN` | `[]SecretInfo` | `[]`                                  | List of secret keys for authentication and their associated connection limits. See details below. Indexed from 0. |
| Enable Auth          | `enable_auth`         | *N/A*          | `WS_ENABLE_AUTH`                              | `bool`         | `false`                               | If `true`, clients must authenticate using a valid secret key from `Secret Info`.                                      |
| Log Level            | `log_level`           | `-log-level`   | `WS_LOG_LEVEL`                                | `string`       | `INFO`                                | Log level. Valid values: `DEBUG`, `INFO`, `WARN`, `ERROR`, `DPANIC`, `PANIC`, `FATAL`.                                 |
| Admin User           | `admin_config.user`   | *N/A*          | `WS_ADMIN_USER`                               | `string`       | `admin`                               | Username for the admin web interface.                                                                                |
| Admin Password       | `admin_config.password`| *N/A*          | `WS_ADMIN_PASSWORD`                           | `string`       | *(generated 12-char ASCII string)*    | Password for the admin web interface. If empty, a random 12-character ASCII password is generated on startup and logged. Must be at least 12 characters if set. |
| Admin Listen Address | `admin_config.addr`   | *N/A*          | `WS_ADMIN_ADDR`                               | `string`       | `0.0.0.0:16780`                       | IP address and port for the admin web interface to listen on.                                                        |
| Config File          | *N/A*                 | `-config`      | *N/A*                                         | `string`       | `""`                                  | Path to a JSON configuration file. If set, other flags (except `-version`) are ignored.                                |
| Use Environment      | *N/A*                 | `-use-env`     | *N/A*                                         | `bool`         | `false`                               | If `true`, configuration is read from environment variables. Other flags (except `-version`) are ignored.              |
| Show Version         | *N/A*                 | `-version`     | *N/A*                                         | `bool`         | `false`                               | Print version information and exit.                                                                                  |

**Note:** In `v0.1.0` and later versions, the command line flag `-enable-auth` has been removed. Please use the environment variable `WS_ENABLE_AUTH` or the JSON configuration `enable_auth` to control it.

**Notes on Environment Variables for Slices and Nested Structures:**

*   **`WS_ID_WHITELIST_<n>`:** For the ID whitelist, use indexed variables starting from 0. Example:
    ```bash
    export WS_ID_WHITELIST_0="client_id_1"
    export WS_ID_WHITELIST_1="client_id_2"
    ```
*   **`WS_SECRET_<n>_KEY` / `WS_SECRET_<n>_MAX_CONN`:** For the Secret Info slice, use indexed variables for each struct field. Example:
    ```bash
    # Secret 0
    export WS_SECRET_0_KEY="mysecret1"
    export WS_SECRET_0_MAX_CONN="5"
    # Secret 1
    export WS_SECRET_1_KEY="mysecret2"
    export WS_SECRET_1_MAX_CONN="10"
    ```
*   **`WS_ADMIN_*`**: For the Admin configuration, use the prefix `WS_ADMIN_` followed by the uppercase field name (`USER`, `PASSWORD`, `ADDR`). Example:
    ```bash
    export WS_ADMIN_USER="myadmin"
    export WS_ADMIN_PASSWORD="a_very_secure_password_at_least_12_chars"
    export WS_ADMIN_ADDR="0.0.0.0:19998"
    ```


## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues on the [GitHub repository](https://github.com/doraemonkeys/WindSend-Relay).
