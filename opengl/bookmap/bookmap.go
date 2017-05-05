package bookmap

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"time"

	font "github.com/lian/gonky/font/terminus"

	"github.com/lian/gdax-bookmap/orderbook"
	"github.com/lian/gdax-bookmap/websocket"
	"github.com/lian/gonky/shader"
	"github.com/lian/gonky/texture"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

type Bookmap struct {
	Texture             *texture.Texture
	PriceScrollPosition float64
	PriceSteps          float64 // zoom
	MaxSizeHisto        float64
	RowHeight           float64
	Book                *orderbook.Book
	gdax                *websocket.Client
	ColumnWidth         float64
	ViewportStep        int
	Graph               *Graph
	Image               *image.RGBA
	IgnoreTexture       bool
	ShowDebug           bool
	AutoHistoSize       bool
}

func New(program *shader.Program, width, height float64, x float64, book *orderbook.Book, gdax *websocket.Client) *Bookmap {
	s := &Bookmap{
		Book:         book,
		gdax:         gdax,
		RowHeight:    14,
		ColumnWidth:  4,
		ViewportStep: 1,
		ShowDebug:    true,
		Texture: &texture.Texture{
			X:      x,
			Y:      height + 10,
			Width:  width,
			Height: height,
		},
	}
	s.PriceSteps = float64(s.Book.ProductInfo.QuoteIncrement) * 10
	if program != nil {
		s.Texture.Setup(program)
	}
	s.Image = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	return s
}

func (s *Bookmap) SetBook(book *orderbook.Book) {
	s.Book = book
	s.PriceSteps = float64(s.Book.ProductInfo.QuoteIncrement) * 10
	s.PriceScrollPosition = 0
	s.MaxSizeHisto = 0
	s.Graph.ProductID = book.ID
	start := s.Graph.CurrentTime.Add(time.Duration((s.Graph.SlotSteps*s.Graph.SlotCount)*-1) * time.Second)
	s.Graph.SetStart(start)
}

// ugly af
func round(k float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	i := fmt.Sprintf(format, k)
	f, _ := strconv.ParseFloat(i, 64)
	return f
}

func (s *Bookmap) InitPriceScrollPosition() {
	if s.PriceScrollPosition != 0.0 {
		return
	}

	if s.Graph == nil {
		return
	}

	rowsCount := s.Texture.Height / s.RowHeight

	centerPrice := s.Graph.Book.Book.CenterPrice()
	if centerPrice != 0.0 {
		//s.PriceScrollPosition = round(centerPrice, 0) + (float64(int(rowsCount/2)) * s.PriceSteps)
		s.PriceScrollPosition = centerPrice + (float64(int(rowsCount/2)) * s.PriceSteps)
	}
}

func (s *Bookmap) WriteTexture() {
	if s.IgnoreTexture {
		return
	}
	s.Texture.Write(&s.Image.Pix)
}

func (s *Bookmap) DrawString(x, y int, text string, color color.RGBA) {
	font.DrawString(s.Image, x, y, text, color)
}

func (s *Bookmap) Render() {
	gc := draw2dimg.NewGraphicContext(s.Image)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	green := color.RGBA{0x4d, 0xa5, 0x3c, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	// fill texture with default background
	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	now := time.Now()

	if s.Graph == nil {
		graph := NewGraph(s.gdax.DB, s.Book.ID, int(s.Texture.Width-80), int(s.ColumnWidth), int(s.ViewportStep))
		if graph.SetStart(now) {
			s.Graph = graph
		}
		s.WriteTexture()
		return
	}

	s.InitPriceScrollPosition()
	x := s.Texture.Width - 80

	if !s.Graph.SetEnd(now) {
		s.WriteTexture()
		return
	}

	statsSlot := NewTimeSlot(s, now, now)
	stats := s.Graph.Book.Book.StateAsStats()
	statsSlot.Fill(stats)
	if s.MaxSizeHisto == 0 || s.AutoHistoSize {
		s.MaxSizeHisto = round(statsSlot.MaxSize/2, 0)
	}

	s.Graph.DrawTimeslots(gc, x, ((s.Texture.Height - s.RowHeight) / s.RowHeight), s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawBidAskLines(gc, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps)
	s.Graph.DrawTradeDots(gc, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawTimeline(gc, s.Image, x, s.Texture.Height-12.0)

	// draw current (statsSlot) volume slot
	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(fg1)
	gc.MoveTo(x, 0)
	//gc.LineTo(x, s.Texture.Height-s.RowHeight)
	gc.LineTo(x, s.Texture.Height)
	gc.Fill()

	xx := float64(x + 2)
	fontPad := int((s.RowHeight - font.Height) / 2.0)
	for n, row := range statsSlot.Rows {
		if math.Mod(float64(n), 3) == 0 {
			s.DrawString(int(xx)-80, int(row.Y)+fontPad, s.Book.ProductInfo.FormatFloat(row.Heigh), fg1)
		}

		width := 80.0
		//draw2dkit.Rectangle(gc, xx, row.Y+2, xx+width, row.Y+s.RowHeight-2)
		//gc.SetFillColor(bg1)
		//gc.Fill()

		if row.Size > 0 {
			if row.BidCount != 0 && row.AskCount != 0 {
				width = 2 + (80 * (row.AskSize / (statsSlot.MaxSize + 10)))
				y1 := row.Y + 1
				y2 := row.Y + s.RowHeight/2
				draw2dkit.Rectangle(gc, xx, y1, xx+width, y2)
				gc.SetFillColor(red)
				gc.Fill()

				width = 2 + (80 * (row.BidSize / (statsSlot.MaxSize + 10)))
				y1 = y2 + 1
				y2 = row.Y + s.RowHeight - 1
				draw2dkit.Rectangle(gc, xx, y1, xx+width, y2)
				gc.SetFillColor(green)
				gc.Fill()
			} else {
				if row.BidCount != 0 {
					gc.SetFillColor(green)
				} else {
					gc.SetFillColor(red)
				}

				width = 2 + (80 * (row.Size / (statsSlot.MaxSize + 10)))
				draw2dkit.Rectangle(gc, xx, row.Y+1, xx+width, row.Y+s.RowHeight-1)
				gc.Fill()
			}
			s.DrawString(int(xx)+4, int(row.Y)+fontPad, fmt.Sprintf("%.2f (%d)", row.Size, row.OrderCount), fg1)
		}

		//gc.MoveTo(0, row.Y+s.RowHeight)
		gc.MoveTo(xx, row.Y+s.RowHeight)
		gc.LineTo(s.Texture.Width, row.Y+s.RowHeight)
		gc.Stroke()
	}

	s.RenderDebug(now)
	s.DrawString(10, 5, fmt.Sprintf("%s %s", s.Book.ID, s.Book.ProductInfo.FormatFloat(s.Graph.Book.Book.LastPrice())), fg1)

	s.WriteTexture()
}

func (s *Bookmap) RenderDebug(now time.Time) {
	if !s.ShowDebug {
		return
	}

	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}

	s.DrawString(10, 25, fmt.Sprintf(
		"%s PriceScrollPos %s PriceSteps %s MaxSizeHisto %.2f ColumnWidth %.0f ViewportStep %d",
		s.Book.ID,
		s.Book.ProductInfo.FormatFloat(s.PriceScrollPosition),
		s.Book.ProductInfo.FormatFloat(s.PriceSteps),
		s.MaxSizeHisto,
		s.ColumnWidth,
		s.ViewportStep,
	), fg1)

	s.DrawString(10, 40, fmt.Sprintf("graph-time-diff: %s", now.Sub(s.Graph.CurrentTime)), fg1)
}
