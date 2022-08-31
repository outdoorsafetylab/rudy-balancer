package controller

import (
	"bytes"
	"errors"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"net/http"
	"sync"

	"service/log"

	"github.com/nfnt/resize"
	"github.com/skip2/go-qrcode"
)

type QRCodeController struct {
	Cache sync.Map
}

func (c *QRCodeController) Generate(w http.ResponseWriter, r *http.Request) {
	var data []byte
	val, _ := c.Cache.Load(r.RequestURI)
	if val != nil {
		data = val.([]byte)
	} else {
		text := stringVar(r, "text", "")
		if text == "" {
			http.Error(w, "Missing 'text'", 500)
			return
		}
		icon := stringVar(r, "icon", "")
		size := intVar(r, "size", 512)
		log.Debugf("Generating QR code: %s", text)
		code, err := qrcode.New(text, qrcode.Highest)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		img := code.Image(int(size))
		if icon != "" {
			iconImage, err := c.getIcon(icon)
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
		var buf bytes.Buffer
		err = png.Encode(&buf, img)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		data = buf.Bytes()
		c.Cache.Store(r.RequestURI, data)
	}
	_, err := w.Write(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
}

func (c *QRCodeController) overlayLogo(srcImage, logoImage image.Image) image.Image {
	offset := image.Pt((srcImage.Bounds().Dx()-logoImage.Bounds().Dx())/2, (srcImage.Bounds().Dy()-logoImage.Bounds().Dy())/2)
	bounds := srcImage.Bounds()
	outout := image.NewNRGBA(bounds)
	draw.Draw(outout, bounds, srcImage, image.Point{}, draw.Src)
	draw.Draw(outout, logoImage.Bounds().Add(offset), logoImage, image.Point{}, draw.Over)
	return outout
}

func (c *QRCodeController) getIcon(url string) (image.Image, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, errors.New(res.Status)
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewBuffer(data))
	if err != nil {
		img, err = jpeg.Decode(bytes.NewBuffer(data))
		if err != nil {
			return nil, err
		}
	}
	return img, nil
}
