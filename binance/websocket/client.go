package websocket

// https://github.com/binance-exchange/binance-official-api-docs/blob/master/web-socket-streams.md
// https://github.com/binance-exchange/binance-official-api-docs/blob/master/rest-api.md

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
	"github.com/lian/gdax-bookmap/binance/orderbook"
)

type Client struct {
	Socket      *websocket.Conn
	Products    []string
	Books       map[string]*orderbook.Book
	ConnectedAt time.Time
	DB          *bolt.DB
	dbEnabled   bool
	LastSync    time.Time
}

func New(bookUpdated, tradesUpdated chan string) *Client {
	c := &Client{
		Products:  []string{},
		Books:     map[string]*orderbook.Book{},
		dbEnabled: true,
	}

	// https://api.binance.com/api/v1/exchangeInfo

	products := []string{"BTC-USDT"}

	for _, name := range products {
		c.AddProduct(name)
	}

	path := "orderbooks.db"
	if os.Getenv("DB_PATH") != "" {
		path = os.Getenv("DB_PATH")
	}

	if c.dbEnabled {
		buckets := []string{}
		for _, name := range products {
			info := orderbook.FetchProductInfo(name)
			buckets = append(buckets, info.DatabaseKey)
		}
		c.DB = OpenDB(path, buckets, false)
	}

	return c
}

func streamNames(name string) (string, string) {
	return name + "@depth", name + "@aggTrade"
}

func (c *Client) AddProduct(name string) {
	c.Products = append(c.Products, name)
	book := orderbook.New(name)
	info := orderbook.FetchProductInfo(name)
	a, b := streamNames(strings.ToLower(info.ID))
	c.Books[a] = book
	c.Books[b] = book
}

func (c *Client) Connect() {
	streams := []string{}
	for _, name := range c.Products {
		info := orderbook.FetchProductInfo(name)
		a, b := streamNames(strings.ToLower(info.ID))
		streams = append(streams, a)
		streams = append(streams, b)
	}
	url := "wss://stream.binance.com:9443/stream?streams=" + strings.Join(streams, "/")

	fmt.Println("connect to websocket", url)
	s, _, err := websocket.DefaultDialer.Dial(url, nil)
	c.Socket = s

	if err != nil {
		log.Fatal("dial:", err)
	}

	c.ConnectedAt = time.Now()
}

type PacketHeader struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type PacketEventHeader struct {
	EventType string `json:"e"`
	EventTime int    `json:"E"`
	//Symbol        string        `json:"s"`
}

type PacketDepthUpdate struct {
	//EventType     string        `json:"e"`
	//EventTime     int           `json:"E"`
	//Symbol        string        `json:"s"`
	FirstUpdateID uint64        `json:"U"`
	FinalUpdateID uint64        `json:"u"`
	Bids          []interface{} `json:"b"` // [ "price", "quantity", []]
	Asks          []interface{} `json:"a"` // [ "price", "quantity", []]
}

type PacketAggTrade struct {
	//EventType        string `json:"e"`
	//EventTime        int    `json:"E"`
	//Symbol           string `json:"s"`
	//AggregateTradeID int    `json:"a"`
	//TradeTime        int    `json:"T"`
	Price         string `json:"p"`
	Quantity      string `json:"q"`
	FirstUpdateID int    `json:"f"`
	FinalUpdateID int    `json:"l"`
	BuyMaker      bool   `json:"m"`
	Ignore        bool   `json:"M"`
}

func (c *Client) UpdateSync(book *orderbook.Book, first, last uint64) error {
	seq := book.Sequence
	next := seq + 1

	if first <= seq {
		return fmt.Errorf("Ignore old messages %d %d", last, seq)
	}

	if book.Synced {
		if first != next {
			c.SyncBook(book)
			return fmt.Errorf("Message lost, resync")
		}
	} else {
		if (first <= next) && (last >= next) {
			book.Synced = true
		}
	}

	book.Sequence = last
	return nil
}

func (c *Client) WriteDB(now time.Time, book *orderbook.Book, buf []byte) {
	c.DB.Update(func(tx *bolt.Tx) error {
		var err error
		var key []byte
		b := tx.Bucket([]byte(book.ProductInfo.DatabaseKey))
		b.FillPercent = 0.9

		nano := now.UnixNano()
		// windows system clock resolution https://github.com/golang/go/issues/8687
		for {
			key = PackUnixNanoKey(nano)
			if b.Get(key) == nil {
				break
			} else {
				nano += 1
			}
		}

		err = b.Put(key, buf)
		if err != nil {
			fmt.Println("WriteDB Error", err)
		}
		return err
	})
}

func (c *Client) HandleMessage(book *orderbook.Book, raw json.RawMessage) {
	var event PacketEventHeader
	if err := json.Unmarshal(raw, &event); err != nil {
		log.Println("PacketEventType-parse:", err)
		return
	}

	eventTime := time.Unix(0, int64(event.EventTime)*int64(time.Millisecond))

	switch event.EventType {
	case "depthUpdate":
		var depthUpdate PacketDepthUpdate
		if err := json.Unmarshal(raw, &depthUpdate); err != nil {
			log.Println("PacketDepthUpdate-parse:", err)
			return
		}

		if err := c.UpdateSync(book, uint64(depthUpdate.FirstUpdateID), uint64(depthUpdate.FinalUpdateID)); err != nil {
			fmt.Println(err)
			return
		}

		for _, d := range depthUpdate.Bids {
			data := d.([]interface{})
			price, _ := strconv.ParseFloat(data[0].(string), 64)
			quantity, _ := strconv.ParseFloat(data[1].(string), 64)
			book.UpdateBidLevel(eventTime, price, quantity)
		}

		for _, d := range depthUpdate.Asks {
			data := d.([]interface{})
			price, _ := strconv.ParseFloat(data[0].(string), 64)
			quantity, _ := strconv.ParseFloat(data[1].(string), 64)
			book.UpdateAskLevel(eventTime, price, quantity)
		}

		if c.dbEnabled {
			now := time.Now()
			if time.Since(c.LastSync) > (time.Minute * 1) {
				c.LastSync = now
				c.WriteDB(now, book, PackSync(book))
			} else {
				c.WriteDB(now, book, PackDiff(&depthUpdate))
			}
		}

	case "aggTrade":
		var trade PacketAggTrade
		if err := json.Unmarshal(raw, &trade); err != nil {
			log.Println("PacketDepthUpdate-parse:", err)
			return
		}

		price, _ := strconv.ParseFloat(trade.Price, 64)
		quantity, _ := strconv.ParseFloat(trade.Quantity, 64)

		side := book.GetSide(price)
		book.AddTrade(eventTime, side, price, quantity)

		if c.dbEnabled {
			now := time.Now()
			c.WriteDB(now, book, PackTrade(side, price, quantity))
		}

	default:
		fmt.Println("unkown event", book.ID, event.EventType, string(raw))
		return
	}
}

func (c *Client) Run() {
	for {
		c.run()
	}
}

func (c *Client) GetBook(id string) *orderbook.Book {
	info := orderbook.FetchProductInfo(id)
	key := strings.ToLower(info.ID) + "@depth"
	return c.Books[key]
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
			for k, book := range c.Books {
				if strings.Contains(k, "@depth") {
					if err := c.SyncBook(book); err != nil {
						fmt.Println("initialSync-error", err)
					}
				}
			}
			initialSync = false
			continue
		}

		var pkt PacketHeader
		if err := json.Unmarshal(message, &pkt); err != nil {
			log.Println("PacketHeader-parse:", err)
			continue
		}

		var book *orderbook.Book
		var ok bool
		if book, ok = c.Books[pkt.Stream]; !ok {
			log.Println("book not found", pkt.Stream)
			continue
		}

		c.HandleMessage(book, pkt.Data)
	}
}
