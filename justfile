set dotenv-load

# Flash the canteen node: stamps ROLE + current laptop LAN IP, uploads, restores.
# Usage: just a | just b | just kitchen   [PORT=/dev/cu.usbmodemXXX]
port := "/dev/cu.usbmodem101"
fqbn := "esp32:esp32:esp32c3"
ino  := "canteen-node/canteen-node.ino"

a: (flash "ROLE_COUNTER_A")
b: (flash "ROLE_COUNTER_B")
kitchen: (flash "ROLE_KITCHEN")

# build/test every Go module in the workspace
build:
    cd backend/food-kiosk && go build -o /dev/null ./...
    cd mqtt/tui && go build -o /dev/null .

test:
    cd backend/food-kiosk && go test ./...
    cd mqtt/tui && go test ./...

# Run the full backend (server + MQTT client)
backend:
    # Set defaults; can be overridden with env vars before calling `just backend`
    cd backend/food-kiosk && go run .

# Build the UI and serve it with the API; prints a LAN URL + QR code at startup
serve:
    #!/bin/zsh
    set -e
    cd frontend
    [ -d node_modules ] || pnpm install
    pnpm build
    cd ../backend/food-kiosk
    exec go run .

# Launch a local Mosquitto broker using the provided config
mqtt:
    # Optional: specify port via MQTT_PORT env var (default 1883)
    mosquitto -c backend/food-kiosk/broker/mosquitto.conf -p $MQTT_PORT

# Run the Svelte UI test-rig (dev server at http://localhost:5173)
ui:
    #!/bin/zsh
    cd frontend
    [ -d node_modules ] || pnpm install
    pnpm dev

[private]
flash role:
    #!/bin/zsh
    set -e
    ip=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1)
    echo "broker=$ip role={{role}} port={{port}}"
    cp {{ino}} /tmp/canteen-node.ino.bak
    trap 'mv /tmp/canteen-node.ino.bak {{ino}}' EXIT
    sed -i '' 's/^#define ROLE .*/#define ROLE {{role}}/' {{ino}}
    sed -i '' 's/^#define MQTT_HOST ".*"/#define MQTT_HOST "'"$ip"'"/' {{ino}}
    grep -E '^#define (ROLE|MQTT_HOST)' {{ino}}
    for pid in $(lsof -t {{port}} 2>/dev/null); do kill $pid 2>/dev/null; done
    sleep 1
    arduino-cli compile --fqbn {{fqbn}} --upload -p {{port}} canteen-node
