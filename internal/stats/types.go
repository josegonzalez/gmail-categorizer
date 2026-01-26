// Package stats provides email statistics aggregation.
package stats

import "time"

// Statistics contains aggregated email statistics.
type Statistics struct {
	TotalMessages int                 `json:"total_messages"`
	ByToAddress   []AddressStat       `json:"by_to_address"`
	ByFromAddress []SpecialAddressStat `json:"by_from_address"`
	ProcessedAt   time.Time           `json:"processed_at"`
	MailboxName   string              `json:"mailbox_name"`
}

// AddressStat represents statistics for a single recipient address.
type AddressStat struct {
	Address string `json:"address"`
	Count   int    `json:"count"`
}

// SpecialAddressStat tracks emails to special addresses grouped by sender.
type SpecialAddressStat struct {
	ToAddress   string `json:"to_address"`
	FromAddress string `json:"from_address"`
	Count       int    `json:"count"`
}
