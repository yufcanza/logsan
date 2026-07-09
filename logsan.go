package main

import (
	"flag"
	"fmt"
	dryrun "logsan/internal/dry-run"
	sanrun "logsan/internal/san-run"
	"os"
)

// var filecount = 0
// var linecount = 0

type Flags struct {
	inDir  string
	outDir string
	report string
	config string
}

func main() {
	var flags Flags

	if len(os.Args) < 2 {
		fmt.Println("Используйте sanitize или dryrun")
		os.Exit(1)
	}
	mode := os.Args[1]

	fmt.Printf("%+v\n", mode)
	fmt.Printf("%+v\n", flags)

	os.Args = append(os.Args[:1], os.Args[2:]...)

	flag.StringVar(&flags.inDir, "in", "./logs", "Вход")
	flag.StringVar(&flags.outDir, "out", "./clean-logs", "Выход")
	flag.StringVar(&flags.report, "report", "report.json", "Репорт")
	flag.StringVar(&flags.config, "config", "detectors.yaml", "Конфиг")

	flag.Parse()
	fmt.Printf("%v", flags)

	if mode != "sanitize" && mode != "dry-run" {
		fmt.Printf("Укажите sanitize или dry-run, сейчас: %s\n", mode)
		os.Exit(1)
	}

	switch mode {
	case "sanitize":
		if err := sanrun.Run(flags.inDir, flags.outDir, flags.config, flags.report); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}
	case "dry-run":
		if err := dryrun.Run(flags.inDir, flags.config, flags.report); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}

	}
}

// reportData := &report.Report{
// 	Detect: make(map[string]int),
// 	Errors: []string{},
// }

// detectors, err := config.LoadConfig(flags.config)
// if err != nil {
// 	reportData.Errors = append(reportData.Errors,
// 		fmt.Sprintf("Ошибка загрузки конфига: %v\n", err))
// 	os.Exit(1)
// }

// files, err := os.ReadDir(flags.inDir)
// if err != nil {
// 	reportData.Errors = append(reportData.Errors,
// 		fmt.Sprintf("Ошибка открытия каталога: %v\n", err))
// 	os.Exit(1)
// }

// for _, file := range files {

// 	inFilePath := fmt.Sprintf("%s/%s", flags.inDir, file.Name())
// 	infile, err := os.Open(inFilePath)
// 	if err != nil {
// 		reportData.Errors = append(reportData.Errors,
// 			fmt.Sprintf("Ошибка открытия файла: %v\n", err))
// 		os.Exit(1)
// 	}
// 	defer infile.Close()

// 	outFilePath := fmt.Sprintf("%s/clean_%s", flags.outDir, file.Name())
// 	outfile, err := os.Create(outFilePath)
// 	if err != nil {
// 		reportData.Errors = append(reportData.Errors,
// 			fmt.Sprintf("Ошибка создания выходного файла %v\n", err))
// 		os.Exit(1)
// 	}
// 	defer outfile.Close()

// 	reader := bufio.NewReader(infile)
// 	writer := bufio.NewWriter(outfile)
// 	defer writer.Flush() //все выходные данные пройдут буфер, для входных данных не требуется, обработка построчно

// 	for {
// 		line, err := reader.ReadString('\n')
// 		if len(line) > 0 {
// 			processed := san.ProcessLine(line, detectors)
// 			if mode == "sanitize" {
// 				writer.WriteString(processed)
// 			}
// 			if mode == "dry-run" {
// 				fmt.Printf("dry-run: %s,%v", file.Name(), processed)
// 			}
// 			linecount++
// 		}
// 		if err != nil {
// 			if err.Error() != "EOF" {
// 				reportData.Errors = append(reportData.Errors,
// 					fmt.Sprintf("Ошибка чтения файла: %v\n", err))
// 			}

// 			break
// 		}

// 	}
// 	filecount++
// }
// report.CreateReport(flags.report, reportData)

// if mode == "dry-run" {
// 	reader := bufio.NewReader(infile)
// 	linecount := 0
// 	replacements := 0
// 	for {
// 		line, err := reader.ReadString('\n')
// 		if len(line) > 0 {
// 			processed := san.ProcessLine(line, detectors)
// 			linecount++
// 			if processed != line {
// 				replacements++
// 			}
// 		}
// 		if err != nil {
// 			if err.Error() != "EOF" {
// 				reportData.Errors = append(reportData.Errors,
// 					fmt.Sprintf("Ошибка чтения файла: %v\n", err))
// 			}

// 			break
// 		}

// 	}

// 	infile.Close()
// 	filecount++

// 	reportData.Detect = san.GetStats()
// 	reportData.LineProc = linecount
// 	reportData.FileProc = filecount

// 	report.CreateReport(flags.report, reportData)
// 	fmt.Print(reportData)
// }

//}
