package output

import (
	"fmt"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/josegonzalez/gmail-categorizer/internal/stats"
)

// tableFormatter outputs statistics as ASCII tables.
type tableFormatter struct {
	opts Options
}

// NewTableFormatter creates a new table formatter.
func NewTableFormatter(opts Options) Formatter {
	return &tableFormatter{opts: opts}
}

// Format outputs statistics as formatted ASCII tables.
func (f *tableFormatter) Format(w io.Writer, s *stats.Statistics) error {
	statsCopy := prepareStatistics(s, f.opts)

	// Print summary
	fmt.Fprintf(w, "Mailbox: %s\n", statsCopy.MailboxName)
	fmt.Fprintf(w, "Total Messages: %d\n", statsCopy.TotalMessages)
	fmt.Fprintf(w, "Processed: %s\n\n", statsCopy.ProcessedAt.Format("2006-01-02 15:04:05"))

	// Regular addresses table
	if len(statsCopy.ByToAddress) > 0 {
		fmt.Fprintln(w, "Emails by Recipient Address:")
		f.renderAddressTable(w, statsCopy.ByToAddress)
	}

	// Special addresses table
	if len(statsCopy.ByFromAddress) > 0 {
		fmt.Fprintln(w, "\nEmails to Special Addresses (grouped by sender):")
		f.renderSpecialTable(w, statsCopy.ByFromAddress)
	}

	return nil
}

// renderAddressTable renders the regular address statistics table.
func (f *tableFormatter) renderAddressTable(w io.Writer, data []stats.AddressStat) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"To Address", "Count"})

	var total int
	for _, stat := range data {
		t.AppendRow(table.Row{stat.Address, stat.Count})
		total += stat.Count
	}

	t.AppendSeparator()
	t.AppendFooter(table.Row{"Total", total})

	t.Render()
}

// renderSpecialTable renders the special address statistics table.
func (f *tableFormatter) renderSpecialTable(w io.Writer, data []stats.SpecialAddressStat) {
	t := table.NewWriter()
	t.SetOutputMirror(w)
	t.SetStyle(table.StyleLight)

	t.AppendHeader(table.Row{"To Address", "From Address", "Count"})

	var total int
	for _, stat := range data {
		t.AppendRow(table.Row{stat.ToAddress, stat.FromAddress, stat.Count})
		total += stat.Count
	}

	t.AppendSeparator()
	t.AppendFooter(table.Row{"", "Total", total})

	t.Render()
}
