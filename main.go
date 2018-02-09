package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/glfw/v3.2/glfw"

	//_ "net/http/pprof"

	binance_websocket "github.com/lian/gdax-bookmap/exchanges/binance/websocket"
	bitfinex_websocket "github.com/lian/gdax-bookmap/exchanges/bitfinex/websocket"
	bitstamp_websocket "github.com/lian/gdax-bookmap/exchanges/bitstamp/websocket"
	gdax_websocket "github.com/lian/gdax-bookmap/exchanges/gdax/websocket"

	opengl_bookmap "github.com/lian/gdax-bookmap/opengl/bookmap"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	"github.com/lian/gdax-bookmap/util"
)

var (
	AppVersion = "unknown"
	AppGitHash = "unknown"
)

func init() {
	runtime.LockOSThread()
}

func SetActiveProduct(index int) {
	if index < len(infos) {
		i := infos[index]
		ActiveProduct = i.DatabaseKey
	}
}

func SetActiveBaseCurrency(base string) {
	for _, info := range infos {
		if info.BaseCurrency == base {
			ActiveBase = base
			ActiveProduct = info.DatabaseKey
			break
		}
	}
}

func keyCallback(window *Window, key glfw.Key, action glfw.Action, mods glfw.ModifierKey) {
	//fmt.Printf("%v %d, %v %v\n", key, scancode, action, mods)

	if key == glfw.KeyEscape && action == glfw.Press {
		window.glfwWindow.SetShouldClose(true)
	} else if key == glfw.Key1 && action == glfw.Press {
		SetActiveBaseCurrency("BTC")
	} else if key == glfw.Key2 && action == glfw.Press {
		SetActiveBaseCurrency("ETH")
	} else if key == glfw.Key3 && action == glfw.Press {
		SetActiveBaseCurrency("BCH")
	} else if key == glfw.KeyS && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceScrollPosition += bm.PriceSteps
		bm.Graph.ClearSlotRows()
	} else if key == glfw.KeyW && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceScrollPosition -= bm.PriceSteps
		bm.Graph.ClearSlotRows()
	} else if key == glfw.KeyD && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ViewportStep = bm.ViewportStep * 2
		bm.Graph.SlotSteps = bm.ViewportStep
		start := bm.Graph.End.Add(time.Duration((bm.ViewportStep*bm.Graph.SlotCount)*-1) * time.Second)
		bm.Graph.SetStart(start)

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.ViewportStep = bm.ViewportStep
			bookmap.Graph.SlotSteps = bm.Graph.SlotSteps
			bookmap.Graph.SetStart(start)
		}
	} else if key == glfw.KeyA && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ViewportStep = bm.ViewportStep / 2
		if bm.ViewportStep <= 0 {
			bm.ViewportStep = 1
		}
		bm.Graph.SlotSteps = bm.ViewportStep
		start := bm.Graph.End.Add(time.Duration((bm.ViewportStep*bm.Graph.SlotCount)*-1) * time.Second)
		bm.Graph.SetStart(start)

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.ViewportStep = bm.ViewportStep
			bookmap.Graph.SlotSteps = bm.Graph.SlotSteps
			bookmap.Graph.SetStart(start)
		}
	} else if key == glfw.KeyJ && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.MaxSizeHisto = bm.MaxSizeHisto * 2

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.MaxSizeHisto = bm.MaxSizeHisto
		}
	} else if key == glfw.KeyK && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.MaxSizeHisto = bm.MaxSizeHisto / 2
		if bm.MaxSizeHisto < 0 {
			bm.MaxSizeHisto = 1
		}

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.MaxSizeHisto = bm.MaxSizeHisto
		}
	} else if key == glfw.KeyDown && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceSteps = bm.PriceSteps * 2
		if bm.PriceSteps >= float64(bm.ProductInfo.BaseMaxSize) {
			//bm.PriceSteps = float64(bm.ProductInfo.BaseMaxSize)
		}
		bm.ForceAutoScroll()

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.PriceSteps = bm.PriceSteps
			bookmap.ForceAutoScroll()
		}
	} else if key == glfw.KeyUp && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceSteps = bm.PriceSteps / 2
		if bm.PriceSteps <= float64(bm.ProductInfo.QuoteIncrement) {
			bm.PriceSteps = float64(bm.ProductInfo.QuoteIncrement)
		}
		bm.ForceAutoScroll()

		for _, info := range infos {
			if info.BaseCurrency != ActiveBase {
				continue
			}
			bookmap := bookmaps[info.DatabaseKey]
			bookmap.PriceSteps = bm.PriceSteps
			bookmap.ForceAutoScroll()
		}
	} else if key == glfw.KeyLeft && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ColumnWidth -= 2
		if bm.ColumnWidth < 0 {
			bm.ColumnWidth = 1
		}
		bm.Graph.SlotWidth = int(bm.ColumnWidth)
	} else if key == glfw.KeyRight && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ColumnWidth += 2
		if bm.ColumnWidth > 30 {
			bm.ColumnWidth = 30
		}
		bm.Graph.SlotWidth = int(bm.ColumnWidth)
	} else if key == glfw.KeyC && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ForceAutoScroll()
	} else if key == glfw.KeyP && action == glfw.Press {
		for _, bm := range bookmaps {
			bm.AutoScroll = !bm.AutoScroll
		}
	} else if key == glfw.KeyR && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.MaxSizeHisto = 0.0
	}
}

func runpprof() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
}

var bookmaps map[string]*opengl_bookmap.Bookmap
var ActiveBase string
var ActiveProduct string
var ActivePlatform string
var infos []*product_info.Info

func main() {
	var db_path string
	var windowWidth int
	var windowHeight int

	fmt.Printf("Starting gdax-bookmap %s-%s\n", AppVersion, AppGitHash)
	//flag.StringVar(&ActivePlatform, "platforms", "gdax-bitstamp-binance-bitfinex", "active platforms")
	flag.StringVar(&ActivePlatform, "platforms", "gdax-bitstamp-binance", "active platforms")
	flag.StringVar(&ActiveBase, "base", "BTC", "active BaseCurrency")
	flag.StringVar(&db_path, "db", "orderbooks.db", "database file")
	flag.IntVar(&windowWidth, "w", 0, "window width")
	flag.IntVar(&windowHeight, "h", 0, "window height")
	flag.Parse()

	//runpprof()

	db, err := util.OpenDB(db_path, []string{}, false)
	if err != nil {
		fmt.Println("OpenDB Error", err)
		os.Exit(0)
	}

	infos = make([]*product_info.Info, 0)

	if strings.Contains(strings.ToLower(ActivePlatform), "gdax") {
		ws := gdax_websocket.New(db, []string{"BTC-USD", "ETH-USD", "BCH-USD"})
		go ws.Run()
		for _, info := range ws.Infos {
			infos = append(infos, info)
		}
		ActiveProduct = infos[0].DatabaseKey
	}
	if strings.Contains(strings.ToLower(ActivePlatform), "bitstamp") {
		ws := bitstamp_websocket.New(db, []string{"BTC-USD", "ETH-USD", "BCH-USD"})
		go ws.Run()
		for _, info := range ws.Infos {
			infos = append(infos, info)
		}
		ActiveProduct = infos[0].DatabaseKey
	}
	if strings.Contains(strings.ToLower(ActivePlatform), "binance") {
		ws := binance_websocket.New(db, []string{"BTC-USDT", "ETH-USDT", "BCH-USDT"})
		go ws.Run()
		for _, info := range ws.Infos {
			infos = append(infos, info)
		}
		ActiveProduct = infos[0].DatabaseKey
	}
	if strings.Contains(strings.ToLower(ActivePlatform), "bitfinex") {
		ws := bitfinex_websocket.New(db, []string{"BTC-USD", "ETH-USD", "BCH-USD"})
		go ws.Run()
		for _, info := range ws.Infos {
			infos = append(infos, info)
		}
		ActiveProduct = infos[0].DatabaseKey
	}

	win, err := NewWindow(windowWidth, windowHeight)
	if err != nil {
		panic(err)
	}
	win.AddKeyCallback(keyCallback)

	bookmaps = map[string]*opengl_bookmap.Bookmap{}

	padding := 10.0
	x := padding

	count := len(infos) / 3
	for _, info := range infos {
		bookmaps[info.DatabaseKey] = opengl_bookmap.New(win.Shader, float64(win.Width)-(padding*2), float64((win.Height-4)/count), x, *info, db)
	}

	pollEventsTimer := time.NewTicker(time.Millisecond * 100)
	second := time.NewTicker(time.Second * 1)

	for !win.ShouldClose() {
		select {
		case <-pollEventsTimer.C:
			win.PollEvents()
			continue
		case <-win.redrawChan:
			// force quick redraw (window resized/moved)
		case <-second.C:
			for _, info := range infos {
				if info.BaseCurrency == ActiveBase {
					bookmaps[info.DatabaseKey].Render()
				} else {
					bookmaps[info.DatabaseKey].Progress()
				}
			}
		}
		win.BeginFrame()

		count := len(infos) / 3
		n := 0
		for _, info := range infos {
			if info.BaseCurrency == ActiveBase {
				bookmaps[info.DatabaseKey].Texture.DrawAt(float32(10), float32(win.Height)-float32(n*(win.Height/count)))
				n += 1
			}
		}

		win.EndFrame()
	}
}
