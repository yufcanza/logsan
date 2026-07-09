package sanrun

import (
	"bufio"
	"fmt"
	"logsan/internal/config"
	processor "logsan/internal/proc"
	"logsan/internal/report"
	"logsan/internal/san"
	"os"
	"path/filepath"
)

func Run(inDir, outDir, configPath, reportPath string) error {
	detectors, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("Ошибка загрузки детекторов: %v", err)
	}
	if _, err := os.Stat(inDir); os.IsNotExist(err) {
		return fmt.Errorf("Директория %v не существует", inDir)
	}

	files, err := processor.GetFiles(inDir)
	if err != nil {
		return err
	}
	reportData := &report.Report{
		Detect: make(map[string]int),
		Errors: []string{},
	}
	filecount := 0
	linecount := 0
	for _, fileName := range files {
		inPath := filepath.Join(inDir, fileName)
		outPath := filepath.Join(outDir, "clean_"+fileName)

		outFile, err := os.Create(outPath)
		if err != nil {
			reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка создания %s: %v", outPath, err))
			continue
		}
		writer := bufio.NewWriter(outFile)
		lines, err := processor.ProcessFileToWrite(inPath, writer, detectors)
		if err != nil {
			reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка обработки %s: %v", fileName, err))
			outFile.Close()
			continue
		}
		writer.Flush()
		outFile.Close()
		filecount++
		linecount += lines
	}

	stats := san.GetStats()
	reportData.Detect = stats
	reportData.FileProc = filecount
	reportData.LineProc = linecount
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка создания отчета: %v", err)
	}
	return nil
}
