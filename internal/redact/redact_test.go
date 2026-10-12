package redact

import "testing"

func TestStringHidesURLAndKeyword(t *testing.T) {
	in := "sqlserver://someuser:S3cret-value@dbhost:1433?database=BHIS password=S3cret-value"
	out := String(in)
	if out == in || contains(out, "S3cret-value") {
		t.Fatalf("not redacted: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
