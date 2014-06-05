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
	fmt.Println(f)

	return uint16(i)
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

	var w = inBounds.Dx()
	//var h = inBounds.Dy()

	for y := inBounds.Min.Y; y < inBounds.Max.Y; y++ {
		for x := inBounds.Min.X; x < inBounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			fimg[4*(y*w+x)+0] = float32(r) / float32(65535.0)
			fimg[4*(y*w+x)+1] = float32(g) / float32(65535.0)
			fimg[4*(y*w+x)+2] = float32(b) / float32(65535.0)
			fimg[4*(y*w+x)+3] = float32(a) / float32(65535.0)
			fimg[4*(y*w+x)+0] *= 0.5
			//fmt.Println(r)
		}
	}

	outimg := image.NewRGBA(inBounds)

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
	
    out, err := os.Create("test_resized.jpg")
    if err != nil {
        log.Fatal(err)
    }
    defer out.Close()

    // write new image to file
    jpeg.Encode(out, outimg, nil)

}
