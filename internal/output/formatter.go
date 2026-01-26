// Package output provides formatters for displaying statistics.
package output

import (
	"fmt"
	"io"

	"github.com/josegonzalez/gmail-categorizer/internal/config"
	"github.com/josegonzalez/gmail-categorizer/internal/stats"
)

// Formatter defines the interface for outputting statistics.
type Formatter interface {
	Format(w io.Writer, s *stats.Statistics) error
}

// Options controls output formatting behavior.
type Options struct {
	SortBy    string
	SortOrder string
	Limit     int
}

// NewFormatter creates a formatter for the given format type.
func NewFormatter(format string, opts Options) (Formatter, error) {
	switch format {
	case config.FormatTable:
		return NewTableFormatter(opts), nil
	case config.FormatJSON:
		return NewJSONFormatter(opts), nil
	case config.FormatCSV:
		return NewCSVFormatter(opts), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// prepareStatistics creates a copy of the statistics and applies sort and limit options.
func prepareStatistics(s *stats.Statistics, opts Options) *stats.Statistics {
	statsCopy := *s
	statsCopy.ByToAddress = make([]stats.AddressStat, len(s.ByToAddress))
	copy(statsCopy.ByToAddress, s.ByToAddress)
	statsCopy.ByFromAddress = make([]stats.SpecialAddressStat, len(s.ByFromAddress))
	copy(statsCopy.ByFromAddress, s.ByFromAddress)

	applySort(&statsCopy, opts)
	applyLimit(&statsCopy, opts.Limit)

	return &statsCopy
}

// applySort sorts the statistics based on options.
func applySort(s *stats.Statistics, opts Options) {
	desc := opts.SortOrder == config.SortDesc

	switch opts.SortBy {
	case config.SortByCount:
		stats.SortByCount(s.ByToAddress, desc)
		stats.SortSpecialByCount(s.ByFromAddress, desc)
	case config.SortByAddress:
		stats.SortByAddress(s.ByToAddress, desc)
		stats.SortSpecialByFrom(s.ByFromAddress, desc)
	default:
		// Default to count descending
		stats.SortByCount(s.ByToAddress, true)
		stats.SortSpecialByCount(s.ByFromAddress, true)
	}
}

// applyLimit limits the number of results if specified.
func applyLimit(s *stats.Statistics, limit int) {
	if limit <= 0 {
		return
	}

	if len(s.ByToAddress) > limit {
		s.ByToAddress = s.ByToAddress[:limit]
	}
	if len(s.ByFromAddress) > limit {
		s.ByFromAddress = s.ByFromAddress[:limit]
	}
}
