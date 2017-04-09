package main

import (
	"fmt"
	"time"

	//_ "net/http/pprof"

	"github.com/lian/gdax-bookmap/websocket"
)

/*
func PrintBooks(gdax *websocket.Client) {
	// clear terminal
	fmt.Print("\033[H\033[2J")

	limit := 10

	for _, key := range gdax.Products {
		book := gdax.Books[key]
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
}
*/

func main() {
	/*
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	*/

	//bookUpdated := make(chan string, 1024)

	gdax := websocket.New([]string{
		"BTC-USD",
		//"BTC-EUR",
		//"ETH-USD",
		//"LTC-USD",
	}, nil, nil)

	go gdax.Run()

	now := time.Now()
	t := time.NewTicker(5 * time.Second)
	book := gdax.Books["BTC-USD"]

	for c := range t.C {
		fmt.Println(c.Sub(now), gdax.DBCount, len(book.Bid)+len(book.Ask))
	}

	/*
		for {
			select {
			case <-bookUpdated:
				PrintBooks(gdax)
			}
		}
	*/
}
