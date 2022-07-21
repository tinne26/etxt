// etxt is a package for font management and text rendering in Golang
// designed to be used mainly with the Ebitengine game engine.
//
// While the API surface can look slightly intimidating at the beginning,
// common usage depends only on a couple types and a few functions...
//
// First, you create a [FontLibrary] and parse the fonts:
//   fontLib := etxt.NewFontLibrary()
//   _, _, err := fontLib.ParseDirFonts("path/to/fonts")
//   if err != nil { ... }
//
// Then, you create a [Renderer]:
//   txtRenderer := etxt.NewStdRenderer()
//   txtRenderer.SetFont(fontLib.GetFont("My Font Name"))
//
// Finally, you set a target and start drawing:
//   txtRenderer.SetTarget(screen)
//   txtRenderer.Draw("Hello world!", x, y)
//
// There are a lot of parameters you can configure, but the critical ones
// are font, size, align, color, cache and target. Take a good look at those
// and have fun exploring the rest!
package etxt
