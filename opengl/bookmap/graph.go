package bookmap

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/boltdb/bolt"
	"github.com/lian/gdax-bookmap/orderbook"
)

type Graph struct {
	CurrentTime time.Time
	Book        *orderbook.Book
	Timeslots   []*TimeSlot
	Width       int
	Height      int
	SlotWidth   int
	SlotCount   int
	SlotSteps   int
	Start       time.Time
	End         time.Time
	DB          *bolt.DB
	ProductID   string
	Red         color.RGBA
	Green       color.RGBA
	Bg1         color.RGBA
	Fg1         color.RGBA
	CurrentSlot *TimeSlot
	NoTimeout   bool
}

func NewGraph(db *bolt.DB, productID string, width, height, slotWidth, slotSteps int) *Graph {
	g := &Graph{
		ProductID: productID,
		DB:        db,
		Width:     width,
		Height:    height,
		SlotWidth: slotWidth,
		SlotCount: width / slotWidth,
		SlotSteps: slotSteps,
		Red:       color.RGBA{0xff, 0x69, 0x39, 0xff},
		Green:     color.RGBA{0x84, 0xf7, 0x66, 0xff},
		Bg1:       color.RGBA{0x15, 0x23, 0x2c, 0xff},
		Fg1:       color.RGBA{0xdd, 0xdf, 0xe1, 0xff},
		Book:      orderbook.New(productID),
	}
	return g
}

func (g *Graph) MaxHistoSize() float64 {
	var max float64
	for _, slot := range g.Timeslots {
		if slot.MaxSize > max {
			max = slot.MaxSize
		}
	}
	return max
}

func (g *Graph) ClearSlotRows() {
	for _, slot := range g.Timeslots {
		slot.ClearRows()
	}
}

func (g *Graph) SetStart(start time.Time) bool {
	var err error
	g.Start = RoundTime(start, g.SlotSteps)

	g.CurrentTime, g.Book, err = g.FetchBook(g.Start)
	if err != nil {
		g.Book = nil
		fmt.Println("ERROR", "SetStart", err)
		return false
	}
	g.Timeslots = make([]*TimeSlot, 0, g.SlotCount)

	return true
}

func (g *Graph) SetEnd(end time.Time) bool {
	if !end.After(g.Start) {
		fmt.Println("ERROR", "SetEnd", "end time before start time", g.Start, end)
		return false
	}

	g.End = end
	g.GenerateTimeslots(end)
	g.ProcessTimeslots()

	return true
}

func (g *Graph) GenerateTimeslots(end time.Time) {
	end = RoundTime(end, g.SlotSteps)

	//lastStart := end.Add(time.Duration(-g.SlotSteps) * time.Second)
	lastStart := g.Start
	if len(g.Timeslots) != 0 {
		lastStart = g.Timeslots[len(g.Timeslots)-1].To
	}

	var slot *TimeSlot
	for {
		if lastStart == end {
			break
		}

		lastEnd := lastStart.Add(time.Duration(g.SlotSteps) * time.Second)
		slot = NewTimeSlot(lastStart, lastEnd)

		if len(g.Timeslots) >= g.SlotCount {
			// remove and free first item
			copy(g.Timeslots[0:], g.Timeslots[1:])
			g.Timeslots[len(g.Timeslots)-1] = slot
		} else {
			if len(g.Timeslots) == 0 {
				g.CurrentSlot = slot
				fmt.Println(g.ProductID, "start new timeslots", slot.From, slot.To)
			}
			g.Timeslots = append(g.Timeslots, slot)
		}

		lastStart = lastEnd
	}
}

func (g *Graph) NextSlot(t time.Time) *TimeSlot {
	nano := t.UnixNano()
	for _, s := range g.Timeslots {
		if nano > s.From.UnixNano() && nano <= s.To.UnixNano() {
			return s
		}
	}
	return nil
}

func (g *Graph) ProcessTimeslots() {
	firstTime := g.Timeslots[0].From
	lastTime := g.Timeslots[len(g.Timeslots)-1].To
	//fmt.Println(g.ProductID, "ProcessTimeslots", firstTime, lastTime)

	if g.CurrentTime.After(lastTime) {
		fmt.Println("g.CurrentTime.After(lastTime)")
		return
	}

	var slot *TimeSlot
	var updateStats bool

	processingStart := time.Now()

	g.DB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(g.ProductID)).Cursor()

		c.Seek(orderbook.PackTimeKey(g.CurrentTime))
		for {
			if !g.NoTimeout && time.Now().Sub(processingStart).Seconds() >= 1.0 {
				fmt.Println(g.ProductID, "defer processing", g.CurrentTime)
				break
			}

			key, buf := c.Next()
			if key == nil {
				break
			}

			t := orderbook.UnpackTimeKey(key)

			// after our wanted range
			if t.After(lastTime) {
				fmt.Println(g.ProductID, "after wanted range", t, lastTime)
				break
			}

			// before our wanted range, process it and move on
			if t.Before(firstTime) {
				//fmt.Println(g.ProductID, "before wanted range", t, firstTime)
				g.CurrentTime = t
				g.Book.Process(t, buf)
				g.Book.ResetStats()
				continue
			}

			slot = g.CurrentSlot

			// move to next slow
			if t.After(slot.To) {
				slot.Stats = g.Book.StatsCopy()
				g.CurrentSlot = g.NextSlot(t)
				if g.NoTimeout {
					fmt.Println("moved to next slot", g.CurrentSlot.From, g.CurrentSlot.To)
				}
				/*
					if g.CurrentSlot == nil {
						fmt.Println(g.ProductID, "next slot nil", lastTime)
						g.CurrentSlot = slot
						break
					}
				*/
				g.Book.ResetStats()
				g.CurrentTime = t
				g.Book.Process(t, buf)
				g.CurrentSlot.Stats = g.Book.StatsCopy()
			} else {
				g.CurrentTime = t
				g.Book.Process(t, buf)

				if slot.Stats == nil {
					slot.Stats = g.Book.StatsCopy()
				} else {
					updateStats = true
				}
			}

		}
		return nil
	})

	if updateStats {
		g.CurrentSlot.Stats = g.Book.StatsCopy()
	}
}

func RoundTime(t time.Time, steps int) time.Time {
	tmp := t.Unix()
	tmp += int64(steps) - int64(math.Mod(float64(tmp), float64(steps)))
	return time.Unix(tmp, 0)
}

func (g *Graph) FetchBook(from time.Time) (time.Time, *orderbook.Book, error) {
	//fmt.Println("Begin FetchBook")
	var err error
	book := orderbook.New(g.ProductID)
	startKey := orderbook.PackTimeKey(from)

	g.DB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(g.ProductID)).Cursor()

		first := true
		var key, buf []byte
		for key, buf = c.Seek(startKey); !bytes.HasPrefix(buf, []byte("\x00")); key, buf = c.Prev() {
			if first == false && key == nil {
				err = errors.New(fmt.Sprintf("FetchBook %s no sync key found", g.ProductID))
				return nil
			}
			first = false
		}

		// apply sync packet
		book.Process(orderbook.UnpackTimeKey(key), buf)
		LastProcessedKey := []byte(string(key))

		// walk and fill book until startKey
		for key, buf = c.Next(); key != nil; key, buf = c.Next() {
			if bytes.Compare(key, startKey) < 0 {
				startKey = key
				book.Process(orderbook.UnpackTimeKey(key), buf)
				LastProcessedKey = []byte(string(key))
			} else {
				break
			}
		}
		startKey = LastProcessedKey

		return nil
	})

	if err == nil {
		fmt.Println(g.ProductID, "FetchBook", "found start", orderbook.UnpackTimeKey(startKey))
	}
	book.ResetStats()

	return orderbook.UnpackTimeKey(startKey), book, err
}
