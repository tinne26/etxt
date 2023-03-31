package sizer

import "golang.org/x/image/font"

// This should have been named quantization, not hinting,
// but whatever...
const hintingNone = font.HintingNone

// Alias for unexported type embedding on other sizers.
type defaultSizer = DefaultSizer
