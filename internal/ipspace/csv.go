package ipspace

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// ImportReport summarizes a CSV import.
type ImportReport struct {
	Added   int      `json:"added"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// ImportCSV adds records from CSV rows (columns: prefix,name,status,tags,notes;
// tags are pipe-separated). A bad CIDR or an overlap conflict skips that row and
// is recorded, never aborting the batch.
func (s *Store) ImportCSV(r io.Reader) (ImportReport, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return ImportReport{}, err
	}
	var rep ImportReport
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(row[0]), "prefix") {
			continue // header
		}
		prefix := strings.TrimSpace(row[0])
		name := col(row, 1)
		status := Status(strings.TrimSpace(col(row, 2)))
		tags := splitTags(col(row, 3))
		notes := col(row, 4)
		if _, err := s.Add(prefix, name, status, tags, notes); err != nil {
			rep.Skipped++
			rep.Errors = append(rep.Errors, fmt.Sprintf("%s: %v", prefix, err))
			continue
		}
		rep.Added++
	}
	return rep, nil
}

// ExportCSV writes every record as CSV, prefix-sorted.
func (s *Store) ExportCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"prefix", "name", "status", "tags", "notes"}); err != nil {
		return err
	}
	for _, r := range s.Records() {
		if err := cw.Write([]string{r.Prefix, r.Name, string(r.Status), strings.Join(r.Tags, "|"), r.Notes}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func col(row []string, i int) string {
	if i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

func splitTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
