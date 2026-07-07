package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aegis-imaging/aegis/api/events"
)

// StudyEvents streams real-time study status changes as Server-Sent Events.
//
// GET /api/studies/events
//
// Optional query param:
//
//	project_id — filter events to a single project
//
// Event format:
//
//	data: {"type":"study.status_changed","study_id":"...","project_id":"...","status":"...","updated_at":"..."}\n\n
//
// The connection is kept open until the client disconnects or the server
// shuts down. A keepalive comment is sent every 30 seconds.
func (s *Server) StudyEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// SSE: stream indefinitely. Without this, the http.Server's WriteTimeout
	// (5 min) would kill the connection mid-stream.
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	projectFilter := r.URL.Query().Get("project_id")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	w.WriteHeader(http.StatusOK)

	// Send an initial connected comment so the client knows the stream is live.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch, unsub := s.bus.Subscribe()
	defer unsub()

	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected.
			return

		case ev := <-ch:
			if projectFilter != "" && ev.ProjectID != projectFilter {
				continue
			}
			data, err := json.Marshal(ev)
			if err != nil {
				log.Printf("events: marshal event: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-keepalive.C:
			// Send a comment to keep the connection alive through proxies.
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// publishStudyEvent is a helper called by write handlers after any study
// status change. It is a no-op when the bus has no subscribers.
func (s *Server) publishStudyEvent(evType, studyID, projectID, status string) {
	s.bus.Publish(events.StudyEvent{
		Type:      evType,
		StudyID:   studyID,
		ProjectID: projectID,
		Status:    status,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
