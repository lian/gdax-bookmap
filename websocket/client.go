package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"

	"github.com/lian/gdax-bookmap/orderbook"
)

const TimeFormat = "2006-01-02T15:04:05.999999Z07:00"

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

func New(products []string, bookUpdated, tradesUpdated chan string) *Client {
	c := &Client{
		BookUpdated:   bookUpdated,
		TradesUpdated: tradesUpdated,
		Products:      []string{},
		Books:         map[string]*orderbook.Book{},
		DBBatch:       []*BatchChunk{},
		DBBatchTime:   time.Now(),
	}

	for _, name := range products {
		c.AddProduct(name)
	}

	path := "gdax_orderbooks.db"
	if os.Getenv("GDAX_DB_PATH") != "" {
		path = os.Getenv("GDAX_DB_PATH")
	}

	if os.Getenv("GDAX_DB_READONLY") != "" {
		c.dbEnabled = false
	} else {
		c.dbEnabled = true
	}
	readonly := !c.dbEnabled

	c.DB = OpenDB(path, products, readonly)

	return c
}

type BatchChunk struct {
	Time time.Time
	Data []byte
}
type Client struct {
	BookUpdated   chan string
	TradesUpdated chan string
	Products      []string
	Books         map[string]*orderbook.Book
	Socket        *websocket.Conn
	DB            *bolt.DB
	DBCount       int
	DBLock        sync.Mutex
	DBBatch       []*BatchChunk
	DBBatchTime   time.Time
	dbEnabled     bool
}

func (c *Client) AddProduct(name string) {
	c.Products = append(c.Products, name)
	c.Books[name] = orderbook.New(name)
	c.Books[name].AlwaysSort = true
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

func (c *Client) BookChanged(book *orderbook.Book) {
	if c.BookUpdated == nil {
		return
	}

	c.BookUpdated <- book.ID
}

func UnpackTimeKey(key []byte) time.Time {
	i, _ := strconv.ParseInt(string(key), 10, 64)
	t := time.Unix(0, i)
	return t
}

func PackTimeKey(t time.Time) []byte {
	return []byte(fmt.Sprintf("%d", t.UnixNano()))
}

func PackUnixNanoKey(nano int64) []byte {
	return []byte(fmt.Sprintf("%d", nano))
}

func (c *Client) WriteDB(book *orderbook.Book, data map[string]interface{}) {
	c.DBCount += 1
	now := time.Now()

	var t time.Time
	if _, ok := data["time"].(string); ok {
		t, _ = time.Parse(TimeFormat, data["time"].(string))
	} else {
		var lastKey []byte
		c.DB.View(func(tx *bolt.Tx) error {
			lastKey, _ = tx.Bucket([]byte(book.ID)).Cursor().Last()
			return nil
		})
		if lastKey != nil {
			t = UnpackTimeKey(lastKey)
		} else {
			t = time.Now().Add(-1 * time.Second)
		}
	}

	c.DBBatch = append(c.DBBatch, &BatchChunk{Time: t, Data: PackPacket(data)})

	if now.Sub(c.DBBatchTime).Seconds() > 0.5 {
		c.DBBatchTime = now
		c.DB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(book.ID))
			b.FillPercent = 0.9
			var err error
			var key []byte
			for _, chunk := range c.DBBatch {
				nano := chunk.Time.UnixNano()
				// hack because gdax api returns 2017-04-06T05:11:37.608000Z
				// instead of docs-stated precision 2014-11-09T08:19:27.028459Z
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
		c.DBBatch = []*BatchChunk{}
	}
}

func (c *Client) HandleMessage(book *orderbook.Book, header PacketHeader, message []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Fatal(err)
	}

	var writeDB bool

	switch header.Type {
	case "received":
		// skip
		break
	case "open":
		writeDB = true
		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		size, _ := strconv.ParseFloat(data["remaining_size"].(string), 64)

		book.Add(map[string]interface{}{
			"id":    data["order_id"].(string),
			"side":  data["side"].(string),
			"price": price,
			"size":  size,
			//"time":           data["time"].(string),
		})
		c.BookChanged(book)

		break
	case "done":
		writeDB = true
		book.Remove(data["order_id"].(string))
		c.BookChanged(book)
		break
	case "match":
		writeDB = true
		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		size, _ := strconv.ParseFloat(data["size"].(string), 64)

		book.Match(map[string]interface{}{
			"size":           size,
			"price":          price,
			"side":           data["side"].(string),
			"maker_order_id": data["maker_order_id"].(string),
			"taker_order_id": data["taker_order_id"].(string),
			//"time":           data["time"].(string),
		}, false)
		c.BookChanged(book)
		break
	case "change":
		if _, ok := book.OrderMap[data["order_id"].(string)]; !ok {
			// if we don't know about the order, it is a change message for a received order
		} else {
			writeDB = true
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
			c.BookChanged(book)
		}
		break
	}

	if c.dbEnabled {
		if math.Mod(float64(c.DBCount), 10000) != 0 {
			if writeDB {
				c.WriteDB(book, data)
			} else {
				c.WriteDB(book, map[string]interface{}{
					"type":     "ignore",
					"sequence": data["sequence"],
					"time":     data["time"],
				})
			}
		} else {
			fmt.Println("==> STORE NEW SYNC")

			bids := [][]interface{}{}
			for _, level := range book.Bid {
				for _, order := range level.Orders {
					bids = append(bids, []interface{}{order.Price, order.Size, order.ID})
				}
			}

			asks := [][]interface{}{}
			for _, level := range book.Ask {
				for _, order := range level.Orders {
					asks = append(asks, []interface{}{order.Price, order.Size, order.ID})
				}
			}

			c.WriteDB(book, map[string]interface{}{
				"type":     "sync_new",
				"sequence": data["sequence"],
				"time":     data["time"],
				"bids":     bids,
				"asks":     asks,
			})
		}
	}
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
