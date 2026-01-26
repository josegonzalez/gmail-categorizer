package stats

import (
	"strings"
	"time"

	"github.com/josegonzalez/gmail-categorizer/internal/imap"
	"github.com/josegonzalez/gmail-categorizer/pkg/mailaddr"
)

// Aggregator collects and computes email statistics.
type Aggregator interface {
	Process(msg *imap.Message) error
	Results(mailboxName string) *Statistics
	Reset()
}

// specialKey uniquely identifies a special address combination.
type specialKey struct {
	to   string
	from string
}

// aggregatorImpl implements the Aggregator interface.
type aggregatorImpl struct {
	specialPrefixes []string
	toAddressCounts map[string]int
	specialCounts   map[specialKey]int
	totalMessages   int
}

// NewAggregator creates a new statistics aggregator.
// specialPrefixes defines address prefixes that trigger grouping by sender.
func NewAggregator(specialPrefixes []string) Aggregator {
	// Normalize prefixes
	normalized := make([]string, len(specialPrefixes))
	for i, p := range specialPrefixes {
		normalized[i] = strings.ToLower(strings.TrimSpace(p))
	}

	return &aggregatorImpl{
		specialPrefixes: normalized,
		toAddressCounts: make(map[string]int),
		specialCounts:   make(map[specialKey]int),
	}
}

// Process adds a message to the aggregation.
func (a *aggregatorImpl) Process(msg *imap.Message) error {
	a.totalMessages++

	// Process each To address
	for _, to := range msg.To {
		toAddr := mailaddr.Normalize(to.String())
		if toAddr == "" {
			continue
		}

		if a.isSpecialAddress(toAddr) {
			// For special addresses, group by sender
			for _, from := range msg.From {
				fromAddr := mailaddr.Normalize(from.String())
				if fromAddr == "" {
					continue
				}
				key := specialKey{to: toAddr, from: fromAddr}
				a.specialCounts[key]++
			}
		} else {
			// Normal grouping by recipient
			a.toAddressCounts[toAddr]++
		}
	}

	return nil
}

// isSpecialAddress checks if an address matches any special prefix.
func (a *aggregatorImpl) isSpecialAddress(addr string) bool {
	for _, prefix := range a.specialPrefixes {
		if mailaddr.HasPrefix(addr, prefix) {
			return true
		}
	}
	return false
}

// Results returns the computed statistics.
func (a *aggregatorImpl) Results(mailboxName string) *Statistics {
	stats := &Statistics{
		TotalMessages: a.totalMessages,
		ProcessedAt:   time.Now(),
		MailboxName:   mailboxName,
	}

	// Convert regular address counts to sorted slice
	stats.ByToAddress = make([]AddressStat, 0, len(a.toAddressCounts))
	for addr, count := range a.toAddressCounts {
		stats.ByToAddress = append(stats.ByToAddress, AddressStat{
			Address: addr,
			Count:   count,
		})
	}

	// Convert special address counts to sorted slice
	stats.ByFromAddress = make([]SpecialAddressStat, 0, len(a.specialCounts))
	for key, count := range a.specialCounts {
		stats.ByFromAddress = append(stats.ByFromAddress, SpecialAddressStat{
			ToAddress:   key.to,
			FromAddress: key.from,
			Count:       count,
		})
	}

	return stats
}

// Reset clears all accumulated data.
func (a *aggregatorImpl) Reset() {
	a.toAddressCounts = make(map[string]int)
	a.specialCounts = make(map[specialKey]int)
	a.totalMessages = 0
}

// SortByCount sorts address statistics by count in the specified order.
func SortByCount(stats []AddressStat, desc bool) {
	SortByInt(stats, func(s AddressStat) int { return s.Count }, desc)
}

// SortByAddress sorts address statistics alphabetically.
func SortByAddress(stats []AddressStat, desc bool) {
	SortByString(stats, func(s AddressStat) string { return s.Address }, desc)
}

// SortSpecialByCount sorts special address statistics by count.
func SortSpecialByCount(stats []SpecialAddressStat, desc bool) {
	SortByInt(stats, func(s SpecialAddressStat) int { return s.Count }, desc)
}

// SortSpecialByFrom sorts special address statistics by sender address.
func SortSpecialByFrom(stats []SpecialAddressStat, desc bool) {
	SortByString(stats, func(s SpecialAddressStat) string { return s.FromAddress }, desc)
}
