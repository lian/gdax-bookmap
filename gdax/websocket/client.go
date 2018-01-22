package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"

	"github.com/lian/gdax-bookmap/gdax/orderbook"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

type Client struct {
	BookUpdated   chan string
	TradesUpdated chan string
	Products      []string
	Books         map[string]*orderbook.Book
	Socket        *websocket.Conn
	DB            *bolt.DB
	dbEnabled     bool
	LastSync      time.Time
	LastDiff      time.Time
	LastDiffSeq   uint64
	BatchWrite    map[string]*util.BookBatchWrite
	Infos         []*product_info.Info
}

func New(db *bolt.DB, bookUpdated, tradesUpdated chan string) *Client {
	c := &Client{
		BookUpdated:   bookUpdated,
		TradesUpdated: tradesUpdated,
		Products:      []string{},
		Books:         map[string]*orderbook.Book{},
		BatchWrite:    map[string]*util.BookBatchWrite{},
		DB:            db,
		Infos:         []*product_info.Info{},
	}
	if c.DB != nil {
		c.dbEnabled = true
	}

	//products := []string{"BTC-USD", "BTC-EUR", "LTC-USD", "ETH-USD", "ETH-BTC", "LTC-BTC", "BCH-USD", "BCH-BTC"}
	products := []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD"}
	for _, name := range products {
		c.AddProduct(name)
	}

	if c.dbEnabled {
		buckets := []string{}
		for _, info := range c.Infos {
			buckets = append(buckets, info.DatabaseKey)
		}
		util.CreateBucketsDB(c.DB, buckets)
	}

	return c
}

func (c *Client) GetBook(id string) *orderbook.Book {
	return c.Books[id]
}

func (c *Client) AddProduct(name string) {
	c.Products = append(c.Products, name)
	c.Books[name] = orderbook.New(name)
	c.BatchWrite[name] = &util.BookBatchWrite{Count: 0, Batch: []*util.BatchChunk{}}
	info := orderbook.FetchProductInfo(name)
	c.Infos = append(c.Infos, &info)

	if c.TradesUpdated != nil {
		c.Books[name].TradesUpdated = c.TradesUpdated
	}
}

func (c *Client) Connect() {
	fmt.Println("connect to websocket")
	s, _, err := websocket.DefaultDialer.Dial("wss://ws-feed.gdax.com", nil)
	c.Socket = s

	if err != nil {
		log.Fatal("dial:", err)
	}

	buf, _ := json.Marshal(map[string]interface{}{"type": "subscribe", "product_ids": c.Products})
	err = c.Socket.WriteMessage(websocket.TextMessage, buf)
}

type PacketHeader struct {
	Type      string `json:"type"`
	Sequence  uint64 `json:"sequence"`
	ProductID string `json:"product_id"`
}

func (c *Client) WriteDB(now time.Time, book *orderbook.Book, buf []byte) {
	batch := c.BatchWrite[book.ID]
	batch.AddChunk(&util.BatchChunk{Time: now, Data: buf})

	if batch.FlushBatch(now) {
		c.DB.Update(func(tx *bolt.Tx) error {
			var err error
			var key []byte
			b := tx.Bucket([]byte(book.ProductInfo.DatabaseKey))
			b.FillPercent = 0.9
			for _, chunk := range batch.Batch {
				nano := chunk.Time.UnixNano()
				// windows system clock resolution https://github.com/golang/go/issues/8687
				for {
					key = PackUnixNanoKey(nano)
					if b.Get(key) == nil {
						break
					} else {
						nano += 1
					}
				}
				err = b.Put(key, chunk.Data)
				if err != nil {
					fmt.Println("HandleMessage DB Error", err)
				}
			}
			return err
		})
		//fmt.Println("flush batch chunks", len(batch.Batch))
		batch.Clear()
	}
}

func (c *Client) HandleMessage(book *orderbook.Book, header PacketHeader, message []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Fatal(err)
	}

	var trade *orderbook.Order

	switch header.Type {
	case "received":
		// skip
	case "open":
		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		size, _ := strconv.ParseFloat(data["remaining_size"].(string), 64)

		book.Add(map[string]interface{}{
			"id":    data["order_id"].(string),
			"side":  data["side"].(string),
			"price": price,
			"size":  size,
			//"time":           data["time"].(string),
		})
	case "done":
		book.Remove(data["order_id"].(string))
	case "match":
		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		size, _ := strconv.ParseFloat(data["size"].(string), 64)

		book.Match(map[string]interface{}{
			"size":           size,
			"price":          price,
			"side":           data["side"].(string),
			"maker_order_id": data["maker_order_id"].(string),
			"taker_order_id": data["taker_order_id"].(string),
			"time":           data["time"].(string),
		}, false)
		trade = book.Trades[len(book.Trades)-1]

	case "change":
		if _, ok := book.OrderMap[data["order_id"].(string)]; !ok {
			// if we don't know about the order, it is a change message for a received order
		} else {
			// change messages are treated as match messages
			old_size, _ := strconv.ParseFloat(data["old_size"].(string), 64)
			new_size, _ := strconv.ParseFloat(data["new_size"].(string), 64)
			price, _ := strconv.ParseFloat(data["price"].(string), 64)
			size_delta := old_size - new_size

			book.Match(map[string]interface{}{
				"size":           size_delta,
				"price":          price,
				"side":           data["side"].(string),
				"maker_order_id": data["order_id"].(string),
				//"time":           data["time"].(string),
			}, true)
		}
	}

	if c.dbEnabled {
		batch := c.BatchWrite[book.ID]
		now := time.Now()
		if trade != nil {
			c.WriteDB(now, book, PackTrade(trade))
		}

		if batch.NextSync(now) {
			fmt.Println("STORE SYNC", book.ID, batch.Count)
			c.WriteSync(batch, book, now)
		} else {
			if batch.NextDiff(now) {
				c.WriteDiff(batch, book, now)
			}
		}
	}
}

func (c *Client) WriteDiff(batch *util.BookBatchWrite, book *orderbook.Book, now time.Time) {
	diff := book.Diff
	if len(diff.Bid) != 0 || len(diff.Ask) != 0 {
		pkt := PackDiff(batch.LastDiffSeq, book.Sequence, diff)
		c.WriteDB(now, book, pkt)
		book.ResetDiff()
		batch.LastDiffSeq = book.Sequence + 1
	}
}

func (c *Client) WriteSync(batch *util.BookBatchWrite, book *orderbook.Book, now time.Time) {
	c.WriteDB(now, book, PackSync(book))
	book.ResetDiff()
	batch.LastDiffSeq = book.Sequence + 1
}

func (c *Client) Run() {
	for {
		c.run()
	}
}

func (c *Client) run() {
	c.Connect()
	defer c.Socket.Close()

	initialSync := true

	for {
		msgType, message, err := c.Socket.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		if msgType != websocket.TextMessage {
			continue
		}

		if initialSync {
			for _, book := range c.Books {
				SyncBook(book, c)
			}
			initialSync = false
			continue
		}

		var header PacketHeader
		if err := json.Unmarshal(message, &header); err != nil {
			log.Println("header-parse:", err)
			continue
		}

		var book *orderbook.Book
		var ok bool
		if book, ok = c.Books[header.ProductID]; !ok {
			log.Println("book not found", header.ProductID)
			continue
		}

		if header.Sequence <= book.Sequence {
			// Ignore old messages
			continue
		}

		if header.Sequence != (book.Sequence + 1) {
			// Message lost, resync
			SyncBook(book, c)
			continue
		}

		book.Sequence = header.Sequence

		c.HandleMessage(book, header, message)
	}
}
