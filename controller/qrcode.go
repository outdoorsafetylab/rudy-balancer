package controller

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"io"
	"service/model"
	"strconv"

	"github.com/crosstalkio/rest"
	"github.com/nfnt/resize"
	"github.com/skip2/go-qrcode"
)

type QRCodeController struct {
	Artifact *model.Artifact
}

func (c *QRCodeController) Get(s *rest.Session) {
	text := s.Var("text", "")
	if text == "" {
		s.Status(400, "Missing 'text'")
		return
	}
	v := s.Var("size", "512")
	size, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		s.Status(400, err)
		return
	}
	var buf bytes.Buffer
	code, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		s.Status(500, err)
		return
	}
	img := code.Image(int(size))
	if c.Artifact.IconImage != nil {
		percent := 20
		logoSize := uint(float64(size) * float64(percent) / 100)
		logoImage := resize.Resize(logoSize, logoSize, c.Artifact.IconImage, resize.Lanczos3)
		img = c.overlayLogo(img, logoImage)
	}
	err = png.Encode(&buf, img)
	if err != nil {
		s.Status(500, err)
		return
	}
	s.ResponseHeader().Set("Content-Type", "image/png")
	_, err = io.Copy(s.ResponseWriter, &buf)
	if err != nil {
		s.Status(500, err)
		return
	}
	s.Status(200, nil)
}

func (c *QRCodeController) overlayLogo(srcImage, logoImage image.Image) image.Image {
	offset := image.Pt((srcImage.Bounds().Dx()-logoImage.Bounds().Dx())/2, (srcImage.Bounds().Dy()-logoImage.Bounds().Dy())/2)
	bounds := srcImage.Bounds()
	outout := image.NewNRGBA(bounds)
	draw.Draw(outout, bounds, srcImage, image.Point{}, draw.Src)
	draw.Draw(outout, logoImage.Bounds().Add(offset), logoImage, image.Point{}, draw.Over)
	return outout
}
