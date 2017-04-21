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
	"github.com/lian/gdax-bookmap/websocket"
)

type Graph struct {
	CurrentKey       []byte
	CurrentTime      time.Time
	Book             *orderbook.DbBook
	Timeslots        []*TimeSlot
	Width            int
	SlotWidth        int
	SlotCount        int
	SlotSteps        int
	Start            time.Time
	End              time.Time
	DB               *bolt.DB
	ProductID        string
	LastProcessedKey []byte
	Red              color.RGBA
	Green            color.RGBA
	Bg1              color.RGBA
	Fg1              color.RGBA
}

func NewGraph(db *bolt.DB, productID string, width, slotWidth, slotSteps int) *Graph {
	g := &Graph{
		ProductID: productID,
		DB:        db,
		Width:     width,
		SlotWidth: slotWidth,
		SlotCount: width / slotWidth,
		SlotSteps: slotSteps,
		Red:       color.RGBA{0xff, 0x69, 0x39, 0xff},
		Green:     color.RGBA{0x84, 0xf7, 0x66, 0xff},
		Bg1:       color.RGBA{0x15, 0x23, 0x2c, 0xff},
		Fg1:       color.RGBA{0xdd, 0xdf, 0xe1, 0xff},
	}
	return g
}

func (g *Graph) ClearSlotRows() {
	for _, slot := range g.Timeslots {
		slot.Rows = make([]*TimeSlotRow, 0, len(slot.Rows))
	}
}

func (g *Graph) SetStart(start time.Time) bool {
	g.Start = RoundTime(start, g.SlotSteps)

	startKey, book, err := g.FetchBook(g.Start)
	if err != nil {
		g.Book = nil
		fmt.Println("ERROR", "SetStart", err)
		return false
	}

	g.Book = book
	g.CurrentKey = startKey
	g.CurrentTime = websocket.UnpackTimeKey(startKey)
	g.Timeslots = make([]*TimeSlot, 0, g.SlotCount)

	return true
}

func (g *Graph) SetEnd(end time.Time) bool {
	if !end.After(g.Start) {
		fmt.Println("ERROR", "SetEnd", "end time before start time", g.Start, end)
		return false
	}
	//fmt.Println("Begin SetEnd", g.Start, end)

	g.End = end
	count := 0

	start := time.Now()

	for {
		count += 1
		newSlot, new, more, err := g.AddTimeslots(end)

		if err != nil {
			fmt.Println("ERROR", "SetEnd", err)
			break
		}

		if new {
			//fmt.Println("added new slot")
			if len(g.Timeslots) >= g.SlotCount {
				// remove and free first item
				copy(g.Timeslots[0:], g.Timeslots[1:])
				g.Timeslots[len(g.Timeslots)-1] = newSlot
			} else {
				g.Timeslots = append(g.Timeslots, newSlot)
			}
		} else {
			//fmt.Println("updated slot")
		}

		if !more || (time.Now().Sub(start).Seconds() >= 1.0) {
			//fmt.Println("done updating slots")
			break
		}
	}

	//fmt.Println("End SetEnd", count, len(g.Timeslots), g.SlotCount, string(g.CurrentKey), g.Start, end)
	return true
}

func RoundTime(t time.Time, steps int) time.Time {
	tmp := t.Unix()
	tmp += int64(steps) - int64(math.Mod(float64(tmp), float64(steps)))
	return time.Unix(tmp, 0)
}

func (g *Graph) AddTimeslots(end time.Time) (*TimeSlot, bool, bool, error) {

	if g.CurrentTime.After(end) {
		// break out for replays
		return nil, false, false, nil
	}

	curSlotStart := RoundTime(g.CurrentTime, g.SlotSteps)
	curSlotEnd := curSlotStart.Add(time.Duration(g.SlotSteps) * time.Second)

	new := true
	more := true
	if curSlotEnd.After(end) {
		// no more wanted
		more = false
	}

	//fmt.Println("Begin AddTimeslots", string(g.CurrentKey), curSlotStart, curSlotEnd, more)

	var slot *TimeSlot

	if len(g.Timeslots) == 0 {
		g.Book.Book.ResetStats()
		slot = NewNewTimeSlot(curSlotStart, curSlotEnd)
	} else {
		slot = g.Timeslots[len(g.Timeslots)-1]
		if slot.From == curSlotStart {
			new = false
		} else {
			g.Book.Book.ResetStats()
			slot = NewNewTimeSlot(curSlotStart, curSlotEnd)
		}
	}

	jumpNext := false
	nextSequence := g.Book.Book.Sequence + 1

	g.DB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(g.ProductID)).Cursor()

		c.Seek(g.LastProcessedKey)
		for {
			key, buf := c.Next()
			if key == nil {
				break
			}

			seq := websocket.UnpackSequence(buf)
			if seq != nextSequence {
				fmt.Println("graph out of sequence", string(g.CurrentKey), string(key), seq, nextSequence, websocket.UnpackPacket(buf)["type"])
				pkt := websocket.UnpackPacket(buf)
				if pkt["type"].(string) == "sync" {
					fmt.Println("found sync packet", seq, nextSequence)
					g.Book.Process(pkt)
				} else {
					fmt.Println("no sync packet. continue search", seq, nextSequence)
					more = true
				}
				g.LastProcessedKey = []byte(string(key))
				jumpNext = true
				break
			}

			if websocket.UnpackTimeKey(key).After(curSlotEnd) {
				//fmt.Println("in sequence defer", string(g.CurrentKey), string(key), seq, nextSequence, websocket.UnpackPacket(buf)["type"])
				more = true
				jumpNext = true
				g.CurrentTime = websocket.UnpackTimeKey(key)
				break
			} else {
				//fmt.Println("in sequence process", string(g.CurrentKey), string(key), seq, nextSequence, websocket.UnpackPacket(buf)["type"])
				g.Book.Process(websocket.UnpackPacket(buf))
				nextSequence = g.Book.Book.Sequence + 1
				g.LastProcessedKey = []byte(string(key))
			}
		}

		if !bytes.Equal(g.LastProcessedKey, g.CurrentKey) {
			g.CurrentKey = []byte(string(g.LastProcessedKey))
			g.CurrentTime = websocket.UnpackTimeKey(g.CurrentKey)
			slot.Stats = g.Book.Book.StatsCopy()
		} else {
			if jumpNext {
				if new {
					slot.Stats = g.Book.Book.StatsCopy()
				}
			} else {
				more = false
			}
		}

		return nil
	})

	if slot.Stats == nil {
		slot.Stats = g.Book.Book.StatsCopy()
	}

	//fmt.Println("End AddTimeslots")
	return slot, new, more, nil
}

func (g *Graph) FetchBook(from time.Time) ([]byte, *orderbook.DbBook, error) {
	//fmt.Println("Begin FetchBook")
	var err error
	book := orderbook.NewDbBook(g.ProductID)
	startKey := websocket.PackTimeKey(from)

	g.DB.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte(g.ProductID)).Cursor()

		first := true
		var key, buf []byte
		for key, buf = c.Seek(startKey); !bytes.HasPrefix(buf, []byte("\x04")); key, buf = c.Prev() {
			if first == false && key == nil {
				err = errors.New(fmt.Sprintf("FetchBook %s no sync key found", g.ProductID))
				return nil
			}
			first = false
		}

		// apply sync packet
		pkt := websocket.UnpackPacket(buf)
		book.Process(pkt)
		g.LastProcessedKey = []byte(string(key))

		// walk and fill book until startKey
		for key, buf = c.Next(); key != nil; key, buf = c.Next() {
			if bytes.Compare(key, startKey) < 0 {
				startKey = key
				pkt := websocket.UnpackPacket(buf)
				book.Process(pkt)
				g.LastProcessedKey = []byte(string(key))
			} else {
				break
			}
		}
		startKey = []byte(string(g.LastProcessedKey))

		return nil
	})

	if err == nil {
		fmt.Println("FetchBook", g.ProductID, "found startKey", string(startKey), book.Book.Sequence)
	}
	return startKey, book, err
}
