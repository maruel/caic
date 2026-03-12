// CI check-run evaluation and failure summary building for bot-driven CI workflows.
package bot

import (
	"fmt"
	"strings"

	"github.com/caic-xyz/caic/backend/internal/cicache"
	"github.com/caic-xyz/caic/backend/internal/forge"
)

// EvaluateCheckRuns inspects runs for a SHA and returns a cicache.Result plus
// whether all checks have completed (done=true). Only call with len(runs)>0.
func EvaluateCheckRuns(owner, repo string, runs []forge.CheckRun) (cicache.Result, bool) {
	for _, r := range runs {
		if r.Status != forge.CheckRunStatusCompleted {
			return cicache.Result{}, false
		}
	}
	checks := make([]cicache.ForgeCheck, 0, len(runs))
	anyFailed := false
	for _, r := range runs {
		checks = append(checks, cicache.ForgeCheck{
			Name:       r.Name,
			Owner:      owner,
			Repo:       repo,
			RunID:      r.RunID,
			JobID:      r.JobID,
			Conclusion: cicache.CheckConclusion(r.Conclusion),
		})
		if r.Conclusion != forge.CheckRunConclusionSuccess &&
			r.Conclusion != forge.CheckRunConclusionNeutral &&
			r.Conclusion != forge.CheckRunConclusionSkipped {
			anyFailed = true
		}
	}
	status := cicache.StatusSuccess
	if anyFailed {
		status = cicache.StatusFailure
	}
	return cicache.Result{Status: status, Checks: checks}, true
}

// FailureSummary builds the agent-facing text summary for a CI failure result,
// listing each failing check with its conclusion and job URL where available.
func FailureSummary(f forge.Forge, result cicache.Result) string {
	var sb strings.Builder
	numFailed := 0
	for _, c := range result.Checks {
		if c.Conclusion != cicache.CheckConclusionSuccess &&
			c.Conclusion != cicache.CheckConclusionNeutral &&
			c.Conclusion != cicache.CheckConclusionSkipped {
			numFailed++
		}
	}
	fmt.Fprintf(&sb, "%s CI: %d check(s) failed:\n", f.Name(), numFailed)
	for _, c := range result.Checks {
		if c.Conclusion != cicache.CheckConclusionSuccess &&
			c.Conclusion != cicache.CheckConclusionNeutral &&
			c.Conclusion != cicache.CheckConclusionSkipped {
			if jobURL := f.CIJobURL(c.Owner, c.Repo, c.RunID, c.JobID); jobURL != "" {
				fmt.Fprintf(&sb, "- %s (%s): %s\n", c.Name, c.Conclusion, jobURL)
			} else {
				fmt.Fprintf(&sb, "- %s (%s)\n", c.Name, c.Conclusion)
			}
		}
	}
	sb.WriteString("\nPlease fix the failures above.")
	return strings.TrimRight(sb.String(), "\n")
}
