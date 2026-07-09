package san

import (
	"regexp"
	"strings"
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
	resetGlobals()
	sameInput := "ivanov@example.com and ivanov@example.com"
	sameWant := "email_1 and email_1"
	sameResult := ProcessLine(sameInput, detectors)

	if sameResult != sameWant {
		t.Errorf("ProcessLine()= %q, want %q", sameResult, sameWant)
	}
	if stats["email"] != 2 {
		t.Errorf("stats[email] = %d, want 2", stats["email"])
	}
	resetGlobals()
	differentInput := "ivanov@example.com and smirnov@example.com"
	differentWant := "email_1 and email_2"
	differentResult := ProcessLine(differentInput, detectors)

	if differentResult != differentWant {
		t.Errorf("ProcessLine()= %q, want %q", differentResult, differentWant)
	}
	if stats["email"] != 2 {
		t.Errorf("stats[email] = %d, want 2", stats["email"])
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

	input := "src=10.1.2.3"
	want := "src=ip_1"
	result := ProcessLine(input, detectors)

	if result != want {
		t.Errorf("ProcessLine()= %q, want %q", result, want)
	}
	if stats["ipv4"] != 1 {
		t.Errorf("stats[ip] = %d, want 1", stats["ipv4"])
	}
	resetGlobals()
	sameInput := "src=10.1.2.3 dst=10.1.2.3"
	sameWant := "src=ip_1 dst=ip_1"
	sameResult := ProcessLine(sameInput, detectors)

	if sameResult != sameWant {
		t.Errorf("ProcessLine()= %q, want %q", sameResult, sameWant)
	}
	if stats["ipv4"] != 2 {
		t.Errorf("stats[ip] = %d, want 2", stats["ipv4"])
	}
	resetGlobals()
	differentInput := "src=10.1.2.5 dst=192.168.1.15"
	differentWant := "src=ip_1 dst=ip_2"
	differentResult := ProcessLine(differentInput, detectors)

	if differentResult != differentWant {
		t.Errorf("ProcessLine()= %q, want %q", differentResult, differentWant)
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
	resetGlobals()

	sameInput := "open https://example.com/login and https://example.com/login"
	sameWant := "open url_1 and url_1"
	sameResult := ProcessLine(sameInput, detectors)

	if sameResult != sameWant {
		t.Errorf("ProcessLine()= %q, want %q", sameResult, sameWant)
	}
	if stats["url"] != 2 {
		t.Errorf("stats[url] = %d, want 2", stats["url"])
	}
	resetGlobals()
	differentInput := "open https://example.com/login and https://example.com/dowload"
	differentWant := "open url_1 and url_2"
	differentResult := ProcessLine(differentInput, detectors)

	if differentResult != differentWant {
		t.Errorf("ProcessLine()= %q, want %q", differentResult, differentWant)
	}
	if stats["url"] != 2 {
		t.Errorf("stats[url] = %d, want 2", stats["url"])
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

func TestLongString(t *testing.T) {
	detectors := []config.Detector{
		{
			ID:                "email",
			Pattern:           `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
			ReplacementPrefix: "email",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		},
		{
			ID:                "url",
			Pattern:           `https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`,
			ReplacementPrefix: "url",
			Enabled:           true,
			Regex:             regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`),
		},
		{
			ID:                "token",
			Pattern:           `[a-fA-F0-9]{12}`,
			ReplacementPrefix: "token",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[a-fA-F0-9]{12}`),
		},
	}

	resetGlobals()

	parts := make([]string, 0, 1000)
	for i := 0; i < 1000; i++ {
		email := "user" + string(rune('a'+i%26)) + string(rune('a'+(i+1)%26)) + "@example.com"
		url := "https://site" + string(rune('a'+i%26)) + ".com/path" + string(rune('a'+(i+1)%26))
		token := strings.Repeat("abcdef", 2) + string(rune('0'+i%10))
		parts = append(parts, email, url, token)
	}
	longLine := strings.Join(parts, " ")

	result := ProcessLine(longLine, detectors)
	if len(result) == 0 {
		t.Error("Результат пустой")
	}
	stats := GetStats()
	if stats["email"] != 1000 {
		t.Errorf("Детектор email: ожидается 1000 замен, получено %d", stats["email"])
	}
	if stats["url"] != 1000 {
		t.Errorf("Детектор url: ожидается 1000 замен, получено %d", stats["url"])
	}
	if stats["token"] != 1000 {
		t.Errorf("Детектор token: ожидается 1000 замен, получено %d", stats["token"])
	}
	if !strings.Contains(result, "email_") {
		t.Error("Маска email не найдена в результате")
	}
	if !strings.Contains(result, "url_") {
		t.Error("Маска url не найдена в результате")
	}
	if !strings.Contains(result, "token_") {
		t.Error("Маска token не найдена в результате")
	}
	emailMasks := 0
	urlMasks := 0
	tokenMasks := 0

	// Подсчитываем количество масок
	for i := 1; i <= 1000; i++ {
		if strings.Contains(result, "email_") {
			emailMasks++
		}
		if strings.Contains(result, "url_") {
			urlMasks++
		}
		if strings.Contains(result, "token_") {
			tokenMasks++
		}
	}

	if emailMasks != 1000 {
		t.Errorf("Найдено %d уникальных email-масок, ожидается 1000", emailMasks)
	}
	if urlMasks != 1000 {
		t.Errorf("Найдено %d уникальных url-масок, ожидается 1000", urlMasks)
	}
	if tokenMasks != 1000 {
		t.Errorf("Найдено %d уникальных token-масок, ожидается 1000", tokenMasks)
	}

}
