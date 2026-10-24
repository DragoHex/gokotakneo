package feed

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
)

// Parser handles little-endian binary batch de-framing and exchange divider scaling.
type Parser struct {
	dividers map[string]float64
	mu       sync.RWMutex
}

// NewParser initializes a binary packet parser.
func NewParser() *Parser {
	return &Parser{
		dividers: make(map[string]float64),
	}
}

// SetDivider stores the price scaling divisor for an exchange segment.
func (p *Parser) SetDivider(segment string, divider float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dividers[segment] = divider
}

// GetDivider returns the divisor, defaulting to 100.0 if not configured.
func (p *Parser) GetDivider(segment string) float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d, ok := p.dividers[segment]
	if !ok || d <= 0 {
		return 100.0
	}
	return d
}

// ParseBatch de-frames a binary buffer containing one or more packets.
func (p *Parser) ParseBatch(buf []byte, defaultSegment string) ([]Tick, error) {
	var ticks []Tick
	offset := 0

	for offset < len(buf) {
		if len(buf)-offset < 4 {
			return nil, fmt.Errorf("truncated packet header at offset %d", offset)
		}

		pktLen := int(binary.LittleEndian.Uint16(buf[offset : offset+2]))
		msgCode := binary.LittleEndian.Uint16(buf[offset+2 : offset+4])

		if offset+4+pktLen > len(buf) {
			return nil, fmt.Errorf("packet length %d exceeds remaining buffer (%d bytes)", pktLen, len(buf)-offset-4)
		}

		payload := buf[offset+4 : offset+4+pktLen]
		offset += 4 + pktLen

		tick, err := p.parsePacket(payload, msgCode, defaultSegment)
		if err != nil {
			continue // skip unsupported or unparseable packet safely
		}
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

func (p *Parser) parsePacket(payload []byte, msgCode uint16, segment string) (Tick, error) {
	if len(payload) < 8 {
		return Tick{}, fmt.Errorf("payload too short for packet")
	}

	token := binary.LittleEndian.Uint32(payload[0:4])
	rawPrice := int32(binary.LittleEndian.Uint32(payload[4:8]))

	divisor := p.GetDivider(segment)
	price := decimal.NewFromFloat(float64(rawPrice) / divisor)

	return Tick{
		InstrumentToken: token,
		ExchangeSegment: segment,
		LastPrice:       price,
	}, nil
}
