package util

import (
	"fmt"
	"strings"
)

func EnsureDot(name string) string {
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}
	return name
}

func ParseRecordID(id string) (name, rrtype string, err error) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("recordID must be 'name:type', got %q", id)
	}
	name = EnsureDot(strings.TrimSpace(parts[0]))
	rrtype = strings.ToUpper(strings.TrimSpace(parts[1]))
	if name == "." || rrtype == "" {
		return "", "", fmt.Errorf("invalid recordID %q", id)
	}
	return name, rrtype, nil
}
