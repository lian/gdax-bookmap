package trades

import (
	"fmt"
	"image"
	"image/color"

	"github.com/lian/gdax/orderbook"
	"github.com/lian/gdax/websocket"

	font "github.com/lian/gonky/font/terminus"

	"github.com/lian/gonky/shader"
	"github.com/lian/gonky/texture"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

type Trades struct {
	ID      string
	Texture *texture.Texture
	gdax    *websocket.Client
}

func New(program *shader.Program, gdax *websocket.Client, id string, height float64, x float64) *Trades {
	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	timePadding := pricePadding + (font.Width * 12)
	width := timePadding + 20
	s := &Trades{
		ID:   id,
		gdax: gdax,
		Texture: &texture.Texture{
			X:      x,
			Y:      height + 10,
			Width:  float64(width),
			Height: height,
		},
	}
	s.Texture.Setup(program)
	s.Render()
	return s
}

func (s *Trades) Render() {
	data := image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	gc := draw2dimg.NewGraphicContext(data)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	green := color.RGBA{0x84, 0xf7, 0x66, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	book := s.gdax.Books[s.ID]

	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	timePadding := pricePadding + (font.Width * 12)
	lineHeight := font.Height + 2

	var latestPrice float64
	i := len(book.Trades)
	if i > 0 {
		latestPrice = book.Trades[i-1].Price
		font.DrawString(data, 10, 5, fmt.Sprintf("%s  %.2f", book.ID, latestPrice), fg1)
	} else {
		font.DrawString(data, 10, 5, book.ID, fg1)
	}

	limit := (int(s.Texture.Height) / lineHeight) - 4
	tradesCount := len(book.Trades)
	if tradesCount < limit {
		limit = tradesCount
	}

	x := 0
	y := lineHeight * 2
	for i := tradesCount - 1; i >= (tradesCount - limit); i-- {
		trade := book.Trades[i]

		var fg color.Color
		if trade.Side == orderbook.BidSide {
			fg = red
		} else {
			fg = green
		}

		size := fmt.Sprintf("%.8f", trade.Size)
		cx := x + (sizePadding - (len(size) * font.Width))
		font.DrawString(data, cx, y, size, fg1)

		price := fmt.Sprintf("%.2f", trade.Price)
		cx = x + (pricePadding - (len(price) * font.Width))
		font.DrawString(data, cx, y, price, fg)

		tradeTime := trade.Time.Format("15:04:05")
		cx = x + (timePadding - (len(tradeTime) * font.Width))
		font.DrawString(data, cx, y, tradeTime, fg1)

		y += lineHeight
	}

	s.Texture.Write(&data.Pix)
}
