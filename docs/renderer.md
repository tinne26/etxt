# Renderers
Random bits of trivia and advice:
- Many small games can do all their text rendering with a single `etxt.Renderer`, by simply changing fonts and sizes as needed. No need for an *army* of renderers.
- You might still need more than one renderer if you need to draw text concurrently, want to use different caches, want to wrap renderers on custom types for advanced use-cases, or simply to organize your code more naturally.
- While renderers aren't too heavy or slow to initialize, do not create new ones on each frame. If you ever get in the business of pooling them (which should be a last resource, but you do you), I recommend setting the font(s) to nil first.
- Even if drawing text is reasonably performant once glyph masks are cached, it's always good to remember that sometimes you can draw to an offscreen image to avoid doing so much work for text rendering on each frame. That said, drawing to an offscreen also has some downsides when the screen size changes, as you might need to re-render.
- If you have a complex UI system, it's advisable to work with color palettes, font sets and sizes at an abstract level (e.g: main, background and highlight colors, main and title font, heading, normal and detail sizes, etc.) instead of passing all that information manually to the renderer. While the `etxt.Renderer` is easy to use directly, in many cases you will want to use it as building block, not as the "definitive" abstraction. It's not and it doesn't try to be.

## Drawing UI at full resolution
To get crisp text at big sizes, it's important that you keep in mind what's your game screen size. When working with Ebitengine, it's very common to use a fixed, small screen size, draw your pixel art there, and then forget that if you also draw your text and UI at that small size it will look terrible when it's scaled up. See the [display scaling](https://github.com/tinne26/etxt/blob/v0.0.9/docs/display-scaling.md) document for further advice.

## Drawing UI at small sizes
To get crisp text at small sizes, I'm sorry, but since this package depends on [**sfnt**](https://pkg.go.dev/golang.org/x/image/font/sfnt) and **sfnt** doesn't have support for hinting instructions, small text is not going to look as good as it can. Maybe some day.

...or you can try to implement [subpixel rendering](https://en.wikipedia.org/wiki/Subpixel_rendering) in a custom rasterizer...
