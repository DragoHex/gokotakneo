package models

import (
	"fmt"
	"strings"
	"time"
)

// BrokerTime provides custom JSON deserialization for Kotak-Neo datetime formats.
type BrokerTime struct {
	time.Time
}

var acceptedFormats = []string{
	"2006/01/02 15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func (bt *BrokerTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		bt.Time = time.Time{}
		return nil
	}

	for _, format := range acceptedFormats {
		if t, err := time.Parse(format, s); err == nil {
			bt.Time = t
			return nil
		}
	}

	return fmt.Errorf("unable to parse datetime %q into BrokerTime", s)
}
