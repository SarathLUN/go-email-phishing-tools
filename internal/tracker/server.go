package tracker

import (
	"fmt"
	"github.com/SarathLUN/go-email-phishing-tools/internal/config" // Adjust path
	"github.com/SarathLUN/go-email-phishing-tools/internal/store"  // Adjust path
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TrackerServer holds dependencies for the tracking HTTP server.
type TrackerServer struct {
	Config     *config.Config
	TargetRepo store.TargetRepository
	Router     *http.ServeMux
}

// NewTrackerServer creates and initializes a new tracker server.
func NewTrackerServer(cfg *config.Config, repo store.TargetRepository) *TrackerServer {
	s := &TrackerServer{
		Config:     cfg,
		TargetRepo: repo,
		Router:     http.NewServeMux(),
	}
	s.routes()
	return s
}

// routes sets up the HTTP routes for the tracker.
func (s *TrackerServer) routes() {
	s.Router.HandleFunc("GET /feedback", s.handleTrackClick()) // Use new Go 1.22+ pattern
	// If not using Go 1.22+ for ServeMux patterns:
	// s.Router.HandleFunc("/track", s.handleTrackClick())
}

// ServeHTTP makes TrackerServer an http.Handler
func (s *TrackerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(w, r)
}

// handleTrackClick returns an http.HandlerFunc that processes click tracking requests.
func (s *TrackerServer) handleTrackClick() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Get UUID from query parameter
		uuidStr := r.URL.Query().Get("id")
		if uuidStr == "" {
			log.Println("Tracker: Received request with missing 'id' query parameter.")
			http.Error(w, "Bad Request: Missing 'id' parameter", http.StatusBadRequest)
			return
		}

		// 2. Validate UUID format
		targetUUID, err := uuid.Parse(uuidStr)
		if err != nil {
			log.Printf("Tracker: Received invalid UUID format: %s. Error: %v", uuidStr, err)
			http.Error(w, "Bad Request: Invalid 'id' parameter format", http.StatusBadRequest)
			return
		}

		// 3. Record the click
		clickedTime := time.Now()
		updated, err := s.TargetRepo.MarkAsClicked(r.Context(), targetUUID, clickedTime)
		if err != nil {
			// This is an internal server error (e.g., DB down)
			log.Printf("Tracker: Error marking target %s as clicked: %v", targetUUID, err)
			// Still redirect, but log the failure. Don't expose DB errors to client.
		} else {
			if updated {
				log.Printf("Tracker: Successfully recorded click for target UUID: %s at %v", targetUUID, clickedTime)
			} else {
				log.Printf("Tracker: Click received for target UUID: %s (already clicked or not found). No new update.", targetUUID)
			}
		}

		// 4. Redirect user
		// Use 302 Found for temporary redirect. Some prefer 307 for non-GET method changes, but 302 is common.
		log.Printf("Tracker: Redirecting user (UUID: %s) to %s", targetUUID, s.Config.RedirectURLAfterClick)
		http.Redirect(w, r, s.Config.RedirectURLAfterClick, http.StatusFound)
	}
}

// Start begins listening for HTTP requests.
func (s *TrackerServer) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", s.Config.TrackerHost, s.Config.TrackerPort)
	log.Printf("Tracker web service starting on %s", listenAddr)
	log.Printf("Redirecting clicks to: %s", s.Config.RedirectURLAfterClick)
	// For simple cases, http.ListenAndServe is fine.
	// For graceful shutdown, you'd use http.Server and its Shutdown method.
	server := &http.Server{
		Addr:         listenAddr,
		Handler:      s.Router, // Or s if TrackerServer implements ServeHTTP directly
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	return server.ListenAndServe()
}
