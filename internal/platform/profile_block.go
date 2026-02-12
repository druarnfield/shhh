package platform

import "strings"

// extractManagedBlock returns the content between the managed block markers,
// or an empty string if no complete managed block is found.
func extractManagedBlock(content string) string {
	start := strings.Index(content, ManagedBlockStart)
	if start == -1 {
		return ""
	}
	end := strings.Index(content, ManagedBlockEnd)
	if end == -1 || end <= start {
		return ""
	}

	block := content[start+len(ManagedBlockStart) : end]
	return strings.TrimSpace(block)
}

// replaceManagedBlock replaces the managed block section in profile with the
// given content. If no managed block exists, one is appended to the end.
func replaceManagedBlock(profile, block string) string {
	section := ManagedBlockStart + "\n" + block + "\n" + ManagedBlockEnd + "\n"

	start := strings.Index(profile, ManagedBlockStart)
	end := strings.Index(profile, ManagedBlockEnd)

	if start == -1 || end == -1 || end <= start {
		// No existing managed block — append.
		if profile != "" && !strings.HasSuffix(profile, "\n") {
			profile += "\n"
		}
		return profile + section
	}

	// Find the end of the end-marker line (consume trailing newline).
	endOfMarker := end + len(ManagedBlockEnd)
	if endOfMarker < len(profile) && profile[endOfMarker] == '\n' {
		endOfMarker++
	}

	return profile[:start] + section + profile[endOfMarker:]
}
