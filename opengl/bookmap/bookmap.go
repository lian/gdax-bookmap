package bookmap

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
	StatusImage         *image.RGBA
	GraphImage          *image.RGBA
	StatsImage          *image.RGBA
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

	s.PriceSteps = float64(s.ProductInfo.QuoteIncrement) * 500
	if program != nil {
		s.Texture.Setup(program)
	}
	s.Image = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	s.GraphImage = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width-145), int(s.Texture.Height-s.RowHeight)))
	s.StatsImage = image.NewRGBA(image.Rect(0, 0, int(145), int(s.Texture.Height-s.RowHeight)))
	s.StatusImage = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.RowHeight)))
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
	price := s.Graph.Book.CenterPrice()
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
		graph := NewGraph(s.DB, s.ProductInfo.DatabaseKey, int(s.Texture.Width-145), int(s.Texture.Height-s.RowHeight), int(s.ColumnWidth), int(s.ViewportStep))
		if graph.SetStart(now) {
			s.Graph = graph
		}
		return false
	}

	s.DoAutoScroll()

	return s.Graph.SetEnd(now)
}

func (s *Bookmap) DrawGraph() {
	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}

	//img := image.NewRGBA(image.Rect(0, 0, int(s.Graph.Width), int(s.Graph.Height)))
	img := s.GraphImage

	gc := draw2dimg.NewGraphicContext(img)

	// fill texture with default background
	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, float64(s.Graph.Width), float64(s.Graph.Height))
	gc.Fill()

	x := float64(s.Graph.Width)
	rowCount := ((float64(s.Graph.Height) - s.RowHeight) / s.RowHeight)
	s.Graph.DrawTimeslots(gc, x, rowCount, s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawTradeDots(gc, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps, s.MaxSizeHisto)
	s.Graph.DrawBidAskLines(img, x, s.RowHeight, s.PriceScrollPosition, s.PriceSteps)
	s.Graph.DrawTimeline(gc, img, x, rowCount*s.RowHeight)

	b := image.Rect(0, int(s.RowHeight), int(s.Graph.Width), int(s.Graph.Height)+int(s.RowHeight))
	draw.Draw(s.Image, b, img, img.Bounds().Min, draw.Src)
}

func (s *Bookmap) DrawGraphStats() {
	zeroTime := time.Time{}
	statsSlot := NewTimeSlot(zeroTime, zeroTime)
	rows := ((float64(s.Graph.Height) - s.RowHeight) / s.RowHeight)
	statsSlot.GenerateRows(rows, s.PriceScrollPosition, s.PriceSteps)
	stats := s.Graph.Book.StateAsStats()
	statsSlot.Fill(stats)

	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	green := color.RGBA{0x4d, 0xa5, 0x3c, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	//img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
	img := s.StatsImage
	gc := draw2dimg.NewGraphicContext(img)

	//width, height := 80, s.Graph.Height
	width, height := 145, s.Graph.Height

	// fill texture with default background
	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, float64(width), float64(height))
	gc.Fill()

	x := float64(0)
	// draw current (statsSlot) volume slot
	gc.SetLineWidth(0.5)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(fg1)
	gc.MoveTo(x, 0)
	gc.LineTo(x, float64(rows*s.RowHeight))
	gc.Fill()

	var y float64
	//xx := x + 1 + 70 // 20 = font width
	xx := x + 4 + (float64(len(s.ProductInfo.FormatFloat(statsSlot.Rows[0].Heigh))) * font.Width) + (2 * font.Width)

	fontPad := int((s.RowHeight - font.Height) / 2.0)
	for n, row := range statsSlot.Rows {
		y = float64(n) * s.RowHeight

		//draw2dkit.Rectangle(gc, xx, y+2, xx+width, y+s.RowHeight-2)
		//gc.SetFillColor(bg1)
		//gc.Fill()

		var size float64

		if row.Size > 0 {
			if row.BidCount != 0 && row.AskCount != 0 {
				size = 2 + (float64(width) * (row.AskSize / (statsSlot.MaxSize)))
				y1 := y
				y2 := y + s.RowHeight/2
				draw2dkit.Rectangle(gc, float64(x+1), y1, float64(x+1)+size, y2)
				gc.SetFillColor(red)
				gc.Fill()

				size = 2 + (float64(width) * (row.BidSize / (statsSlot.MaxSize)))
				y1 = y2
				y2 = y + s.RowHeight
				draw2dkit.Rectangle(gc, float64(x+1), y1, float64(x+1)+size, y2)
				gc.SetFillColor(green)
				gc.Fill()
			} else {
				if row.BidCount != 0 {
					gc.SetFillColor(green)
				} else {
					gc.SetFillColor(red)
				}

				size := 2 + (float64(width) * (row.Size / (statsSlot.MaxSize)))
				draw2dkit.Rectangle(gc, float64(x+1), y, float64(x+1)+size, y+s.RowHeight)
				gc.Fill()
			}
			font.DrawString(img, int(xx), int(y)+fontPad, fmt.Sprintf("%.2f (%d)", row.Size, row.OrderCount), fg1)
		}

		/*
			//gc.MoveTo(0, y+s.RowHeight)
			gc.MoveTo(float64(x), y+s.RowHeight)
			gc.LineTo(float64(width), y+s.RowHeight)
			gc.Stroke()
		*/

		//if math.Mod(float64(n), 2) == 0 {
		font.DrawString(img, int(x+4), int(y)+fontPad, s.ProductInfo.FormatFloat(row.Heigh), fg1)
		//}
	}

	//b := image.Rect(0, 0, s.Graph.Width, int(height))
	b := image.Rect(int(s.Graph.Width), int(s.RowHeight), int(s.Graph.Width+width), int(s.Graph.Height)+int(s.RowHeight))
	draw.Draw(s.Image, b, img, img.Bounds().Min, draw.Src)
}

func (s *Bookmap) Render() {
	if !s.Progress() {
		s.WriteTexture()
		return
	}

	if s.MaxSizeHisto == 0 || s.AutoHistoSize {
		s.MaxSizeHisto = round(s.Graph.MaxHistoSize()*0.60, 0)
	}

	s.DrawGraph()
	s.DrawGraphStats()

	now := time.Now()
	s.DrawStatus(now)

	s.WriteTexture()
}

func (s *Bookmap) DrawStatus(now time.Time) {
	//img := image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.RowHeight)))
	img := s.StatusImage
	gc := draw2dimg.NewGraphicContext(img)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}

	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.RowHeight)
	gc.Fill()

	text := fmt.Sprintf(
		"%s %s   PriceSteps %s MaxSizeHisto %.2f ColumnWidth %.0f ViewportStep %d time-diff %s",
		s.ProductInfo.DatabaseKey,
		s.ProductInfo.FormatFloat(s.Graph.Book.LastPrice()),
		s.ProductInfo.FormatFloat(s.PriceSteps),
		s.MaxSizeHisto,
		s.ColumnWidth,
		s.ViewportStep,
		now.Sub(s.Graph.CurrentTime),
	)

	font.DrawString(img, 10, 2, text, fg1)
	b := image.Rect(0, 0, int(s.Texture.Width), int(s.RowHeight))
	draw.Draw(s.Image, b, img, img.Bounds().Min, draw.Src)
}
