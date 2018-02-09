package main

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/lian/gonky/shader"
)

type KeyCallback func(*Window, glfw.Key, glfw.Action, glfw.ModifierKey)

type Window struct {
	Width      int
	Height     int
	glfwWindow *glfw.Window
	Shader     *shader.Program

	redrawChan        chan bool
	redrawChanHalfLen int
	KeyCallbacks      []KeyCallback
}

func NewWindow(width, height int) (*Window, error) {
	w := &Window{
		Width:             width,
		Height:            height,
		redrawChan:        make(chan bool, 2),
		redrawChanHalfLen: 1,
	}

	var err error

	if err = w.InitGL(); err != nil {
		return nil, err
	}

	if err = w.InitShader(); err != nil {
		return nil, err
	}

	// configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.18, 0.23, 0.27, 1.0)

	return w, nil
}

func (w *Window) TriggerRedraw() {
	if len(w.redrawChan) < w.redrawChanHalfLen {
		w.redrawChan <- true
	}
}

func (w *Window) InitGL() error {
	var err error
	if err = glfw.Init(); err != nil {
		return err
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	padding := 40

	screenInfo := glfw.GetPrimaryMonitor().GetVideoMode()
	if w.Width == 0 {
		w.Width = screenInfo.Width - padding
	}
	if w.Height == 0 {
		w.Height = screenInfo.Height - padding
	}

	w.glfwWindow, err = glfw.CreateWindow(w.Width, w.Height, "gdax-bookmap", nil, nil)
	if err != nil {
		return err
	}

	w.glfwWindow.MakeContextCurrent()
	//w.glfwWindow.SetSizeCallback(resizeCallback)
	w.glfwWindow.SetFramebufferSizeCallback(w.resizeCallback)
	w.glfwWindow.SetRefreshCallback(w.refreshCallback)
	w.glfwWindow.SetFocusCallback(w.focusCallback)
	w.glfwWindow.SetKeyCallback(w.keyCallback)

	if err = gl.Init(); err != nil {
		return err
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	return nil
}

func (w *Window) InitShader() error {
	var err error
	w.Shader, err = shader.DefaultShader()
	if err != nil {
		return err
	}

	width, height := w.glfwWindow.GetFramebufferSize()
	w.resizeCallback(nil, width, height)

	return nil
}

func (w *Window) focusCallback(_ *glfw.Window, focused bool) {
	w.TriggerRedraw()
}

func (w *Window) refreshCallback(_ *glfw.Window) {
	w.TriggerRedraw()
}

func (w *Window) resizeCallback(_ *glfw.Window, width int, height int) {
	fmt.Println("RESIZE", width, height)
	w.SetupPerspective(width, height, w.Shader)
	w.TriggerRedraw()
}

func (w *Window) keyCallback(_ *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	for _, cb := range w.KeyCallbacks {
		cb(w, key, action, mods)
	}
	w.TriggerRedraw()
}

func (w *Window) AddKeyCallback(cb KeyCallback) {
	w.KeyCallbacks = append(w.KeyCallbacks, cb)
}

func (w *Window) SetupPerspective(width, height int, program *shader.Program) {
	program.Use()

	fov := float32(60.0)
	eyeX := float32(w.Width) / 2.0  // NOTE: does the scaling
	eyeY := float32(w.Height) / 2.0 // NOTE: does the scaling
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

func (w *Window) Close() {
	glfw.Terminate()
}

func (w *Window) ShouldClose() bool {
	return w.glfwWindow.ShouldClose()
}

func (w *Window) SwapBuffers() {
	w.glfwWindow.SwapBuffers()
}

func (w *Window) PollEvents() {
	glfw.PollEvents()
}

func (w *Window) BeginFrame() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	w.Shader.Use()
}

func (w *Window) EndFrame() {
	w.glfwWindow.SwapBuffers()
	glfw.PollEvents()
}
