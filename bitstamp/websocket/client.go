package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"

	"github.com/lian/gdax-bookmap/bitstamp/orderbook"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

type Client struct {
	Products      []string
	Books         map[string]*orderbook.Book
	ChannelLookup map[string]string
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
		Products:      []string{},
		Books:         map[string]*orderbook.Book{},
		ChannelLookup: map[string]string{},
		BatchWrite:    map[string]*util.BookBatchWrite{},
		DB:            db,
		Infos:         []*product_info.Info{},
	}

	if c.DB != nil {
		c.dbEnabled = true
	}

	//products := []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD", "XRP-USD"}
	products := []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD"}
	for _, name := range products {
		c.AddProduct(name)
	}

	if c.dbEnabled {
		buckets := []string{}
		for _, info := range c.Infos {
			buckets = append(buckets, info.DatabaseKey)
		}
		util.CreateBucketsDB(db, buckets)
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
}

func (c *Client) Connect() {
	fmt.Println("connect to websocket")
	url := "wss://ws.pusherapp.com/app/de504dc5763aeef9ff52?protocol=7&client=js&version=2.1.6&flash=false"
	s, _, err := websocket.DefaultDialer.Dial(url, nil)
	c.Socket = s

	if err != nil {
		log.Fatal("dial:", err)
	}

	for _, book := range c.Books {
		a, b := c.GetChannelNames(book)
		c.ChannelLookup[a] = book.ID
		c.ChannelLookup[b] = book.ID
		c.Subscribe(a)
		c.Subscribe(b)
	}
}

func (c *Client) Subscribe(channel string) {
	a := map[string]interface{}{"event": "pusher:subscribe", "data": map[string]interface{}{"channel": channel}}
	c.Socket.WriteJSON(a)
}

func (c *Client) GetChannelNames(book *orderbook.Book) (string, string) {
	if book.ID == "BTC-USD" {
		return "diff_order_book", "live_trades"
	} else {
		return fmt.Sprintf("diff_order_book_%s", book.WebsocketID), fmt.Sprintf("live_trades_%s", book.WebsocketID)
	}
}

type Packet struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
	Data    string `json:"data"`
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

func (c *Client) UpdateSync(book *orderbook.Book, last uint64) error {
	seq := book.Sequence
	/*
		if book.ID == "BTC-USD" {
			fmt.Println("UpdateSync", book.ID, seq, last, last < seq)
		}
	*/

	if last < seq {
		return fmt.Errorf("Ignore old messages %d %d", last, seq)
	}

	book.Sequence = last
	return nil
}

func (c *Client) HandleMessage(book *orderbook.Book, pkt Packet) {
	eventTime := time.Now()
	var trade *orderbook.Trade

	switch pkt.Event {
	case "data":
		//fmt.Println("diff", book.ID, string(pkt.Data))

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(pkt.Data), &data); err != nil {
			log.Println(err)
			return
		}
		seq, _ := strconv.ParseInt(data["timestamp"].(string), 10, 64)

		if err := c.UpdateSync(book, uint64(seq)); err != nil {
			fmt.Println(err)
			return
		}

		for _, d := range data["bids"].([]interface{}) {
			data := d.([]interface{})
			price, _ := strconv.ParseFloat(data[0].(string), 64)
			size, _ := strconv.ParseFloat(data[1].(string), 64)
			book.UpdateBidLevel(eventTime, price, size)
		}

		for _, d := range data["asks"].([]interface{}) {
			data := d.([]interface{})
			price, _ := strconv.ParseFloat(data[0].(string), 64)
			size, _ := strconv.ParseFloat(data[1].(string), 64)
			book.UpdateAskLevel(eventTime, price, size)
		}

	case "trade":
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(pkt.Data), &data); err != nil {
			log.Println(err)
			return
		}

		price, _ := strconv.ParseFloat(data["price_str"].(string), 64)
		size, _ := strconv.ParseFloat(data["amount_str"].(string), 64)
		side := book.GetSide(price)

		book.AddTrade(eventTime, side, price, size)
		trade = book.Trades[len(book.Trades)-1]

	default:
		fmt.Println("unkown event", book.ID, pkt.Event, string(pkt.Data))
		return
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
				//fmt.Println("STORE DIFF", book.ID, batch.Count)
				c.WriteDiff(batch, book, now)
			}
		}
	}
}

func (c *Client) WriteDiff(batch *util.BookBatchWrite, book *orderbook.Book, now time.Time) {
	book.FixBookLevels() // TODO fix/remove
	diff := book.Diff
	if len(diff.Bid) != 0 || len(diff.Ask) != 0 {
		pkt := PackDiff(batch.LastDiffSeq, book.Sequence, diff)
		c.WriteDB(now, book, pkt)
		book.ResetDiff()
		batch.LastDiffSeq = book.Sequence + 1
	}
}

func (c *Client) WriteSync(batch *util.BookBatchWrite, book *orderbook.Book, now time.Time) {
	book.FixBookLevels() // TODO fix/remove
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

	for {
		msgType, message, err := c.Socket.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}

		if msgType != websocket.TextMessage {
			continue
		}

		var pkt Packet
		if err := json.Unmarshal(message, &pkt); err != nil {
			log.Println("header-parse:", err)
			continue
		}

		switch pkt.Event {
		// pusher stuff
		case "pusher:connection_established":
			log.Println("Connected")
			continue
		case "pusher_internal:subscription_succeeded":
			log.Println("Subscribed")
			continue
		case "pusher:pong":
			// ignore
			continue
		case "pusher:ping":
			c.Socket.WriteJSON(map[string]interface{}{"event": "pusher:pong"})
			continue
		}

		var book *orderbook.Book
		if id, ok := c.ChannelLookup[pkt.Channel]; ok {
			if book, ok = c.Books[id]; !ok {
				log.Println("book not found", pkt.Channel, id)
				continue
			}
		} else {
			log.Println("book lookup failed", pkt.Channel)
			continue
		}

		if book.Sequence == 0 {
			c.SyncBook(book)
			continue
		}

		c.HandleMessage(book, pkt)
	}
}
