package main

import (
	//"github.com/nfnt/resize"
	"os"
	"log"
	"fmt"
	"flag"
	"image"
	"image/jpeg"
	"image/color"
)

func clamp(f float32) uint16 {
	i := int32(f * 65535)
	if i < 0 {
		i = 0
	}
	if (i > 65535) {
		i = 65535
	}

	return uint16(i)
}

func accumulate(dst []float32, src image.Image) {
	inBounds := src.Bounds()

	var w = inBounds.Dx()

	for y := inBounds.Min.Y; y < inBounds.Max.Y; y++ {
		for x := inBounds.Min.X; x < inBounds.Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			dst[4*(y*w+x)+0] += float32(r) / float32(65535.0)
			dst[4*(y*w+x)+1] += float32(g) / float32(65535.0)
			dst[4*(y*w+x)+2] += float32(b) / float32(65535.0)
			dst[4*(y*w+x)+3] += float32(a) / float32(65535.0)
		}
	}
}

func divBy(dst []float32, n float32) {

	invN := 1.0 / n

	for i := 0; i < len(dst); i++ {
		dst[i] *= invN
	}

}

func main() {

	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("needs input")
		os.Exit(1)
	}

	filename := flag.Arg(0)
	f, err := os.Open(filename)
    if err != nil {
    	log.Fatal(err)
	}

    img, err := jpeg.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
    f.Close()

	inBounds := img.Bounds()
	len := inBounds.Dx() * inBounds.Dy()
	fmt.Println(len)
	var fimg = make([]float32, len * 4)
	for i := 0; i < len * 4; i++ {
		fimg[i] = 0.0
	}

	n := 16

	for i := 0; i < n; i++ {
		accumulate(fimg, img);
	}

	divBy(fimg, float32(n))

	outimg := image.NewRGBA(inBounds)

	var w = inBounds.Dx()

	for y := inBounds.Min.Y; y < inBounds.Max.Y; y++ {
		for x := inBounds.Min.X; x < inBounds.Max.X; x++ {
			r := fimg[4*(y*w+x)+0]
			g := fimg[4*(y*w+x)+1]
			b := fimg[4*(y*w+x)+2]
			a := fimg[4*(y*w+x)+3]
			rgba := color.RGBA64{clamp(r), clamp(g), clamp(b), clamp(a)}
			outimg.Set(x, y, rgba)
		}
	}
	
    out, err := os.Create("output.jpg")
    if err != nil {
        log.Fatal(err)
    }
    defer out.Close()

    // write new image to file
    jpeg.Encode(out, outimg, nil)

}
