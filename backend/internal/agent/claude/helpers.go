package claude

import "encoding/json"

// makeSet builds a map[string]struct{} from keys for O(1) lookup.
func makeSet(keys ...string) map[string]struct{} {
	s := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		s[k] = struct{}{}
	}
	return s
}

// collectUnknown returns entries from raw whose keys are not in known.
func collectUnknown(raw map[string]json.RawMessage, known map[string]struct{}) map[string]json.RawMessage {
	var extra map[string]json.RawMessage
	for k, v := range raw {
		if _, ok := known[k]; !ok {
			if extra == nil {
				extra = make(map[string]json.RawMessage)
			}
			extra[k] = v
		}
	}
	return extra
}
