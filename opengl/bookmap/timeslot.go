package bookmap

import (
	"fmt"
	"time"

	"github.com/lian/gdax-bookmap/orderbook"
)

type TimeSlotRow struct {
	Low        float64
	Heigh      float64
	Size       float64
	Type       int
	OrderCount int
	AskSize    float64
	BidSize    float64
	BidCount   int
	AskCount   int
}

type TimeSlot struct {
	From         time.Time
	To           time.Time
	Rows         []*TimeSlotRow
	MaxSize      float64
	BidPrice     float64
	AskPrice     float64
	BidTradeSize float64
	AskTradeSize float64
	Stats        *orderbook.BookMapStatsCopy
	Cleared      bool
}

func NewTimeSlot(from time.Time, to time.Time) *TimeSlot {
	v := &TimeSlot{
		From:    from,
		To:      to,
		Rows:    []*TimeSlotRow{},
		Cleared: true,
	}
	return v
}

func (s *TimeSlot) noStats() bool {
	return s.Stats == nil
}

func (s *TimeSlot) isEmpty() bool {
	for _, row := range s.Rows {
		if row.Size > 0 {
			return false
		}
	}
	return true
}

func (s *TimeSlot) FindRow(price float64) *TimeSlotRow {
	for _, row := range s.Rows {
		if (price <= row.Heigh) && (price > row.Low) {
			return row
		}
	}
	return nil
}
func (s *TimeSlot) ClearRows() {
	s.Cleared = true
	//s.Rows = make([]*TimeSlotRow, 0)
}

func (s *TimeSlot) GenerateRows(count, priceOffset, steps float64) {
	if s.Cleared || len(s.Rows) != int(count) {
		s.Cleared = false

		s.Rows = make([]*TimeSlotRow, int(count))

		for i := 0; i < int(count); i++ {
			heigh := priceOffset - (float64(i) * steps)
			low := heigh - steps

			s.Rows[i] = &TimeSlotRow{Low: low, Heigh: heigh}
		}
	} else {
		for i := 0; i < int(count); i++ {
			heigh := priceOffset - (float64(i) * steps)
			low := heigh - steps

			s.Rows[i].Low = low
			s.Rows[i].Heigh = heigh
		}
	}
}

func (s *TimeSlot) Refill() {
	for _, row := range s.Rows {
		row.BidSize = 0
		row.BidCount = 0
		row.AskSize = 0
		row.AskCount = 0
		row.OrderCount = 0
		row.Size = 0
	}
	if s.Stats != nil {
		s.Fill(s.Stats)
	} else {
		fmt.Println("no stats pointer")
	}
}

func (s *TimeSlot) Fill(stats *orderbook.BookMapStatsCopy) {
	maxSize := 0.0
	s.AskTradeSize = 0.0
	s.BidTradeSize = 0.0

	high := s.Rows[0].Heigh
	low := s.Rows[len(s.Rows)-1].Low

	for i := len(stats.Bid) - 1; i >= 0; i-- {
		state := stats.Bid[i]

		if state.TradeSize > 0 {
			s.BidTradeSize += state.TradeSize
		}

		if state.Price > high || state.Price <= low {
			continue
		}

		row := s.FindRow(state.Price)

		if row == nil {
			//fmt.Println("bid row not found for price", state.Price, low, high)
			continue
		}

		row.Type = 0
		row.Size += state.Size
		row.BidSize += state.Size
		row.BidCount += state.OrderCount
		row.OrderCount += state.OrderCount

		if s.BidPrice == 0 {
			s.BidPrice = state.Price
		} else {
			if state.Price > s.BidPrice {
				s.BidPrice = state.Price
			}
		}

		if row.Size > maxSize {
			maxSize = row.Size
		}
	}

	for _, state := range stats.Ask {
		if state.TradeSize > 0 {
			s.AskTradeSize += state.TradeSize
		}

		if state.Price > high || state.Price <= low {
			continue
		}

		row := s.FindRow(state.Price)

		if row == nil {
			//fmt.Println("ask row not found for price", state.Price, low, high)
			continue
		}

		row.Type = 1
		row.Size += state.Size
		row.AskSize += state.Size
		row.AskCount += state.OrderCount
		row.OrderCount += state.OrderCount

		if s.AskPrice == 0 {
			s.AskPrice = state.Price
		} else {
			if state.Price < s.AskPrice {
				s.AskPrice = state.Price
			}
		}

		if row.Size > maxSize {
			maxSize = row.Size
		}
	}

	s.MaxSize = maxSize
}
