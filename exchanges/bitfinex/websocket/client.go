package websocket

// https://github.com/thrasher-/gocryptotrader/blob/master/exchanges/bitfinex/bitfinex_websocket.go
// api version 2: https://docs.bitfinex.com/v2/reference#ws-public-order-books

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
	book_info "github.com/lian/gdax-bookmap/exchanges/bitfinex/product_info"
	"github.com/lian/gdax-bookmap/exchanges/common/orderbook"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

type Client struct {
	Platform      string
	Socket        *websocket.Conn
	Products      []string
	Books         map[string]*orderbook.Book
	ConnectedAt   time.Time
	DB            *bolt.DB
	dbEnabled     bool
	BatchWrite    map[string]*util.BookBatchWrite
	Infos         []*product_info.Info
	Subscriptions map[int]SubscriptionInfo
}

func New(db *bolt.DB, products []string) *Client {
	c := &Client{
		Platform:      "Bitfinex",
		Products:      []string{},
		Books:         map[string]*orderbook.Book{},
		BatchWrite:    map[string]*util.BookBatchWrite{},
		DB:            db,
		Infos:         []*product_info.Info{},
		Subscriptions: map[int]SubscriptionInfo{},
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
	id := fmt.Sprintf("t%s%s", info.BaseCurrency, info.QuoteCurrency)
	c.Books[id] = book
}

type WebsocketHandshake struct {
	Event   string  `json:"event"`
	Code    int64   `json:"code"`
	Version float64 `json:"version"`
}

func (c *Client) Subscribe(channel string, params map[string]string) {
	request := make(map[string]string)
	request["event"] = "subscribe"
	request["channel"] = channel

	if len(params) > 0 {
		for k, v := range params {
			request[k] = v
		}
	}

	c.Socket.WriteJSON(request)
}

func (c *Client) Connect() error {
	//url := "wss://api.bitfinex.com/ws"
	url := "wss://api.bitfinex.com/ws/2"
	fmt.Println("connect to websocket", url)
	s, _, err := websocket.DefaultDialer.Dial(url, nil)

	if err != nil {
		return err
	}

	msgType, resp, err := s.ReadMessage()
	if err != nil {
		return err
	}

	if msgType != websocket.TextMessage {
		return fmt.Errorf("invalid websocket message")
	}

	var hs WebsocketHandshake
	if err := json.Unmarshal(resp, &hs); err != nil {
		return err
	}

	if hs.Event == "info" {
		log.Println(c.Platform, "Connected")
	} else {
		return fmt.Errorf("no handshake")
	}

	c.Socket = s
	c.ConnectedAt = time.Now()

	for _, channel := range []string{"book", "trades"} {
		for symbol, _ := range c.Books {
			params := make(map[string]string)
			if channel == "book" {
				params["prec"] = "P0"
				params["freq"] = "F0"
				params["len"] = "100"
			}
			params["symbol"] = symbol
			c.Subscribe(channel, params)
		}
	}

	return nil
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

type SubscriptionInfo struct {
	Channel string
	Symbol  string
}

func (c *Client) AddSubscriptionChannel(chanID int, channel, symbol string) {
	c.Subscriptions[chanID] = SubscriptionInfo{Symbol: symbol, Channel: channel}
	log.Printf("%s Subscribed to Channel: %s Symbol: %s ChannelID: %d\n", c.Platform, channel, symbol, chanID)
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

		var pkt interface{}
		if err := json.Unmarshal(message, &pkt); err != nil {
			log.Println("PacketHeader-parse:", err)
			continue
		}

		if eventData, ok := pkt.(map[string]interface{}); ok {
			if event, ok := eventData["event"].(string); ok {
				switch event {
				case "subscribed":
					c.AddSubscriptionChannel(int(eventData["chanId"].(float64)), eventData["channel"].(string), eventData["symbol"].(string))
				default:
					fmt.Println("unkown event", eventData)
				}
			}
			continue
		}

		if data, ok := pkt.([]interface{}); ok {
			if len(data) == 0 {
				continue
			}

			chanInfo, ok := c.Subscriptions[int(data[0].(float64))]

			if !ok {
				log.Println("Unable to locate chanID", int(data[0].(float64)))
				continue
			}

			if len(data) == 2 {
				if text, ok := data[1].(string); ok && text == "hb" {
					// received heartbeat
					continue
				}
			}

			book := c.Books[chanInfo.Symbol]
			//fmt.Println(book.ProductInfo.DatabaseKey, chanInfo.Channel, data)
			now := time.Now()

			var trade *orderbook.Trade

			//fmt.Println(chanInfo.Channel, data)

			switch chanInfo.Channel {
			case "book":
				if len(data) != 2 {
					fmt.Println("wrong book packet length", chanInfo)
				}

				list := data[1].([]interface{})

				if _, ok := list[0].(float64); ok {
					// update

					price, count, amount := list[0].(float64), list[1].(float64), list[2].(float64)
					if amount < 0 {
						// ask
						amount = math.Abs(amount)
						if count == 0 {
							amount = 0
						}
						book.UpdateAskLevel(now, price, amount)
					} else {
						// bid
						if count == 0 {
							amount = 0
						}
						book.UpdateBidLevel(now, price, amount)
					}
				} else {
					// snapshot

					book.Clear()
					//book.Sequence = uint64(now.Unix())
					book.Sequence = uint64(0)

					for _, item := range list {
						values := item.([]interface{})
						price, count, amount := values[0].(float64), values[1].(float64), values[2].(float64)

						if amount < 0 {
							// ask
							amount = math.Abs(amount)
							if count == 0 {
								amount = 0
							}
							book.UpdateAskLevel(now, price, amount)
						} else {
							// bid
							if count == 0 {
								amount = 0
							}
							book.UpdateBidLevel(now, price, amount)
						}
					}
				}
			case "trades":
				if len(data) != 3 {
					// skip snapshot
					//fmt.Println("wrong trades packet length", chanInfo, data)
				}

				if pktType, ok := data[1].(string); ok && pktType == "te" {
					values := data[2].([]interface{})
					amount, price := values[2].(float64), values[3].(float64)
					if amount < 0 {
						// sell
						amount = math.Abs(amount)
						book.AddTrade(now, uint8(orderbook.BidSide), price, amount)
					} else {
						// buy
						book.AddTrade(now, uint8(orderbook.AskSide), price, amount)
					}
					trade = book.Trades[len(book.Trades)-1]
				}

			default:
				fmt.Println("unkown channel", chanInfo)
			}

			book.Sequence += 1

			if c.dbEnabled {
				batch := c.BatchWrite[book.ID]
				now := time.Now()
				if trade != nil {
					batch.Write(c.DB, now, book.ProductInfo.DatabaseKey, orderbook.PackTrade(trade))
				}

				if batch.NextSync(now) {
					fmt.Println("STORE SYNC", book.ProductInfo.DatabaseKey, batch.Count)
					c.WriteSync(batch, book, now)
				} else {
					if batch.NextDiff(now) {
						//fmt.Println("STORE DIFF", book.ProductInfo.DatabaseKey, batch.Count)
						c.WriteDiff(batch, book, now)
					}
				}
			}

		}
	}
}
