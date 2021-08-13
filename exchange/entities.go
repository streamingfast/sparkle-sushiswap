// Code generated by sparkle.

package exchange

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (t *Token) Sanitize() {
	t.Name = strings.ReplaceAll(t.Name, "\u0000", "")
	t.Symbol = strings.ReplaceAll(t.Symbol, "\u0000", "")
}

func (p *Pair) Sanitize() {
	p.Name = strings.ReplaceAll(p.Name, "\u0000", "")
}

func (h *HourData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	hourId := blockTime.Unix() / 3600
	activeId := strconv.FormatInt(hourId, 10)

	return h.ID != activeId
}

func (d *DayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := strconv.FormatInt(dayId, 10)

	return d.ID != activeId
}

func (t *TokenDayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := fmt.Sprintf("%s-%d", t.Token, dayId)

	return t.ID != activeId
}

func (p *PairHourData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	hourId := blockTime.Unix() / 3600
	activeId := fmt.Sprintf("%s-%d", p.Pair, hourId)

	return p.ID != activeId
}

func (p *PairDayData) IsFinal(blockNum uint64, blockTime time.Time) bool {
	dayId := blockTime.Unix() / 86400
	activeId := fmt.Sprintf("%s-%d", p.Pair, dayId)

	return p.ID != activeId
}

func (t *Transaction) IsFinal(blockNum uint64, blockTime time.Time) bool {
	return true
}

func (s *Swap) IsFinal(blockNum uint64, blockTime time.Time) bool {
	return true
}
