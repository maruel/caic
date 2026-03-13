// CI check-run evaluation and failure summary building for bot-driven CI workflows.
package bot

import (
	"fmt"
	"strings"

	"github.com/caic-xyz/caic/backend/internal/forge"
	"github.com/caic-xyz/caic/backend/internal/forge/forgecache"
)

// EvaluateCheckRuns inspects runs for a SHA and returns a forgecache.Result plus
// whether all checks have completed (done=true). Only call with len(runs)>0.
func EvaluateCheckRuns(owner, repo string, runs []forge.CheckRun) (forgecache.Result, bool) {
	checks := make([]forge.Check, len(runs))
	allDone := true
	anyFailed := false
	for i := range runs {
		checks[i] = forge.CheckFromRun(owner, repo, &runs[i])
		if runs[i].Status != forge.CheckRunStatusCompleted {
			allDone = false
		} else if runs[i].Conclusion.IsFailed() {
			anyFailed = true
		}
	}
	if !allDone {
		return forgecache.Result{Checks: checks}, false
	}
	status := forge.CIStatusSuccess
	if anyFailed {
		status = forge.CIStatusFailure
	}
	return forgecache.Result{Status: status, Checks: checks}, true
}

// InterimCIStatus returns the CI status and checks to display while checks are
// still running. Returns CIStatusFailure as soon as any completed check has a
// failing conclusion, otherwise CIStatusPending.
func InterimCIStatus(runs []forge.CheckRun, checks []forge.Check) (forge.CIStatus, []forge.Check) {
	status := forge.CIStatusPending
	for i := range runs {
		if runs[i].Status == forge.CheckRunStatusCompleted && runs[i].Conclusion.IsFailed() {
			status = forge.CIStatusFailure
			break
		}
	}
	return status, checks
}

// FailureSummary builds the agent-facing text summary for a CI failure result,
// listing each failing check with its conclusion and job URL where available.
func FailureSummary(f forge.Forge, result forgecache.Result) string {
	var sb strings.Builder
	numFailed := 0
	for i := range result.Checks {
		if result.Checks[i].Conclusion.IsFailed() {
			numFailed++
		}
	}
	fmt.Fprintf(&sb, "%s CI: %d check(s) failed:\n", f.Name(), numFailed)
	for i := range result.Checks {
		c := &result.Checks[i]
		if !c.Conclusion.IsFailed() {
			continue
		}
		if jobURL := f.CIJobURL(c.Owner, c.Repo, c.RunID, c.JobID); jobURL != "" {
			fmt.Fprintf(&sb, "- %s (%s): %s\n", c.Name, c.Conclusion, jobURL)
		} else {
			fmt.Fprintf(&sb, "- %s (%s)\n", c.Name, c.Conclusion)
		}
	}
	sb.WriteString("\nPlease fix the failures above.")
	return strings.TrimRight(sb.String(), "\n")
}
