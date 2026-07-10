package san

import (
	"fmt"
	"logsan/internal/config"
	"strings"
	"sync"
)

var counter = make(map[string]int)    //счётчик для замен, привязанный к паттерну замены
var mapping = make(map[string]string) //маппинг для стабильной псевдонимизации
var stats = make(map[string]int)      //связывает детектор и количество замен
var mu sync.Mutex

func ProcessLine(line string, detectors []config.Detector) string {
	mu.Lock()
	defer mu.Unlock()
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
			result = strings.Replace(result, replaceWhat, mask, 1) //результат замены: заношу в результат маску на выделенное ранее место
			stats[detector.ID]++                                   //для отчета кол-во замен
		}

	}
	return result
}

func GetStats() map[string]int {
	result := make(map[string]int)
	for key, counts := range stats { //key - ID детектора, counts - количество замен
		result[key] = counts
	}
	return result
}
