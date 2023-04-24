# Display scaling

One of the top mistakes among Ebitengine game devs is failing to use [`ebiten.DeviceScaleFactor()`](https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#DeviceScaleFactor) correctly. There are some reasons for this:
- Many developers haven't internalized that pixels in physical monitors are not all the same size.
- Ebitengine treats display scaling as an optional feature instead of as a primary concern.
- UI frameworks for Ebitengine also have an unhealthy tendency to neglect display scaling.
- Almost no one understands how `Game.Layout` really works.

The main victim in all this? Text. According to scientific studies™, text looks jaggy in 13 out of 14 Ebitengine games.

Thankfully, the path towards betterment is only five steps away:
1. Repent.
2. Learn how `Game.Layout` works by [reading this guide](https://github.com/tinne26/kage-desk/blob/main/docs/tutorials/ebitengine_game.md#layout).
3. Repent harder.
4. Delete your whole game and rewrite it using what you have learned.
5. Meditate until enlightened.

Technically, the critical code when using `etxt` can be as simple as this:
```Golang
func (game *Game) Layout(_, _ int) (int, int) { panic("use Ebitengine >=v2.5.0") }
func (game *Game) LayoutF(logicWinWidth, logicWinHeight float64) (float64, float64) {
	scale := ebiten.DeviceScaleFactor()
	game.TextRenderer.SetScale(scale)
	canvasWidth  := math.Ceil(logicWinWidth*scale)
	canvasHeight := math.Ceil(logicWinHeight*scale)
	return canvasWidth, canvasHeight
}
```
The main problem is that if you have already a game half written and you are only understanding `Game.Layout` now, this code will most likely break everything for you.

The summary would be that if you want crisp text you need to use the full resolution of the player's screen, and that can only be achieved by understanding `Game.Layout` and using `DeviceScaleFactor()` correctly. From there on, `etxt` makes your life easy by separating logical sizes (`Renderer.SetSize()`) and scale (`Renderer.SetScale()`).

## FAQ

**You haven't actually explained anything.**

Yeah, the explanations are actually on the [`Game.Layout` tutorial](https://github.com/tinne26/kage-desk/blob/main/docs/tutorials/ebitengine_game.md#layout). You either understand how to make full use of the player screen's resolution through `Game.Layout` and `ebiten.DeviceScaleFactor()` or you don't. If you do, `etxt` makes it trivial to adapt to that with `Renderer.SetScale()`, and there's not much more to say. If you don't, you are missing the pre-required knowledge, so don't ask me for miracles.

**But what if I'm making a low-resolution pixel art game? Do I still need display scaling?**

That's a fair question. In those cases, you would ideally use a bitmap font, not a vectorial one, which is what `etxt` focuses on. We don't have any good package for that yet, so you may have to default to a monospaced bitmap font stored as a simple PNG or something, going retro. I may write something more decent in the future. You may also pray for someone to port [BMFont](https://www.angelcode.com/products/bmfont/) to Go.

But yeah, you can actually work on many parts of your game without display scaling if you are doing pure pixel art. I only want to highlight that people will often think their games are pure pixel art when they are not. For example, if you want to add a blur shadow effect to your text, doing that without display scaling will result in a jaggy mess. It may be acceptable in some cases, but it already feels like incorrectly trying to mix non-pixel art effects with pure pixel art without acknowledging that those are different things.

**I've been following your teachings diligently, but... I don't see any difference?**

Some developers work with screens that have a standard display scaling of 100% and they never notice. You can go to your system configuration and play around with the display scaling to test. Notice, though, that Ebitengine doesn't detect the display scaling changes dynamically once your game is already running, so you will have to restart the game each time you change the display scaling.

**Ok, now I've seen the difference... but it doesn't look that bad to me.**

Consider scheduling an appointment with the nearest ophthalmologist.

Nah, it's probably ok for a game jam or when starting to experiment with Ebitengine, but don't start making a serious game without properly understanding display scaling and `Game.Layout` or you will have to rewrite large parts of your game further down the road and heavily regret it.

*(The full disclaimer is that for best quality you would still have to adjust the renderer's quantization and I would have to add gamma correction and subpixel antialiasing to `etxt`, but that's a topic for another day...)*

**But then how do I adapt the rest of my game to display scaling?**

That's a bit outside the scope of this document, but I generally recommend having separate logical sizes and scaling factors and bringing them together at rendering time. I think writers of libraries and packages for Ebitengine should also adopt this approach when relevant, with a few extra optimizations, but sadly the situation is that more often than not they don't include this in their examples and have no specific model for it, leaving you to do it manually.
