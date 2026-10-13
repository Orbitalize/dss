// Package client implements the interfaces of the admin package by calling the admin API of a DSS instance, for dssctl.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interuss/dss/pkg/admin"
	"github.com/interuss/stacktrace"
)

var (
	adminURL = flag.String("admin_url", "http://localhost:8078", "URL of the admin API of the DSS instance, matching its --admin_addr")
	timeout  = flag.Duration("admin_timeout", 90*time.Second, "Timeout of the requests to the admin API of the DSS instance")
)

// group is an admin.RaftGroup managed through the admin API of a DSS instance.
type group struct {
	httpClient *http.Client
	adminURL   string
	name       string
}

// InitRaftGroup returns the Raft group name of the DSS instance whose admin API is configured by the --admin_url flag.
func InitRaftGroup(name string) (admin.RaftGroup, error) {
	if _, err := url.ParseRequestURI(*adminURL); err != nil {
		return nil, stacktrace.Propagate(err, "invalid --admin_url %q", *adminURL)
	}

	return &group{
		httpClient: &http.Client{Timeout: *timeout},
		adminURL:   strings.TrimSuffix(*adminURL, "/"),
		name:       name,
	}, nil
}

func (g *group) Members(ctx context.Context) (admin.Membership, error) {
	var status admin.Membership
	err := g.do(ctx, http.MethodGet, admin.MembersPath(g.name), nil, &status)
	return status, err
}

func (g *group) AddNode(ctx context.Context, nodeID uint64, peerURL string) error {
	return g.do(ctx, http.MethodPost, admin.MembersPath(g.name), admin.AddNodeRequest{NodeID: nodeID, URL: peerURL}, nil)
}

func (g *group) RemoveNode(ctx context.Context, nodeID uint64) error {
	return g.do(ctx, http.MethodDelete, admin.MemberPath(g.name, nodeID), nil, nil)
}

func (g *group) do(ctx context.Context, method, path string, reqBody, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		buf, err := json.Marshal(reqBody)
		if err != nil {
			return stacktrace.Propagate(err, "failed to encode request")
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, g.adminURL+path, body)
	if err != nil {
		return stacktrace.Propagate(err, "failed to create request")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return stacktrace.Propagate(err, "failed to send request to %s", g.adminURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var errResp admin.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil || errResp.Error == "" {
			return stacktrace.NewError("%s", resp.Status)
		}
		return stacktrace.NewError("%s: %s", resp.Status, errResp.Error)
	}

	if respBody == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
		return stacktrace.Propagate(err, "failed to decode response")
	}
	return nil
}
