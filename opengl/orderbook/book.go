package orderbook

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

type Orderbook struct {
	ID      string
	Texture *texture.Texture
	gdax    *websocket.Client
}

func New(program *shader.Program, gdax *websocket.Client, id string, n int) *Orderbook {
	width := 300
	padding := 10
	s := &Orderbook{
		ID:   id,
		gdax: gdax,
		Texture: &texture.Texture{
			X:      float64(padding + (n * (width + padding))),
			Y:      510,
			Width:  float64(width),
			Height: 500,
		},
	}
	s.Texture.Setup(program)
	return s
}

func (s *Orderbook) Render() {
	data := image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	gc := draw2dimg.NewGraphicContext(data)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	//bg2 := color.RGBA{0x2f, 0x3d, 0x45, 0xff}
	fg1 := color.RGBA{0xce, 0xd2, 0xd5, 0xff}

	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	book := s.gdax.Books[s.ID]

	limit := ((int(s.Texture.Height) / 2) / font.Height) - 1
	bids, asks := book.StateCombined()

	var latestPrice float64
	i := len(book.Trades)
	if i > 0 {
		latestPrice = book.Trades[i-1].Price
	} else {
		if len(book.Ask) > 0 {
			latestPrice = bids[0].Price
		}
	}

	var spread float64
	if len(bids) > 0 && len(asks) > 0 {
		spread = asks[0].Price - bids[0].Price
	}

	font.DrawString(data, 10, 5, fmt.Sprintf("%s  %.2f", book.ID, latestPrice), fg1)

	ask_limit := limit
	if len(asks) < ask_limit {
		ask_limit = len(asks)
	}

	x := 10
	y := (int(s.Texture.Height) / 2)
	for _, s := range asks[:ask_limit] {
		text := fmt.Sprintf("%.8f    %.2f", s.Size, s.Price)
		y -= font.Height
		font.DrawString(data, x, y, text, fg1)
	}

	bid_limit := limit
	if len(bids) < bid_limit {
		bid_limit = len(bids)
	}

	x = 10
	y = (int(s.Texture.Height) / 2)
	for _, s := range bids[:bid_limit] {
		text := fmt.Sprintf("%.8f    %.2f", s.Size, s.Price)
		y += font.Height
		font.DrawString(data, x, y, text, fg1)
	}

	y = (int(s.Texture.Height) / 2)
	text := fmt.Sprintf("%.2f", spread)
	font.DrawString(data, x, y, text, fg1)

	x = 180
	y = 20 - font.Height
	for i := len(book.Trades) - 1; i >= 0; i-- {
		trade := book.Trades[i]

		var fg color.Color
		if trade.Side == orderbook.BidSide {
			fg = color.RGBA{0xff, 0x00, 0xd5, 0xff}
		} else {
			fg = color.RGBA{0x00, 0xff, 0xd5, 0xff}
		}
		text := fmt.Sprintf("%.8f  %.2f", trade.Size, trade.Price)
		y += font.Height
		font.DrawString(data, x, y, text, fg)
	}

	s.Texture.Write(&data.Pix)
}
