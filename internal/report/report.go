package report

import (
	"encoding/json"
	"fmt"
	"os"
)

type Report struct {
	FileProc     int            `json:"files_processed"`
	LineProc     int            `json:"lines_processed"`
	ReplaceTotal int            `json:"replacement_total"`
	Detect       map[string]int `json:"by_detector"`
	Errors       []string       `json:"errors"`
}

func CreateReport(path string, report *Report) {

	data, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		fmt.Printf("Ошибка создания репорта: %v\n", err)

	}
	os.WriteFile(path, data, 0644)

}
