package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	"goratt/cmd/labelprinter/assets"
)

const (
	labelWidthLines = 960
	labelHeightBPL  = 38
	labelHeightPx   = labelHeightBPL * 8 // 304 pixels
)

// Printer manages Dymo label printing with serialized access.
type Printer struct {
	device string
	mu     sync.Mutex
	font   *truetype.Font

	storageTemplate image.Image
	freeTemplate    image.Image
	milsmLogo       image.Image
}

// NewPrinter creates and initializes a Printer instance.
func NewPrinter(device string) (*Printer, error) {
	parsedFont, err := truetype.Parse(assets.FontUbuntuR)
	if err != nil {
		return nil, fmt.Errorf("parse embedded font: %w", err)
	}

	p := &Printer{
		device: device,
		font:   parsedFont,
	}

	// Decode embedded templates
	if len(assets.StoragePassTemplatePNG) > 0 {
		img, err := png.Decode(bytes.NewReader(assets.StoragePassTemplatePNG))
		if err == nil {
			p.storageTemplate = img
		} else {
			log.Printf("Warning: decode storage pass template: %v", err)
		}
	}

	if len(assets.FreeTableTemplatePNG) > 0 {
		img, err := png.Decode(bytes.NewReader(assets.FreeTableTemplatePNG))
		if err == nil {
			p.freeTemplate = img
		} else {
			log.Printf("Warning: decode free table template: %v", err)
		}
	}

	if len(assets.MilsmPNG) > 0 {
		img, err := png.Decode(bytes.NewReader(assets.MilsmPNG))
		if err == nil {
			p.milsmLogo = img
		} else {
			log.Printf("Warning: decode milsm logo: %v", err)
		}
	}

	return p, nil
}

func (p *Printer) face(size float64) font.Face {
	return truetype.NewFace(p.font, &truetype.Options{
		Size: size,
	})
}

func (p *Printer) openDevice() (*os.File, error) {
	return os.OpenFile(p.device, os.O_RDWR, 0644)
}

func (p *Printer) exportbmpDymo(dc *gg.Context, file *os.File) error {
	img := dc.Image()
	bpl := dc.Height() / 8

	header := []byte{
		27, 0x44, byte(bpl), // Printer Width (Bytes)
	}
	if _, err := file.Write(header); err != nil {
		return fmt.Errorf("write dymo header: %w", err)
	}

	lineBuf := make([]byte, 1+bpl)
	lineBuf[0] = 0x16 // SYN byte starting raster line

	for x := 0; x < img.Bounds().Max.X; x++ {
		byteIdx := 1
		for y := img.Bounds().Max.Y - 1; y >= 0; y -= 8 {
			var data byte
			for i := 0; i < 8; i++ {
				r, _, _, _ := img.At(x, y+i).RGBA()
				if r <= 0x8000 { // Dark pixel -> print dot
					data |= (1 << i)
				}
			}
			lineBuf[byteIdx] = data
			byteIdx++
		}
		if _, err := file.Write(lineBuf); err != nil {
			return fmt.Errorf("write raster line: %w", err)
		}
	}

	// Form Feed (ejects label to tear bar position)
	if _, err := file.Write([]byte{27, 'E'}); err != nil {
		return fmt.Errorf("write form feed: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}

// PrintStoragePass prints a 3-day temporary storage pass for the given member.
func (p *Printer) PrintStoragePass(member string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dc := gg.NewContext(labelWidthLines, labelHeightPx)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	dc.SetRGB(0, 0, 0)

	offset := float64(0)
	if p.storageTemplate != nil {
		dc.DrawImage(p.storageTemplate, 0, 0)
		offset = 100
	} else if p.milsmLogo != nil {
		dc.DrawImage(p.milsmLogo, 20, 20)
		offset = float64(p.milsmLogo.Bounds().Dx()) / 2.0
	}

	currentDate := time.Now()
	futureDate := currentDate.AddDate(0, 0, 3)
	futureDateString := futureDate.Format("Mon, 02-Jan-06")
	formattedDateTime := currentDate.Format("Mon, 02-Jan-2006 01:04 PM")

	centerX := float64(labelWidthLines/2) + offset

	dc.SetFontFace(p.face(84))
	dc.DrawStringAnchored(futureDateString, centerX, 180, 0.5, 0.5)

	dc.SetFontFace(p.face(48))
	//dc.DrawStringAnchored("Temporary Storage Pass", centerX, 50, 0.5, 0.5)
	dc.DrawStringAnchored(strings.ReplaceAll(member, ".", " "), centerX, 120, 0.5, 0.5)

	dc.SetFontFace(p.face(24))
	dc.DrawStringAnchored(fmt.Sprintf("Left on: %s", formattedDateTime), centerX, 240, 0.5, 0.5)

	dc.SetLineWidth(2)
	dc.DrawRectangle(10, 10, float64(labelWidthLines-20), float64(labelHeightPx-20))
	dc.Stroke()

	dc.SetFontFace(p.face(18))
	textbody := "Items may be discarded and disposal charges may be incurred if items are left after specified date."
	dc.DrawStringWrapped(textbody, centerX, float64(labelHeightPx-32), 0.5, 0.5, float64(labelWidthLines/2), 1.2, gg.AlignCenter)

	file, err := p.openDevice()
	if err != nil {
		return fmt.Errorf("open printer %s: %w", p.device, err)
	}
	defer file.Close()

	log.Printf("Printing Storage Pass for member %q", member)
	return p.exportbmpDymo(dc, file)
}

// PrintFreeTable prints an 8-day free table pass for the given member.
func (p *Printer) PrintFreeTable(member string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dc := gg.NewContext(labelWidthLines, labelHeightPx)
	dc.SetRGB(1, 1, 1)
	dc.Clear()
	dc.SetRGB(0, 0, 0)

	offset := float64(0)
	if p.freeTemplate != nil {
		dc.DrawImage(p.freeTemplate, 0, 0)
	} else if p.milsmLogo != nil {
		dc.DrawImage(p.milsmLogo, 20, 20)
		offset = float64(p.milsmLogo.Bounds().Dx()) / 2.0
	}

	currentDate := time.Now()
	futureDate := currentDate.AddDate(0, 0, 8)
	futureDateString := futureDate.Format("Mon, 02-Jan-06")
	formattedDateTime := currentDate.Format("Mon, 02-Jan-2006 01:04 PM")

	centerX := float64(labelWidthLines/2) + offset

	dc.SetFontFace(p.face(84))
	dc.DrawStringAnchored(futureDateString, centerX, 180, 0.5, 0.5)

	dc.SetFontFace(p.face(48))
	dc.DrawStringAnchored(strings.ReplaceAll(member, ".", " "), centerX, 120, 0.5, 0.5)

	dc.SetFontFace(p.face(24))
	dc.DrawStringAnchored(fmt.Sprintf("Left on: %s", formattedDateTime), centerX, 240, 0.5, 0.5)

	dc.SetLineWidth(2)
	dc.DrawRectangle(10, 10, float64(labelWidthLines-20), float64(labelHeightPx-20))
	dc.Stroke()

	dc.SetFontFace(p.face(18))
	textbody := "Items will be discarded after specified date."
	dc.DrawStringWrapped(textbody, centerX, float64(labelHeightPx-32), 0.5, 0.5, float64(labelWidthLines/2), 1.2, gg.AlignCenter)

	file, err := p.openDevice()
	if err != nil {
		return fmt.Errorf("open printer %s: %w", p.device, err)
	}
	defer file.Close()

	log.Printf("Printing Free Table pass for member %q", member)
	return p.exportbmpDymo(dc, file)
}

// PrintAruco prints an ArUco code bin label with the code marker and member name.
func (p *Printer) PrintAruco(codeIndex int, member string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if codeIndex < 0 || codeIndex >= len(aruco) {
		return fmt.Errorf("code index %d out of bounds (0-%d)", codeIndex, len(aruco)-1)
	}

	dc := gg.NewContext(labelWidthLines, labelHeightPx)
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	arucoPxSize := float64(labelHeightPx)
	arucoBlockPx := arucoPxSize / 6.0

	// Draw black background box for the ArUco marker
	dc.SetRGB(0, 0, 0)
	dc.DrawRectangle(0, 0, arucoPxSize, arucoPxSize)
	dc.Fill()

	// Split member name into display lines
	var rawMembers []string
	if strings.Contains(member, ".") {
		rawMembers = strings.Split(member, ".")
	} else {
		rawMembers = strings.Fields(member)
	}

	var members []string
	for _, m := range rawMembers {
		trimmed := strings.TrimSpace(m)
		if trimmed != "" {
			members = append(members, trimmed)
		}
	}

	maxTextWidth := float64(labelWidthLines) - arucoPxSize - 20.0
	maxTextHeight := float64(labelHeightPx) / 2.5
	verticalSpacing := 10.0

	fontSize := float64(84)
	lineHeight := 0.0

	if len(members) > 0 {
		for _, line := range members {
			for {
				dc.SetFontFace(p.face(fontSize))
				lineWidth, lh := dc.MeasureString(line)
				if lineWidth <= maxTextWidth || fontSize <= 8 {
					lineHeight = lh
					break
				}
				fontSize -= 2
			}
			if ((lineHeight*float64(len(members)))+(verticalSpacing*float64(len(members)-1))) > maxTextHeight && fontSize > 8 {
				fontSize -= 2
			}
		}

		totalDrawnHeight := (lineHeight * float64(len(members))) + (verticalSpacing * float64(len(members)-1))
		startY := (float64(labelHeightPx) / 2.0) - (totalDrawnHeight / 2.0) + (lineHeight / 2.0)
		textCenterX := (float64(labelWidthLines)-arucoPxSize)/2.0 + arucoPxSize

		dc.SetFontFace(p.face(fontSize))
		dc.SetRGB(0, 0, 0)
		for _, line := range members {
			dc.DrawStringAnchored(line, textCenterX, startY, 0.5, 0.5)
			startY += lineHeight + verticalSpacing
		}
	}

	// Render the ArUco code marker interior
	byts := aruco[codeIndex].v
	codewidth := 4
	for y := 0; y < codewidth; y++ {
		for x := 0; x < codewidth; x++ {
			bitnum := y*codewidth + x
			byteIndex := bitnum >> 3
			if byteIndex < len(byts) {
				byteVal := byts[byteIndex]
				bit := byteVal & (0x80 >> (bitnum % 8))
				if bit != 0 {
					// White cell
					dc.SetRGB(1, 1, 1)
					dc.DrawRectangle(
						arucoBlockPx*float64(x+1),
						arucoBlockPx*float64(y+1),
						arucoBlockPx+1.0,
						arucoBlockPx+1.0,
					)
					dc.Fill()
				}
			}
		}
	}

	file, err := p.openDevice()
	if err != nil {
		return fmt.Errorf("open printer %s: %w", p.device, err)
	}
	defer file.Close()

	log.Printf("Printing ArUco label: code=%d member=%q", codeIndex, member)
	return p.exportbmpDymo(dc, file)
}
