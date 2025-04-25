package pkg

import "strings"

// ParseAgentID splits a combined agent:id string into agent and id components
func ParseAgentID(combined string) (string, string) {
	parts := strings.SplitN(combined, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
