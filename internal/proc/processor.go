package processor

import (
	"bufio"
	"fmt"
	"logsan/internal/san"
	"os"
)

type ProcessFileResult struct {
	Lines       int
	Replacement int
}

func ProcessFile(inPath string, sanitizer *san.Sanitizer, dryrun bool) (*ProcessFileResult, error) {
	inFile, err := os.Open(inPath)
	if err != nil {
		return nil, fmt.Errorf("Не удалось открыть %s: %v", inPath, err)
	}
	defer func() {
		if err := inFile.Close(); err != nil {
			fmt.Printf("Ошибка закрытия %s: %v", inPath, err)
		}
	}()

	reader := bufio.NewReaderSize(inFile, 256*1024*1024)
	result := &ProcessFileResult{}

	for {
		line, err := reader.ReadString('\n')

		if len(line) > 0 {
			processed := sanitizer.ProcessLine(line)
			result.Lines++
			if processed != line {
				result.Replacement++
			}
			if !dryrun {

			}
		}
		if err != nil {
			if err.Error() != "EOF" {
				return result, fmt.Errorf("Ошибка чтения %s: %v", inPath, err)

			}
			break
		}

	}
	return result, nil
}

func ProcessFileToWrite(inPath string, writer *bufio.Writer, sanitizer *san.Sanitizer) (int, error) {
	inFile, err := os.Open(inPath)
	if err != nil {
		return 0, fmt.Errorf("Ошибка чтения директории %s: %v", inPath, err)
	}
	defer inFile.Close()

	reader := bufio.NewReaderSize(inFile, 256*1024*1024)
	lines := 0
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			processed := sanitizer.ProcessLine(line)
			if _, err := writer.WriteString(processed); err != nil {
				return lines, fmt.Errorf("Ошибка записи: %v", err)
			}
			lines++
		}
		if err != nil {
			if err.Error() != "EOF" {
				return lines, fmt.Errorf("Ошибка чтения %s : %v", inPath, err)
			}
			break
		}
	}
	if err := writer.Flush(); err != nil {
		return lines, fmt.Errorf("Ошибка сброса буфера: %v", err)
	}
	return lines, nil
}

func GetFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения директории %s: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	return files, nil
}
