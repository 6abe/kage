package capture

import (
	"fmt"
	"image/png"
	"os"
)

// PNGSize returns the pixel size of a PNG without decoding the full raster.
func PNGSize(path string) (width, height int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return 0, 0, fmt.Errorf("not a PNG: %w", err)
	}
	return cfg.Width, cfg.Height, nil
}
