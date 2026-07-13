package report

import (
	"encoding/json"
	"fmt"
	"os"
)

type ReplacementExample struct {
	DetectorID   string `json:"detector_id"`
	OriginalKind string `json:"original_kind"`
	Replacement  string `json:"replacement"`
	Count        int    `json:"count"`
}

type Report struct {
	FileProc            int                  `json:"files_processed"`
	LineProc            int                  `json:"lines_processed"`
	ReplaceTotal        int                  `json:"replacement_total"`
	Detect              map[string]int       `json:"by_detector"`
	Errors              []string             `json:"errors"`
	ReplacementExamples []ReplacementExample `json:"replacement_examples,omitempty"`
}

func (r *Report) AddReplacementExample(DetectorID, OriginalKind, replacement string, count int) {

	for i, ex := range r.ReplacementExamples {
		if ex.DetectorID == DetectorID && ex.Replacement == replacement {
			r.ReplacementExamples[i].Count += count
			return
		}
	}
	r.ReplacementExamples = append(r.ReplacementExamples, ReplacementExample{
		DetectorID:   DetectorID,
		OriginalKind: OriginalKind,
		Replacement:  replacement,
		Count:        count,
	})
}
func CreateReport(path string, report *Report) error {

	total := 0
	for _, count := range report.Detect {
		total += count
	}
	report.ReplaceTotal = total

	data, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		fmt.Printf("Ошибка создания репорта: %v\n", err)

	}
	os.WriteFile(path, data, 0644)
	return nil

}
