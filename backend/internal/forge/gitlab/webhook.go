// Payload types for GitLab webhook events.

package gitlab

// PipelineEvent is the payload for X-Gitlab-Event: Pipeline Hook.
type PipelineEvent struct {
	ObjectAttributes struct {
		SHA    string `json:"sha"`
		Status string `json:"status"` // "success", "failed", "canceled", "skipped"
		ID     int64  `json:"id"`
	} `json:"object_attributes"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"` // "group/repo"
	} `json:"project"`
}
