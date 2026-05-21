package params

import (
	"flag"
	"net/url"
	"strconv"
	"strings"

	"github.com/interuss/stacktrace"
)

type (
	// ConnectParameters bundles up parameters used for connecting nodes in a raftstore cluster.
	ConnectParameters struct {
		ID    uint64
		Peers string
		TLS   string
	}

	// TLSCertificates bundles up TLS certificates parsed from the TLS field of ConnectParameters.
	TLSCertificates struct {
		CAFile   string
		CertFile string
		KeyFile  string
	}
)

// PeerMap parses the Peers string into a map of node ID to peer URL.
func (c ConnectParameters) PeerMap() (map[uint64]*url.URL, error) {
	peers := make(map[uint64]*url.URL)

	for entry := range strings.SplitSeq(c.Peers, ",") {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			return nil, stacktrace.NewError("invalid peer entry %s: must be in format nodeID=peerURL", entry)
		}

		id, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return nil, stacktrace.Propagate(err, "invalid peer ID %s", parts[0])
		}

		if id == 0 {
			return nil, stacktrace.NewError("invalid peer ID 0: peer IDs must be greater than 0")
		}

		if _, exists := peers[id]; exists {
			return nil, stacktrace.NewError("duplicate peer ID %d", id)
		}

		peerURL, err := url.Parse(parts[1])
		if err != nil {
			return nil, stacktrace.Propagate(err, "invalid peer URL %s", parts[1])
		}

		peers[id] = peerURL
	}

	return peers, nil
}

// TLSCertificates parses the TLS string into a TLSCertificates struct.
func (c ConnectParameters) TLSCertificates() (TLSCertificates, error) {
	tlsCerts := TLSCertificates{}

	if c.TLS == "" {
		return TLSCertificates{}, stacktrace.NewError("TLS configuration is empty")
	}

	for _, part := range strings.Split(c.TLS, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return TLSCertificates{}, stacktrace.NewError("invalid TLS parameter '%s': must be in format key=value", part)
		}

		key, value := kv[0], kv[1]
		if value == "" {
			return TLSCertificates{}, stacktrace.NewError("invalid TLS parameter '%s': value cannot be empty", key)
		}

		switch key {
		case "ca":
			tlsCerts.CAFile = value
		case "cert":
			tlsCerts.CertFile = value
		case "key":
			tlsCerts.KeyFile = value
		default:
			return TLSCertificates{}, stacktrace.NewError("invalid TLS parameter '%s': must be one of ca, cert, or key", key)
		}
	}

	if tlsCerts.CAFile == "" {
		return TLSCertificates{}, stacktrace.NewError("missing required TLS parameter: ca")
	}
	if tlsCerts.CertFile == "" {
		return TLSCertificates{}, stacktrace.NewError("missing required TLS parameter: cert")
	}
	if tlsCerts.KeyFile == "" {
		return TLSCertificates{}, stacktrace.NewError("missing required TLS parameter: key")
	}

	return tlsCerts, nil
}

var (
	connectParameters ConnectParameters
)

func init() {
	flag.Uint64Var(&connectParameters.ID, "raft_node_id", 0, "raft node ID for this instance (must be non-zero and unique within the cluster)")
	flag.StringVar(&connectParameters.Peers, "raft_peers", "", `comma-separated "nodeID=peerURL" pairs for all cluster members, including the current node, e.g. "1=http://node1:9021,2=http://node2:9021,3=http://node3:9021"`)
	flag.StringVar(&connectParameters.TLS, "raft_tls", "", `TLS certificates, format: ca=/path/to/ca.crt,cert=/path/to/node.crt,key=/path/to/node.key"`)
}

// GetConnectParameters returns a ConnectParameters instance that gets populated from well-known CLI flags.
func GetConnectParameters() ConnectParameters {
	return connectParameters
}
