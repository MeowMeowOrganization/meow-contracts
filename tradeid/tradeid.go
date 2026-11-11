// Package tradeid is the canonical, versioned textual encoding of a matching
// execution identity, shared by market data, accounting and order services.
package tradeid

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Format encodes MatchingEventId as <market_id>:<market_sequence>:<event_ordinal>.
// The sequence is positive; market ID and ordinal may be zero.
func Format(marketID uint32, sequence, ordinal uint64) (string, error) {
	if sequence == 0 {
		return "", errors.New("tradeid: zero sequence")
	}
	return fmt.Sprintf("%d:%d:%d", marketID, sequence, ordinal), nil
}

func Parse(s string) (uint32, uint64, uint64, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, 0, 0, errors.New("tradeid: invalid format")
	}
	id, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	seq, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	ord, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return 0, 0, 0, err
	}
	canonical, err := Format(uint32(id), seq, ord)
	if err != nil || s != canonical {
		return 0, 0, 0, errors.New("tradeid: noncanonical encoding")
	}
	return uint32(id), seq, ord, nil
}
