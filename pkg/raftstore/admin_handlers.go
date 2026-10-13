package raftstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/interuss/dss/pkg/admin"
	"github.com/interuss/dss/pkg/raftstore/consensus"
	"github.com/interuss/stacktrace"
	"go.uber.org/zap"
)

// requestTimeout bounds the time a membership change may take.
const requestTimeout = time.Minute

type adminHandler struct {
	logger *zap.Logger
	group  admin.RaftGroup
}

// RegisterAdminHandlers registers on mux the admin API of the Raft group, served under its name.
func RegisterAdminHandlers(mux *http.ServeMux, logger *zap.Logger, name string, group admin.RaftGroup) {
	h := &adminHandler{logger: logger.With(zap.String("raft_group", name)), group: group}

	mux.HandleFunc("GET "+admin.MembersPath(name), h.listMembers)
	mux.HandleFunc("POST "+admin.MembersPath(name), h.addNode)
	mux.HandleFunc("DELETE "+admin.MembersPath(name)+"/{nodeID}", h.removeNode)
}

func (h *adminHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	status, err := h.group.Members(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err)
		return
	}
	h.writeJSON(w, http.StatusOK, status)
}

func (h *adminHandler) addNode(w http.ResponseWriter, r *http.Request) {
	var req admin.AddNodeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, stacktrace.Propagate(err, "invalid request body"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if err := h.group.AddNode(ctx, req.NodeID, req.URL); err != nil {
		h.writeError(w, errorStatus(err), err)
		return
	}

	h.logger.Info("added node to raft group", zap.Uint64("nodeID", req.NodeID), zap.String("url", req.URL))
	w.WriteHeader(http.StatusNoContent)
}

func (h *adminHandler) removeNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := strconv.ParseUint(r.PathValue("nodeID"), 10, 64)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, stacktrace.Propagate(err, "invalid node ID %q", r.PathValue("nodeID")))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if err := h.group.RemoveNode(ctx, nodeID); err != nil {
		h.writeError(w, errorStatus(err), err)
		return
	}

	h.logger.Info("removed node from raft group", zap.Uint64("nodeID", nodeID))
	w.WriteHeader(http.StatusNoContent)
}

func errorStatus(err error) int {
	switch {
	case errors.Is(err, consensus.ErrConfChangeRejected):
		return http.StatusBadRequest
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func (h *adminHandler) writeError(w http.ResponseWriter, status int, err error) {
	h.logger.Warn("raft admin request failed", zap.Int("status", status), zap.Error(err))
	// %#s formats stacktrace errors on a single line, without the stack.
	h.writeJSON(w, status, admin.ErrorResponse{Error: fmt.Sprintf("%#s", err)})
}

func (h *adminHandler) writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.Error("failed to encode raft admin response", zap.Error(err))
	}
}
