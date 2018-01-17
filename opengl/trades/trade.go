package trades

import (
	"fmt"
	"image"
	"image/color"

	"github.com/lian/gdax-bookmap/opengl/bookmap"
	"github.com/lian/gdax-bookmap/orderbook"
	"github.com/lian/gdax-bookmap/orderbook/product_info"

	font "github.com/lian/gonky/font/terminus"

	"github.com/lian/gonky/shader"
	"github.com/lian/gonky/texture"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/llgcode/draw2d/draw2dkit"
)

type Trades struct {
	ProductInfo product_info.Info
	ID          string
	Texture     *texture.Texture
	bookmap     *bookmap.Bookmap
	Image       *image.RGBA
}

func New(program *shader.Program, bookmap *bookmap.Bookmap, info product_info.Info, height float64, x float64) *Trades {
	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	timePadding := pricePadding + (font.Width * 12)
	width := timePadding + 20
	s := &Trades{
		ID:          info.ID,
		ProductInfo: info,
		bookmap:     bookmap,
		Texture: &texture.Texture{
			X:      x,
			Y:      height + 10,
			Width:  float64(width),
			Height: height,
		},
	}
	s.Texture.Setup(program)
	s.Image = image.NewRGBA(image.Rect(0, 0, int(s.Texture.Width), int(s.Texture.Height)))
	s.Render()
	return s
}

func (s *Trades) Render() {
	data := s.Image
	gc := draw2dimg.NewGraphicContext(data)

	bg1 := color.RGBA{0x15, 0x23, 0x2c, 0xff}
	fg1 := color.RGBA{0xdd, 0xdf, 0xe1, 0xff}
	//green := color.RGBA{0x84, 0xf7, 0x66, 0xff}
	green := color.RGBA{0x4d, 0xa5, 0x3c, 0xff}
	red := color.RGBA{0xff, 0x69, 0x39, 0xff}

	gc.SetFillColor(bg1)
	draw2dkit.Rectangle(gc, 0, 0, s.Texture.Width, s.Texture.Height)
	gc.Fill()

	if s.bookmap.Graph == nil {
		return
	}
	book := s.bookmap.Graph.Book.Book

	sizePadding := font.Width * 15
	pricePadding := sizePadding + (font.Width * 12)
	timePadding := pricePadding + (font.Width * 12)
	lineHeight := font.Height + 2

	font.DrawString(data, 10, 5, fmt.Sprintf("%s  %.2f", book.ID, book.LastPrice()), fg1)

	limit := (int(s.Texture.Height) / lineHeight) - 3

	book.MuTrades.Lock()
	trades := book.Trades
	book.MuTrades.Unlock()

	tradesCount := len(trades)
	if tradesCount < limit {
		limit = tradesCount
	}

	x := 0
	y := lineHeight * 2
	for i := tradesCount - 1; i >= (tradesCount - limit); i-- {
		trade := trades[i]

		var fg color.Color
		if trade.Side == orderbook.BidSide {
			fg = red
		} else {
			fg = green
		}

		size := fmt.Sprintf("%.8f", trade.Size)
		cx := x + (sizePadding - (len(size) * font.Width))
		font.DrawString(data, cx, y, size, fg1)

		price := s.ProductInfo.FormatFloat(trade.Price)
		cx = x + (pricePadding - (len(price) * font.Width))
		font.DrawString(data, cx, y, price, fg)

		tradeTime := trade.Time.Format("15:04:05")
		cx = x + (timePadding - (len(tradeTime) * font.Width))
		font.DrawString(data, cx, y, tradeTime, fg1)

		y += lineHeight
	}

	s.Texture.Write(&data.Pix)
}
