//go:build !gtxt

package fract

import "github.com/hajimehoshi/ebiten/v2"

// Ebitengine-related additional utility methods.

// Utility function to retrieve the subimage corresponding
// to the rect area.
func (self Rect) Clip(image *ebiten.Image) *ebiten.Image {
	return image.SubImage(self.ImageRect()).(*ebiten.Image)
}
