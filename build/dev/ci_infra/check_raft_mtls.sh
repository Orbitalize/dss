#!/usr/bin/env bash
# Checks the raft ports (rid 95NN, scd 96NN, aux 97NN) of every node of the local pool started by run_locally.sh
# A client with a node certificate is accepted, and clients without a certificate, with an untrusted certificate or
# over plain HTTP are rejected.
# Usage: NUM_USS=<n> [NUM_NODES=<n>] check_raft_mtls.sh

set -eo pipefail

NUM_USS=${NUM_USS:-2}
NUM_NODES=${NUM_NODES:-1}

cd "$(dirname "$0")/raft-certs"

failures=0

# check <description> <expected HTTP status, 000 when the connection is rejected> <curl arguments>
check() {
  status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "${@:3}" || true)
  if [ "$status" = "$2" ]; then
    echo "ok: $1"
  else
    echo "FAILED: $1: got HTTP status $status, expected $2"
    failures=$((failures + 1))
  fi
}

for ((node=1; node<=NUM_USS*NUM_NODES; node++)); do
  for prefix in 95 96 97; do
    port=$prefix$(printf "%02d" "$node")
    url=https://127.0.0.1:$port/raft/probing
    check "port $port accepts a node certificate" 200 --cacert trusted-cas.crt --cert node1.crt --key node1.key "$url"
    check "port $port rejects no certificate" 000 --cacert trusted-cas.crt "$url"
    check "port $port rejects an untrusted certificate" 000 --cacert trusted-cas.crt --cert untrusted-node.crt --key untrusted-node.key "$url"
    check "port $port rejects plain HTTP" 000 "http://127.0.0.1:$port/raft/probing"
  done
done

[ "$failures" -eq 0 ]
