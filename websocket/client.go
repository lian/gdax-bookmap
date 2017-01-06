package websocket

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/gorilla/websocket"

	"github.com/lian/gdax/orderbook"
)

func New(products []string, bookUpdated, tradesUpdated chan string) *Client {
	c := &Client{
		BookUpdated:   bookUpdated,
		TradesUpdated: tradesUpdated,
		Products:      []string{},
		Books:         map[string]*orderbook.Book{},
	}

	for _, name := range products {
		c.AddProduct(name)
	}

	return c
}

type Client struct {
	BookUpdated   chan string
	TradesUpdated chan string
	Products      []string
	Books         map[string]*orderbook.Book
	Socket        *websocket.Conn
}

func (c *Client) AddProduct(name string) {
	c.Products = append(c.Products, name)
	c.Books[name] = orderbook.New(name)
	if c.TradesUpdated != nil {
		c.Books[name].TradesUpdated = c.TradesUpdated
	}
}

func (c *Client) Connect() {
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

func (c *Client) HandleMessage(book *orderbook.Book, header PacketHeader, message []byte) {
	var data map[string]interface{}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Fatal(err)
	}

	switch header.Type {
	case "received":
		// skip
		break
	case "open":
		price, _ := strconv.ParseFloat(data["price"].(string), 64)
		size, _ := strconv.ParseFloat(data["remaining_size"].(string), 64)

		book.Add(map[string]interface{}{
			"id":    data["order_id"].(string),
			"side":  data["side"].(string),
			"price": price,
			"size":  size,
		})
		c.BookChanged(book)

		break
	case "done":
		book.Remove(data["order_id"].(string))
		c.BookChanged(book)
		break
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
		c.BookChanged(book)
		break
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
			c.BookChanged(book)
		}
		break
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
				SyncBook(book)
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
			SyncBook(book)
			continue
		}

		book.Sequence = header.Sequence

		c.HandleMessage(book, header, message)
	}
}
