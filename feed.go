package etxt

import "golang.org/x/image/font/sfnt"
import "github.com/tinne26/etxt/fract"

// TODO: add tests comparing with Draw().

// Feeds are the lowest level mechanism to draw text in etxt,
// allowing the user to issue each glyph draw call individually
// and modifying positions or configurations in between.
//
// As a rule of thumb, you should only resort to feeds if
// neither [Renderer.Draw](), [RendererComplex.Draw](), nor
// [RendererGlyph.SetDrawFunc]() give you enough control to do
// what you want. Make sure you are well acquainted with the
// basic methods first.
type Feed struct {
	Renderer *Renderer             // associated renderer
	Position fract.Point           // the feed's working pen position or origin
	LineBreakX fract.Unit          // the x coordinate set after a line break
	LineBreakAcc uint16            // consecutive accumulated line breaks
	PrevGlyphIndex sfnt.GlyphIndex // previous glyph index. used for kern
}

// Creates a [Feed] associated to the given [Renderer].
//
// Feeds are the lowest level mechanism to draw text in etxt, as they
// expose and allow one to modify the drawing position manually.
//
// Notice that the high coupling to the renderer makes working with
// feeds potentially finicky and unsafe. In most cases, creating a
// feed for each function where a feed is required is the most sane
// approach.
func NewFeed(renderer *Renderer) *Feed {
	return &Feed{ Renderer: renderer }
}

// Handy method to set the feed's Position and LineBreakX fields.
// Often chained on feed creation as follows:
//   feed := etxt.NewFeed(renderer).At(x, y)
//
// Notice that the given y will be adjusted based on the associated
// renderer's align. While feeds can't automatically align text because 
// the content to work with isn't known ahead of time, the vertical
// align is considered in this method in order to set the Position.Y
// field to its baseline value.
//
// For more precise positioning, you can always manipulate the Position
// field directly. This method also sets the LineBreakX field.
func (self *Feed) At(x, y int) *Feed {
	renderer := self.Renderer
	vertAlign := renderer.GetAlign().Vert()
	self.Position.X = fract.FromInt(x)
	self.LineBreakX = self.Position.X
	fractY := fract.FromInt(y)
	if vertAlign == Baseline || vertAlign == LastBaseline {
		self.Position.Y = fractY
		return self // basic case
	}
	
	// prepare for complex cases
	font   := renderer.state.activeFont
	sizer  := renderer.state.fontSizer
	ascent := sizer.Ascent(font, &renderer.buffer, renderer.state.scaledSize)

	// code based on Renderer.fractDraw // adjust Y position
	qtVert := fract.Unit(renderer.state.vertQuantization)
	switch vertAlign {
	case Top:	
		self.Position.Y = (fractY + ascent).QuantizeUp(qtVert)
	case Midline, LastMidline:
		self.Position.Y = (fractY + ascent - renderer.xheight(font)).QuantizeUp(qtVert)
	case VertCenter:
		height := sizer.LineHeight(font, &renderer.buffer, renderer.state.scaledSize)
		self.Position.Y = (fractY + ascent - (height >> 1)).QuantizeUp(qtVert)
	case Bottom:
		height := sizer.LineHeight(font, &renderer.buffer, renderer.state.scaledSize)
		self.Position.Y = (fractY + ascent - height).QuantizeUp(qtVert)
	default:
		panic(vertAlign)
	}
	return self
}

// Utility method for setting all the feed fields to their zero values.
// After a reset, you will need to set the feed's [Renderer] again manually
// if you want to use it again.
func (self *Feed) Reset() {
	self.Renderer = nil
	self.Position = fract.Point{}
	self.LineBreakX = 0
	self.LineBreakAcc = 0
	self.PrevGlyphIndex = 0
}

// Draws the given rune and advances the feed's Position.
//
// The drawing configuration is taken from the feed's associated renderer.
//
// Quantization will be checked before every drawing operation and adjusted
// if necessary (even vertical quantization).
func (self *Feed) Draw(target TargetImage, codePoint rune) {
	self.DrawGlyph(target, self.Renderer.Glyph().GetRuneIndex(codePoint))
}

// Same as [Feed.Draw](), but taking a glyph index instead of a rune.
func (self *Feed) DrawGlyph(target TargetImage, glyphIndex sfnt.GlyphIndex) {
	self.traverseGlyph(target, glyphIndex, true)
}

// Draws the given shape and advances the feed's position based on
// the image bounds width (rect.Dx). Notice that shapes are not cached,
// so they may be expensive to use. See also DrawImage.
// func (self *Feed) DrawShape(shape emask.Shape) {
//
// }

// TODO: ebiten version vs gtxt version
// func (self *Feed) DrawImage(img image.Image)

// Advances the feed's position without drawing anything.
func (self *Feed) Advance(codePoint rune) {
	// Note: while this method may seem superfluous, at least it can be
	//       useful when drawing paragraphs at an abitrary scroll position.
	//       Efficient implementations would split by lines from the start,
	//       but that's tricky, so the cheap approach could be reasonable
	//       and one would rather Advance() than Draw(). Fair enough?
	if codePoint == '\n' {
		self.LineBreak()
	} else {
		self.AdvanceGlyph(self.Renderer.Glyph().GetRuneIndex(codePoint))
	}
}

// Advances the feed's position without drawing anything.
func (self *Feed) AdvanceGlyph(glyphIndex sfnt.GlyphIndex) {
	self.traverseGlyph(nil, glyphIndex, false)
}

// Advances the feed's position with a line break.
func (self *Feed) LineBreak() {
	renderer := self.Renderer

	// advance
	self.Position.Y += renderer.state.fontSizer.LineAdvance(
		renderer.state.activeFont, &renderer.buffer, 
		renderer.state.scaledSize, int(self.LineBreakAcc),
	)
	
	// y position must be quantized for conformity with Renderer operations
	qtVert := fract.Unit(renderer.state.vertQuantization)
	self.Position.Y = self.Position.Y.QuantizeUp(qtVert)
	self.Position.X = self.LineBreakX // doesn't matter if it's unquantized
	self.LineBreakAcc += 1
}

// Private traverse method used for Draw and Advance.
func (self *Feed) traverseGlyph(target TargetImage, glyphIndex sfnt.GlyphIndex, drawMode bool) {
	// make sure all relevant properties are initialized
	renderer := self.Renderer
	font   := renderer.state.activeFont
	sizer  := renderer.state.fontSizer
	qtHorz, qtVert := renderer.fractGetQuantization()

	// traverse in the proper direction
	dir := self.Renderer.Complex().GetDirection()
	switch dir {
	case LeftToRight:
		// apply kerning unless coming from line break
		if self.LineBreakAcc == 0 {
			self.Position.X += sizer.Kern(font, &renderer.buffer, renderer.state.scaledSize, self.PrevGlyphIndex, glyphIndex)	
		}

		// always apply quantization inconditionally before drawing.
		// not always strictly necessary, but users can mess up with
		// position at many different points in dangerous ways
		self.Position.X = self.Position.X.QuantizeUp(qtHorz)
		self.Position.Y = self.Position.Y.QuantizeUp(qtVert)
		renderer.cacheHandler.NotifyFractChange(self.Position)

		// traverse glyph
		if drawMode {
			renderer.internalGlyphDraw(target, glyphIndex, self.Position, font)
		}

		// advance
		self.Position.X += sizer.GlyphAdvance(font, &renderer.buffer, renderer.state.scaledSize, glyphIndex)
	case RightToLeft:
		// advance
		self.Position.X -= sizer.GlyphAdvance(font, &renderer.buffer, renderer.state.scaledSize, glyphIndex)

		// apply kerning unless coming from line break
		if self.LineBreakAcc == 0 {
			self.Position.X -= sizer.Kern(font, &renderer.buffer, renderer.state.scaledSize, self.PrevGlyphIndex, glyphIndex)	
		}

		// quantize
		self.Position.X = self.Position.X.QuantizeUp(qtHorz)
		self.Position.Y = self.Position.Y.QuantizeUp(qtVert)
		renderer.cacheHandler.NotifyFractChange(self.Position)

		// traverse glyph
		if drawMode {
			renderer.internalGlyphDraw(target, glyphIndex, self.Position, font)
		}
	default:
		panic(dir)
	}

	// update prev glyph tracking
	self.LineBreakAcc   = 0
	self.PrevGlyphIndex = glyphIndex
}
