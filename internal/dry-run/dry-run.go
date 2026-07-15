package dryrun

import (
	"fmt"
	"logsan/internal/config"
	processor "logsan/internal/proc"
	"logsan/internal/report"
	"logsan/internal/san"
	"os"
	"path/filepath"
	"sync"
)

func Run(inPath, configPath, reportPath, mappingIn, mappingOut string) error {
	info, err := os.Stat(inPath)
	if err != nil {
		return fmt.Errorf("путь %s не существует: %v", inPath, err)
	}

	if info.IsDir() {
		return RunDir(inPath, configPath, reportPath, mappingIn, mappingOut)
	}
	return RunFile(inPath, configPath, reportPath, mappingIn, mappingOut)
}

func RunDir(inDir, configPath, reportPath, mappingIn, mappingOut string) error {
	detectors, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("Ошибка загрузки конфига: %v", configPath)
	}
	sanitizer := san.NewSanitizer(detectors)
	if mappingIn != "" {
		err := sanitizer.LoadMapping(mappingIn)
		if err != nil {
			return fmt.Errorf("Ошибка загрузки словаря: %v", err)
		}
	}

	if _, err := os.Stat(inDir); os.IsNotExist(err) {
		return fmt.Errorf("Директория %s не существует", inDir)
	}
	files, err := processor.GetFiles(inDir)
	if err != nil {
		return err
	}

	reportData := &report.Report{
		Detect: make(map[string]int),
		Errors: []string{},
		//DryRun: true,
	}

	filecount := 0
	linecount := 0
	totalReplacement := 0
	var wg sync.WaitGroup
	var mu sync.Mutex

	errorsChan := make(chan string, len(files)) // канал для сбора ошибок

	for _, fileName := range files {
		wg.Add(1)
		fileName := fileName
		go func() {
			defer wg.Done()
			inPath := filepath.Join(inDir, fileName)
			result, err := processor.ProcessFile(inPath, sanitizer, true)
			if err != nil {
				errorsChan <- fmt.Sprintf("Ошибка обработки %s: %v", fileName, err)
				return
			}

			mu.Lock()
			filecount++
			linecount += result.Lines
			totalReplacement += result.Replacement
			mu.Unlock()

		}()

	}
	wg.Wait()
	close(errorsChan)

	for errMsg := range errorsChan {
		reportData.Errors = append(reportData.Errors, errMsg)
	}

	if mappingOut != "" {
		if err := sanitizer.SaveMapping(mappingOut); err != nil {
			return fmt.Errorf("Ошибка сохранения словаря: %v", err)
		}
	}

	stats := sanitizer.GetStats()
	reportData.Detect = stats
	reportData.FileProc = filecount
	reportData.LineProc = linecount
	reportData.ReplaceTotal = totalReplacement

	examples := sanitizer.GetReplacementExamples()
	for _, ex := range examples {
		reportData.AddReplacementExample(ex.DetectorID, ex.OriginalKind, ex.Replacement, ex.Count)
	}
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка сохранения отчета: %v", err)
	}
	return nil
	// fmt.Print("Dry-run\n")
}
func RunFile(inFile, configPath, reportPath, mappingIn, mappingOut string) error {
	detectors, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("Ошибка загрузки детекторов: %v", err)
	}

	sanitizer := san.NewSanitizer(detectors)

	if mappingIn != "" {
		err := sanitizer.LoadMapping(mappingIn)
		if err != nil {
			return fmt.Errorf("Ошибка загрузки словаря: %v", err)
		}
	}

	if _, err := os.Stat(inFile); os.IsNotExist(err) {
		return fmt.Errorf("Файл %v не существует", inFile)
	}

	reportData := &report.Report{
		Detect: make(map[string]int),
		Errors: []string{},
	}

	result, err := processor.ProcessFile(inFile, sanitizer, true)
	if err != nil {
		return fmt.Errorf("ошибка обработки файла: %v", err)
	}

	if mappingOut != "" {
		if err := sanitizer.SaveMapping(mappingOut); err != nil {
			return fmt.Errorf("Ошибка сохранения словаря: %v", err)
		}
	}
	stats := sanitizer.GetStats()
	reportData.Detect = stats
	reportData.FileProc = 1
	reportData.LineProc = result.Lines
	reportData.ReplaceTotal = result.Replacement

	examples := sanitizer.GetReplacementExamples()
	for _, ex := range examples {
		reportData.AddReplacementExample(ex.DetectorID, ex.OriginalKind, ex.Replacement, ex.Count)
	}
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка создания отчета: %v", err)
	}

	return nil

}
