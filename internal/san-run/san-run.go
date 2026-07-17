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

func Run(inPath, outPath, configPath, reportPath, mappingIn, mappingOut string) error {
	info, err := os.Stat(inPath)
	if err != nil {
		return fmt.Errorf("путь %s не существует: %v", inPath, err)
	}

	if info.IsDir() {
		return RunDir(inPath, outPath, configPath, reportPath, mappingIn, mappingOut)
	}
	return RunFile(inPath, outPath, configPath, reportPath, mappingIn, mappingOut)
}

func RunDir(inDir, outDir, configPath, reportPath, mappingIn, mappingOut string) error {
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

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %v", outDir, err)
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

	errorsChan := make(chan string, len(files)) // канал для сбора ошибок

	for _, fileName := range files {
		wg.Add(1)
		fileName := fileName
		go func() {

			defer wg.Done()
			inPath := filepath.Join(inDir, fileName)
			outPath := filepath.Join(outDir, "clean_"+fileName)
			outFile, err := os.Create(outPath)
			if err != nil {
				errorsChan <- fmt.Sprintf("Ошибка создания %s: %v", outPath, err)
				return
			}
			defer func() {
				if err := outFile.Close(); err != nil {
					errorsChan <- fmt.Sprintf("Ошибка закрытия %s: %v", outPath, err)
				}
			}()

			writer := bufio.NewWriterSize(outFile, 5*1024*1024)
			lines, err := processor.ProcessFileToWrite(inPath, writer, sanitizer)
			if err != nil {
				errorsChan <- fmt.Sprintf("Ошибка обработки %s: %v", fileName, err)
				return
			}
			if err := writer.Flush(); err != nil {
				errorsChan <- fmt.Sprintf("Ошибка сброса буфера %s: %v", fileName, err)
				return
			}
			//outFile.Close()
			mu.Lock()
			filecount++
			linecount += lines
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

	examples := sanitizer.GetReplacementExamples()
	for _, ex := range examples {
		reportData.AddReplacementExample(ex.DetectorID, ex.OriginalKind, ex.Replacement, ex.Count)
	}
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка создания отчета: %v", err)
	}

	return nil
}

func RunFile(inFile, outFile, configPath, reportPath, mappingIn, mappingOut string) error {
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

	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %v", err)
	}

	reportData := &report.Report{
		Detect: make(map[string]int),
		Errors: []string{},
	}

	outFileHandle, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("Jшибка создания %s: %v", outFile, err)
	}
	defer func() {
		if closeErr := outFileHandle.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("Ошибка закрытия %s: %v", outFile, err)
		}
	}()

	writer := bufio.NewWriterSize(outFileHandle, 5*1024*1024)
	lineCount, err := processor.ProcessFileToWrite(inFile, writer, sanitizer)
	if err != nil {
		return fmt.Errorf("Ошибка обработки файла: %v", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("ошибка сброса буфера: %v", err)
	}

	if mappingOut != "" {
		if err := sanitizer.SaveMapping(mappingOut); err != nil {
			return fmt.Errorf("Ошибка сохранения словаря: %v", err)
		}
	}
	stats := sanitizer.GetStats()
	reportData.Detect = stats
	reportData.FileProc = 1
	reportData.LineProc = lineCount

	examples := sanitizer.GetReplacementExamples()
	for _, ex := range examples {
		reportData.AddReplacementExample(ex.DetectorID, ex.OriginalKind, ex.Replacement, ex.Count)
	}
	if err := report.CreateReport(reportPath, reportData); err != nil {
		return fmt.Errorf("Ошибка создания отчета: %v", err)
	}

	return nil

}
