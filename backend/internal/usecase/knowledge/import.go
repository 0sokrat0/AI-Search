package knowledge

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"unicode"

	"MRG/internal/infrastructure/search"
)

const importBatchSize = 64

type ImportService struct {
	sieve *search.QdrantSieve
}

type ImportResult struct {
	FileName     string `json:"fileName"`
	Imported     int    `json:"imported"`
	Merchants    int    `json:"merchants"`
	PSOffers     int    `json:"psOffers"`
	TraderSearch int    `json:"traderSearch"`
	Traders      int    `json:"traders"`
	Noise        int    `json:"noise"`
}

type importRow struct {
	text      string
	direction string
	isLead    bool
}

func NewImportService(sieve *search.QdrantSieve) *ImportService {
	return &ImportService{sieve: sieve}
}

func (s *ImportService) ImportCSV(ctx context.Context, fileName, content string) (*ImportResult, error) {
	if s == nil || s.sieve == nil {
		return nil, fmt.Errorf("knowledge import is unavailable")
	}

	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		fileName = "knowledge.csv"
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("file is empty")
	}

	rows, err := parseRows(content)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("file does not contain importable rows")
	}

	result := &ImportResult{FileName: fileName}
	for start := 0; start < len(rows); start += importBatchSize {
		end := start + importBatchSize
		if end > len(rows) {
			end = len(rows)
		}

		texts := make([]string, 0, end-start)
		labels := make([]bool, 0, end-start)
		directions := make([]string, 0, end-start)
		for _, row := range rows[start:end] {
			texts = append(texts, row.text)
			labels = append(labels, row.isLead)
			directions = append(directions, row.direction)
			result.Imported++
			switch row.direction {
			case "merchant":
				result.Merchants++
			case "ps_offer":
				result.PSOffers++
			case "trader_search":
				result.TraderSearch++
			case "trader":
				result.Traders++
			case "noise":
				result.Noise++
			}
		}

		if err := s.sieve.AddReferencePointsBatchWithMeta(ctx, texts, labels, directions); err != nil {
			return nil, fmt.Errorf("import reference batch: %w", err)
		}
	}

	return result, nil
}

func parseRows(content string) ([]importRow, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = detectDelimiter(content)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("file is empty")
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	textIdx := -1
	categoryIdx := -1
	for i, raw := range header {
		switch normalizeColumnName(raw) {
		case "text", "targetmessage", "targetmsg", "целевоесообщение", "сообщение":
			textIdx = i
		case "category", "leadtype", "leadkind", "типлида", "тип":
			categoryIdx = i
		}
	}
	if textIdx < 0 || categoryIdx < 0 {
		return nil, fmt.Errorf("csv must contain target message and lead type columns")
	}

	rows := make([]importRow, 0, 128)
	var problems []string
	line := 1

	for {
		line++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			problems = append(problems, fmt.Sprintf("line %d: %v", line, err))
			continue
		}
		if len(record) <= textIdx || len(record) <= categoryIdx {
			problems = append(problems, fmt.Sprintf("line %d: target message or lead type is missing", line))
			continue
		}

		text := strings.TrimSpace(record[textIdx])
		if text == "" {
			problems = append(problems, fmt.Sprintf("line %d: text is empty", line))
			continue
		}

		direction, isLead, ok := normalizeCategory(record[categoryIdx])
		if !ok {
			problems = append(problems, fmt.Sprintf("line %d: unsupported category %q", line, strings.TrimSpace(record[categoryIdx])))
			continue
		}

		rows = append(rows, importRow{text: text, direction: direction, isLead: isLead})
	}

	if len(problems) > 0 {
		if len(problems) > 5 {
			problems = append(problems[:5], fmt.Sprintf("and %d more errors", len(problems)-5))
		}
		return nil, fmt.Errorf("invalid csv: %s", strings.Join(problems, "; "))
	}

	return rows, nil
}

func detectDelimiter(content string) rune {
	firstLine := content
	if idx := strings.IndexAny(content, "\r\n"); idx >= 0 {
		firstLine = content[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		return ';'
	}
	return ','
}

func normalizeColumnName(raw string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func normalizeCategory(raw string) (direction string, isLead bool, ok bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "merchants", "merchant", "merch":
		return "merchant", true, true
	case "processing_requests", "processing_request", "processing":
		return "merchant", true, true
	case "ps_offers", "ps_offer", "offer":
		return "ps_offer", true, true
	case "trader_search", "search_trader", "search_traders", "looking_for_trader":
		return "trader_search", true, true
	case "traders", "trader":
		return "trader", true, true
	case "noise", "negative", "spam":
		return "noise", false, true
	default:
		return "", false, false
	}
}
