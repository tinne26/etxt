# Tips for pixel-art-like vectorial fonts

Pixel-art-like fonts can be used with etxt as long as they also contain glyph outlines (as opposed to fonts with only glyph bitmaps). This document is focused on etxt, but some of the advice is general and also applicable to `ebiten/text/v2`.

Quick practical advice:
- Enable full horizontal quantization with [`Renderer.Fract().SetHorzQuantization(etxt.QtFull)`](https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9#RendererFract.SetHorzQuantization). Vertical quantization is already set to full pixels by default. Ebitengine text packages determine ["glyph variations" automatically](https://github.com/hajimehoshi/ebiten/blob/v2.8.5/text/v2/text.go#L96-L114), and this isn't directly customizable there.
- Pixel-art-like vectorial fonts are designed for a single size (or their multiples). 16px is common... but single sizes have some implications:
	- You don't want to use [Renderer.SetScale](). Leave the scale to 1. You want to be rendering on your logical, fixed-size canvas, and only doing scaling later on the projection from the logical canvas to the full resolution screen.
	- If your font doesn't look sharp at the intended size, DPI may be to blame. There are two common DPI values used in the wild: 72DPI and 96DPI. On etxt, 72DPI is implicitly used. If a font is designed for 96DPI, you may need to multiply its size by 4/3 or similar conversions.
	- If you still want to use pixel art fonts at arbitrary sizes, you might consider using the [`SharpRasterizer`](https://pkg.go.dev/github.com/tinne26/etxt@v0.0.9/mask#SharpRasterizer) to avoid blurriness (`rasterizer.Glyph().SetRasterizer(&mask.SharpRasterizer{})`).

In general, etxt is not optimized or oriented to pixel art fonts, and sfnt, the underlying library used to parse the fonts, doesn't have support for glyph bitmaps. This doesn't mean that using etxt is crazy if you are working with such fonts; etxt still provides many useful features no matter the type of font you are using. That being said, if a specialized package existed for dealing with this kind of fonts on Ebitengine, that could easily become a better alternative. I'm working on [ptxt](https://github.com/tinne26/ptxt), but it still has a long way to go. For a simpler approach, you might also be interested in [ingenten](https://github.com/Frabjous-Studios/ingenten)).
