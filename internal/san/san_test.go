package san

import (
	"regexp"
	"testing"

	"logsan/internal/config"
)

func resetGlobals() {
	counter = make(map[string]int)
	mapping = make(map[string]string)
	stats = make(map[string]int)
}

func TestEmailDetector(t *testing.T) {

	detectors := []config.Detector{
		{
			ID:                "email",
			Pattern:           `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
			ReplacementPrefix: "email",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		},
	}

	resetGlobals()
	//тест корректной замены
	input := "user email=ivanov@example.com"
	want := "user email=email_1"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["email"] != 1 {
		t.Errorf("stats[email] = %d, want 1", stats["email"])
	}

}

func TestIp(t *testing.T) {
	detectors := []config.Detector{
		{
			ID:                "ipv4",
			Pattern:           `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			ReplacementPrefix: "ip",
			Enabled:           true,
			Regex:             regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		},
	}

	resetGlobals()
	input := "src=10.1.2.3 dst=10.1.2.3"
	want := "src=ip_1 dst=ip_1"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["ipv4"] != 2 {
		t.Errorf("stats[ip] = %d, want 2", stats["ipv4"])
	}

}
func TestUrl(t *testing.T) {
	detectors := []config.Detector{
		{
			ID:                "url",
			Pattern:           `https?://[^\s<>"{}|\\^]+`,
			ReplacementPrefix: "url",
			Enabled:           true,
			Regex:             regexp.MustCompile(`https?://[^\s<>"{}|\\^]+`),
		},
	}

	resetGlobals()
	input := "open https://example.com/login"
	want := "open url_1"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["url"] != 1 {
		t.Errorf("stats[url] = %d, want 1", stats["url"])
	}

}

func TestNoMatch(t *testing.T) {
	detectors := []config.Detector{
		{
			ID:                "email",
			Pattern:           `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
			ReplacementPrefix: "email",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		},
	}

	resetGlobals()
	input := "normal log line without secrets"
	want := "normal log line without secrets"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["url"] != 0 {
		t.Errorf("stats should be empty, got %v", stats)
	}

}
func TestDisabledDetector(t *testing.T) {

	detectors := []config.Detector{
		{
			ID:                "email",
			Pattern:           `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
			ReplacementPrefix: "email",
			Enabled:           false,
			Regex:             regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		},
	}

	resetGlobals()
	//тест корректной замены
	input := "user email=ivanov@example.com"
	want := "user email=ivanov@example.com"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["email"] != 0 {
		t.Errorf("stats[email] = %d, want 1", stats["email"])
	}

}
