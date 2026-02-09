package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maruel/wmao/backend/internal/task"
)

func TestHandleTaskEventsNotFound(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/99/events", http.NoBody)
	req.SetPathValue("id", "99")
	w := httptest.NewRecorder()
	s.handleTaskEvents(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleTaskEventsInvalidID(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/abc/events", http.NoBody)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	s.handleTaskEvents(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTaskInputNotRunning(t *testing.T) {
	s := &Server{}
	s.tasks = append(s.tasks, &taskEntry{
		task: &task.Task{Prompt: "test"},
		done: make(chan struct{}),
	})

	body := strings.NewReader(`{"prompt":"hello"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/0/input", body)
	req.SetPathValue("id", "0")
	w := httptest.NewRecorder()
	s.handleTaskInput(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleTaskInputEmptyPrompt(t *testing.T) {
	s := &Server{}
	s.tasks = append(s.tasks, &taskEntry{
		task: &task.Task{Prompt: "test"},
		done: make(chan struct{}),
	})

	body := strings.NewReader(`{"prompt":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/0/input", body)
	req.SetPathValue("id", "0")
	w := httptest.NewRecorder()
	s.handleTaskInput(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleFinishNotWaiting(t *testing.T) {
	s := &Server{}
	s.tasks = append(s.tasks, &taskEntry{
		task: &task.Task{Prompt: "test", State: task.StatePending},
		done: make(chan struct{}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/0/finish", http.NoBody)
	req.SetPathValue("id", "0")
	w := httptest.NewRecorder()
	s.handleTaskFinish(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestHandleFinishWaiting(t *testing.T) {
	tk := &task.Task{Prompt: "test", State: task.StateWaiting}
	tk.InitDoneCh()
	s := &Server{}
	s.tasks = append(s.tasks, &taskEntry{
		task: tk,
		done: make(chan struct{}),
	})

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/0/finish", http.NoBody)
	req.SetPathValue("id", "0")
	w := httptest.NewRecorder()
	s.handleTaskFinish(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify doneCh is closed.
	select {
	case <-tk.Done():
	default:
		t.Error("doneCh not closed after finish")
	}
}

func TestHandleCreateTaskReturnsID(t *testing.T) {
	s := &Server{runner: &task.Runner{BaseBranch: "main"}}
	handler := s.handleCreateTask(t.Context())

	body := strings.NewReader(`{"prompt":"test task"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", body)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["id"]; !ok {
		t.Error("response missing 'id' field")
	}
}
