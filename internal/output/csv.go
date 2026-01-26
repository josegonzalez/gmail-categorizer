package output

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/josegonzalez/gmail-categorizer/internal/stats"
)

// csvFormatter outputs statistics as CSV.
type csvFormatter struct {
	opts Options
}

// NewCSVFormatter creates a new CSV formatter.
func NewCSVFormatter(opts Options) Formatter {
	return &csvFormatter{opts: opts}
}

// Format outputs statistics as CSV.
func (f *csvFormatter) Format(w io.Writer, s *stats.Statistics) error {
	statsCopy := prepareStatistics(s, f.opts)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"type", "to_address", "from_address", "count"}); err != nil {
		return err
	}

	// Write regular address stats
	for _, stat := range statsCopy.ByToAddress {
		record := []string{
			"regular",
			stat.Address,
			"",
			strconv.Itoa(stat.Count),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	// Write special address stats
	for _, stat := range statsCopy.ByFromAddress {
		record := []string{
			"special",
			stat.ToAddress,
			stat.FromAddress,
			strconv.Itoa(stat.Count),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return writer.Error()
}
