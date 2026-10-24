package feed_test

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/DragoHex/gokotakneo/feed"
	"github.com/shopspring/decimal"
)

func TestNativeBatchBinaryParser(t *testing.T) {
	t.Run("Positive: Single LTP Packet with Divider Scaling", func(t *testing.T) {
		// Mock LTP Packet:
		// [0..1] Length = 8
		// [2..3] MessageCode = 1 (LTP)
		// [4..7] InstrumentToken = 11536 (little endian uint32)
		// [8..11] Scaled Price = 345050 (little endian int32) -> 3450.50 with divider 100
		buf := make([]byte, 12)
		binary.LittleEndian.PutUint16(buf[0:2], 8)
		binary.LittleEndian.PutUint16(buf[2:4], 1)
		binary.LittleEndian.PutUint32(buf[4:8], 11536)
		binary.LittleEndian.PutUint32(buf[8:12], 345050)

		parser := feed.NewParser()
		parser.SetDivider("nse_cm", 100.0)

		ticks, err := parser.ParseBatch(buf, "nse_cm")
		if err != nil {
			t.Fatalf("ParseBatch failed: %v", err)
		}
		if len(ticks) != 1 {
			t.Fatalf("expected 1 tick, got %d", len(ticks))
		}
		tick := ticks[0]
		if tick.InstrumentToken != 11536 {
			t.Errorf("token = %d, want 11536", tick.InstrumentToken)
		}
		expectedPrice := decimal.NewFromFloat(3450.50)
		if !tick.LastPrice.Equal(expectedPrice) {
			t.Errorf("lastPrice = %s, want %s", tick.LastPrice, expectedPrice)
		}
	})

	t.Run("Positive: Multi-Packet Batch De-framing", func(t *testing.T) {
		// Construct two concatenated packets
		pkt1 := make([]byte, 12)
		binary.LittleEndian.PutUint16(pkt1[0:2], 8)
		binary.LittleEndian.PutUint16(pkt1[2:4], 1)
		binary.LittleEndian.PutUint32(pkt1[4:8], 1001)
		binary.LittleEndian.PutUint32(pkt1[8:12], 10000) // 100.00

		pkt2 := make([]byte, 12)
		binary.LittleEndian.PutUint16(pkt2[0:2], 8)
		binary.LittleEndian.PutUint16(pkt2[2:4], 1)
		binary.LittleEndian.PutUint32(pkt2[4:8], 1002)
		binary.LittleEndian.PutUint32(pkt2[8:12], 25050) // 250.50

		batch := append(pkt1, pkt2...)

		parser := feed.NewParser()
		parser.SetDivider("nse_cm", 100.0)

		ticks, err := parser.ParseBatch(batch, "nse_cm")
		if err != nil {
			t.Fatalf("ParseBatch failed: %v", err)
		}
		if len(ticks) != 2 {
			t.Fatalf("expected 2 ticks, got %d", len(ticks))
		}
		if ticks[0].InstrumentToken != 1001 || ticks[1].InstrumentToken != 1002 {
			t.Errorf("unexpected tokens: %v", ticks)
		}
	})

	t.Run("Negative: Truncated Buffer", func(t *testing.T) {
		parser := feed.NewParser()
		_, err := parser.ParseBatch([]byte{0x08, 0x00}, "nse_cm")
		if err == nil {
			t.Fatalf("expected error on truncated packet, got nil")
		}
	})

	t.Run("Edge Case: Zero Divider Fallback", func(t *testing.T) {
		buf := make([]byte, 12)
		binary.LittleEndian.PutUint16(buf[0:2], 8)
		binary.LittleEndian.PutUint16(buf[2:4], 1)
		binary.LittleEndian.PutUint32(buf[4:8], 11536)
		binary.LittleEndian.PutUint32(buf[8:12], 10000)

		parser := feed.NewParser()
		// No divider set -> should fallback safely to 100.0 without divide by zero
		ticks, err := parser.ParseBatch(buf, "unknown_seg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ticks[0].LastPrice.Equal(decimal.NewFromFloat(100.00)) {
			t.Errorf("lastPrice = %s, want 100.00", ticks[0].LastPrice)
		}
	})
}

// Suppress unused math import if needed
var _ = math.Exp2
