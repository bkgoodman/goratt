package main

import (
	"fmt"
  "time"
  "os"
  "github.com/fogleman/gg"
)

func dymo_print(dc *gg.Context, lpfile string) error {
    usbDeviceFile, err := os.OpenFile(lpfile, os.O_RDWR, 0644)

    if err != nil {
        return err
    }
    defer usbDeviceFile.Close()

    img := dc.Image()
    fmt.Println("XY Are",dc.Width(),dc.Height())
    
    bpl := dc.Height()/ 8
    lines  := dc.Width()

    usbDeviceFile.Write([]byte{27,0x44,byte(bpl)}) // Printer Width (Bytes)
    usbDeviceFile.Write([]byte{27,0x4c,byte((lines >> 8)&0xff),byte(lines &0xff)}) // 16 lines on lable

    for x := 0; x < img.Bounds().Max.X; x++ {
      usbDeviceFile.Write([]byte{byte(0x16)})
    for y := img.Bounds().Max.Y-1;y>=0; y-=8 {
      data := 0
      for i:=0;i<8;i++ {
            r, _, _, _ := img.At(x, y+i).RGBA()
            if (r <= 0x8000) {
              //data |= (1 << (7-i))
              data |= (1 << i)
              //fmt.Println("-- PT",x+i,y,r)
            }
          }
          usbDeviceFile.Write([]byte{byte(data)})
        }
    }
    usbDeviceFile.Write([]byte{27,'E'}) // Form Feed
    return nil
}




func dymo_label(name string) {
    lines :=960
    bpl := 38 // Bytes Per Line

    // We are doing this rotated
    var HEIGHT = (bpl * 8)
    var WIDTH = lines
    dc := gg.NewContext(WIDTH,HEIGHT)
    dc.SetRGB(1, 1, 1)
    dc.Clear()
    dc.SetRGB(0, 0, 0)
    if err := dc.LoadFontFace("Ubuntu-R.ttf", float64(84)); err != nil {
      panic(err)
    }

    offset := float64(0)
    im, err := gg.LoadPNG("StoragePassTemplate.png")
    if err == nil {
      dc.DrawImage(im, 0, 0)
      //offset =float64(im.Bounds().Dx()) /float64(2 )
      offset=100
    }

    currentDate := time.Now()
    futureDate := currentDate.AddDate(0, 0, 3)
    futureDateString := futureDate.Format("Mon, 02-Jan-06")

    formattedDateTime := currentDate.Format("Mon, 02-Jan-2006 01:04 PM")

    dc.DrawStringAnchored(futureDateString, float64(WIDTH/2)+offset, 180, 0.5, 0.5)
    dc.LoadFontFace("Ubuntu-R.ttf", float64(48))
    //dc.DrawStringAnchored("Temporary Storage Pass", float64(WIDTH/2)+offset, 50, 0.5, 0.5)
    dc.DrawStringAnchored(name, float64(WIDTH/2)+offset, 120, 0.5, 0.5)
    dc.LoadFontFace("Ubuntu-R.ttf", float64(24))
    dc.DrawStringAnchored(fmt.Sprintf("Left on: %s",formattedDateTime), float64(WIDTH/2)+offset, 240, 0.5, 0.5)
    dc.SetLineWidth(2)
    dc.DrawRectangle(10, 10, float64(WIDTH-10), float64(HEIGHT-10))
    dc.Stroke()

    if err := dc.LoadFontFace("Ubuntu-R.ttf", float64(18)); err != nil {
      panic(err)
    }

    textbody := "Items may be discarded and disposal charges may be incurred if items are left after specified date."
    dc.DrawStringWrapped(textbody,float64(WIDTH/2)+offset,float64(HEIGHT-32) , 0.5, 0.5, float64(WIDTH/2), 1.2, gg.AlignCenter)

    err = dymo_print(dc, "/dev/usb/lp0")
    if (err != nil) {
      fmt.Println("Error",err)
    }
  }
