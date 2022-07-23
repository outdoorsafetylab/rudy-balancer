package cache

import (
	"bytes"
	"image"
	"image/jpeg"
	"io/ioutil"
	"net/http"
)

func GetImage(url string) (image.Image, error) {
	data, err := Get(url)
	if err != nil {
		return nil, err
	}
	if data == nil {
		res, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		data, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
		err = Set(url, data)
		if err != nil {
			return nil, err
		}
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
