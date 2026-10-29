package server

import (
	"errors"
	"net/http"
	"strconv"

	"food-kiosk/backend"
	"food-kiosk/database"
)

type queueResponse struct {
	Head  *int64  `json:"head"`
	Depth int     `json:"depth"`
	Items []int64 `json:"items"`
}

// handleQueue reports the head and depth of one queue lane. It reads the same
// fulfillment service the MQTT client publishes from, so the HTTP view can
// never drift from what the stations display.
func (s *server) handleQueue(w http.ResponseWriter, r *http.Request) {
	lane := backend.Lane(r.URL.Query().Get("lane"))

	q, err := s.queues.Queue(r.Context(), lane)
	if err != nil {
		writeDBError(w, err)
		return
	}

	resp := queueResponse{Depth: len(q.Items), Items: make([]int64, len(q.Items))}
	for i, e := range q.Items {
		resp.Items[i] = e.ID
	}
	if len(q.Items) > 0 {
		head := q.Items[0].ID
		resp.Head = &head
	}

	writeJSON(w, http.StatusOK, resp)
}

type ackQueueRequest struct {
	ID int64 `json:"id"`
}

// handleAckQueue acknowledges the head of a lane, exactly like a station
// button press. Cancelling an order is a different operation and refunds it.
func (s *server) handleAckQueue(w http.ResponseWriter, r *http.Request) {
	lane := backend.Lane(r.PathValue("lane"))

	var req ackQueueRequest
	if err := decodeJSON(r, &req); err != nil || req.ID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}

	err := s.queues.Acknowledge(r.Context(), lane, strconv.FormatInt(req.ID, 10))
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, backend.ErrUnexpectedAck):
		writeError(w, http.StatusConflict, "order is not the head of that lane")
	case errors.Is(err, backend.ErrInvalidAck):
		writeError(w, http.StatusBadRequest, "invalid lane or token")
	case errors.Is(err, database.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid lane or token")
	default:
		writeDBError(w, err)
	}
}
