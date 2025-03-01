package handler

import (
	"encoding/json"
	"fmt"
	"github.com/mdfriday/hugoverse/internal/domain/content/valueobject"
	"github.com/mdfriday/hugoverse/internal/domain/host"
	"net/http"
	"time"

	"github.com/mdfriday/hugoverse/internal/domain/host/factory"
	"github.com/mdfriday/hugoverse/pkg/loggers"
)

// SSEWriter wraps a http.ResponseWriter to provide SSE functionality
type SSEWriter struct {
	w http.ResponseWriter
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(w http.ResponseWriter) *SSEWriter {
	writer := &SSEWriter{w: w}
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	return writer
}

// SendEvent sends an SSE event
func (sw *SSEWriter) SendEvent(event *ProgressEvent) {
	fmt.Fprintf(sw.w, "data: %s\n\n", event.ToJSON())
	sw.w.(http.Flusher).Flush()
}

// DeploymentInfo stores information about an ongoing deployment
type DeploymentInfo struct {
	Username   string
	Password   string
	Host       string
	Port       string
	RemotePath string
	Status     string
}

// Progress represents the current progress of an operation
type Progress struct {
	Current int64  `json:"current"`
	Total   int64  `json:"total"`
	Status  string `json:"status"`
}

// ProgressEvent represents a SSE event for progress updates
type ProgressEvent struct {
	Event string    `json:"event"`
	Data  *Progress `json:"data"`
}

// ToJSON converts the progress event to JSON string
func (pe *ProgressEvent) ToJSON() string {
	data, _ := json.Marshal(pe)
	return string(data)
}

func (s *Handler) deployPrivateHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")

	loggers.SetGlobalFields(s.newLogFields("deploy private"))

	// Get deployment parameters
	hostName := req.FormValue("host_name")
	if hostName != "Private" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("this is private handler")).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	username := req.FormValue("username")
	password := req.FormValue("password")
	address := req.FormValue("host")
	port := req.FormValue("port")
	remotePath := req.FormValue("remote_path")

	if port == "" {
		port = "22"
	}

	// Validate required fields
	if username == "" || password == "" || address == "" || remotePath == "" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("missing required fields")).
			Logf("username: %v, password: %v, address: %v, remote_path: %v",
				username != "", password != "", address != "", remotePath != "")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Generate a unique session ID for this deployment
	sessionID := fmt.Sprintf("deploy-%d", time.Now().UnixNano())

	// Store deployment info in memory
	s.deployments.Store(sessionID, &DeploymentInfo{
		Username:   username,
		Password:   password,
		Host:       address,
		Port:       port,
		RemotePath: remotePath,
		Status:     "pending",
	})

	// Return the session ID to the client
	response := map[string]string{
		"session_id": sessionID,
		"status":     "initialized",
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("error marshaling response")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	j, err := s.res.FmtJSON(jsonResponse)
	if err != nil {
		s.log.Errorf("Error formatting JSON: %v", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
	s.res.Json(res, j)
}

// DeployProgressHandler handles SSE connection for deployment progress
func (s *Handler) DeployProgressHandler(res http.ResponseWriter, req *http.Request) {
	q := req.URL.Query()
	id := q.Get("id")
	t := q.Get("type")

	loggers.SetGlobalFields(s.newLogFields("deploy progress"))

	// Get session ID from query parameters
	sessionID := q.Get("session_id")
	if sessionID == "" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("missing session_id")).
			Logf("session_id not provided")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Retrieve deployment info
	deployInfoRaw, ok := s.deployments.Load(sessionID)
	if !ok {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("invalid session_id")).
			Logf("session_id: %s not found", sessionID)
		res.WriteHeader(http.StatusNotFound)
		return
	}

	deployInfo := deployInfoRaw.(*DeploymentInfo)
	if deployInfo.Status != "pending" {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("invalid deployment status")).
			Logf("status: %s", deployInfo.Status)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	sc, err := s.contentApp.GetContentObject(t, id)
	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(fmt.Errorf("error GetContentObject: %v", err)).
			Logf("t: %s, id: %s", t, id)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	var target string
	if site, ok := sc.(*valueobject.Site); ok {
		target = site.WorkingDir
	}

	scpHost, err := factory.NewPasswordScpHost(
		deployInfo.Username, deployInfo.Password,
		deployInfo.Host, deployInfo.Port,
		deployInfo.RemotePath)

	if err != nil {
		s.log.Error().
			WithFields(loggers.GetGlobalFields()).
			WithError(err).
			Logf("error creating SCP host")
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	sseWriter := NewSSEWriter(res)

	// Create a channel for progress updates
	progressChan := make(chan *Progress)
	defer close(progressChan)

	// Create a channel to detect client disconnection
	closeNotify := req.Context().Done()

	// Start a goroutine to send progress updates
	go func() {
		for {
			select {
			case progress := <-progressChan:
				event := &ProgressEvent{
					Event: "progress",
					Data:  progress,
				}
				sseWriter.SendEvent(event)
			case <-closeNotify:
				return
			}
		}
	}()

	scpHost.SetProgress(func(current, total int64) {
		progress := &Progress{
			Current: current,
			Total:   total,
			Status:  "uploading",
		}
		progressChan <- progress
	})

	// Start deployment in a goroutine
	go func() {
		defer func() {
			s.deployments.Delete(sessionID)
			close(progressChan)
		}()

		deployInfo.Status = "in_progress"
		result, err := scpHost.Deploy(target)
		if err != nil {
			s.log.Error().
				WithFields(loggers.GetGlobalFields()).
				WithError(err).
				Logf("Deploy failed for session: %s", sessionID)

			event := &ProgressEvent{
				Event: "error",
				Data: &Progress{
					Current: 0,
					Total:   0,
					Status:  "error",
				},
			}
			sseWriter.SendEvent(event)
			return
		}

		scpResult := result.(host.Result)
		event := &ProgressEvent{
			Event: "complete",
			Data: &Progress{
				Status:  "complete",
				Current: scpResult.GetSize(),
				Total:   scpResult.GetSize(),
			},
		}
		sseWriter.SendEvent(event)
		deployInfo.Status = "completed"
	}()

	// Keep the connection alive until client disconnects
	<-closeNotify
}
