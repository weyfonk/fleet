#!/bin/bash

providers=(
    'github.com'
    'gitlab.com'
    'bitbucket.org'
    'ssh.dev.azure.com'
    'vs-ssh.visualstudio.com'
)

for prov in "${providers[@]}"; do
    echo "Provider: $prov"

    grep "$prov" charts/fleet/templates/configmap_known_hosts.yaml | sort -b > current_entries
    ssh-keyscan "$prov" | grep "^$prov" | sort -b  > new_entries

    diff -w current_entries new_entries
done

rm current_entries new_entries
