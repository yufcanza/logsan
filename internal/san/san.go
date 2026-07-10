package san

import (
	"encoding/json"
	"fmt"
	"logsan/internal/config"
	"os"
	"strings"
	"sync"
)

var counter = make(map[string]int)    //счётчик для замен, привязанный к паттерну замены
var mapping = make(map[string]string) //маппинг для стабильной псевдонимизации
var stats = make(map[string]int)      //связывает детектор и количество замен
var mu sync.Mutex

type MappingData struct {
	Counter map[string]int    `json: "counter"`
	Mapping map[string]string `json: "mapping"`
	Stats   map[string]int    `json:"stats"`
}

func ProcessLine(line string, detectors []config.Detector) string {
	//mu.Lock()
	//defer mu.Unlock()
	result := line
	//counter := make(map[string]int)

	for _, detector := range detectors {
		if !detector.Enabled {
			continue
		}
		matches := detector.Regex.FindAllStringSubmatch(result, -1) //ищет все совпадения регулярных значений в строке,
		//использую FindAllStringSubmatch, потому что некоторые регулярки написанны группами

		for _, match := range matches {
			var replaceWhat string
			var key string
			if len(match) > 1 { //если в регулярке больше одной группы, заменяю только вторую ([1])группу
				replaceWhat = match[1]
				key = detector.ID + replaceWhat //использую дальше для маппинга стабильной псевдонимизации
			} else {
				replaceWhat = match[0] //а если в регулярке нет групп, беру всю найденную строку целиком
				key = detector.ID + replaceWhat
			}
			mask, exists := mapping[key] //использую key для маппинга

			if !exists { //если такое значение ранее не повторялось, доваляю счетчик и делаю новый маппинг
				counter[detector.ID]++
				mask = fmt.Sprintf("%s_%d", detector.ReplacementPrefix, counter[detector.ID])
				mapping[key] = mask
			}
			result = strings.ReplaceAll(result, replaceWhat, mask) //результат замены: заношу в результат маску на выделенное ранее место
			stats[detector.ID]++                                   //для отчета кол-во замен
		}

	}
	return result
}

func SaveMapping(path string) error {
	data := MappingData{
		Counter: counter,
		Mapping: mapping,
		Stats:   stats,
	}
	jsonData, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("Ошибка создания словаря: %v", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("Ошибка записи словаря:%v", err)

	}
	return nil
}

func LoadMapping(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Ошибка чтения словаря замен: %v", err)
	}
	var mappingData MappingData
	if err := json.Unmarshal(data, &mappingData); err != nil {
		return fmt.Errorf("Ошибка парсинга словоря: %v", err)
	}
	counter = mappingData.Counter
	mapping = mappingData.Mapping
	if mappingData.Stats != nil {
		stats = mappingData.Stats
	}
	return nil
}

func GetStats() map[string]int {
	result := make(map[string]int)
	for key, counts := range stats { //key - ID детектора, counts - количество замен
		result[key] = counts
	}
	return result
}
