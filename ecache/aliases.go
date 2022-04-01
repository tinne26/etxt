package ecache

import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/internal"

// Ignore this file, it only contains aliases to avoid cyclic imports
// but still allow the code to be split more modularly.

// Same as etxt.Font.
type Font = sfnt.Font

// Same as etxt.GlyphMask.
type GlyphMask = internal.GlyphMask

// Same as etxt.GlyphIndex.
type GlyphIndex = sfnt.GlyphIndex
