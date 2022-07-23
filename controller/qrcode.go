package controller

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"service/cache"

	"github.com/nfnt/resize"
	log "github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

type QRCodeController struct {
}

func (c *QRCodeController) Generate(w http.ResponseWriter, r *http.Request) {
	text := stringVar(r, "text", "")
	if text == "" {
		http.Error(w, "Missing 'text'", 500)
		return
	}
	icon := stringVar(r, "icon", "")
	size := intVar(r, "size", 512)
	var buf bytes.Buffer
	log.Debugf("Generating QR code: %s", text)
	code, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	img := code.Image(int(size))
	if icon != "" {
		iconImage, err := cache.GetImage(icon)
		if err != nil {
			log.Errorf("Failed to get icon: %s: %s", icon, err.Error())
			http.Error(w, err.Error(), 500)
			return
		}
		percent := 20
		logoSize := uint(float64(size) * float64(percent) / 100)
		logoImage := resize.Resize(logoSize, logoSize, iconImage, resize.Lanczos3)
		img = c.overlayLogo(img, logoImage)
	}
	err = png.Encode(&buf, img)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, err = io.Copy(w, &buf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (c *QRCodeController) overlayLogo(srcImage, logoImage image.Image) image.Image {
	offset := image.Pt((srcImage.Bounds().Dx()-logoImage.Bounds().Dx())/2, (srcImage.Bounds().Dy()-logoImage.Bounds().Dy())/2)
	bounds := srcImage.Bounds()
	outout := image.NewNRGBA(bounds)
	draw.Draw(outout, bounds, srcImage, image.Point{}, draw.Src)
	draw.Draw(outout, logoImage.Bounds().Add(offset), logoImage, image.Point{}, draw.Over)
	return outout
}
