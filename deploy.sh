#!/bin/bash

SOURCE_FILE="check_mem.go"
BINARY_NAME="check_mem"

echo "Building $SOURCE_FILE..."
GOOS=linux GOARCH=amd64 go build -o "$BINARY_NAME" "$SOURCE_FILE"

if [ $? -ne 0 ]; then
  echo "Build failed. Aborting."
  exit 1
fi

echo "Enter target servers (format: user@host). Press Enter on an empty line to finish."

TARGETS=()
while true; do
  read -p "Target (user@host): " TARGET
  [ -z "$TARGET" ] && break
  TARGETS+=("$TARGET")
done

for HOST in "${TARGETS[@]}"; do
  echo "Deploying to $HOST..."

  scp "$BINARY_NAME" "$HOST:/tmp/$BINARY_NAME"
  if [ $? -ne 0 ]; then
    echo "Failed to copy file to $HOST. Skipping."
    continue
  fi

  ssh "$HOST" "sudo mv /tmp/$BINARY_NAME /usr/lib/nagios/plugins/$BINARY_NAME && sudo chmod +x /usr/lib/nagios/plugins/$BINARY_NAME"
  if [ $? -ne 0 ]; then
    echo "Failed to move or set permissions on $HOST."
  else
    echo "Deployment to $HOST successful."
  fi
done

echo "Deployment process completed."
