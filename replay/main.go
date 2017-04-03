package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/lian/gdax-bookmap/orderbook"
	"github.com/lian/gdax-bookmap/websocket"
)

const TimeFormat = "2006-01-02T15:04:05.999999Z07:00"

func PrintBook(book *orderbook.Book) {
	// clear terminal
	//fmt.Print("\033[H\033[2J")
	limit := 5

	bids, asks := book.StateCombined()
	fmt.Printf("\n= %s =====================================================\n", book.ID)
	fmt.Println("= asks =================================================")
	ask_limit := limit
	if len(asks) < ask_limit {
		ask_limit = len(asks)
	}
	asks_limited := asks[:ask_limit]

	for i := len(asks_limited) - 1; i >= 0; i-- {
		fmt.Printf("%.8f    %.2f\n", asks_limited[i].Size, asks_limited[i].Price)
	}
	fmt.Println("\n========================================================")

	bid_limit := limit
	if len(bids) < bid_limit {
		bid_limit = len(bids)
	}
	bids_limited := bids[:bid_limit]

	for _, s := range bids_limited {
		fmt.Printf("%.8f    %.2f\n", s.Size, s.Price)
	}
	fmt.Println("= bids =================================================")
}

func PrintStats(book *orderbook.Book) {
	// clear terminal
	//fmt.Print("\033[H\033[2J")
	limit := 5

	bids := book.Stats.Bid
	asks := book.Stats.Ask

	fmt.Printf("\n= %s ========================================= STATS =======\n", book.ID)
	fmt.Println("= asks =================================================")
	ask_limit := limit
	if len(asks) < ask_limit {
		ask_limit = len(asks)
	}
	asks_limited := asks[:ask_limit]

	for i := len(asks_limited) - 1; i >= 0; i-- {
		fmt.Printf("%.8f    %.2f\n", asks_limited[i].Size, asks_limited[i].Price)
	}
	fmt.Println("\n========================================================")

	bid_limit := limit
	if len(bids) < bid_limit {
		bid_limit = len(bids)
	}

	n := 0
	for i := len(bids) - 1; i >= 0; i-- {
		n += 1
		if n >= bid_limit {
			break
		}
		fmt.Printf("%.8f    %.2f\n", bids[i].Size, bids[i].Price)
	}
	fmt.Println("= bids =================================================")
}

func OpenDB(path string, buckets []string, readOnly bool) *bolt.DB {
	db, err := bolt.Open(path, 0600, &bolt.Options{ReadOnly: readOnly})
	if err != nil {
		log.Fatal(err)
	}

	db.Update(func(tx *bolt.Tx) error {
		for _, name := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return fmt.Errorf("create bucket: %s %s", name, err)
			}
		}
		return nil
	})

	return db
}

func stats(db *bolt.DB) {
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("BTC-USD"))
		c := b.Cursor()

		count := 0
		bytes := 0

		for k, v := c.First(); k != nil; k, v = c.Next() {
			count += 1
			bytes += len(v)
			//fmt.Printf("key=%s, value=%s\n", k, v)
		}

		first, _ := c.First()
		last, _ := c.Last()

		fmt.Println(string(first), string(last))
		fmt.Println(count, bytes)
		return nil
	})
}
func main() {
	products := []string{"BTC-USD"}
	db := OpenDB("/tmp/orderbooks.db", products, true)

	stats(db)

	return
	fmt.Println("find book")
	dbbook := orderbook.NewDbBook("BTC-USD")

	from, _ := time.Parse(TimeFormat, "2017-04-04T16:57:36.857194+02:00")
	to, _ := time.Parse(TimeFormat, "2017-04-04T16:57:38.857194+02:00")

	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("BTC-USD"))
		c := b.Cursor()

		fromKey := []byte(from.Format(TimeFormat))
		toKey := []byte(to.Format(TimeFormat))

		// find starting book point
		var prevSyncKey []byte
		for prevSyncKey, _ = c.Seek(fromKey); !bytes.HasSuffix(prevSyncKey, []byte("_sync")); prevSyncKey, _ = c.Prev() {
			//fmt.Printf("key=%s\n", k)
		}
		prevSyncPkt := websocket.UnpackPacket(b.Get(prevSyncKey))
		fmt.Printf("found prevSyncKey=%s\n", prevSyncKey)
		dbbook.Process(prevSyncPkt)
		PrintBook(dbbook.Book)
		fmt.Println("|||||||||||||||||||||||||||||||||||||||||||||")

		// fill book until from starting point
		var nextFillKey, buf []byte
		for nextFillKey, buf = c.Next(); bytes.Compare(nextFillKey, fromKey) <= 0; nextFillKey, buf = c.Next() {
			nextPkt := websocket.UnpackPacket(buf)
			//fmt.Printf("found nextFillKey=%s\n", nextFillKey)
			dbbook.Process(nextPkt)
			//PrintBook(dbbook.Book)
		}

		firstFromKey := nextFillKey
		fmt.Printf("found firstFromKey=%s\n", firstFromKey)
		nextPkt := websocket.UnpackPacket(buf)
		dbbook.Process(nextPkt)
		dbbook.Book.ResetStats()

		var nextKey []byte
		for nextKey, buf = c.Next(); bytes.Compare(nextKey, toKey) <= 0; nextKey, buf = c.Next() {
			nextPkt := websocket.UnpackPacket(buf)
			dbbook.Process(nextPkt)
		}

		PrintBook(dbbook.Book)
		fmt.Println("|||||||||||||||||||||||||||||||||||||||||||||")
		PrintStats(dbbook.Book)
		fmt.Println("|||||||||||||||||||||||||||||||||||||||||||||")

		return nil
	})
}
