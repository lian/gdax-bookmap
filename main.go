package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"runtime"
	"strings"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/lian/gonky/shader"

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

var redrawChan chan bool = make(chan bool, 10)

const redrawChanHalfLen = 5

func triggerRedraw() {
	if len(redrawChan) < redrawChanHalfLen {
		redrawChan <- true
	}
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

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	//fmt.Printf("%v %d, %v %v\n", key, scancode, action, mods)
	if key == glfw.KeyEscape && action == glfw.Press {
		window.SetShouldClose(true)
	} else if key == glfw.Key1 && action == glfw.Press {
		SetActiveBaseCurrency("BTC")
	} else if key == glfw.Key2 && action == glfw.Press {
		SetActiveBaseCurrency("ETH")
	} else if key == glfw.Key3 && action == glfw.Press {
		SetActiveBaseCurrency("BCH")
		/*
			} else if key == glfw.Key1 && action == glfw.Press {
				SetActiveProduct(0)
			} else if key == glfw.Key2 && action == glfw.Press {
				SetActiveProduct(1)
			} else if key == glfw.Key3 && action == glfw.Press {
				SetActiveProduct(2)
			} else if key == glfw.Key4 && action == glfw.Press {
				SetActiveProduct(3)
			} else if key == glfw.Key5 && action == glfw.Press {
				SetActiveProduct(4)
			} else if key == glfw.Key6 && action == glfw.Press {
				SetActiveProduct(5)
			} else if key == glfw.Key7 && action == glfw.Press {
				SetActiveProduct(6)
			} else if key == glfw.Key8 && action == glfw.Press {
				SetActiveProduct(7)
			} else if key == glfw.Key9 && action == glfw.Press {
				SetActiveProduct(8)
		*/
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

func SetupPerspective(width, height int, program *shader.Program) {
	program.Use()

	fov := float32(60.0)
	eyeX := float32(WindowWidth) / 2.0
	eyeY := float32(WindowHeight) / 2.0
	ratio := float32(width) / float32(height)
	halfFov := (math.Pi * fov) / 360.0
	theTan := math.Tan(float64(halfFov))
	dist := eyeY / float32(theTan)
	nearDist := dist / 10.0
	farDist := dist * 10.0

	projection := mgl32.Perspective(mgl32.DegToRad(fov), ratio, nearDist, farDist)
	projectionUniform := program.UniformLocation("projection")
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

	camera := mgl32.LookAtV(mgl32.Vec3{eyeX, eyeY, dist}, mgl32.Vec3{eyeX, eyeY, 0}, mgl32.Vec3{0, 1, 0})
	cameraUniform := program.UniformLocation("camera")
	gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

	//model := mgl32.Ident4()
	//modelUniform := program.UniformLocation("model")
	//gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

	textureUniform := program.UniformLocation("tex")
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(program.ID, 0, gl.Str("outputColor\x00"))

	gl.Viewport(0, 0, int32(width), int32(height))
}

func resizeCallback(_ *glfw.Window, width int, height int) {
	//fmt.Println("RESIZE", width, height)
	SetupPerspective(width, height, program)
}

var WindowWidth int
var WindowHeight int

var program *shader.Program

//var trades map[string]*opengl_trades.Trades
var bookmaps map[string]*opengl_bookmap.Bookmap

var ActiveBase string
var ActiveProduct string
var ActivePlatform string
var infos []*product_info.Info

func main() {
	var db_path string
	fmt.Printf("Starting gdax-bookmap %s-%s\n", AppVersion, AppGitHash)
	flag.StringVar(&ActivePlatform, "platforms", "gdax-bitstamp-binance-bitfinex", "active platforms")
	//flag.StringVar(&ActivePlatform, "platforms", "gdax-bitstamp-bitfinex", "active platforms")
	flag.StringVar(&ActiveBase, "base", "BTC", "active BaseCurrency")
	flag.StringVar(&db_path, "db", "orderbooks.db", "database file")
	flag.IntVar(&WindowWidth, "w", 1920, "window width")
	flag.IntVar(&WindowHeight, "h", 1080, "window height")
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

	window, err := glfw.CreateWindow(WindowWidth, WindowHeight, "gdax-bookmap", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	//window.SetSizeCallback(resizeCallback)
	window.SetFramebufferSizeCallback(resizeCallback)
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

	w, h := window.GetFramebufferSize()
	SetupPerspective(w, h, program)

	db := util.OpenDB(db_path, []string{}, false)

	//bookUpdated := make(chan string, 1024)
	//tradesUpdated := make(chan string)
	infos = make([]*product_info.Info, 0)

	if strings.Contains(strings.ToLower(ActivePlatform), "gdax") {
		//ws := gdax_websocket.New(db, []string{"BTC-USD", "BTC-EUR", "LTC-USD", "ETH-USD", "ETH-BTC", "LTC-BTC", "BCH-USD", "BCH-BTC"})
		//ws := gdax_websocket.New(db, []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD"})
		ws := gdax_websocket.New(db, []string{"BTC-USD", "ETH-USD", "BCH-USD"})
		//ws := gdax_websocket.New(db, []string{"BCH-EUR"})
		go ws.Run()
		for _, info := range ws.Infos {
			infos = append(infos, info)
		}
		ActiveProduct = infos[0].DatabaseKey
	}
	if strings.Contains(strings.ToLower(ActivePlatform), "bitstamp") {
		//ws := bitstamp_websocket.New(db, []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD", "XRP-USD"})
		//ws := bitstamp_websocket.New(db, []string{"BTC-USD", "ETH-USD", "LTC-USD", "BCH-USD"})
		ws := bitstamp_websocket.New(db, []string{"BTC-USD", "ETH-USD", "BCH-USD"})
		//ws := bitstamp_websocket.New(db, []string{"BCH-EUR"})
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

	bookmaps = map[string]*opengl_bookmap.Bookmap{}
	//trades = map[string]*opengl_trades.Trades{}

	padding := 10.0
	x := padding

	count := len(infos) / 2
	for _, info := range infos {
		//bookmaps[info.DatabaseKey] = opengl_bookmap.New(program, 1260, 680/3, x, *info, db)
		bookmaps[info.DatabaseKey] = opengl_bookmap.New(program, float64(WindowWidth)-20, float64((WindowHeight-4)/count), x, *info, db)
	}
	x += bookmaps[ActiveProduct].Texture.Width + padding
	/*
		for _, info := range infos {
			trades[info.DatabaseKey] = opengl_trades.New(program, bookmaps[info.DatabaseKey], *info, 700, x)
		}
		x += trades[ActiveProduct].Texture.Width + padding
	*/

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.18, 0.23, 0.27, 1.0)

	pollEventsTimer := time.NewTicker(time.Millisecond * 100)
	second := time.NewTicker(time.Second * 1)
	halfsecond := time.NewTicker(time.Millisecond * 500)

	for !window.ShouldClose() {
		select {
		case <-pollEventsTimer.C:
			glfw.PollEvents()
			continue
		case <-halfsecond.C:
			//trades[ActiveProduct].Render()
		case <-second.C:
			/*
				for _, info := range infos {
					if ActiveProduct == info.DatabaseKey {
						bookmaps[info.DatabaseKey].Render()
					} else {
						bookmaps[info.DatabaseKey].Progress()
					}
				}
			*/
			for _, info := range infos {
				if info.BaseCurrency == ActiveBase {
					bookmaps[info.DatabaseKey].Render()
				} else {
					bookmaps[info.DatabaseKey].Progress()
				}
			}
		case <-redrawChan:
			//fmt.Println("forced redraw")
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		program.Use()

		//bookmaps[ActiveProduct].Texture.Draw()
		//trades[ActiveProduct].Texture.Draw()

		count := len(infos) / 2
		n := 0
		for _, info := range infos {
			if info.BaseCurrency == ActiveBase {
				bookmaps[info.DatabaseKey].Texture.DrawAt(float32(10), float32(WindowHeight)-float32(n*(WindowHeight/count)))
				n += 1
			}
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
