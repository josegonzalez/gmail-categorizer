package output

import (
	"encoding/json"
	"io"

	"github.com/josegonzalez/gmail-categorizer/internal/stats"
)

// jsonFormatter outputs statistics as JSON.
type jsonFormatter struct {
	opts Options
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter(opts Options) Formatter {
	return &jsonFormatter{opts: opts}
}

// Format outputs statistics as formatted JSON.
func (f *jsonFormatter) Format(w io.Writer, s *stats.Statistics) error {
	statsCopy := prepareStatistics(s, f.opts)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(statsCopy)
}
