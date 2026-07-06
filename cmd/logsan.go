package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type detectorConfig struct {
	Detectors []Detector `yaml:"detectors"`
}
type Detector struct {
	ID                string `yaml:"detector_id"`
	Type              string `yaml:"type"`
	Pattern           string `yaml:"pattern"`
	ReplacementPrefix string `yaml:"replacement_prefix"`
	Enabled           bool   `yaml:"enabled"`
	regex             *regexp.Regexp
}

var counter = make(map[string]int)
var mapping = make(map[string]string)

func main() {
	detectors, err := loadConfig()
	if err != nil {
		fmt.Printf("Ошибка загрузки конфига: %v\n", err)
		os.Exit(1)
	}

	inFilePath := "../testdata/in.log"
	infile, err := os.Open(inFilePath)
	if err != nil {
		fmt.Printf("Ошибка открытия файла: %v\n", err)
		os.Exit(1)
	}
	defer infile.Close()

	outFilePath := "../testdata/out.log"
	outfile, err := os.Create(outFilePath)
	if err != nil {
		fmt.Printf("Ошибка создания выходного файла %v\n", err)
		os.Exit(1)
	}
	defer outfile.Close()

	reader := bufio.NewReader(infile)
	writer := bufio.NewWriter(outfile)
	defer writer.Flush()
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			processed := processLine(line, detectors)
			writer.WriteString(processed)
		}
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Printf("Ошибка чтения файла: %v\n", err)
			}

			break
		}

	}
}

func loadConfig() ([]Detector, error) {
	data, err := os.ReadFile("config/detectors.yaml")
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения yaml файла, %v", err)
	}
	var config detectorConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения данных из yaml: %v\n", err)
	}

	for i := range config.Detectors {
		if config.Detectors[i].Enabled {
			pattern := config.Detectors[i].Pattern
			re, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("Не скомпилировалось рег для %s: %v", config.Detectors[i].ID, err)
			}
			config.Detectors[i].regex = re
		}
	}
	return config.Detectors, nil
}

func processLine(line string, detectors []Detector) string {
	result := line
	//counter := make(map[string]int)

	for _, detector := range detectors {
		if !detector.Enabled {
			continue
		}
		matches := detector.regex.FindAllStringSubmatch(result, -1)

		for _, match := range matches {
			var replaceWhat string
			var key string
			if len(match) > 1 {
				replaceWhat = match[1]
				key = detector.ID + replaceWhat
			} else {
				replaceWhat = match[0]
				key = detector.ID + "|" + replaceWhat
			}
			mask, exists := mapping[key]

			if !exists {
				counter[detector.ID]++
				mask = fmt.Sprintf("%s_%d", detector.ReplacementPrefix, counter[detector.ID])
				mapping[key] = mask
			}
			result = strings.Replace(result, replaceWhat, mask, 1)

		}

	}

	//fmt.Printf("%s\n", result)
	return result
}
