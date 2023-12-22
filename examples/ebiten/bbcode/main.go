package main

import "os"
import "log"
import "fmt"
import "math"
import "time"
import "image"
import "regexp"
import "strings"
import "strconv"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

import "github.com/tinne26/etxt"
import "github.com/tinne26/etxt/font"
import "github.com/tinne26/etxt/fract"
import "github.com/tinne26/etxt/mask"

// This example showcases how to embed the Renderer and Twine types
// within your own types to create easy to use text renderers with
// custom preprocessing. We implement a subset of BBCode for this
// example.
//
// For a simpler and more direct Twine example, check ebiten/twine
// instead.
//
// Notice that we are embedding both Renderer and Twine in our
// own types, but sometimes you only need one of the two. For custom
// formatting, in general, wrapping the Twine type is more common than
// wrapping Renderer.
// 
// You can run the example like this:
//   go run github.com/tinne26/etxt/examples/ebiten/bbcode@latest path/to/font.ttf
// Optionally, you can pass a second font that's monospaced, for better
// results with the [code][/code] format directive.

// ---- palette and misc. constants ----

var backColor  = color.RGBA{ 255, 255, 255, 255 }
var shadeColor = color.RGBA{ 230, 230, 230, 255 }
var hintColor  = color.RGBA{ 144, 144, 144, 255 }
const hintColorHex = "#909090"
var textColor  = color.RGBA{   0,   0,   0, 255 }
const sizeNormal = 16
const sizeSmall  = 14
const sizeTime   = 12

// ---- renderer wrapping ----

// This definition will grant BBCodeRenderer all the capabilities of
// etxt.Renderer, with any new methods or overrides we define. This
// is a classic example of type embedding.
type BBCodeRenderer struct {
	etxt.Renderer // type embedding
	twine BBCodeTwine // reusable twine for preprocessing
	preprocess bool
}

// Notice that we are overriding the main renderer draw method. Sometimes
// you will prefer using a different name instead, like DrawBB.
func (self *BBCodeRenderer) Draw(target etxt.Target, text string, x, y int) {
	if self.preprocess {
		self.twine.Reset()
		self.twine.Preprocess(text)
		self.Renderer.Twine().Draw(target, self.twine.Twine, x, y)
	} else {
		self.Renderer.Draw(target, text, x, y)
	}
	// note: it's worth pointing out that in more demanding scenarios you
	// would generally try to optimize this a bit more; rebuilding the twine
	// on each frame, in particular, is rather undesirable... but you can
	// get away with it in most cases if you are lazy like me.
}

// ---- twine wrapping ----

// Yet another example of type embedding.
// 
// This type extends the etxt.Twine with the Preprocess() method, which
// can detect and apply the common [b][/b], [i][/i], [code][/code], [s][/s],
// [size=NUM][/size] and [color=#CODE][/color] BBCode formatting directives.
type BBCodeTwine struct {
	etxt.Twine
	codeFontIndex etxt.FontIndex
}

// Adds content to the twine based on the given text, after applying
// bbcode-style preprocessing.
var fmtOpeningRegexp = regexp.MustCompile(`\[(b|i|s|code|size=[1-9][0-9]?|color=#[0-9A-F]{6})\]`)
func (self *BBCodeTwine) Preprocess(text string) {
	// format processing loop
	safetyBreak := 100
	for {
		// (in case I messed up something)
		safetyBreak -= 1
		if safetyBreak == 0 { panic("infinite loop?") }

		// find next opening format code
		leftRight := fmtOpeningRegexp.FindStringIndex(text)
		
		// if no formatting left, add the remaining string and stop
		if leftRight == nil {
			self.Add(text)
			return
		}

		// if formatting doesn't start immediately, draw text before it
		left, right := leftRight[0], leftRight[1]
		if left != 0 {
			self.Add(text[ : left])
			text = text[left : ]
		}
		right -= left

		// see what kind of formatting tag we have and find the closing match
		codeLen := right
		var fmtcode string
		switch codeLen {
		case 3    : fmtcode = text[1 : right - 1]
		case 6    : fmtcode = "code"
		case 8, 9 : fmtcode = "size"
		case 15   : fmtcode = "color"
		default:
			panic("broken code")
		}

		// see if there's a closing tag
		index := strings.Index(text[right : ], "[/" + fmtcode + "]")
		if index == -1 { // no tag, show format tag as regular text
			self.Add(text[ : right])
			text = text[right : ]
			continue
		}
		index += right // because the search started on 'right'

		// apply format
		switch fmtcode {
		case "b": // bold
			self.PushEffect(etxt.EffectFauxBold, etxt.SinglePass)
		case "i": // italics
			self.PushEffect(etxt.EffectOblique, etxt.SinglePass)
		case "s": // strikethrough
			self.PushEffect(etxt.EffectCrossOut, etxt.SinglePass)
		case "code": // code
			self.PushFont(self.codeFontIndex)
			self.ShiftSize(-1)
			self.PushColor(textColor) // force text color, don't color code from other markup
			pad := fract.FromFloat64(6.0*ebiten.DeviceScaleFactor())
			spacing := etxt.TwineEffectSpacing{ PrePad: pad, PostPad: pad }
			r, g, b, a := uint8(96), uint8(96), uint8(96), uint8(128)
			self.PushEffectWithSpacing(etxt.EffectHighlightB, etxt.DoublePass, spacing, r, g, b, a)
		case "size": // size
			size, err := strconv.ParseInt(text[6 : right - 1], 10, 7)
			if err != nil { panic(err) }
			self.PushEffect(etxt.EffectSetSize, etxt.SinglePass, uint8(size))
		case "color": // color
			hex := text[8 : right - 1]
			if len(hex) != 6 { panic("broken code") }
			r, g, b := parseHexColor(hex[0 : 2]), parseHexColor(hex[2 : 4]), parseHexColor(hex[4 : 6])
			self.PushColor(color.RGBA{r, g, b, 255})
		default:
			panic("broken code")
		}

		// we recursively preprocess the inner text fragment.
		// this is not super efficient, but it's easier to code
		self.Preprocess(text[right : index])
		text = text[index + len(fmtcode) + 3 : ]
		if fmtcode == "code" { // life is messy
			for i := 0; i < 4; i++ { self.Pop() }
		} else {
			self.Pop()
		}
	}
}

func parseHexColor(cc string) uint8 {
	return (runeDigit(cc[0]) << 4) + runeDigit(cc[1])
}
func runeDigit(r uint8) uint8 {
	if r > '9' { return uint8(r) - 55 }
	return uint8(r) - 48
}

// ---- game interface ----

type Game struct {
	text *BBCodeRenderer
	startTime time.Time
	initialized bool
	mPressed bool
}

func (self *Game) Layout(winWidth, winHeight int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	self.text.SetScale(scale) // relevant for HiDPI
	canvasWidth  := int(math.Ceil(float64(winWidth)*scale))
	canvasHeight := int(math.Ceil(float64(winHeight)*scale))
	return canvasWidth, canvasHeight
}

func (self *Game) Update() error {
	if !self.initialized {
		self.startTime = time.Now()
		self.initialized = true
	}
	mPressed := ebiten.IsKeyPressed(ebiten.KeyM)
	if mPressed != self.mPressed {
		self.mPressed = mPressed
		if mPressed {
			self.text.preprocess = !self.text.preprocess
		}
	}
	return nil
}

// Doing everything on draw is not recommended in general, but
// in this particular it helps make code simpler. Slow, but simpler.
func (self *Game) Draw(canvas *ebiten.Image) {
	// fill background
	canvas.Fill(backColor)

	// get available space and relevant metrics
	bounds := canvas.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	margin := int(16.0*ebiten.DeviceScaleFactor())
	lineHeight := self.text.Utils().GetLineHeight()
	lineOffsetStd   := int(lineHeight + 0.5)
	lineOffsetExtra := int(lineHeight*1.4 + 0.5)
	x := margin
	y := h - margin
	
	// draw user message box
	rect := image.Rect(x, y - lineOffsetStd - margin, w - margin, y)
	canvas.SubImage(rect).(*ebiten.Image).Fill(shadeColor)
	self.text.SetColor(hintColor)
	self.text.SetSize(sizeNormal)
	self.text.Draw(canvas, ChatWhyNot, x + margin, y - (lineOffsetStd + margin)/2)
	y -= (lineOffsetStd + margin + margin/3)

	// draw who's typing
	self.text.SetColor(hintColor)
	self.text.SetSize(sizeSmall)
	self.text.SetAlign(etxt.Baseline)

	second := int(time.Now().Sub(self.startTime).Seconds())
	var usersTyping []string
	for _, event := range Chat {
		if event.WasTypingAt(second) {
			usersTyping = append(usersTyping, event.Username())
		}
	}
	if len(usersTyping) > 0 {
		if len(usersTyping) == 1 {
			self.text.Draw(canvas, usersTyping[0] + " is typing...", x, y)
		} else {
			self.text.Draw(canvas, joinNames(usersTyping) + " are typing...", x, y)
		}
	}
	
	// draw preprocessing toggle hint
	self.text.SetAlign(etxt.Right)
	if self.text.preprocess {
		self.text.Draw(canvas, "markup preprocessing ON [M]", w - margin, y)
	} else {
		self.text.Draw(canvas, "markup preprocessing OFF [M]", w - margin, y)
	}
	self.text.SetAlign(etxt.VertCenter | etxt.Left)
	y -= (lineOffsetExtra + margin)

	// collect chat messages and write them from bottom to top
	// until we run out of space. very raw and inefficient, sure
	self.text.SetColor(textColor)
	self.text.SetSize(sizeNormal)
	
	var hasPrevEvent bool
	var prevEvent Event
	var index = len(Chat)
	for {
		var event Event
		event, index = fetchNextMessageBefore(index, second)

		// format message
		if index >= 0 {
			// add user name of previous chatter if changing message authors
			if hasPrevEvent && event.UserID != prevEvent.UserID {
				preprocess := self.text.preprocess
				self.text.preprocess = true
				self.text.Draw(canvas, self.fmtUserName(prevEvent), x, y)
				self.text.preprocess = preprocess
				y -= lineOffsetExtra
			}
			
			// write new message
			self.text.Draw(canvas, event.Message, x, y)
			y -= lineOffsetStd
		}

		// write username if this is the last message
		stop := (index < 0 || y < margin/2)
		if hasPrevEvent && stop {
			preprocess := self.text.preprocess
			self.text.preprocess = true
			self.text.Draw(canvas, self.fmtUserName(event), x, y)
			self.text.preprocess = preprocess
			y -= lineOffsetExtra
		}

		// stop or update looping variables
		if stop { break }
		hasPrevEvent = true
		prevEvent = event
	}

	// re-fill top margin in case we drew some text over it
	rect = image.Rect(0, 0, w, margin)
	canvas.SubImage(rect).(*ebiten.Image).Fill(backColor)
}

// formats and returns the user name text to be displayed
// above its messages
func (self *Game) fmtUserName(event Event) string {
	messageTimeOffset := time.Duration(event.MessageSendTime)*time.Second
	return fmt.Sprintf(
		"[b][color=%s]%s[/color][/b]  [color=%s][size=%d]%s[/size][/color]",
		event.UserColorHex(), event.Username(), hintColorHex, sizeTime,
		self.startTime.Add(messageTimeOffset).Format("15:04 PM"),
	)
}

// helper function
func joinNames(names []string) string {
	var namesChain string
	for i, name := range names {
		namesChain += name
		if i == len(names) - 2 {
			namesChain += " and "
		} else if i != len(names) - 1 {
			namesChain += ", "
		}
	}
	return namesChain
}

// ---- main ----

func main() {
	// get font path
	if len(os.Args) < 2 || len(os.Args) > 3 {
		msg := "Usage: expects one argument with the path to the font to be used, plus optionally a second monospaced font\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// create and configure renderer (except fonts)
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	var rasterizer mask.FauxRasterizer
	renderer.Glyph().SetRasterizer(&rasterizer)
	renderer.SetAlign(etxt.VertCenter | etxt.Left)

	// parse and set font(s)
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)
	renderer.SetFont(sfntFont)
	
	// (optional code font)
	codeFontIndex := renderer.Twine().GetFontIndex()
	if len(os.Args) == 3 {
		codeFont, codeFontName, err := font.ParseFromPath(os.Args[2])
		if err != nil { log.Fatal(err) }
		fmt.Printf("Code font loaded: %s\n", codeFontName)
		codeFontIndex = renderer.Twine().RegisterFont(etxt.NextFontIndex, codeFont)
	}

	// set up bbcode renderer
	bbcodeRenderer := BBCodeRenderer{ Renderer: *renderer, preprocess: true }
	bbcodeRenderer.twine.codeFontIndex = codeFontIndex

	// run the game
	ebiten.SetWindowTitle("etxt/examples/ebiten/bbcode")
	ebiten.SetWindowSize(960, 600)
	err = ebiten.RunGame(&Game{ text: &bbcodeRenderer })
	if err != nil { log.Fatal(err) }
}

// ---- chat content ----
// The remaining content is just for fun, not relevant to learning etxt.

var ChatWhyNot string = "You have been temporarily banned for spamming."

// I put this at the end to avoid spoilers, it's more fun like this.
type Event struct {
	UserID uint8
	Message string
	TypingStartTime int // in seconds
	MessageSendTime int // in seconds
}
func (self *Event) WasTypingAt(sec int) bool {
	return sec >= self.TypingStartTime && sec < self.MessageSendTime
}
func (self *Event) Username() string {
	return Users[self.UserID].Name
}
func (self *Event) UserColorHex() string {
	return Users[self.UserID].ColorHex
}

type User struct {
	Name string
	ColorHex string
}

var Users = []User{
	User{ "gopher42", "#880000" },
	User{ "mommyfan", "#BD0658" },
	User{ "bustedGhost", "#009999" },
	User{ "shrimper3000", "#EB7100" },
	User{ "transcendent", "#CCA90C" },
}

var Chat = []Event{
	Event{ 0, "Hello, anyone around?", 1, 2 },
	Event{ 1, "helloo [i]*waves gopher*[/i]", 4, 5 },
	Event{ 2, "Silently lurking, we are always around...", 8, 10 },
	Event{ 2, "You are new here, aren't you [b]@gopher42[/b]..?", 11, 14 },
	Event{ 3, "Don't mind ghost, gopher, he's always creepy like that.", 14, 17 },
	Event{ 3, "Also, hello", 18, 19 },
	Event{ 0, "Hi everyone! I was trying to print centered text with Ebitengine", 18, 27 },
	Event{ 0, "but I couldn't figure it out. Does anyone know how to do centered text?", 28, 28 },
	Event{ 3, "Are you using [code]ebiten/v2/text[/code]?", 36, 39 },
	Event{ 0, "Yes.", 42, 43 },
	Event{ 0, "I managed to show text, but I don't know how to center it.", 46, 50 },
	Event{ 3, "You should look into [code]ebiten/v2/text/[b]v2[/b][/code] instead.", 48, 53 },
	Event{ 3, "It was added on v2.7.0, and you have an explicit [code]Align[/code] that you can", 54, 59 },
	Event{ 3, "pass to [code]Draw[/code] through [code]*DrawOptions[/code].", 60, 60 },
	Event{ 2, "Hajime never sleeps. I haunt him at night to keep him writing code.", 63, 68 },
	Event{ 2, "He he he...", 69, 71 },
	Event{ 1, "you were already busted, shuuush and let people sleep~!", 74, 78 },
	Event{ 1, "bad ghost! <3", 79, 81 },
	Event{ 4, "yo, wassup", 89, 91 },
	Event{ 0, "Oh, I didn't know about text/v2, thanks shrimper3000, I'll look into it!", 86, 92 },
	Event{ 1, "heya! [i]*waves transcendent*[/i]", 94, 97 },
	Event{ 2, "I have [s]never tormented anyone[/s]...", 95, 100 },
	Event{ 2, "mommy I have been a bad ghost... :'(", 101, 104 },
	Event{ 1, "you gotta learn to be a good boi [i]*pats ghost*[/i]", 112, 117 },
	Event{ 2, "*hand passes through ghost head*", 122, 125 },
	Event{ 4, "[b]@gopher42[/b] there's also the unofficial [code]etxt[/code] package", 116, 127 },
	Event{ 4, "[b]@mommyfan[/b] [b]@bustedGhost[/b] you two get a room and be weird [i]in private[/i] ^^", 129, 136 },
	Event{ 1, ">.<", 144, 145 },
	Event{ 0, "Thanks transcendent, I'll look into that too!", 146, 150 },
	Event{ 3, "Oh, etxt. Don't remind me of that guy...", 155, 157 },
	Event{ 3, "He spammed the chat with, like, two thousand links the other day.", 158, 164 },
	Event{ 3, "Etxt this, etxt that. Calm down already.", 165, 168 },
	Event{ 4, "unhinged", 170, 173 },
	Event{ 2, "Egg bannend him right after you left..", 178, 181 },
	Event{ 2, "*banned", 182, 183 },
	Event{ 4, "eggplantz our savior", 190, 195 },
	Event{ 3, "Really ghost? [color=#00AA00]Good[/color]. He was driving me insane.", 193, 198 },
	Event{ 2, "yeah", 207, 208 },
	Event{ 1, "[i]spammer wiiiiped yiii[/i]  \\o/", 207, 209 },
	Event{ 2, "I have to vanish for today. Haunt you all...", 223, 226 },
	Event{ 4, "have a good one [b]@bustedGhost[/b]", 230, 234 },
	Event{ 1, "byyeee", 237, 238 },
}

func init() {
	// ensure chat events are sorted by MessageSendTime, and that they don't overlap
	var minSendTime int
	for i, _ := range Chat {
		sendTime := Chat[i].MessageSendTime
		if sendTime <= minSendTime {
			panic("chat events not properly sorted by MessageSendTime")
		}
		minSendTime = sendTime
	}
}

func fetchNextMessageBefore(index, second int) (Event, int) {
	var event Event

	// first valid index to look up is the previous to the given one
	index -= 1
	if index < 0 { return event, -1 }
	if index > len(Chat) - 1 {
		index = len(Chat) - 1
	}

	// ensure that we are getting something <= the given time
	for Chat[index].MessageSendTime > second {
		index -= 1
		if index < 0 { return event, -1 }
	}

	return Chat[index], index
}
