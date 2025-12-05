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

const (
        OS_WIDTH = 200
        OS_HEIGHT = 20
)

const (
        FONT_STRING = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
)
// This clear function works for any BPP by just zeroing out bytes, resulting in black
func clearFramebuffer(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

func setFontSize(size int) {
	fontPathGG := FONT_STRING
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

    // backBuffer is where we will blit the data to, to quickly copy out to video FB
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


// Offscreen buffer for FAST copies - realtime UI updates
var os_backBuffer []byte // This is where we will blit ofscreen data to for fast copy-out
func init_offscreen_buffer() {

	// --- NEW: Create a separate 32-bit RGBA image buffer for drawing ---
	// This is where gg and x/image/font will draw. It's in system RAM, not directly FB.
    rgbaImage := image.NewRGBA(image.Rect(0, 0, OS_WIDTH, OS_HEIGHT))
    os_dc := gg.NewContextForRGBA(rgbaImage) // gg context now draws to this 32-bit image
	// --- END NEW ---
	os_dc.SetRGB(0, 0.5, 0)
	os_dc.DrawRectangle(0, 0, OS_WIDTH, OS_HEIGHT)
	os_dc.Fill()

    err := os_dc.LoadFontFace(FONT_STRING, 32)
	if err != nil {
		log.Fatalf("Failed to load font face '%s' for gg: %v. Ensure the font file is present and accessible.", FONT_STRING, err)
	}

    os_dc.DrawString("@", 20,20)
}

func video_updateknob(evt UIEvent) {
    v:= fmt.Sprintf("%d",knobpos)
    setFontSize(32)
	dc.SetRGB(0, 0, 1)
	dc.DrawRectangle(120, 120-32, 64, 64)
    dc.Fill()
	dc.SetRGB(1, 1, 1)
    dc.DrawString(v, 120,120)
    video_partial_update(120,120-32,64,64)
    copy(pixBuffer,backBuffer)
}

func video_clear() {
	clearFramebuffer(pixBuffer) // Clear the actual framebuffer directly
    video_update()
}
func video_comein() {


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

func video_draw() {

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
    video_partial_update(0,0,HEIGHT,WIDTH)
    copy(pixBuffer,backBuffer)
}
func video_partial_update(xStart int, yStart int, height int, width int) {
	// Assuming common RGB565 format for 16-bit framebuffer, and Little Endian byte order.
	// You might need to adjust the conversion formula or byte order (binary.BigEndian)
	// if your framebuffer uses a different 16-bit format (e.g., ARGB1555) or byte order.
	// See varInfo.Red.Offset, varInfo.Green.Offset, etc. for precise format.
    height += yStart
    width += xStart
	for y := yStart; y < height; y++ {
		for x := xStart; x < width; x++ {
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

    if ((xStart==0) && (yStart==0) && (width == WIDTH) && (height == HEIGHT)) {
            copy(pixBuffer,backBuffer)
    } else {
            copy(pixBuffer,backBuffer)
    }


	// --- END NEW BLITTING ---

        //time.Sleep(1 * time.Second)
    // for }
}
