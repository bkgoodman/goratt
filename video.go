package main

import (
	"encoding/binary" // For byte order conversion (PutUint16)
	"fmt"
	"image"
	"log"
	"os"
	"time"
    "strings"
    "golang.org/x/image/draw"
    _ "image/png" 
	"github.com/d21d3q/framebuffer"
	"github.com/fogleman/gg"
)

// This clear function works for any BPP by just zeroing out bytes, resulting in black
func clearFramebuffer(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

func setFontSize(size int) {
	fontPathGG := "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
	fontSizeGG := float64(size)
    err := dc.LoadFontFace(fontPathGG, fontSizeGG)
	if err != nil {
		log.Fatalf("Failed to load font face '%s' for gg: %v. Ensure the font file is present and accessible.", fontPathGG, err)
	}
}

func HowLongAgo(t time.Time) string {
	// Calculate the duration since the given time.
	duration := time.Since(t)

	// Get the total duration in full minutes (truncated, not rounded).
	// This simplifies the logic and ensures "59.9 minutes" is not "1 hour".
	totalMinutes := int(duration.Minutes())

	// Handle the minimum granularity of 1 minute.
	if totalMinutes < 1 {
		return "1 minute"
	}

	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	// Use a slice to build the output string parts, which handles different combinations.
	var parts []string
	if hours > 0 {
		if hours == 1 {
			parts = append(parts, "1 hour")
		} else {
			parts = append(parts, fmt.Sprintf("%d hours", hours))
		}
	}

	if minutes > 0 {
		if minutes == 1 {
			parts = append(parts, "1 minute")
		} else {
			parts = append(parts, fmt.Sprintf("%d minutes", minutes))
		}
	}

	// This case should not be hit due to the totalMinutes < 1 check,
	// but it's a good safety fallback.
	if len(parts) == 0 {
		return "1 minute"
	}

	return strings.Join(parts, ", ")
}
// loadImage loads a PNG from the specified path.
func loadImage(path string) (image.Image, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    img, _, err := image.Decode(file)
    if err != nil {
        return nil, err
    }

    return img, nil
}

func loadAndScaleImage(filePath string, maxW, maxH int) (*image.RGBA, error, int, int) {
	file, err := os.Open(filePath)
	if err != nil {
        fmt.Errorf("Error loading image \"%s\": %w\n",filePath,err)
		return nil, err, 0, 0
	}
	defer file.Close()

	originalImage, _, err := image.Decode(file)
	if err != nil {
        fmt.Errorf("Error decoding image \"%s\": %w\n",filePath,err)
		return nil, err, 0, 0
	}

	// Get the original dimensions of the image
	origBounds := originalImage.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()

	// Calculate the scaling factor
	scaleFactor := 1.0
	if maxW > 0 && origWidth > maxW {
		scaleFactor = float64(maxW) / float64(origWidth)
	}
	if maxH > 0 && origHeight > maxH {
		// If the height constraint results in a smaller scale, use that one
		heightScale := float64(maxH) / float64(origHeight)
		if heightScale < scaleFactor {
			scaleFactor = heightScale
		}
	}

	// If no constraints were specified or the image is already smaller,
	// just use the original size.
	if scaleFactor >= 1.0 {
		return imageToRGBA(originalImage), nil, origWidth, origHeight
	}

	// Calculate the new dimensions
	newWidth := int(float64(origWidth) * scaleFactor)
	newHeight := int(float64(origHeight) * scaleFactor)

	fmt.Printf("Scaling image from %dx%d to %dx%d\n", origWidth, origHeight, newWidth, newHeight)

	// Create a new image with the target dimensions
	downscaledImage := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Use a high-quality scaler.
	draw.CatmullRom.Scale(downscaledImage, downscaledImage.Bounds(), originalImage, originalImage.Bounds(), draw.Over, nil)

	return downscaledImage, nil, newWidth, newHeight
}

// imageToRGBA converts an image.Image to *image.RGBA, if it's not already.
func imageToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

var dc *gg.Context
var pixBuffer []byte
var backBuffer []byte
var WIDTH int
var HEIGHT int
var rgbaImage *image.RGBA
var lineLengthBytes int

func video_init() {
	// Reconciled: Using two parameters for OpenFrameBuffer as per your system's requirement
	fbLowLevel, err := framebuffer.OpenFrameBuffer("/dev/fb0", os.O_RDWR)
	if err != nil {
		log.Fatalf("Failed to open framebuffer: %v\n"+
			"Please ensure /dev/fb0 exists and you have proper permissions (e.g., run with sudo or add user to 'video' group).", err)
	}

	varInfo, err := fbLowLevel.VarScreenInfo()
	if err != nil {
		log.Fatalf("Failed to get variable screen info: %v", err)
	}
	fixedInfo, err := fbLowLevel.FixScreenInfo()
	if err != nil {
		log.Fatalf("Failed to get fixed screen info: %v", err)
	}

	// pixBuffer is the actual 16-bit framebuffer memory
	pixBuffer, err = fbLowLevel.Pixels()
	if err != nil {
		log.Fatalf("Failed to get pixel data from framebuffer: %v", err)
	}


	WIDTH = int(varInfo.XRes)
	HEIGHT = int(varInfo.YRes)
	bitsPerPixel := varInfo.BitsPerPixel
	lineLengthBytes = int(fixedInfo.LineLength) // Stride in bytes for framebuffer

    backBuffer =  make([]byte, HEIGHT*lineLengthBytes)

	fmt.Printf("Framebuffer opened:\n")
	fmt.Printf("  Resolution: %dx%d\n", WIDTH, HEIGHT)
	fmt.Printf("  Bits Per Pixel: %d\n", bitsPerPixel)
	fmt.Printf("  Line Length (stride): %d bytes\n", lineLengthBytes)
	fmt.Printf("  Buffer Size: %d bytes\n", len(pixBuffer))

	// --- NEW: Create a separate 32-bit RGBA image buffer for drawing ---
	// This is where gg and x/image/font will draw. It's in system RAM, not directly FB.
	rgbaImage = image.NewRGBA(image.Rect(0, 0, WIDTH, HEIGHT))
	dc = gg.NewContextForRGBA(rgbaImage) // gg context now draws to this 32-bit image
	// --- END NEW ---
	clearFramebuffer(pixBuffer) // Clear the actual framebuffer directly
}

func video_clear() {
	clearFramebuffer(pixBuffer) // Clear the actual framebuffer directly
    video_update()
}
func video_available() {


    // Background
	dc.SetRGB(0, 0.5, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

	dc.SetRGB(1, 1, 1)

    if (nextCalEntry == nil) {
            setFontSize(64)
            h := float64((HEIGHT-64)/2)
            dc.DrawStringAnchored("Room Available", float64(WIDTH/2), h, 0.5, 0.5)
            setFontSize(32)
            h+=50
            dc.DrawStringAnchored("Swipe fob to use room", float64(WIDTH/2), h, 0.5, 0.5)
    } else {
            setFontSize(64)
            h := float64(110)
            dc.DrawStringAnchored("Room Available", float64(WIDTH/2), h, 0.5, 0.5)
            setFontSize(32)
            h+=50
            dc.DrawStringAnchored("Swipe fob to use room", float64(WIDTH/2), h, 0.5, 0.5)


            h+=35
            dc.SetRGB(1, 1, 1)
            dc.DrawRectangle(20, h, float64(WIDTH-40), 180)
            dc.Fill()
            dc.SetRGB(0.2, 0.5, 0.2)
            h+=20
            dc.DrawStringAnchored("-Next Reservaion -", float64(WIDTH/2), h, 0.5, 0.5)
            h += 55
            dc.DrawStringAnchored(nextCalEntry.SUMMARY, float64(WIDTH/2), h, 0.5, 0.5)
            h+=40
            dc.DrawStringAnchored(nextCalEntry.ORGANIZER, float64(WIDTH/2), h, 0.5, 0.5)
            h+=40
            dc.DrawStringAnchored(nextCalEntry.WHEN, float64(WIDTH/2), h, 0.5, 0.5)

    }


    /* Lower Banner */

    setFontSize(42)
	dc.SetRGB(0.3, 0.5, 0.3)
	now := time.Now()
    TIMEYPOS:=float64(HEIGHT-76)
	dc.DrawRectangle(0, TIMEYPOS, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0, 0.25, 0)

	now = time.Now()
    formattedDateTime := now.Format("Jan 2 3:04pm")
	dc.DrawStringAnchored(formattedDateTime, float64(WIDTH/2), TIMEYPOS+30, 0.5, 0.5)


    // Top User Banner
	dc.SetRGB(0.3, 0.5, 0.3)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0.0, 0.0, 0.0)

}
func video_comein() {

	fmt.Println("Clearing framebuffer (16-bit)...")

	fmt.Println("Drawing a red Background")
	dc.SetRGB(0, 0, 0.5)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

	dc.SetRGB(1, 1, 1)
    setFontSize(64)
    h := float64((HEIGHT-64)/2)
	dc.DrawStringAnchored("Room in Use", float64(WIDTH/2), h, 0.5, 0.5)
    setFontSize(32)
    h+=50
	dc.DrawStringAnchored("Swipe fob badge-out of room", float64(WIDTH/2), h, 0.5, 0.5)


    /* Lower Banner */
    setFontSize(42)
	dc.SetRGB(0.2, 0.2, 0.5)
    TIMEYPOS:=float64(HEIGHT-76)
	dc.DrawRectangle(0, TIMEYPOS, float64(WIDTH), 66)
	dc.Fill()

	dc.SetRGB(0.0, 0.2, 0.0)
	dc.DrawStringAnchored(HowLongAgo(lastEventTime), float64(WIDTH/2), TIMEYPOS+30, 0.5, 0.5)


    // User Banner
    if (occupiedBy != nil) {
	dc.SetRGB(0.2, 0.2, 0.5)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0.0, 0.0, 0.0)
	dc.DrawStringAnchored(strings.Replace(*occupiedBy,"."," ",-1), float64(WIDTH/2), 40, 0.5, 0.5)
    }

}
func video_alert() {
	dc.SetRGB(1, 1, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()


    pngImage, err, w, h := loadAndScaleImage("alert.png",WIDTH/4,0)
    if (err != nil) {
            fmt.Errorf("Error loading image %w\n",err)
    }
    dc.DrawImage(pngImage, 20, (HEIGHT-h)/2)

	dc.SetRGB(1, 0, 0)
    setFontSize(42)
	dc.DrawString(alertMessage, float64(20+w), 200)
}

func video_draw() {

	fmt.Println("Drawing a red Background")
	dc.SetRGB(1, 0, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

    /*
	fmt.Println("Drawing a blue circle with gg (to 32-bit buffer)...")
	dc.SetRGB(0, 0, 1)
	dc.DrawCircle(200, 200, 75)
	dc.Fill()
    */

	dc.SetRGB(1, 1, 1)


	//now := time.Now()
	//formattedDateTime := now.Format("2006-01-02 15:04:05")

	//dc.DrawStringAnchored(fmt.Sprintf("Left on: %s", formattedDateTime), float64(WIDTH/2)+offset, 240, 0.5, 0.5)

	//dc.DrawStringAnchored("More gg text below!", float64(WIDTH/2), 300, 0.5, 0.5)


	// --- NEW: Convert 32-bit image to 16-bit framebuffer and blit ---

    setFontSize(42)

	dc.SetRGB(1, 0.3, 0.3)
    TIMEYPOS:=float64(HEIGHT-76)
	dc.DrawRectangle(0, TIMEYPOS, float64(WIDTH), 66)
	dc.Fill()

	dc.SetRGB(0.5, 0.0, 0.0)
	dc.DrawStringAnchored(HowLongAgo(lastEventTime), float64(WIDTH/2), TIMEYPOS+30, 0.5, 0.5)


    // User Banner
	dc.SetRGB(1, 0.3, 0.3)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
    if (occupiedBy != nil) {
            dc.SetRGB(0.0, 0.0, 0.0)
            dc.DrawStringAnchored(*occupiedBy, float64(WIDTH/2), 40, 0.5, 0.5)
    }


    pngImage, err, w, h := loadAndScaleImage("DarkroomInUse.png",(3*WIDTH)/4,0)
    _ = err
    dc.DrawImage(pngImage, (WIDTH-w) /2, (HEIGHT-h)/2)

    /* End Update Time */

}

func video_update() {

	// Assuming common RGB565 format for 16-bit framebuffer, and Little Endian byte order.
	// You might need to adjust the conversion formula or byte order (binary.BigEndian)
	// if your framebuffer uses a different 16-bit format (e.g., ARGB1555) or byte order.
	// See varInfo.Red.Offset, varInfo.Green.Offset, etc. for precise format.
	for y := 0; y < HEIGHT; y++ {
		for x := 0; x < WIDTH; x++ {
			// Get 32-bit RGBA pixel from the in-memory image
			r, g, b, _ := rgbaImage.At(x, y).RGBA() // Returns 0-65535, so convert to 0-255
			
			// Convert to 16-bit RGB565
			r5 := uint16(r >> (16 - 5)) // 16-bit (0-65535) to 5-bit (0-31)
			g6 := uint16(g >> (16 - 6)) // 16-bit (0-65535) to 6-bit (0-63)
			b5 := uint16(b >> (16 - 5)) // 16-bit (0-65535) to 5-bit (0-31)

			pixel16 := (r5 << 11) | (g6 << 5) | b5 // Combine into 16-bit value

			// Calculate index in the 16-bit framebuffer. LineLength is in bytes.
			fbIdx := (y * lineLengthBytes) + (x * 2) 

			if fbIdx+1 < len(pixBuffer) {
				// Write 16-bit pixel to framebuffer. Assuming Little Endian.
				//binary.LittleEndian.PutUint16(pixBuffer[fbIdx:], pixel16) 
				binary.LittleEndian.PutUint16(backBuffer[fbIdx:], pixel16) 
			}
		}
	}

    copy(pixBuffer,backBuffer)


	// --- END NEW BLITTING ---

        //time.Sleep(1 * time.Second)
    // for }
	fmt.Println("Done. The framebuffer content should be visible.")
	fmt.Println("The program will exit in 5 seconds.")
}
