package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type detectorConfig struct { //для хранения всех детекторов
	Detectors []Detector `yaml:"detectors"`
}
type Detector struct { //для хранения каждого отдельного детектора
	ID                string         `yaml:"detector_id"`
	Type              string         `yaml:"type"`
	Pattern           string         `yaml:"pattern"`
	ReplacementPrefix string         `yaml:"replacement_prefix"`
	Enabled           bool           `yaml:"enabled"`
	Regex             *regexp.Regexp //храним тут скомпилированное регулярное значение pattern
}

func LoadConfig(path string) ([]Detector, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения yaml файла, %v", err)
	}
	var config detectorConfig
	err = yaml.Unmarshal(data, &config) //преобразуем детекторы yaml в стуктуры Detector
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения данных из yaml: %v\n", err)
	}

	for i := range config.Detectors {
		if config.Detectors[i].Enabled {
			pattern := config.Detectors[i].Pattern
			re, err := regexp.Compile(pattern) //компилируем регулярное значение для каждого детектора
			if err != nil {
				return nil, fmt.Errorf("Не скомпилировалось рег для %s: %v", config.Detectors[i].ID, err)
			}
			config.Detectors[i].Regex = re
		}
	}
	return config.Detectors, nil
}
