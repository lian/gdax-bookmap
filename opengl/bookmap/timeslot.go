package bookmap

import (
	"fmt"
	"time"

	"github.com/lian/gdax-bookmap/orderbook"
)

type TimeSlotRow struct {
	Y          float64
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
}

func NewNewTimeSlot(from time.Time, to time.Time) *TimeSlot {
	v := &TimeSlot{
		From: from,
		To:   to,
		Rows: []*TimeSlotRow{},
	}
	return v
}

func NewTimeSlot(bookmap *Bookmap, from time.Time, to time.Time) *TimeSlot {
	v := &TimeSlot{
		From: from,
		To:   to,
		//ColumnWidth: bookmap.ColumnWidth,
		Rows: []*TimeSlotRow{},
	}

	rows := (bookmap.Texture.Height / bookmap.RowHeight)

	for i := 0.0; i < rows; i++ {
		y := i * bookmap.RowHeight

		heigh := bookmap.PriceScrollPosition - (i * bookmap.PriceSteps)
		low := heigh - bookmap.PriceSteps

		v.Rows = append(v.Rows, &TimeSlotRow{
			Y:     y,
			Low:   low,
			Heigh: heigh,
		})
	}

	return v
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

func (s *TimeSlot) GenerateRows(count, priceOffset, steps float64) {
	s.Rows = []*TimeSlotRow{}

	for i := 0.0; i < count; i++ {
		//y := i * bookmap.RowHeight

		heigh := priceOffset - (i * steps)
		low := heigh - steps

		s.Rows = append(s.Rows, &TimeSlotRow{
			//Y:     y,
			Low:   low,
			Heigh: heigh,
		})
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

	for i := len(stats.Bid) - 1; i >= 0; i-- {
		state := stats.Bid[i]
		row := s.FindRow(state.Price)

		if row == nil {
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

		if state.TradeSize > 0 {
			s.BidTradeSize += state.TradeSize
		}

		if row.Size > maxSize {
			maxSize = row.Size
		}
	}

	for _, state := range stats.Ask {
		row := s.FindRow(state.Price)

		if row == nil {
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

		if state.TradeSize > 0 {
			s.AskTradeSize += state.TradeSize
		}

		if row.Size > maxSize {
			maxSize = row.Size
		}
	}

	s.MaxSize = maxSize
}
