// etxt is a package for text rendering designed to be used with
// [Ebitengine], a 2D game engine made by Hajime Hoshi for Golang.
//
// To get started, you should create a [Renderer] and set up
// a font and cache:
//   text := etxt.NewRenderer()
//   text.SetFont(font)
//   text.SetCache8MiB()
//
// Then, you can further adjust the renderer properties with functions
// like [Renderer.SetColor](), [Renderer.SetSize](), [Renderer.SetAlign](),
// [Renderer.SetScale]() and many others.
//
// Once you have everything configured to your liking, drawing should be
// very straightforward:
//   text.Draw(canvas, "Hello world!", x, y)
//
// To learn more, make sure to check the [examples]!
//
// [examples]: https://github.com/tinne26/etxt/tree/main/examples
// [Ebitengine]: https://ebitengine.org
package etxt
