package orderbook

import (
	"fmt"
	"image"
	"image/color"

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

func New(program *shader.Program, gdax *websocket.Client, id string, height float64, x float64) *Orderbook {
	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	width := pricePadding + 20

	s := &Orderbook{
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
	return s
}

func (s *Orderbook) Render() {
	data := image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	gc := draw2dimg.NewGraphicContext(data)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	//fg1 := color.RGBA{0xce, 0xd2, 0xd5, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	green := color.RGBA{0x84, 0xf7, 0x66, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	book := s.gdax.Books[s.ID]

	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	lineHeight := font.Height + 2

	limit := ((int(s.Texture.Height) / 2) / lineHeight) - 2
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

	font.DrawString(data, 10, 5, fmt.Sprintf("%s  %.2f", book.ID, latestPrice), fg1)

	ask_limit := limit
	if len(asks) < ask_limit {
		ask_limit = len(asks)
	}

	x := 0
	y := (int(s.Texture.Height) / 2)
	for _, s := range asks[:ask_limit] {
		y -= lineHeight

		size := fmt.Sprintf("%.8f", s.Size)
		cx := x + (sizePadding - (len(size) * font.Width))
		font.DrawString(data, cx, y, size, fg1)

		price := fmt.Sprintf("%.2f", s.Price)
		cx = x + (pricePadding - (len(price) * font.Width))
		font.DrawString(data, cx, y, price, red)
	}

	bid_limit := limit
	if len(bids) < bid_limit {
		bid_limit = len(bids)
	}

	x = 0
	y = (int(s.Texture.Height) / 2)
	for _, s := range bids[:bid_limit] {
		y += lineHeight

		size := fmt.Sprintf("%.8f", s.Size)
		cx := x + (sizePadding - (len(size) * font.Width))
		font.DrawString(data, cx, y, size, fg1)

		price := fmt.Sprintf("%.2f", s.Price)
		cx = x + (pricePadding - (len(price) * font.Width))
		font.DrawString(data, cx, y, price, green)
	}

	var spread float64
	if len(bids) > 0 && len(asks) > 0 {
		spread = asks[0].Price - bids[0].Price
	}

	text := fmt.Sprintf("%.2f", spread)
	y = (int(s.Texture.Height) / 2)
	x = (pricePadding - (len(text) * font.Width))
	font.DrawString(data, x, y, text, fg1)

	s.Texture.Write(&data.Pix)
}
