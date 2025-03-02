# Display scaling

One of the most common mistakes among Ebitengine game devs is failing to use [`ebiten.Monitor().DeviceScaleFactor()`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#MonitorType.DeviceScaleFactor). There are some reasons for this:
- We tend to forget that pixel size and density can vary between monitors.
- Ebitengine treats display scaling as an optional feature instead of as a primary concern.
- UI frameworks for Ebitengine also have an unhealthy tendency to neglect display scaling.
- Almost no one understands how `Game.Layout` really works.

A common victim of all this? Text. The solution? Learn how `Game.Layout` works by [reading this guide](https://github.com/tinne26/kage-desk/blob/main/docs/tutorials/ebitengine_game.md#layout).

In the case of `etxt`, you can adjust the text scale through the `Renderer.SetScale()` function. If you want crisp text, the summary is that you need to use the full resolution of the player's screen, and that can only be achieved by understanding `Game.Layout` and using `DeviceScaleFactor()` correctly. From there on, `etxt` makes your life easy by separating logical sizes (`Renderer.SetSize()`) and scale (`Renderer.SetScale()`).

## Text scaling contexts

There are two main ways in which you may want to scale your text:

The first one is scaling text based on the `DeviceScaleFactor()`, but preserving text size regardless of the window size. This is common for general GUI applications. You may make the window bigger or smaller, but you still want the text to be rendered at the same size. *Text size is independent from the window size*.

To achieve this, you simply need to apply `Renderer.SetScale(ebiten.Monitor().DeviceScaleFactor())` on the `Game.Layout()` function or similar:
```Golang
func (game *Game) Layout(_, _ int) (int, int) { panic("use Ebitengine >=v2.5.0") }
func (game *Game) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	scale := ebiten.Monitor().DeviceScaleFactor()
	game.TextRenderer.SetScale(scale)
	return logicWinWidth*scale, logicWinHeight*scale
}
```

The second approach is having text size scale along the window size, as it happens in many games. You make the window bigger? Everything gets bigger. You make the window smaller? Everything get smaller. *Text size is proportional to the window size*.

In this case, you have to do two or three things:
1. Apply `ebiten.Monitor().DeviceScaleFactor()` on the `Game.Layout` function to get a canvas of the maximum possible resolution.
2. If your game graphics are made with pixel art and you expect a specific canvas size for them, draw that on an offscreen and then project it to the high-resolution canvas.
3. Draw your text directly on the high-resolution canvas. Your text scale should be set to `HighResolution/LogicalResolution`. For example, if your logical canvas for pixel art is 640x360, but you are projecting to a screen of resolution 1920x1080, your scaling factor would be `3.0`. There can be many intricacies here if you want to do stretching, integer scaling or stuff like that, but this is the main idea.

Here's some *bad but illustrative* code for this approach:
```Golang
func (game *Game) Draw(canvas *ebiten.Image) {
	// draw pixel art stuff
	game.LogicalCanvas.Clear()
	game.DrawPixelArt(g.LogicalCanvas)

	// project from logical resolution to high-resolution
	uiCanvas, uiScale := game.Project(game.LogicalCanvas, canvas) // *
	// * The uiCanvas may be the same as 'canvas', or it may
	//   be a canvas.SubImage() that accounts for black borders
	//   (e.g., caused by integer scaling or to avoid stretching).

	// draw UI
	game.TextRenderer.SetScale(uiScale)
	game.DrawUI(uiCanvas, uiScale)
}
```

## FAQ

**But what if I'm making a low-resolution pixel art game? Do I still need display scaling?**

That's a fair question. In those cases, you would ideally use a bitmap font, not a vectorial one, which is what `etxt` focuses on. We don't have any good package for that yet, so you may have to default to a monospaced bitmap font stored as a simple PNG or something, going retro. I may write something more decent in the future. You may also pray for someone to port [BMFont](https://www.angelcode.com/products/bmfont/) to Go.

But yeah, you can actually work on many parts of your game without display scaling if you are doing pure pixel art. I only want to highlight that people will often think their games are pure pixel art when they are not. For example, if you want to add a blur shadow effect to your text, doing that without display scaling will result in a jaggy mess. It may be acceptable in some cases, but it already feels like incorrectly trying to mix non-pixel art effects with pure pixel art without acknowledging that those are different things.

**I've been following your teachings diligently, but... I don't see any difference?**

Many developers work with screens that have a standard display scaling of 100% and they never notice. You can go to your system configuration and play around with the display scaling to test. Notice, though, that Ebitengine doesn't detect the display scaling changes dynamically once your game is already running, so you will have to restart the game each time you change the display scaling.

**Ok, now I've seen the difference... but it doesn't look that bad to me.**

Consider scheduling an appointment with the nearest ophthalmologist.

Nah, it's probably ok for a game jam or when starting to experiment with Ebitengine, but don't start making a serious game without properly understanding display scaling and `Game.Layout` or you will have to rewrite large parts of your game further down the road and heavily regret it.

*(The full disclaimer is that for best quality you would still have to adjust the renderer's quantization and I would have to add gamma correction and subpixel antialiasing to `etxt`, but that's a topic for another day...)*

**But then how do I adapt the rest of my game to display scaling?**

That's a bit outside the scope of this document, but I generally recommend having separate logical sizes and scaling factors and bringing them together at rendering time. I think writers of libraries and packages for Ebitengine should also adopt this approach when relevant, with a few extra optimizations, but sadly the situation is that more often than not they don't include this in their examples and have no specific model for it, leaving you to do it manually.
