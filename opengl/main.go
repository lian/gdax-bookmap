package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"

	"github.com/lian/gdax/websocket"

	"github.com/lian/gonky/shader"

	opengl_orderbook "github.com/lian/gdax/opengl/orderbook"
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

func keyCallback(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	//fmt.Printf("%v %d, %v %v\n", key, scancode, action, mods)
	if key == glfw.KeyEscape && action == glfw.Press {
		window.SetShouldClose(true)
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

var WindowWidth int = 800
var WindowHeight int = 600

var program *shader.Program

func main() {
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

	//bookUpdated := make(chan string, 1024)
	//bookUpdated := make(chan string)
	gdax := websocket.New([]string{
		"BTC-USD",
		"BTC-EUR",
		"ETH-USD",
		"LTC-USD",
	}, nil)
	//}, bookUpdated)
	go gdax.Run()

	orderbooks := map[string]*opengl_orderbook.Orderbook{}

	for n, name := range gdax.Products {
		orderbooks[name] = opengl_orderbook.New(program, gdax, name, n)
	}

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.18, 0.23, 0.27, 1.0)

	pollEventsTimer := time.NewTicker(time.Millisecond * 100)
	tick := time.NewTicker(time.Millisecond * 500)

	for !window.ShouldClose() {
		select {
		case <-pollEventsTimer.C:
			glfw.PollEvents()
			continue
		case <-tick.C:
			for _, orderbook := range orderbooks {
				orderbook.Render()
			}
		//case <-bookUpdated:
		//	orderbook.Render()
		case <-redrawChan:
			//fmt.Println("forced redraw")
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		program.Use()
		for _, orderbook := range orderbooks {
			orderbook.Texture.Draw()
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}
