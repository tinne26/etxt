package eglyr

import "golang.org/x/image/font/sfnt"

// Ignore this file, it only contains aliases to avoid cyclic imports
// but still allow the code to be split more modularly.

// Same as [etxt.Font].
type Font = sfnt.Font

// Same as [etxt.GlyphIndex].
type GlyphIndex = sfnt.GlyphIndex
