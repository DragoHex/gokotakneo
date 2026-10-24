package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/DragoHex/gokotakneo/models"
)

func TestBrokerTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		inputJSON  string
		wantYear   int
		wantMonth  time.Month
		wantDay    int
		wantHour   int
		wantMinute int
		wantSecond int
		wantErr    bool
	}{
		{
			name:       "Positive: Kotak standard slash format",
			inputJSON:  `{"time": "2022/06/21 15:11:02"}`,
			wantYear:   2022,
			wantMonth:  time.June,
			wantDay:    21,
			wantHour:   15,
			wantMinute: 11,
			wantSecond: 2,
			wantErr:    false,
		},
		{
			name:       "Positive: Standard ISO8601 dash format",
			inputJSON:  `{"time": "2023-01-15 09:30:00"}`,
			wantYear:   2023,
			wantMonth:  time.January,
			wantDay:    15,
			wantHour:   9,
			wantMinute: 30,
			wantSecond: 0,
			wantErr:    false,
		},
		{
			name:       "Positive: Date only format",
			inputJSON:  `{"time": "2023-11-20"}`,
			wantYear:   2023,
			wantMonth:  time.November,
			wantDay:    20,
			wantHour:   0,
			wantMinute: 0,
			wantSecond: 0,
			wantErr:    false,
		},
		{
			name:       "Edge Case: Null JSON value",
			inputJSON:  `{"time": null}`,
			wantYear:   1, // zero time
			wantErr:    false,
		},
		{
			name:       "Edge Case: Empty string value",
			inputJSON:  `{"time": ""}`,
			wantYear:   1, // zero time
			wantErr:    false,
		},
		{
			name:      "Negative: Corrupt string format",
			inputJSON: `{"time": "corrupt-datetime-value"}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload struct {
				Time models.BrokerTime `json:"time"`
			}

			err := json.Unmarshal([]byte(tt.inputJSON), &payload)
			if (err != nil) != tt.wantErr {
				t.Fatalf("json.Unmarshal() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.wantYear > 1 {
				got := payload.Time.Time
				if got.Year() != tt.wantYear || got.Month() != tt.wantMonth || got.Day() != tt.wantDay {
					t.Errorf("got date %v-%v-%v, want %v-%v-%v", got.Year(), got.Month(), got.Day(), tt.wantYear, tt.wantMonth, tt.wantDay)
				}
				if got.Hour() != tt.wantHour || got.Minute() != tt.wantMinute || got.Second() != tt.wantSecond {
					t.Errorf("got time %v:%v:%v, want %v:%v:%v", got.Hour(), got.Minute(), got.Second(), tt.wantHour, tt.wantMinute, tt.wantSecond)
				}
			}
		})
	}
}
