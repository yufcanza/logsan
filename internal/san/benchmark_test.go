package san

import (
	"bufio"
	"fmt"
	"logsan/internal/config"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

func createSyntheticLog(b *testing.B, size int) string {
	tmpDir := os.TempDir()
	logPath := filepath.Join(tmpDir, "benchmark_log.log")

	file, err := os.Create(logPath)
	if err != nil {
		b.Fatalf("Ошибка создания файла: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, 256*1024*1024)

	templates := []string{
		"2026-07-10 10:15:22 INFO user=%s email=%s@%s ip=%d.%d.%d.%d token=%s\n",
		"2026-07-10 10:23:45 ERROR user=%s email=%s@%s ip=%d.%d.%d.%d token=%s\n",
		"2026-07-10 10:31:12 DEBUG user=%s email=%s@%s ip=%d.%d.%d.%d token=%s\n",
		"2026-07-10 10:42:08 WARN user=%s email=%s@%s ip=%d.%d.%d.%d token=%s\n",
	}

	users := []string{"ivanov", "petrova", "smirnov", "kozlov", "novikova", "morozov", "volkova"}
	domains := []string{"example.com", "company.ru", "mail.org", "domain.net", "test.io", "server.gov", "cloud.edu"}

	written := int(0)
	lineNum := 0

	for written < size {
		template := templates[lineNum%len(templates)]
		user := users[lineNum%len(users)]
		domain := domains[lineNum%len(domains)]
		ip1 := lineNum % 255
		ip2 := (lineNum + 10) % 255
		ip3 := (lineNum + 20) % 255
		ip4 := (lineNum + 30) % 255
		token := fmt.Sprintf("%x", lineNum)

		line := fmt.Sprintf(template, user, user, domain, ip1, ip2, ip3, ip4, token, user)
		lineNum++

		writer.WriteString(line)
		written += int(len(line))

		if lineNum%10000 == 0 {
			writer.Flush()
		}
	}
	writer.Flush()
	return logPath

}

func getDetectors() []config.Detector {
	return []config.Detector{
		{
			ID:                "email",
			Pattern:           `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`,
			ReplacementPrefix: "email",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`),
		},
		{
			ID:                "ipv4",
			Pattern:           `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			ReplacementPrefix: "ip",
			Enabled:           true,
			Regex:             regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		},
		{
			ID:                "token",
			Pattern:           `[a-fA-F0-9]{12}`,
			ReplacementPrefix: "token",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[a-fA-F0-9]{12}`),
		},
		{
			ID:                "url",
			Pattern:           `https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`,
			ReplacementPrefix: "url",
			Enabled:           true,
			Regex:             regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`),
		},
		{
			ID:                "windows_username",
			Pattern:           `[A-Z]:\\Users\\([^\\]+)`,
			ReplacementPrefix: "user",
			Enabled:           true,
			Regex:             regexp.MustCompile(`[A-Z]:\\Users\\([^\\]+)`),
		},
	}
}

func BenchmarkSmall(b *testing.B) {
	detectors := getDetectors()
	sanitizer := NewSanitizer(detectors)

	line := "user=ivanov email=ivanov@example.com ip=10.1.2.3 token=ab12cd34ef56 url=https://example.com/login"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sanitizer.ProcessLine(line)
	}
}

func Benchmark_GBLog(b *testing.B) {
	logPath := createSyntheticLog(b, 1024*1024*1024)
	defer os.Remove(logPath)

	detectors := getDetectors()
	sanitizer := NewSanitizer(detectors)

	inFile, err := os.Open(logPath)
	if err != nil {
		b.Fatalf("Ошибка открытия файла: %v", err)
	}
	defer inFile.Close()

	outFile, err := os.CreateTemp("", "benchmark_output.log")
	if err != nil {
		b.Fatalf("Ошибка создания временного файла: %v", err)
	}
	defer os.Remove(outFile.Name())
	defer outFile.Close()

	reader := bufio.NewReaderSize(inFile, 10*1024*1024)
	writer := bufio.NewWriterSize(outFile, 10*1024*1024)

	b.ResetTimer()
	start := time.Now()
	lineCount := 0
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			processed := sanitizer.ProcessLine(line)
			writer.WriteString(processed)
			lineCount++
		}
		if err != nil {
			if err.Error() != "EOF" {
				b.Fatalf("Ошибка чтения: %v", err)
			}
			break
		}
	}

	writer.Flush()

	elapsed := time.Since(start)
	fileInfo, _ := os.Stat(logPath)
	fileSize := fileInfo.Size()
	speed := float64(fileSize) / elapsed.Seconds() / 1024 / 1024 // MB/s

	b.ReportMetric(float64(lineCount), "lines")
	b.ReportMetric(speed, "MB/s")
	b.ReportMetric(float64(elapsed.Milliseconds()), "ms")

	b.Logf(" Файл: %d байт (%.2f MB)", fileSize, float64(fileSize)/1024/1024)
	b.Logf(" Строк: %d", lineCount)
	b.Logf(" Время: %.2f секунд", elapsed.Seconds())
	b.Logf(" Скорость: %.2f MB/s", speed)

}
