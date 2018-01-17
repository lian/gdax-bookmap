package bookmap

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/lian/gdax-bookmap/orderbook/product_info"
	font "github.com/lian/gonky/font/terminus"

	"github.com/lian/gonky/shader"
	"github.com/lian/gonky/texture"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

type Bookmap struct {
	ID                  string
	ProductInfo         product_info.Info
	Texture             *texture.Texture
	PriceScrollPosition float64
	PriceSteps          float64 // zoom
	MaxSizeHisto        float64
	RowHeight           float64
	DB                  *bolt.DB
	ColumnWidth         float64
	ViewportStep        int
	Graph               *Graph
	Image               *image.RGBA
	IgnoreTexture       bool
	ShowDebug           bool
	AutoHistoSize       bool
	AutoScroll          bool
}

func New(program *shader.Program, width, height float64, x float64, info product_info.Info, db *bolt.DB) *Bookmap {
	s := &Bookmap{
		ID:           info.ID,
		ProductInfo:  info,
		DB:           db,
		RowHeight:    14,
		ColumnWidth:  4,
		ViewportStep: 1,
		ShowDebug:    true,
		AutoScroll:   true,
		Texture: &texture.Texture{
			X:      x,
			Y:      height + 10,
			Width:  width,
			Height: height,
		},
	}

	s.PriceSteps = float64(s.ProductInfo.QuoteIncrement) * 50
	if program != nil {
		s.Texture.Setup(program)
	}
	s.Image = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	return s
}

// ugly af
func round(k float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	i := fmt.Sprintf(format, k)
	f, _ := strconv.ParseFloat(i, 64)
	return f
}

func (s *Bookmap) ForceAutoScroll() {
	if s.Graph == nil {
		return
	}

	rowsCount := s.Texture.Height / s.RowHeight

	last := s.PriceScrollPosition

	//price := s.Graph.Book.Book.LastPrice()
	price := s.Graph.Book.Book.CenterPrice()
	if price != 0.0 {
		s.PriceScrollPosition = (price - math.Mod(price, s.PriceSteps)) + (float64(rowsCount/2) * s.PriceSteps)
		if last != s.PriceScrollPosition {
			s.Graph.ClearSlotRows()
		}
	}
}

func (s *Bookmap) DoAutoScroll() {
	if !s.AutoScroll {
		return
	}

	s.ForceAutoScroll()
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

func (s *Bookmap) Progress() bool {
	now := time.Now()

	if s.Graph == nil {
		graph := NewGraph(s.DB, s.ID, int(s.Texture.Width-80), int(s.ColumnWidth), int(s.ViewportStep))
		if graph.SetStart(now) {
			s.Graph = graph
		}
		return false
	}

	s.DoAutoScroll()

	return s.Graph.SetEnd(now)
}

func (s *Bookmap) Render() {
	if !s.Progress() {
		s.WriteTexture()
		return
	}

	now := time.Now()

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

	if s.MaxSizeHisto == 0 || s.AutoHistoSize {
		s.MaxSizeHisto = round(s.Graph.MaxHistoSize()*0.60, 0)
	}

	x := s.Texture.Width - 80
	s.Graph.DrawTimeslots(gc, x, ((s.Texture.Height - s.RowHeight) / s.RowHeight), s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawTradeDots(gc, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawBidAskLines(s.Image, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps)
	s.Graph.DrawTimeline(gc, s.Image, x, s.Texture.Height-12.0)

	// draw current (statsSlot) volume slot
	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(fg1)
	gc.MoveTo(x, 0)
	gc.LineTo(x, s.Texture.Height)
	gc.Fill()

	statsSlot := NewTimeSlot(s, now, now)
	stats := s.Graph.Book.Book.StateAsStats()
	statsSlot.Fill(stats)

	var y float64
	xx := float64(x + 2)
	fontPad := int((s.RowHeight - font.Height) / 2.0)
	for n, row := range statsSlot.Rows {
		y = float64(n) * s.RowHeight

		if math.Mod(float64(n), 3) == 0 {
			s.DrawString(int(xx)-80, int(y)+fontPad, s.ProductInfo.FormatFloat(row.Heigh), fg1)
		}

		width := 80.0
		//draw2dkit.Rectangle(gc, xx, y+2, xx+width, y+s.RowHeight-2)
		//gc.SetFillColor(bg1)
		//gc.Fill()

		if row.Size > 0 {
			if row.BidCount != 0 && row.AskCount != 0 {
				width = 2 + (80 * (row.AskSize / (statsSlot.MaxSize + 10)))
				y1 := y + 1
				y2 := y + s.RowHeight/2
				draw2dkit.Rectangle(gc, xx, y1, xx+width, y2)
				gc.SetFillColor(red)
				gc.Fill()

				width = 2 + (80 * (row.BidSize / (statsSlot.MaxSize + 10)))
				y1 = y2 + 1
				y2 = y + s.RowHeight - 1
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
				draw2dkit.Rectangle(gc, xx, y+1, xx+width, y+s.RowHeight-1)
				gc.Fill()
			}
			s.DrawString(int(xx)+4, int(y)+fontPad, fmt.Sprintf("%.2f (%d)", row.Size, row.OrderCount), fg1)
		}

		//gc.MoveTo(0, y+s.RowHeight)
		gc.MoveTo(xx, y+s.RowHeight)
		gc.LineTo(s.Texture.Width, y+s.RowHeight)
		gc.Stroke()
	}

	s.RenderDebug(now)
	s.DrawString(10, 5, fmt.Sprintf("%s %s", s.ID, s.ProductInfo.FormatFloat(s.Graph.Book.Book.LastPrice())), fg1)

	s.WriteTexture()
}

func (s *Bookmap) RenderDebug(now time.Time) {
	if !s.ShowDebug {
		return
	}

	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}

	s.DrawString(10, 25, fmt.Sprintf(
		"%s PriceScrollPos %s PriceSteps %s MaxSizeHisto %.2f ColumnWidth %.0f ViewportStep %d",
		s.ID,
		s.ProductInfo.FormatFloat(s.PriceScrollPosition),
		s.ProductInfo.FormatFloat(s.PriceSteps),
		s.MaxSizeHisto,
		s.ColumnWidth,
		s.ViewportStep,
	), fg1)

	s.DrawString(10, 40, fmt.Sprintf("graph-time-diff: %s", now.Sub(s.Graph.CurrentTime)), fg1)
}
