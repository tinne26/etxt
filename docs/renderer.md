# Renderers
Random bits of trivia and advice:
- Most small games can do all their text rendering with a single `etxt.Renderer`, by simply changing fonts and sizes as needed. No need for an *army* of renderers.
- You might still need more than one renderer if you need to draw text concurrently, want to use different caches, want to wrap renderers on custom types for advanced use-cases, or simply to organize your code more naturally.
- While renderer's aren't too heavy or slow to initialize, you shouldn't create new ones on each frame. If you ever get in the business of pooling them (which should be a last resource, but you do you), make sure to clear the rendering target and the font pointers first.
- Even if drawing text is reasonably performant once glyph masks are cached, it's always good to remember that sometimes you can draw to an offscreen image to avoid doing so much work for text rendering on each frame. That said, drawing to an offscreen also has some downsides when the screen size changes, as you might need to re-render.
- If you have a complex UI system, it's advisable to work with color palettes, font sets and sizes at an abstract level (e.g: main, background and highlight colors, main and title font, heading, normal and detail sizes, etc.) instead of passing all that information manually to the renderer. While the `etxt.Renderer` is easy to use directly, in many cases you will want to use it as building block, not as the "definitive" abstraction. It's not and it doesn't try to be.

## Drawing UI at full resolution
To get crisp text at big sizes, it's important that you keep in mind what's your game screen size. When working with Ebiten, it's very common to use a fixed, small screen size, draw your pixel art there, and then forget that if you also draw your text and UI at that small size it will look terrible when it's scaled up.

To get this right, you can do the following:
- Make your `Game.Layout` function return the full screen size (or even higher than that if you are supporting [high-DPI](https://github.com/hajimehoshi/ebiten/blob/main/examples/highdpi/main.go)).
- Keep a smaller offscreen image to draw your pixel art to (and only pixel art). There's no good reason to draw your pixel art directly to a bigger screen size, it only makes your life harder for no good reason.
- Once you are done with the previous step, draw the offscreen image to the main screen image by scaling it as necessary (using the standard `DrawImageOptions.GeoM`). I always recommend adding a game option (*optional*) to allow scaling to integer multiples of the main resolution. While in many cases it loses too much screen real-state, having a full-screen pixel-perfect mode is always nice.
- Now you can finally draw your UI and text... and scalable vectors... and high-resolution images... and whatever else you might have to the main screen. As a downside, if the screen size changes you will also have to re-adjust your font sizes.

## Drawing UI at small sizes
To get crisp text at small sizes, I'm sorry, but since this package depends on [**sfnt**](https://pkg.go.dev/golang.org/x/image/font/sfnt) and **sfnt** doesn't have support for hinting instructions, small text is not going to look as good as it can. Maybe some day.

...or you can try to implement [subpixel rendering](https://en.wikipedia.org/wiki/Subpixel_rendering) in a custom rasterizer...
