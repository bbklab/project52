package main

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"unicode/utf8"

	zbar "github.com/PeterCxy/gozbar" // zbar decoder, required: yum -y install zbar zbar-devel,  require CGO building
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/makiuchi-d/gozxing" // zxing decoer
	"github.com/makiuchi-d/gozxing/qrcode"

	enc "github.com/skip2/go-qrcode"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "qrcode"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "decode,d",
			Usage: "decode given qrcode image file",
		},
	}

	app.Action = func(c *cli.Context) error {
		var (
			arg    = c.Args().First()
			decode = c.Bool("decode")
		)
		if arg == "" {
			return cli.ShowSubcommandHelp(c)
		}

		// decode qrcode png file
		if decode {
			png, err := ioutil.ReadFile(arg)
			if err != nil {
				return err
			}

			content, err := ZbarDecode(png)
			if err != nil {
				content, err = ZxingDecode(png)
				if err != nil {
					return err
				}
			}

			content = string(GBK2UTF8([]byte(content)))
			os.Stdout.Write(append([]byte(content), '\r', '\n'))
			return nil
		}

		// encode any string
		png, err := Encode(arg)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile("qrcode.png", png, os.FileMode(0644))
		if err != nil {
			return err
		}

		os.Stdout.Write(append([]byte("qrcode.png"), '\r', '\n'))
		return nil
	}

	app.RunAndExitOnError()
}

// Encode is exported
func Encode(data string) ([]byte, error) {
	return enc.Encode(data, enc.Medium, 256)
}

func ZxingDecode(png []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewBuffer(png))
	if err != nil {
		return "", err
	}

	// prepare BinaryBitmap
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}

	// decode image
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", errors.New("zxing: no symbols found")
	}

	return result.String(), nil
}

// ZBarDecode is exported
func ZbarDecode(png []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewBuffer(png))
	if err != nil {
		return "", err
	}

	s := zbar.NewScanner()
	s.SetConfig(zbar.QRCODE, zbar.CFG_ENABLE, 1) // sometimes some qrcode image can't be recognized

	image := zbar.FromImage(img)
	result := s.Scan(image)
	if result <= 0 {
		return "", errors.New("zbar: no symbols found")
	}

	var res []string
	image.First().Each(func(item string) {
		res = append(res, item)
	})

	if len(res) == 0 {
		return "", errors.New("zbar: no symbols found")
	}

	return res[0], nil
}

// utils
//
func isUTF8(b []byte) bool {
	return utf8.Valid(b)
}

// GBK2UTF8 is exported
func GBK2UTF8(b []byte) []byte {
	if isUTF8(b) {
		return b
	}
	reader := transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewDecoder())
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return b
	}
	return bs
}

// UTF82GBK is exported
func UTF82GBK(b []byte) []byte {
	if !isUTF8(b) {
		return b
	}
	reader := transform.NewReader(bytes.NewReader(b), simplifiedchinese.GBK.NewEncoder())
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return b
	}
	return bs
}
