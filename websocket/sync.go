package websocket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/lian/gdax-bookmap/orderbook"
)

func SyncBook(book *orderbook.Book, client *Client) error {
	fmt.Println("sync", book.ID)

	full, err := FetchRawBook(3, book.ID)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if seq, ok := full["sequence"]; ok {
		book.Clear()
		book.Sequence = uint64(seq.(float64))

		packBids := [][]interface{}{}
		packAsks := [][]interface{}{}

		if bids, ok := full["bids"].([]interface{}); ok {
			for i := len(bids) - 1; i >= 0; i-- {
				data := bids[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				size, _ := strconv.ParseFloat(data[1].(string), 64)
				book.Add(map[string]interface{}{
					"id":    data[2].(string),
					"side":  "buy",
					"price": price,
					"size":  size,
				})
				packBids = append(packBids, []interface{}{price, size, data[2].(string)})
			}
		}
		if asks, ok := full["asks"].([]interface{}); ok {
			for i := len(asks) - 1; i >= 0; i-- {
				data := asks[i].([]interface{})
				price, _ := strconv.ParseFloat(data[0].(string), 64)
				size, _ := strconv.ParseFloat(data[1].(string), 64)
				book.Add(map[string]interface{}{
					"id":    data[2].(string),
					"side":  "sell",
					"price": price,
					"size":  size,
				})
				packAsks = append(packAsks, []interface{}{price, size, data[2].(string)})
			}
		}

		client.WriteDB(book, map[string]interface{}{
			"type":     "sync",
			"sequence": full["sequence"],
			"bids":     packBids,
			"asks":     packAsks,
		})
	}

	return nil
}

func FetchRawBook(level int, product string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://api.gdax.com/products/%s/book?level=%d", product, level)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
