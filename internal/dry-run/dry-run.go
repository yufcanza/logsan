package dryrun

import (
	"fmt"
	"logsan/internal/config"
	processor "logsan/internal/proc"
	"logsan/internal/report"
	"logsan/internal/san"
	"os"
	"path/filepath"
)

func Run(inDir, configPath, reportPath string) error {

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

	for _, fileName := range files {
		inPath := filepath.Join(inDir, fileName)
		result, err := processor.ProcessFile(inPath, detectors, true)
		if err != nil {
			reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка обработки %s : %v", fileName, err))
			continue
		}
		filecount++
		linecount += result.Lines
		totalReplacement += result.Replacement
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
