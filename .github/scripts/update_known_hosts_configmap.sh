#!/bin/bash

providers=(
    "bitbucket.org"
    "github.com"
    "gitlab.com"
    "ssh.dev.azure.com"
    "vs-ssh.visualstudio.com"
)

dst=charts/fleet/templates/configmap_known_hosts.yaml
echo "apiVersion: v1" > "$dst"
echo "kind: ConfigMap" >> "$dst"
echo "metadata:" >> "$dst"
echo "  name: known-hosts" >> "$dst"
echo "data:" >> "$dst"
echo "  known_hosts: |" >> "$dst"

for prov in "${providers[@]}"; do
    ssh-keyscan "$prov" | grep "^$prov" | sort -b | sed 's/^/    /' >> "$dst"
done
