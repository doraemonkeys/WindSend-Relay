<h3 align="center"> English | <a href='README-ZH.md'>简体中文</a></h3>


# WindSend-Relay

[![Go Report Card](https://goreportcard.com/badge/github.com/doraemonkeys/WindSend-Relay)](https://goreportcard.com/report/github.com/doraemonkeys/WindSend-Relay)
[![LICENSE](https://img.shields.io/github/license/doraemonkeys/WindSend-Relay)](https://github.com/doraemonkeys/WindSend-Relay/blob/main/LICENSE)



WindSend-Relay is a Go implementation of the [WindSend](https://github.com/doraemonkeys/WindSend) relay server. WindSend uses TLS certificates for authentication and encrypts the relay traffic, ensuring data security even when using third-party relay servers.



## Installation

You can run WindSend-Relay using pre-built binaries, by building from source, or using Docker.



**Option 1: Pre-built Binaries**

Check the [Releases](https://github.com/doraemonkeys/WindSend-Relay/releases) page for pre-compiled binaries for your operating system. Download the appropriate archive, extract it, and run the executable. Usually, you should download **Linux** version to run on most Linux distributions.



**Option 2: Using Docker**

```
docker pull doraemonkey/windsend-relay:latest
```



**Option 3: Build from Source**

1. **Clone the repository:**

   ```bash
   git clone https://github.com/doraemonkeys/WindSend-Relay.git
   cd WindSend-Relay
   ```


2. **Build the application:**

   ```bash
   # Go 1.24+
   go build -o windsend-relay
   ```

3. **Run the executable:**

   ```bash
   ./windsend-relay [flags]
   ```

## Usage Examples

**Run with defaults (listening on `0.0.0.0:16779`, no auth):**

```bash
./windsend-relay -max-conn=50
# Note: SecretInfo and IDWhitelist cannot be set via simple flags. Use JSON or Env Vars for those.
```

**Run on a different port with authentication enabled (using env vars):**

```bash
# Listen on port 19999
export WS_LISTEN_ADDR="0.0.0.0:19999"
# Enable authentication
export WS_ENABLE_AUTH="true"
# Set secret key for the first secret
export WS_SECRET_0_KEY="your_secret_key_0"
# Set max connections for the first secret
export WS_SECRET_0_MAX_CONN="5"
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
  "enable_auth": true
}
```

**Run with docker defaults (no auth)**

```bash
docker run -d \
  --name ws-relay \
  -p 16779:16779 \
  -e WS_MAX_CONN="100" \
  doraemonkey/windsend-relay:latest
```
**Run with Docker Compose**

```yaml
services:
  windsend-relay:
    image: doraemonkey/windsend-relay:latest
    container_name: windsend-relay-app
    restart: unless-stopped            
    ports:
      - "16779:16779" 
    environment:
      # --- Basic Configuration ---
      WS_MAX_CONN: "100"             # Overall max connections (adjust as needed)
      WS_ENABLE_AUTH: "true"         # Set to "false" to disable authentication

      # --- Authentication & Whitelisting ---
      # Configure EITHER SecretInfo OR IDWhitelist based on your needs.
      
      # Example: Using Secret Keys (WS_SECRET prefix)
      # Add more WS_SECRET_<index>_* variables for multiple secrets. Index starts at 0.
      WS_SECRET_0_KEY: "YOUR_VERY_SECRET_KEY_HERE"  # !!! IMPORTANT: CHANGE THIS !!!
      WS_SECRET_0_MAX_CONN: "10"                    # Max connections allowed for this specific key

      # Example for a second secret key:
      # WS_SECRET_1_KEY: "ANOTHER_SECRET_KEY"
      # WS_SECRET_1_MAX_CONN: "5"

    # Optional: Add volumes if you need persistent logs outside the container
    # volumes:
    #   - ./logs:/app/logs
    #   - ./config.json:/app/config.json

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
2.  **Environment Variables:** Used if the `-use-env` flag is passed or if running the default Docker entrypoint.
3.  **JSON Configuration File:** Used if the `-config` flag is passed with a file path.
4.  **Default Values:** Lowest precedence, used if no other configuration is provided for a specific option.

### Configuration Options

| Parameter       | JSON Key       | Flag           | Environment Variable                          | Type           | Default         | Description                                                  |
| :-------------- | :------------- | :------------- | :-------------------------------------------- | :------------- | :-------------- | :----------------------------------------------------------- |
| Listen Address  | `listen_addr`  | `-listen-addr` | `WS_LISTEN_ADDR`                              | `string`       | `0.0.0.0:16779` | IP address and port for the relay server to listen on.       |
| Max Connections | `max_conn`     | `-max-conn`    | `WS_MAX_CONN`                                 | `int`          | `100`           | Global maximum number of concurrent client connections allowed. |
| ID Whitelist    | `id_whitelist` | *N/A*          | `WS_ID_WHITELIST_<n>`                         | `[]string`     | `[]`            | List of client IDs allowed to connect. If empty or omitted, all IDs are allowed (subject to auth). |
| Secret Info     | `secret_info`  | *N/A*          | `WS_SECRET_<n>_KEY`, `WS_SECRET_<n>_MAX_CONN` | `[]SecretInfo` | `[]`            | List of secret keys for authentication and their associated connection limits. See details below. |
| Enable Auth     | `enable_auth`  | `-enable-auth` | `WS_ENABLE_AUTH`                              | `bool`         | `false`         | If `true`, clients must authenticate using a valid secret key from `Secret Info`. |
| Config File     | *N/A*          | `-config`      | *N/A*                                         | `string`       | `""`            | Path to a JSON configuration file. If set, other flags (except `-version`) are ignored. |
| Use Environment | *N/A*          | `-use-env`     | *N/A*                                         | `bool`         | `false`         | If `true`, configuration is read from environment variables. Other flags (except `-version`) are ignored. |
| Show Version    | *N/A*          | `-version`     | *N/A*                                         | `bool`         | `false`         | Print version information and exit.                          |
| Log Level      | `log_level`     | `-log-level`   | `WS_LOG_LEVEL`                                | `string`       | `INFO`          | Log level. Valid values: `DEBUG`, `INFO`, `WARN`, `ERROR`, `DPANIC`, `PANIC`, `FATAL`. |

**Note:** In `v0.1.0` and later versions, the command line flag `-enable-auth` has been removed. Please use the environment variable `WS_ENABLE_AUTH` or the JSON configuration `enable_auth` to control it.

**Notes on Environment Variables for Slices:**

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



## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues on the [GitHub repository](https://github.com/doraemonkeys/WindSend-Relay).



