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
	inmap  string
	outmap string
}

func main() {
	var flags Flags

	if len(os.Args) < 2 {
		fmt.Println("Используйте sanitize или dryrun")
		os.Exit(1)
	}
	mode := os.Args[1]

	os.Args = append(os.Args[:1], os.Args[2:]...)

	flag.StringVar(&flags.inDir, "in", "./logs", "Вход")
	flag.StringVar(&flags.outDir, "out", "./clean-logs", "Выход")
	flag.StringVar(&flags.report, "report", "report.json", "Репорт")
	flag.StringVar(&flags.config, "config", "detectors.yaml", "Конфиг")
	flag.StringVar(&flags.inmap, "mapping-in", "", "Загрузить словарь замен")
	flag.StringVar(&flags.outmap, "mapping-out", "", "Сохранить словарь замен")
	flag.Parse()

	if mode != "sanitize" && mode != "dry-run" {
		fmt.Printf("Укажите sanitize или dry-run, сейчас: %s\n", mode)
		os.Exit(1)
	}

	switch mode {
	case "sanitize":
		if err := sanrun.Run(flags.inDir, flags.outDir, flags.config, flags.report, flags.inmap, flags.outmap); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}
	case "dry-run":
		if err := dryrun.Run(flags.inDir, flags.config, flags.report, flags.inmap, flags.outmap); err != nil {
			fmt.Printf("Ошибка: %v\n", err)
			os.Exit(1)
		}

	}
}
