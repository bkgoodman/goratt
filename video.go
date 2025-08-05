package main

import (
	"encoding/binary" // For byte order conversion (PutUint16)
	"fmt"
	"image"
	"log"
	"os"
	"time"
    "strings"
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

func video_available() {

	fmt.Println("Clearing framebuffer (16-bit)...")
	clearFramebuffer(pixBuffer) // Clear the actual framebuffer directly

    // Background
	dc.SetRGB(0, 0.5, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

	dc.SetRGB(1, 1, 1)

    setFontSize(64)
	dc.DrawStringAnchored("Room Available", float64(WIDTH/2), 280, 0.5, 0.5)
    setFontSize(32)
	dc.DrawStringAnchored("Swipe fob to use room", float64(WIDTH/2), 340, 0.5, 0.5)


    /* Update Time */

    setFontSize(60)
	dc.SetRGB(0.3, 0.5, 0.3)
	now := time.Now()
    TIMEYPOS:=float64(550)
	dc.DrawRectangle(0, TIMEYPOS-30, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0, 0.25, 0)

	now = time.Now()
    formattedDateTime := now.Format("Jan-02 3:04pm")
	dc.DrawStringAnchored(formattedDateTime, float64(WIDTH/2)+25, TIMEYPOS, 0.5, 0.5)


    // User Banner
	dc.SetRGB(0.3, 0.5, 0.3)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0.0, 0.0, 0.0)
	//dc.DrawStringAnchored("Eric Roth", float64(WIDTH/2)+25, 40, 0.5, 0.5)

}
func video_comein() {

	fmt.Println("Clearing framebuffer (16-bit)...")

	fmt.Println("Drawing a red Background")
	dc.SetRGB(0, 0, 0.5)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()

	dc.SetRGB(1, 1, 1)
    setFontSize(64)
	dc.DrawStringAnchored("Room in Use", float64(WIDTH/2), 300, 0.5, 0.5)


    /* Update Time */
    setFontSize(60)

	dc.SetRGB(0.2, 0.2, 0.5)
    TIMEYPOS:=float64(550)
	dc.DrawRectangle(0, TIMEYPOS-30, float64(WIDTH), 66)
	dc.Fill()

	dc.SetRGB(0.0, 0.2, 0.0)
	dc.DrawStringAnchored(HowLongAgo(lastEventTime), float64(WIDTH/2)+25, TIMEYPOS, 0.5, 0.5)


    // User Banner
    if (occupiedBy != nil) {
	dc.SetRGB(0.2, 0.2, 0.5)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
	dc.SetRGB(0.0, 0.0, 0.0)
	dc.DrawStringAnchored(*occupiedBy, float64(WIDTH/2)+25, 40, 0.5, 0.5)
    }

}
func video_alert() {
	dc.SetRGB(1, 1, 0)
	dc.DrawRectangle(0, 0, 1024, 600)
	dc.Fill()


    pngImage, err := loadImage("alert.png")
    _ = err
    dc.DrawImage(pngImage, 100, 80)

	dc.SetRGB(1, 0, 0)
    setFontSize(42)
	dc.DrawStringAnchored("Unrecognized Tag", float64(WIDTH/2)+180, 200, 0.5, 0.5)
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

	fmt.Println("Drawing text with gg (to 32-bit buffer)...")
	dc.SetRGB(1, 1, 1)


	//now := time.Now()
	//formattedDateTime := now.Format("2006-01-02 15:04:05")

	//dc.DrawStringAnchored(fmt.Sprintf("Left on: %s", formattedDateTime), float64(WIDTH/2)+offset, 240, 0.5, 0.5)

	//dc.DrawStringAnchored("More gg text below!", float64(WIDTH/2), 300, 0.5, 0.5)


	// --- NEW: Convert 32-bit image to 16-bit framebuffer and blit ---
	fmt.Println("Converting and blitting 32-bit image to 16-bit framebuffer...")

    setFontSize(60)

	dc.SetRGB(1, 0.3, 0.3)
    TIMEYPOS:=float64(550)
	dc.DrawRectangle(0, TIMEYPOS-30, float64(WIDTH), 66)
	dc.Fill()

	dc.SetRGB(0.5, 0.0, 0.0)
	dc.DrawStringAnchored(HowLongAgo(lastEventTime), float64(WIDTH/2)+25, TIMEYPOS, 0.5, 0.5)


    // User Banner
	dc.SetRGB(1, 0.3, 0.3)
	dc.DrawRectangle(0, 10, float64(WIDTH), 66)
	dc.Fill()
    if (occupiedBy != nil) {
            dc.SetRGB(0.0, 0.0, 0.0)
            dc.DrawStringAnchored(*occupiedBy, float64(WIDTH/2)+25, 40, 0.5, 0.5)
    }


    pngImage, err := loadImage("DarkroomInUse.png")
    _ = err
    dc.DrawImage(pngImage, 60, 80)

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
