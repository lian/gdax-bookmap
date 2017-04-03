package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"

	"github.com/lian/gdax-bookmap/websocket"

	"github.com/lian/gonky/shader"

	opengl_bookmap "github.com/lian/gdax-bookmap/opengl/bookmap"
	opengl_orderbook "github.com/lian/gdax-bookmap/opengl/orderbook"
	opengl_trades "github.com/lian/gdax-bookmap/opengl/trades"
	//_ "net/http/pprof"
)

var ActiveProduct string = "BTC-USD"

func init() {
	runtime.LockOSThread()
}

var redrawChan chan bool = make(chan bool, 10)

const redrawChanHalfLen = 5

func triggerRedraw() {
	if len(redrawChan) < redrawChanHalfLen {
		redrawChan <- true
	}
}

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	//fmt.Printf("%v %d, %v %v\n", key, scancode, action, mods)
	if key == glfw.KeyEscape && action == glfw.Press {
		window.SetShouldClose(true)
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
		bm.Graph.SetStart(bm.Graph.Start)
	} else if key == glfw.KeyA && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ViewportStep = bm.ViewportStep / 2
		bm.Graph.SlotSteps = bm.ViewportStep
		bm.Graph.SetStart(bm.Graph.Start)
	} else if key == glfw.KeyJ && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.MaxSizeHisto = bm.MaxSizeHisto * 2
	} else if key == glfw.KeyK && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.MaxSizeHisto = bm.MaxSizeHisto / 2
	} else if key == glfw.KeyUp && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceScrollPosition = 0
		bm.PriceSteps = bm.PriceSteps * 2
		bm.InitPriceScrollPosition()
		bm.Graph.ClearSlotRows()
	} else if key == glfw.KeyDown && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceScrollPosition = 0
		bm.PriceSteps = bm.PriceSteps / 2
		bm.InitPriceScrollPosition()
		bm.Graph.ClearSlotRows()
	} else if key == glfw.KeyLeft && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ColumnWidth -= 2
	} else if key == glfw.KeyRight && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.ColumnWidth += 2
	} else if key == glfw.KeyC && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.PriceScrollPosition = 0.0
		bm.InitPriceScrollPosition()
	} else if key == glfw.KeyT && action == glfw.Press {
		bm := bookmaps[ActiveProduct]
		bm.Graph.SetStart(bm.Graph.Start)
	}
	triggerRedraw()
}

func focusCallback(window *glfw.Window, focused bool) {
	//fmt.Println("focus:", focused)
	triggerRedraw()
}

func refreshCallback(window *glfw.Window) {
	//fmt.Println("refreshCallback")
	triggerRedraw()
}

func resizeCallback(w *glfw.Window, width int, height int) {
	//fmt.Println("RESIZE", width, height)
	WindowWidth = width
	WindowHeight = height
	shader.SetupPerspective(width, height, program)
}

//var WindowWidth int = 1250
var WindowWidth int = 1280
var WindowHeight int = 720

var program *shader.Program
var bookmaps map[string]*opengl_bookmap.Bookmap

func main() {
	flag.StringVar(&ActiveProduct, "pair", "BTC-USD", "gdax Product ID")
	flag.Parse()

	/*
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	*/

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	//screenInfo := glfw.GetPrimaryMonitor().GetVideoMode()
	//WindowWidth := screenInfo.Width
	//WindowHeight := screenInfo.Height

	window, err := glfw.CreateWindow(WindowWidth, WindowHeight, "gdax-go", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	window.SetSizeCallback(resizeCallback)
	window.SetRefreshCallback(refreshCallback)
	window.SetFocusCallback(focusCallback)
	window.SetKeyCallback(keyCallback)

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	program, err = shader.DefaultShader()
	if err != nil {
		panic(err)
	}
	//fmt.Printf("program: %v\n", program)
	program.Use()

	shader.SetupPerspective(WindowWidth, WindowHeight, program)

	bookUpdated := make(chan string, 1024)
	tradesUpdated := make(chan string)
	gdax := websocket.New([]string{ActiveProduct}, bookUpdated, tradesUpdated)
	go gdax.Run()

	orderbooks := map[string]*opengl_orderbook.Orderbook{}
	bookmaps = map[string]*opengl_bookmap.Bookmap{}
	trades := map[string]*opengl_trades.Trades{}

	padding := 10.0
	x := padding
	updatedOrderbook := map[string]bool{}
	for _, name := range gdax.Products {
		orderbooks[name] = opengl_orderbook.New(program, gdax, name, 700, x)
		x += orderbooks[name].Texture.Width + padding
		bookmaps[name] = opengl_bookmap.New(program, 800, 700, x, gdax.Books[ActiveProduct], gdax)
		//width := float64(WindowWidth) - x - 254 // 254 from trades widget
		//bookmaps[name] = opengl_bookmap.New(program, width, 700, x, gdax.Books[ActiveProduct], gdax)
		x += bookmaps[name].Texture.Width + padding
		trades[name] = opengl_trades.New(program, gdax, name, 700, x)
		x += trades[name].Texture.Width + padding
		updatedOrderbook[name] = true
	}

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.18, 0.23, 0.27, 1.0)

	pollEventsTimer := time.NewTicker(time.Millisecond * 100)
	tick := time.NewTicker(time.Millisecond * 500)
	second := time.NewTicker(time.Second * 1)

	//lastWidth := float64(WindowWidth)

	for !window.ShouldClose() {
		select {
		case <-pollEventsTimer.C:
			glfw.PollEvents()
			continue
		case id := <-bookUpdated:
			updatedOrderbook[id] = true
			//bookmaps[id].BookUpdated(gdax.Books[id])

			s := len(bookUpdated)
			for i := s; i < s; i += 1 {
				id = <-bookUpdated
				//bookmaps[id].BookUpdated(gdax.Books[id])
				updatedOrderbook[id] = true
			}
			continue
		case id := <-tradesUpdated:
			trades[id].Render()
		case <-tick.C:
			none := true
			for id, ok := range updatedOrderbook {
				if ok {
					none = false
					updatedOrderbook[id] = false
					orderbooks[id].Render()
				}
			}
			if none {
				continue
			}
		case <-second.C:
			bookmap := bookmaps[ActiveProduct]
			/*
				if lastWidth != float64(WindowWidth) {
					lastWidth = float64(WindowWidth)
					bookmap.UpdateTexture(lastWidth-(orderbooks[ActiveProduct].Texture.Width+padding)-254-(padding*3), float64(WindowHeight), bookmap.Texture.X, program)
					trade := trades[ActiveProduct]
					trade.Texture.UpdatePosition(lastWidth-254-padding, trade.Texture.Y)
				}
			*/
			bookmap.Render()

		case <-redrawChan:
			//fmt.Println("forced redraw")
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		program.Use()

		for _, orderbook := range orderbooks {
			orderbook.Texture.Draw()
		}
		for _, bookmap := range bookmaps {
			bookmap.Texture.Draw()
		}
		for _, trade := range trades {
			trade.Texture.Draw()
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}