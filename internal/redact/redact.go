// Package redact strips connection secrets from strings before they are logged or printed.
package redact

import "regexp"

var (
	reURLUser = regexp.MustCompile(`://([^/\s:@]+):([^@\s]+)@`)
	reKV      = regexp.MustCompile(`(?i)(password|pwd|secret)\s*=\s*[^;\s&"]+`)
)

// String hides userinfo passwords and password= key/value pairs.
func String(s string) string {
	s = reURLUser.ReplaceAllString(s, "://$1:<redacted>@")
	s = reKV.ReplaceAllString(s, "${1}=<redacted>")
	return s
}
