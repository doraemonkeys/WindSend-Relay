## Run the container

*   **Using environment variables (default CMD):**
    ```bash
    docker run -d --name relay \
      -p 16779:16779 \
      -e WS_LISTEN_ADDR="0.0.0.0:16779" \
      -e WS_ENABLE_AUTH="false" \
      -e WS_MAX_CONN="200" \

      # e.g., for SecretInfo:
      # -e WS_SECRET_0_KEY="somekey" -e WS_SECRET_0_MAX_CONN="10"
      # -e WS_SECRET_1_KEY="anotherkey" -e WS_SECRET_1_MAX_CONN="5"
      # e.g., for IDWhitelist:
      # -e WS_ID_WHITELIST_0="id1" -e WS_ID_WHITELIST_1="id2"
      windsend-relay:latest
    ```

*   **Using a mounted configuration file:**
    Create a `config.json` on your host machine, for example at `/path/to/your/config.json`.
    ```json
    // /path/to/your/config.json
    {
      "listen_addr": "0.0.0.0:16779",
      "max_conn": 150,
      "enable_auth": false,
      "secret_info": [
        { "secret_key": "mysecret", "max_conn": 10 }
      ],
      "id_whitelist": ["allowed_id1", "allowed_id2"]
    }
    ```
    Then run the container, overriding the default `CMD` to use the config file flag:
    ```bash
    docker run -d --name relay \
      -p 16779:16779 \
      -v /path/to/your/config.json:/app/config.json \
      windsend-relay:latest --config /app/config.json
    ```
    *Note:* We mount the host file `/path/to/your/config.json` to `/app/config.json` inside the container and tell the application to use `/app/config.json` via the `--config` flag.

*   **Using command-line flags:**
    ```bash
    docker run -d --name relay \
      -p 16779:16779 \
      windsend-relay:latest --listen-addr 0.0.0.0:16779 --max-conn 50
    ```







```yaml
services:
  windsend-relay:
    image: windsend-relay:latest
    container_name: windsend-relay-app # A specific name for the running container
    restart: unless-stopped            
    ports:
      - "16779:16779" 
    environment:
      # --- Basic Configuration ---
      WS_MAX_CONN: "100"              # Overall max connections (adjust as needed)
      WS_ENABLE_AUTH: "false"         # Set to "true" to enable authentication

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
```

