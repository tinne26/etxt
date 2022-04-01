//go:build !gtxt

package internal

import "github.com/hajimehoshi/ebiten/v2"

type GlyphMask = *EbitenGlyphMask

// TODO: https://github.com/hajimehoshi/ebiten/issues/2013
type EbitenGlyphMask struct {
	Image *ebiten.Image
	XOffset int
	YOffset int
}
