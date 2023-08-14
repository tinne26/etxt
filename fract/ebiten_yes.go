//go:build !gtxt

package fract

import "github.com/hajimehoshi/ebiten/v2"

// Ebitengine-related additional utility methods.

// Returns the subimage corresponding to the area delimited
// by the Rect.
func (self Rect) Clip(image *ebiten.Image) *ebiten.Image {
	return image.SubImage(self.ImageRect()).(*ebiten.Image)
}
