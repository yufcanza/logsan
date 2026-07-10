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

func Run(inDir, configPath, reportPath, mappingIn, mappingOut string) error {

	if mappingIn != "" {
		err := san.LoadMapping(mappingIn)
		if err != nil {
			return fmt.Errorf("Ошибка загрузки словаря: %v", err)
		}
	}
	detectors, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("Ошибка загрузки конфига: %v", configPath)
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

	for _, fileName := range files {
		wg.Add(1)
		fileName := fileName
		go func() {
			defer wg.Done()
			inPath := filepath.Join(inDir, fileName)
			result, err := processor.ProcessFile(inPath, detectors, true)
			if err != nil {
				reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка обработки %s : %v", fileName, err))
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

	if mappingOut != "" {
		if err := san.SaveMapping(mappingOut); err != nil {
			return fmt.Errorf("Ошибка сохранения словаря: %v", err)
		}
	}

	stats := san.GetStats()
	reportData.Detect = stats
	reportData.FileProc = filecount
	reportData.LineProc = linecount
	reportData.ReplaceTotal = totalReplacement

	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка сохранения отчета: %v", err)
	}
	return nil
	// fmt.Print("Dry-run\n")
}
