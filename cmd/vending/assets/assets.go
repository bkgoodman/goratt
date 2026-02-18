package assets

import _ "embed"

var (
	//go:embed canceled_16.pcm
	Audio_canceled []byte

	//go:embed complete_16.pcm
	Audio_complete []byte

	//go:embed confirm_16.pcm
	Audio_confirm []byte

	//go:embed error_16.pcm
	Audio_error []byte

	//go:embed notrecognized_16.pcm
	Audio_notrecognized []byte

	//go:embed purchase_16.pcm
	Audio_purchase []byte

	//go:embed reup_16.pcm
	Audio_reup []byte
)
