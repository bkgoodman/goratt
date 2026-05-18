//go:build screen

package screen

import (
	_ "embed"

	"github.com/golang/freetype/truetype"
)

//go:embed DejaVuSans-Bold.ttf
var embeddedFontData []byte

// embeddedFont is the parsed font, initialized once at startup.
var embeddedFont *truetype.Font

func init() {
	var err error
	embeddedFont, err = truetype.Parse(embeddedFontData)
	if err != nil {
		panic("failed to parse embedded font: " + err.Error())
	}
}
