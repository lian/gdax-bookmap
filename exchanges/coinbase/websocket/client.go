package websocket

// https://docs.pro.coinbase.com/#websocket-feed

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	book_info "github.com/lian/gdax-bookmap/exchanges/coinbase/product_info"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
	"github.com/lian/gdax-bookmap/exchanges/common/orderbook"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

type Client struct {
	Socket      *websocket.Conn
	Products    []string
	Books       map[string]*orderbook.Book
	ConnectedAt time.Time
	DB          *bolt.DB
	dbEnabled   bool
	BatchWrite  map[string]*util.BookBatchWrite
	Infos       []*product_info.Info
}

func New(db *bolt.DB, products []string) *Client {
	c := &Client{
		Products:   []string{},
		Books:      map[string]*orderbook.Book{},
		BatchWrite: map[string]*util.BookBatchWrite{},
		DB:         db,
		Infos:      []*product_info.Info{},
	}
	if c.DB != nil {
		c.dbEnabled = true
	}

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

func (c *Client) AddProduct(name string) {
	c.Products = append(c.Products, name)
	c.BatchWrite[name] = &util.BookBatchWrite{Count: 0, Batch: []*util.BatchChunk{}}
	book := orderbook.New(name)
	info := book_info.FetchProductInfo(name)
	c.Infos = append(c.Infos, &info)
	book.SetProductInfo(info)
	c.Books[name] = book
}

func (c *Client) Connect() error {
	url := "wss://ws-feed.pro.coinbase.com"

	fmt.Println("connect to websocket", url)
	s, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return err
	}

	c.Socket = s
	c.ConnectedAt = time.Now()

	buf, _ := json.Marshal(map[string]interface{}{"type": "subscribe", "product_ids": c.Products, "channels": []string{"level2", "heartbeat", "ticker"}})
	err = c.Socket.WriteMessage(websocket.TextMessage, buf)

	return nil
}

type PacketHeader struct {
	Type      string `json:"type"`
	ProductID string `json:"product_id"`
}

type Snapshot struct {
	Bids [][]string `json:"bids"`
	Asks [][]string `json:"asks"`
}

type L2Update struct {
	Changes [][]string `json:"changes"`
}

type Ticker struct {
	Price    float64 `json:"price,string"`
	Quantity float64 `json:"last_size,string"`
}

type Heartbeat struct {
	Sequence uint64 `json:"sequence"`
}

func (c *Client) HandleMessage(book *orderbook.Book, header PacketHeader, message []byte) {
	var trade *orderbook.Trade
	now := time.Now()

	switch header.Type {
	case "snapshot":
		var s Snapshot
		if err := json.Unmarshal(message, &s); err != nil {
			log.Println("HandleMessage:", err)
			return
		}

		book.Clear()

		for _, data := range s.Bids {
			price, _ := strconv.ParseFloat(data[0], 64)
			size, _ := strconv.ParseFloat(data[1], 64)
			book.UpdateBidLevel(now, price, size)
		}

		for _, data := range s.Asks {
			price, _ := strconv.ParseFloat(data[0], 64)
			size, _ := strconv.ParseFloat(data[1], 64)
			book.UpdateAskLevel(now, price, size)
		}
	case "l2update":
		var s L2Update
		if err := json.Unmarshal(message, &s); err != nil {
			log.Println("HandleMessage:", err)
			return
		}

		for _, data := range s.Changes {
			price, _ := strconv.ParseFloat(data[1], 64)
			size, _ := strconv.ParseFloat(data[2], 64)
			if data[0] == "buy" {
				book.UpdateBidLevel(now, price, size)
			} else {
				book.UpdateAskLevel(now, price, size)
			}
		}

	case "ticker":
		var s Ticker
		if err := json.Unmarshal(message, &s); err != nil {
			log.Println("HandleMessage:", err)
			return
		}

		side := book.GetSide(s.Price)
		book.AddTrade(now, side, s.Price, s.Quantity)
		trade = book.Trades[len(book.Trades)-1]

	case "heartbeat":
		var s Heartbeat
		if err := json.Unmarshal(message, &s); err != nil {
			log.Println("HandleMessage:", err)
			return
		}

		book.Sequence = s.Sequence

	default:
		fmt.Println("unkown event", book.ID, now, string(message))
		return
	}

	if c.dbEnabled {
		batch := c.BatchWrite[book.ID]

		if trade != nil {
			batch.Write(c.DB, now, book.ProductInfo.DatabaseKey, orderbook.PackTrade(trade))
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
		pkt := orderbook.PackDiff(batch.LastDiffSeq, book.Sequence, diff)
		batch.Write(c.DB, now, book.ProductInfo.DatabaseKey, pkt)
		book.ResetDiff()
		batch.LastDiffSeq = book.Sequence + 1
	}
}

func (c *Client) WriteSync(batch *util.BookBatchWrite, book *orderbook.Book, now time.Time) {
	book.FixBookLevels() // TODO fix/remove
	batch.Write(c.DB, now, book.ProductInfo.DatabaseKey, orderbook.PackSync(book))
	book.ResetDiff()
	batch.LastDiffSeq = book.Sequence + 1
}

func (c *Client) Run() {
	for {
		c.run()
	}
}

func (c *Client) run() {
	if err := c.Connect(); err != nil {
		fmt.Println("failed to connect", err)
		time.Sleep(1000 * time.Millisecond)
		return
	}
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

		var header PacketHeader
		if err := json.Unmarshal(message, &header); err != nil {
			log.Println("header-parse:", err)
			continue
		}

		if header.Type == "subscriptions" {
			fmt.Println("Coinbase Websocket subscriptions", message)
			continue
		}

		var book *orderbook.Book
		var ok bool
		if book, ok = c.Books[header.ProductID]; !ok {
			log.Println("book not found", header.ProductID)
			continue
		}

		c.HandleMessage(book, header, message)
	}
}
