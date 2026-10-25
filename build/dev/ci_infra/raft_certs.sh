#!/usr/bin/env bash
# Creates in raft-certs/ the certificates for mutual TLS between the raft nodes of the local pool:
# - one CA per USS, and one certificate per node signed by the CA of its USS
# - trusted-cas.crt, the bundle of the CAs of all USSs, trusted by every node
# - an untrusted CA and certificate, used by check_raft_mtls.sh
# Node certificates are valid for the node IP and for 127.0.0.1, where the raft ports are published.
# Usage: raft_certs.sh <number of USSs> <number of nodes per USS>

set -eo pipefail

NUM_USS=${1:?number of USSs is required}
NUM_NODES=${2:?number of nodes per USS is required}

cd "$(dirname "$0")"
rm -rf raft-certs
mkdir raft-certs
cd raft-certs

# new_ca <name>
new_ca() {
  openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -keyout "$1-ca.key" -out "$1-ca.crt" -days 30 \
    -subj "/CN=$1 CA" -addext basicConstraints=critical,CA:TRUE -addext keyUsage=critical,keyCertSign
}

# new_cert <name> <CA name> <IP>
new_cert() {
  openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -keyout "$1.key" -out "$1.csr" -subj "/CN=$1"
  printf 'subjectAltName=IP:%s,IP:127.0.0.1\nextendedKeyUsage=serverAuth,clientAuth\n' "$3" > "$1.ext"
  openssl x509 -req -in "$1.csr" -CA "$2-ca.crt" -CAkey "$2-ca.key" -CAcreateserial -days 30 -out "$1.crt" -extfile "$1.ext"
}

for ((i=1; i<=NUM_USS; i++)); do
  new_ca "uss$i"
  for ((j=1; j<=NUM_NODES; j++)); do
    new_cert "node$(( (i-1) * NUM_NODES + j ))" "uss$i" "172.27.$i.$j"
  done
done
cat uss*-ca.crt > trusted-cas.crt

new_ca untrusted
new_cert untrusted-node untrusted 127.0.0.1
