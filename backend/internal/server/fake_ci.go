// Fake CI simulation for e2e tests: sets a PR and cycles checks to success.
//go:build e2e

package server

import (
	"time"

	"github.com/caic-xyz/caic/backend/internal/forge"
	"github.com/caic-xyz/caic/backend/internal/task"
)

// maybeFakeCI polls until the task reaches a non-running state, then sets a
// fake PR and transitions CI from pending to success with progressive check
// completion.
func (s *Server) maybeFakeCI(t *task.Task) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
		switch t.GetState() {
		case task.StateWaiting, task.StateAsking, task.StateHasPlan:
			goto ready
		case task.StatePurged, task.StateFailed:
			return
		default:
		}
	}
ready:
	t.SetPR("fake-owner", "fake-repo", 1)
	checks := []task.CICheck{
		{Name: "build", Owner: "fake-owner", Repo: "fake-repo", RunID: 1, JobID: 1},
		{Name: "test", Owner: "fake-owner", Repo: "fake-repo", RunID: 1, JobID: 2},
		{Name: "lint", Owner: "fake-owner", Repo: "fake-repo", RunID: 1, JobID: 3},
	}
	t.SetCIStatus(task.CIStatusPending, checks)
	for i := range checks {
		select {
		case <-time.After(time.Second):
		case <-s.ctx.Done():
			return
		}
		checks[i].Conclusion = forge.CheckRunConclusionSuccess
		t.SetCIStatus(task.CIStatusPending, checks)
	}
	t.SetCIStatus(task.CIStatusSuccess, checks)
}
