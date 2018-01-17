package bookmap

import (
	"image"
	"image/color"
	"math"

	font "github.com/lian/gonky/font/terminus"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

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

const circleStartAngle float64 = 0 * (math.Pi / 180.0)
const circleAngle float64 = 360 * (math.Pi / 180.0)

func DrawCircle(gc *draw2dimg.GraphicContext, color color.RGBA, x, y, size float64) {
	gc.ArcTo(x, y, size, size, circleStartAngle, circleAngle)
	gc.SetFillColor(color)
	gc.Fill()
}

func (g *Graph) DrawTradeDots(gc *draw2dimg.GraphicContext, x, rowHeight, pricePosition, priceSteps, maxSizeHisto float64) {
	var xx, y float64

	// trade volume ask/bid dots
	for idx := len(g.Timeslots) - 1; idx > 0; idx-- {
		x -= float64(g.SlotWidth)

		if x < 0 {
			break
		}

		slot := g.Timeslots[idx]
		if slot.isEmpty() || (slot.AskTradeSize == 0 && slot.BidTradeSize == 0) {
			continue
		}

		xx = (x + (float64(g.SlotWidth) / 2))

		if slot.AskTradeSize != 0 {
			y = ((pricePosition - slot.AskPrice) / priceSteps) * rowHeight
			t := (slot.AskTradeSize / (maxSizeHisto * 0.8))
			if t > 1.0 {
				t = 1.0
			}
			size := 4 + float64(t*15)
			DrawCircle(gc, g.Green, xx, y, size)
		}

		if slot.BidTradeSize != 0 {
			y = ((pricePosition - slot.BidPrice) / priceSteps) * rowHeight
			t := (slot.BidTradeSize / (maxSizeHisto * 0.8))
			if t > 1.0 {
				t = 1.0
			}
			size := 4 + float64(t*15)
			DrawCircle(gc, g.Red, xx, y, size)
		}
	}
}

func (g *Graph) DrawBidAskLines(img *image.RGBA, x, rowHeight, pricePosition, priceSteps float64) {
	askgc := draw2dimg.NewGraphicContext(img)
	askgc.SetLineWidth(2.0)
	askgc.SetStrokeColor(g.Red)
	askstart := true

	bidgc := draw2dimg.NewGraphicContext(img)
	bidgc.SetLineWidth(2.0)
	bidgc.SetStrokeColor(g.Green)
	bidstart := true

	var y float64

	for idx := len(g.Timeslots) - 1; idx > 0; idx-- {
		slot := g.Timeslots[idx]

		x -= float64(g.SlotWidth)
		if x < 0 {
			break
		}

		if slot.isEmpty() {
			askgc.Stroke()
			askstart = true
			bidgc.Stroke()
			bidstart = true
			continue
		}

		if slot.AskPrice == 0.0 {
			askgc.Stroke()
			askstart = true
		} else {
			y = ((pricePosition - slot.AskPrice) / priceSteps) * rowHeight
			if askstart {
				askstart = false
				askgc.MoveTo(x+float64(g.SlotWidth), y)
			} else {
				askgc.LineTo(x+float64(g.SlotWidth), y)
			}
			askgc.LineTo(x, y)
		}

		if slot.BidPrice == 0.0 {
			bidgc.Stroke()
			bidstart = true
		} else {
			y = ((pricePosition - slot.BidPrice) / priceSteps) * rowHeight
			if bidstart {
				bidstart = false
				bidgc.MoveTo(x+float64(g.SlotWidth), y)
			} else {
				bidgc.LineTo(x+float64(g.SlotWidth), y)
			}
			bidgc.LineTo(x, y)
		}
	}
	askgc.Stroke()
	bidgc.Stroke()
}

func (g *Graph) DrawTimeslots(gc *draw2dimg.GraphicContext, x, rowsCount, rowHeight, pricePosition, priceSteps, maxSizeHisto float64) {
	var x2, y float64

	maxIdx := len(g.Timeslots) - 1
	for idx := maxIdx; idx > 0; idx-- {
		slot := g.Timeslots[idx]

		x -= float64(g.SlotWidth)
		if x < 0 {
			break
		}

		//if len(slot.Rows) == 0 {
		if slot.Cleared {
			slot.GenerateRows(rowsCount, pricePosition, priceSteps)
			slot.Refill()
		} else {
			if idx >= (maxIdx - 1) { // only need to refill last/current two
				slot.Refill()
			}
		}

		x2 = x + float64(g.SlotWidth)

		for i, row := range slot.Rows {
			strength := (row.Size / maxSizeHisto)
			if strength > 0 {
				y = float64(i) * rowHeight
				draw2dkit.Rectangle(gc, x, y, x2, y+rowHeight)
				gc.SetFillColor(colourGradientor(strength, g.Fg1, g.Bg1))
				gc.Fill()
			}
		}
	}
}

func (g *Graph) DrawTimeline(gc *draw2dimg.GraphicContext, image *image.RGBA, x, y float64) {
	for idx := len(g.Timeslots) - 1; idx > 0; idx-- {
		slot := g.Timeslots[idx]

		x -= float64(g.SlotWidth)
		if x < 0 {
			break
		}

		if math.Mod(float64(idx), 30) == 0 {
			/*
				gc.SetLineWidth(1.0)
				gc.SetFillColor(g.Bg1)
				gc.MoveTo(cx, 0)
				gc.LineTo(cx, y)
				gc.Fill()
			*/
			font.DrawString(image, int(x), int(y), slot.From.Format("15:04:05"), g.Fg1)
		}
	}
}
