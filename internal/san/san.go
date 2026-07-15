package san

import (
	"encoding/json"
	"fmt"
	"logsan/internal/config"
	"os"
	"strings"
	"sync"
)

// var counter = make(map[string]int)    //счётчик для замен, привязанный к паттерну замены
// var mapping = make(map[string]string) //маппинг для стабильной псевдонимизации
// var stats = make(map[string]int)      //связывает детектор и количество замен
// var mu sync.Mutex
type Sanitizer struct {
	detectors []config.Detector
	counter   map[string]int
	mapping   map[string]string
	stats     map[string]int
	mu        sync.Mutex
}
type MappingData struct {
	Counter map[string]int    `json:"counter"`
	Mapping map[string]string `json:"mapping"`
	Stats   map[string]int    `json:"stats"`
}
type ReplacementExample struct {
	DetectorID   string
	OriginalKind string
	Replacement  string
	Count        int
}

func NewSanitizer(detectors []config.Detector) *Sanitizer {
	return &Sanitizer{
		detectors: detectors,
		counter:   make(map[string]int),
		mapping:   make(map[string]string),
		stats:     make(map[string]int),
	}
}

func (s *Sanitizer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter = make(map[string]int)
	s.mapping = make(map[string]string)
	s.stats = make(map[string]int)
}

func (s *Sanitizer) ProcessLine(line string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := line

	for _, detector := range s.detectors {
		if !detector.IsEnabled() {
			continue
		}
		matches := detector.Regex.FindAllStringSubmatch(result, -1) //ищет все совпадения регулярных значений в строке,
		//использую FindAllStringSubmatch, потому что некоторые регулярки написанны группами

		for _, match := range matches {
			var replaceWhat string
			var key string
			if len(match) > 1 { //если в регулярке больше одной группы, заменяю только вторую ([1])группу
				replaceWhat = match[1]
				key = detector.ID + "|" + replaceWhat //использую дальше для маппинга стабильной псевдонимизации
			} else {
				replaceWhat = match[0] //а если в регулярке нет групп, беру всю найденную строку целиком
				key = detector.ID + "|" + replaceWhat
			}
			mask, exists := s.mapping[key] //использую key для маппинга

			if !exists { //если такое значение ранее не повторялось, доваляю счетчик и делаю новый маппинг
				s.counter[detector.ID]++
				mask = fmt.Sprintf("%s_%d", detector.ReplacementPrefix, s.counter[detector.ID])
				s.mapping[key] = mask
			}
			s.stats[detector.ID]++
			result = strings.ReplaceAll(result, replaceWhat, mask) //результат замены: заношу в результат маску на выделенное ранее место
			//для отчета кол-во замен
		}

	}
	return result
}

func (s *Sanitizer) SaveMapping(path string) error {
	data := MappingData{
		Counter: s.counter,
		Mapping: s.mapping,
		Stats:   s.stats,
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

func (s *Sanitizer) LoadMapping(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("Ошибка чтения словаря замен: %v", err)
	}
	var mappingData MappingData
	if err := json.Unmarshal(data, &mappingData); err != nil {
		return fmt.Errorf("Ошибка парсинга словоря: %v", err)
	}
	s.counter = mappingData.Counter
	s.mapping = mappingData.Mapping
	if mappingData.Stats != nil {
		s.stats = mappingData.Stats
	}
	return nil
}

func (s *Sanitizer) GetStats() map[string]int {
	result := make(map[string]int)
	for key, counts := range s.stats { //key - ID детектора, counts - количество замен
		result[key] = counts
	}
	return result
}

func (s *Sanitizer) GetReplacementExamples() []ReplacementExample {
	maskStats := make(map[string]map[string]int)

	for key, mask := range s.mapping {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		detectorID := parts[0]

		if maskStats[detectorID] == nil {
			maskStats[detectorID] = make(map[string]int)
		}
		maskStats[detectorID][mask]++
	}
	var examples []ReplacementExample
	for detectorID, masks := range maskStats {
		for mask, count := range masks {
			examples = append(examples, ReplacementExample{
				DetectorID:   detectorID,
				OriginalKind: detectorID,
				Replacement:  mask,
				Count:        count,
			})
		}
	}
	return examples
}
