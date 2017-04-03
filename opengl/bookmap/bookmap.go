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

const TimeFormat = "2006-01-02T15:04:05.999999Z07:00"

type Bookmap struct {
	Texture             *texture.Texture
	PriceScrollPosition float64
	PriceSteps          float64 // zoom
	MaxSizeHisto        float64
	RowHeight           float64
	book                *orderbook.Book
	gdax                *websocket.Client
	ColumnWidth         float64
	ViewportStep        int
	Graph               *Graph
}

func New(program *shader.Program, width, height float64, x float64, book *orderbook.Book, gdax *websocket.Client) *Bookmap {
	s := &Bookmap{
		book: book,
		gdax: gdax,
		//PriceScrollPosition: 0,
		PriceSteps: 0.50,
		//PriceSteps: 0.02,
		RowHeight: 20,
		//MaxSizeHisto: 80.0,
		//MaxSizeHisto: 1600.0,
		ColumnWidth:  4,
		ViewportStep: 1,
		Texture: &texture.Texture{
			X:      x,
			Y:      height + 10,
			Width:  width,
			Height: height,
		},
	}
	s.Texture.Setup(program)
	return s
}

// ugly af
func round(k float64, precision int) float64 {
	format := fmt.Sprintf("%%.%df", precision)
	i := fmt.Sprintf(format, k)
	f, _ := strconv.ParseFloat(i, 64)
	return f
}

func colourGradientor(p float64, begin, end color.RGBA) color.RGBA {
	if p > 1.0 {
		p = 1.0
	}
	w := p*2 - 1
	w1 := (w + 1) / 2.0
	w2 := 1 - w1

	r := uint8(float64(begin.R)*w1 + float64(end.R)*w2)
	g := uint8(float64(begin.G)*w1 + float64(end.G)*w2)
	b := uint8(float64(begin.B)*w1 + float64(end.B)*w2)

	return color.RGBA{R: r, G: g, B: b, A: 0xff}
}

func (s *Bookmap) InitPriceScrollPosition() {
	if s.PriceScrollPosition != 0.0 {
		return
	}

	rowsCount := s.Texture.Height / s.RowHeight

	if len(s.book.Ask) > 0 {
		s.PriceScrollPosition = round(s.book.Ask[0].Price, 0) + (float64(int(rowsCount/2)) * s.PriceSteps)
	}
}

func (s *Bookmap) Render() {
	data := image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	gc := draw2dimg.NewGraphicContext(data)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	green := color.RGBA{0x4d, 0xa5, 0x3c, 0xff}
	green2 := color.RGBA{0x84, 0xf7, 0x66, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	now := time.Now()

	if s.Graph == nil {
		graph := NewGraph(s.gdax.DB, s.book.ID, int(s.Texture.Width-80), int(s.ColumnWidth), int(s.ViewportStep), s.gdax)

		tnow := now.Unix() - 1
		tnow += int64(graph.SlotSteps) - int64(math.Mod(float64(tnow), float64(graph.SlotSteps)))
		start := time.Unix(tnow-int64(1*graph.SlotSteps), 0)

		if graph.SetStart(start) {
			s.Graph = graph
		}

		s.Texture.Write(&data.Pix)
		return
	}

	s.InitPriceScrollPosition()
	x := s.Texture.Width - 80

	tnow := now.Unix()
	tnow += int64(s.Graph.SlotSteps) - int64(math.Mod(float64(tnow), float64(s.Graph.SlotSteps)))
	end := time.Unix(tnow-int64(0*s.Graph.SlotSteps), 0)
	if !s.Graph.SetEnd(end) {
		s.Texture.Write(&data.Pix)
		return
	}

	statsSlot := NewTimeSlot(s, now, now)
	stats := s.Graph.Book.Book.StateAsStats()
	statsSlot.Fill(stats)
	if s.MaxSizeHisto == 0 {
		s.MaxSizeHisto = round(statsSlot.MaxSize/2, 0)
	}

	cx := x

	maxIdx := len(s.Graph.Timeslots)
	for idx := maxIdx - 1; idx >= 0; idx-- {
		slot := s.Graph.Timeslots[idx]
		//fmt.Println("slot", slot.From, slot.To, slot.Stats == nil)

		cx -= s.ColumnWidth
		if cx < 0 {
			break
		}

		if len(slot.Rows) == 0 {
			count := (s.Texture.Height / s.RowHeight)
			slot.GenerateRows(count, s.PriceScrollPosition, s.PriceSteps)
			slot.Refill()
		} else {
			if idx >= (maxIdx - 3) { // only need to refill last/current two
				slot.Refill()
			}
		}

		x1 := cx
		x2 := cx + s.ColumnWidth

		for i, row := range slot.Rows {
			strength := (row.Size / s.MaxSizeHisto)
			if strength > 0 {
				y := float64(i) * s.RowHeight
				draw2dkit.Rectangle(gc, x1, y, x2, y+s.RowHeight)
				gc.SetFillColor(colourGradientor(strength, fg1, bg1))
				gc.SetStrokeColor(color.Black)
				//gc.FillStroke()
				gc.Fill()
			}
		}
	}

	gc.SetLineWidth(2.0)

	// ask line
	cx = x
	y := 0.0
	start := true
	maxIdx = len(s.Graph.Timeslots)
	for idx := maxIdx - 1; idx >= 0; idx-- {
		slot := s.Graph.Timeslots[idx]

		cx -= s.ColumnWidth
		if cx < 0 {
			break
		}
		if slot.isEmpty() || slot.AskPrice == 0.0 {
			continue
		}

		y = ((s.PriceScrollPosition - slot.AskPrice) / s.PriceSteps) * s.RowHeight

		if start {
			start = false
			gc.MoveTo(cx+s.ColumnWidth, y)
		} else {
			gc.LineTo(cx+s.ColumnWidth, y)
		}
		gc.LineTo(cx, y)
	}
	gc.SetStrokeColor(red)
	gc.Stroke()

	// bid line
	cx = x
	y = 0.0
	start = true
	maxIdx = len(s.Graph.Timeslots)
	for idx := maxIdx - 1; idx >= 0; idx-- {
		slot := s.Graph.Timeslots[idx]
		cx -= s.ColumnWidth
		if cx < 0 {
			break
		}
		if slot.isEmpty() || slot.BidPrice == 0.0 {
			continue
		}

		y = ((s.PriceScrollPosition - slot.BidPrice) / s.PriceSteps) * s.RowHeight

		if start {
			start = false
			gc.MoveTo(cx+s.ColumnWidth, y)
		} else {
			gc.LineTo(cx+s.ColumnWidth, y)
		}
		gc.LineTo(cx, y)
	}
	gc.SetStrokeColor(green2)
	gc.Stroke()

	dotGreen := green2 // color.RGBA{0x84, 0xf7, 0x66, 0xaa}
	dotRed := red      // color.RGBA{0xff, 0x69, 0x39, 0xaa}

	// trade ask dots
	cx = x
	y = 0.0
	maxIdx = len(s.Graph.Timeslots)
	for idx := maxIdx - 1; idx >= 0; idx-- {
		slot := s.Graph.Timeslots[idx]
		cx -= s.ColumnWidth
		if cx < 0 {
			break
		}
		if slot.isEmpty() || slot.AskTradeSize == 0 {
			continue
		}

		y = ((s.PriceScrollPosition - slot.AskPrice) / s.PriceSteps) * s.RowHeight

		startAngle := 0 * (math.Pi / 180.0)
		angle := 360 * (math.Pi / 180.0)

		xx := (cx + (s.ColumnWidth / 2))
		//size := 6.0
		t := (slot.AskTradeSize / s.MaxSizeHisto)
		if t > 1.0 {
			t = 1.0
		}
		size := 4 + float64(t*10)
		gc.ArcTo(xx, y, size, size, startAngle, angle)
		gc.SetFillColor(dotGreen)
		gc.Fill()
	}

	// trade bid dots
	cx = x
	y = 0.0
	maxIdx = len(s.Graph.Timeslots)
	for idx := maxIdx - 1; idx >= 0; idx-- {
		slot := s.Graph.Timeslots[idx]
		cx -= s.ColumnWidth
		if cx < 0 {
			break
		}
		if slot.isEmpty() || slot.BidTradeSize == 0 {
			continue
		}

		y = ((s.PriceScrollPosition - slot.BidPrice) / s.PriceSteps) * s.RowHeight

		startAngle := 0 * (math.Pi / 180.0)
		angle := 360 * (math.Pi / 180.0)

		xx := (cx + (s.ColumnWidth / 2))
		//size := 6.0
		t := (slot.BidTradeSize / s.MaxSizeHisto)
		if t > 1.0 {
			t = 1.0
		}
		size := 4 + float64(t*10)
		gc.ArcTo(xx, y, size, size, startAngle, angle)
		gc.SetFillColor(dotRed)
		gc.Fill()
	}

	gc.SetLineWidth(1.0)
	gc.SetStrokeColor(fg1)
	gc.SetFillColor(fg1)
	gc.MoveTo(x, 0)
	gc.LineTo(x, s.Texture.Height)
	gc.Fill()

	xx := float64(x + 2)
	for n, row := range statsSlot.Rows {
		if math.Mod(float64(n), 3) == 0 {
			font.DrawString(data, int(xx)-100, int(row.Y)+4, fmt.Sprintf("%.2f", row.Heigh), red)
		}

		width := 80.0
		draw2dkit.Rectangle(gc, xx, row.Y+2, xx+width, row.Y+s.RowHeight-2)
		//gc.SetFillColor(fg1)
		gc.SetFillColor(bg1)
		gc.Fill()

		if row.Size > 0 {
			if row.Type == 0 {
				gc.SetFillColor(green)
			} else {
				gc.SetFillColor(red)
			}

			maxSize := statsSlot.MaxSize
			width = 80 * (row.Size / (maxSize + 10))
			draw2dkit.Rectangle(gc, xx, row.Y+2, xx+width, row.Y+s.RowHeight-2)
			gc.Fill()

			font.DrawString(data, int(xx)+4, int(row.Y)+4, fmt.Sprintf("%.2f (%d)", row.Size, row.OrderCount), fg1)
		}

		//gc.MoveTo(float64(x), y+s.RowHeight)
		//gc.LineTo(float64(x)+200, y+s.RowHeight)
		gc.MoveTo(0, row.Y+s.RowHeight)
		gc.LineTo(s.Texture.Width, row.Y+s.RowHeight)
		gc.Stroke()
	}

	font.DrawString(data, 10, 5, fmt.Sprintf(
		"PriceScrollPosition %.2f PriceSteps %.2f MaxSizeHisto %.2f ColumnWidth %.0f ViewportStep %d",
		s.PriceScrollPosition,
		s.PriceSteps,
		s.MaxSizeHisto,
		s.ColumnWidth,
		s.ViewportStep,
	), fg1)

	font.DrawString(data, 10, 25, fmt.Sprintf("graph-time-diff: %s", now.Sub(s.Graph.CurrentTime)), fg1)

	s.Texture.Write(&data.Pix)
}
