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
	"sync"
)

func Run(inDir, outDir, configPath, reportPath, mappingIn, mappingOut string) error {
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
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, fileName := range files {
		wg.Add(1)
		fileName := fileName
		go func() {

			defer wg.Done()
			inPath := filepath.Join(inDir, fileName)
			outPath := filepath.Join(outDir, "clean_"+fileName)
			outFile, err := os.Create(outPath)
			if err != nil {
				reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка создания %s: %v", outPath, err))
				return
			}
			defer outFile.Close()

			writer := bufio.NewWriterSize(outFile, 256*1024*1024)
			lines, err := processor.ProcessFileToWrite(inPath, writer, sanitizer)
			if err != nil {
				mu.Lock()
				reportData.Errors = append(reportData.Errors, fmt.Sprintf("Ошибка обработки %s: %v", fileName, err))
				mu.Unlock()
				outFile.Close()
				return
			}
			writer.Flush()
			//outFile.Close()
			mu.Lock()
			filecount++
			linecount += lines
			mu.Unlock()
		}()
	}
	wg.Wait()

	if mappingOut != "" {
		if err := sanitizer.SaveMapping(mappingOut); err != nil {
			return fmt.Errorf("Ошибка сохранения словаря: %v", err)
		}
	}
	stats := sanitizer.GetStats()
	reportData.Detect = stats
	reportData.FileProc = filecount
	reportData.LineProc = linecount

	examples := sanitizer.GetReplacementExamples()
	for _, ex := range examples {
		reportData.AddReplacementExample(ex.DetectorID, ex.OriginalKind, ex.Replacement, ex.Count)
	}
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка создания отчета: %v", err)
	}

	return nil
}
