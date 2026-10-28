package assets

import _ "embed"

var (
	//go:embed Ubuntu-R.ttf
	FontUbuntuR []byte

	//go:embed milsm.png
	MilsmPNG []byte

	//go:embed makeit_logo_lable.png
	MakeitLogoPNG []byte

	//go:embed StoragePassTemplate.png
	StoragePassTemplatePNG []byte

	//go:embed FreeTableTemplate.png
	FreeTableTemplatePNG []byte
)
