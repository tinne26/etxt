package main

import "os"
import "log"
import "fmt"
import "time"
import "math"
import "strconv"
import "strings"
import "image"
import "image/color"
import "image/png"
import "math/rand"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/ebitenutil"
import "github.com/tinne26/etxt/emask"
import "golang.org/x/image/math/fixed"

// One day I was checking the results of emask.EdgeMarkerRasterizer
// and I decided to compare them with vector.Rasterizer. I made tests
// be randomized so I could be more confident that everything was
// good... but after having some trouble matching the results of the
// two, I exported the images for visual comparison and found out
// they were actually really cool!
//
// Imagine, writing tests leading to fancier results than when I
// intentionally try to make something look good >.<
//
// And that's the story of how this example was born. Does it have
// anything to do with etxt? Not really, but it's cool. I even added
// symmetries for extra fun!

func init() {
	rand.Seed(time.Now().UnixNano())
}

var keys = []ebiten.Key {
	ebiten.KeySpace, ebiten.KeyArrowUp, ebiten.KeyArrowDown,
	ebiten.KeyE, ebiten.KeyM, ebiten.KeyH, ebiten.KeyF,
}

type Game struct {
	keyPressed map[ebiten.Key]bool
	rasterizer *emask.EdgeMarkerRasterizer
	shape emask.Shape
	size int
	hideShortcuts bool
	segments int
	symmetryMode int // 0 = none, 1 = mirror, 2 = x2, 3 = diag.x2
	originalImg *image.Alpha
	symmetryImg *image.Alpha
	ebiImg *ebiten.Image
}

func (self *Game) Layout(w, h int) (int, int) {
	scale := ebiten.DeviceScaleFactor()
	return int(math.Ceil(float64(w)*scale)), int(math.Ceil(float64(h)*scale))
}
func (self *Game) Update() error {
	for _, key := range keys {
		wasPressed := self.keyPressed[key]
		isPressed  := ebiten.IsKeyPressed(key)
		self.keyPressed[key] = isPressed
		if !wasPressed && isPressed {
			switch key {
			case ebiten.KeyH:
				self.hideShortcuts = !self.hideShortcuts
			case ebiten.KeyF:
				ebiten.SetFullscreen(!ebiten.IsFullscreen())
			case ebiten.KeySpace:
				err := self.newImage()
				if err != nil { return err }
			case ebiten.KeyArrowUp:
				slow := ebiten.IsKeyPressed(ebiten.KeyShiftLeft)
				if ebiten.IsKeyPressed(ebiten.KeyS) {
					self.segments += 1
				} else {
					change := 20
					if slow { change = 1 }
					self.size += change
				}
			case ebiten.KeyArrowDown:
				slow := ebiten.IsKeyPressed(ebiten.KeyShiftLeft)
				if ebiten.IsKeyPressed(ebiten.KeyS) {
					self.segments -= 1
					if self.segments < 3 { self.segments = 3 }
				} else {
					change := 20
					if slow { change = 1 }
					self.size -= change
					if self.size < 50 { self.size = 50 }
				}
			case ebiten.KeyM: // increase symmetry num
				self.symmetryMode += 1
				if self.symmetryMode == 4 { self.symmetryMode = 0 }
				self.refreshSymmetry()
			case ebiten.KeyE:
				// export image
				file, err := os.Create("rng_shape.png")
				if err != nil { return err }
				err = png.Encode(file, self.symmetryImg)
				if err != nil { return err }
				err = file.Close()
				if err != nil { return err }

				// export raw data
				file, err = os.Create("rng_shape.txt")
				if err != nil { return err }
				var strBuilder strings.Builder
				for _, segment := range self.shape.Segments() {
					strBuilder.WriteString(strconv.Itoa(int(segment.Op)))
					for i := 0; i < 3; i++ {
						strBuilder.WriteRune(' ')
						point := segment.Args[i]
						strBuilder.WriteString(strconv.Itoa(int(point.X)))
						strBuilder.WriteRune(' ')
						strBuilder.WriteString(strconv.Itoa(int(point.Y)))
					}
					strBuilder.WriteRune('\n')
				}
				_, err = file.WriteString(strBuilder.String())
				if err != nil { return err }
				err = file.Close()
				if err != nil { return err }
				fmt.Print("Exported shape data successfully!\n")
			default:
				panic(key)
			}
		}
	}

	return nil
}

func (self *Game) newImage() error {
	fsw, fsh := float64(self.size)*64, float64(self.size)*64
	var makeXY = func() (fixed.Int26_6, fixed.Int26_6) {
		return fixed.Int26_6(rand.Float64()*fsw), fixed.Int26_6(rand.Float64()*fsh)
	}
	startX, startY := makeXY()
	self.shape = emask.NewShape(self.segments + 3)
	self.shape.InvertY(true)

	// trick to expand bounds
	self.shape.MoveTo(0, 0)
	self.shape.MoveTo(self.size, self.size)

	// actual shape generation
	self.shape.MoveToFract(startX, startY)
	for i := 0; i < self.segments; i++ {
		x, y := makeXY()
		switch rand.Intn(3) {
		case 0: // LineTo
			self.shape.LineToFract(x, y)
		case 1: // QuadTo
			cx, cy := makeXY()
			self.shape.QuadToFract(cx, cy, x, y)
		case 2: // CubeTo
			cx1, cy1 := makeXY()
			cx2, cy2 := makeXY()
			self.shape.CubeToFract(cx1, cy1, cx2, cy2, x, y)
		}
	}
	self.shape.LineToFract(startX, startY)
	var err error
	self.originalImg, err = emask.Rasterize(self.shape.Segments(), self.rasterizer, fixed.Point26_6{})
	if err != nil { return err }
	self.refreshSymmetry()
	return nil
}

func (self *Game) refreshSymmetry() {
	bounds := self.originalImg.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w != h { panic("what?") }

	self.symmetryImg = image.NewAlpha(bounds)
	copy(self.symmetryImg.Pix, self.originalImg.Pix)

	switch self.symmetryMode {
	case 0:
		// nothing to do here
	case 1: // mirror
		xStart, yStart, xEnd, yEnd := 0, 0, w/2, h
		pix := getImgRect(self.originalImg, xStart, yStart, xEnd, yEnd)
		xStart, xEnd = w - 1, w - w/2 - 1
		setImgRect(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
	case 2: // x2
		odd := (w % 2)
		xStart, yStart, xEnd, yEnd := 0, 0, w/2 + odd, h/2 + odd
		pix := getImgRect(self.originalImg, xStart, yStart, xEnd, yEnd)
		xStart, xEnd = w - 1, w - w/2 - 1 - odd
		setImgRect(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		yStart, yEnd = h - 1, h - h/2 - 1 - odd
		setImgRect(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		xStart, xEnd = 0, w/2 + odd
		setImgRect(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
	case 3: // diag. x2
		pix := getImgTopToCenterTriangle(self.originalImg)
		setImgBottomToCenterTriangle(self.symmetryImg, pix)
		setImgLeftToCenterTriangle(self.symmetryImg, pix)
		setImgRightToCenterTriangle(self.symmetryImg, pix)
	}

	self.ebiImg = ebiten.NewImage(w, h)
	self.ebiImg.Fill(color.Black)
	img := ebiten.NewImageFromImage(self.symmetryImg)
	self.ebiImg.DrawImage(img, nil)
}

func (self *Game) Draw(screen *ebiten.Image) {
	sw, sh := screen.Size()
	iw, ih := self.ebiImg.Size()
	tx := (sw - iw)/2
	ty := (sh - ih)/2

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(tx), float64(ty))
	screen.DrawImage(self.ebiImg, opts)

	if !self.hideShortcuts {
		content := "export [E]\ngenerate [SPACE]\nsize "
		content += strconv.Itoa(self.size) + " [UP/DOWN](+shift)\n"
		switch self.symmetryMode {
		case 0: content += "no"
		case 1: content += "mirror"
		case 2: content += "x2"
		case 3: content += "diag.x2"
		}
		content += " symmetry [M]\nsegments "
		content += strconv.Itoa(self.segments) + " [S + UP/DOWN]"
		ebitenutil.DebugPrint(screen, content)
	}
}

func main() {
	fmt.Print("Instructions can be hidden with [H]\n")
	fmt.Print("Fullscreen can be switched with [F]\n")
	ebiten.SetWindowTitle("rng shapes")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(640, 480)
	game := &Game{
		rasterizer: emask.NewStdEdgeMarkerRasterizer(),
		keyPressed: make(map[ebiten.Key]bool),
		size: int(476*ebiten.DeviceScaleFactor()),
		segments: 16,
		symmetryMode: 1,
	}
	err := game.newImage()
	if err != nil { log.Fatal(err) }
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}

// --- lots of helper functions for symmetries ---

// Precondition: xStart <= xEnd, yStart <= yEnd
// Returns the colors of the given rect for the given image (xEnd and yEnd
// not included) as a single slice, from the top left to the bottom right.
func getImgRect(img *image.Alpha, xStart, yStart, xEnd, yEnd int) []color.Alpha {
	result := make([]color.Alpha, 0, (xEnd - xStart)*(yEnd - yStart))
	for y := yStart; y < yEnd; y += 1 {
		for x := xStart; x < xEnd; x += 1 {
			result = append(result, img.AlphaAt(x, y))
		}
	}
	return result
}

// Sets the colors of the given rect on the given image with the contents
// of pix (xEnd and yEnd not included). xStart and yStart may be smaller
// than xEnd and yEnd, respectively, and the iteration direction will be
// changed.
func setImgRect(img *image.Alpha, xStart, yStart, xEnd, yEnd int, pix []color.Alpha) {
	xChange, yChange := 1, 1
	if xEnd < xStart { xChange = -1 }
	if yEnd < yStart { yChange = -1 }
	index := 0
	for y := yStart; y != yEnd; y += yChange {
		for x := xStart; x != xEnd; x += xChange {
			img.SetAlpha(x, y, pix[index])
			index += 1
		}
	}
	if index != len(pix) { panic("incorrect pix len") }
}

// Returns the colors of the triangle that goes from the top corners of
// the given image to its center, as a single slice, from left to right,
// top to bottom.
func getImgTopToCenterTriangle(img *image.Alpha) []color.Alpha {
	bounds := img.Bounds()
	height := bounds.Dy()
	xStart, xEnd := bounds.Min.X, bounds.Max.X
	yStart, yEnd := bounds.Min.Y, bounds.Min.Y + height/2

	result := make([]color.Alpha, 0, (height*height)/4)
	offset := 0 // increased for each triangle row
	for y := yStart; y < yEnd; y += 1 {
		for x := xStart + offset; x < xEnd - offset; x += 1 {
			result = append(result, img.AlphaAt(x, y))
		}
		offset += 1
	}
	return result
}

// Sets the colors of the triangle that goes from the bottom corners of
// the given image to its center, from right to left and bottom to top.
func setImgBottomToCenterTriangle(img *image.Alpha, pix []color.Alpha) {
	bounds := img.Bounds()
	xStart, xEnd := bounds.Min.X, bounds.Max.X - 1
	yStart, yEnd := bounds.Max.Y - 1, bounds.Max.Y - bounds.Dy()/2

	index  := 0 // pix index
	offset := 0 // increased for each triangle row
	for y := yStart; y >= yEnd; y -= 1 {
		for x := xEnd - offset; x >= xStart + offset; x -= 1 {
			img.SetAlpha(x, y, pix[index])
			index += 1
		}
		offset += 1
	}
	if index != len(pix) { panic("incorrect pix len") }
}

// Sets the colors of the triangle that goes from the left corners of
// the given image to its center, from top to bottom and left to right.
func setImgLeftToCenterTriangle(img *image.Alpha, pix []color.Alpha) {
	bounds := img.Bounds()
	xStart, xEnd := bounds.Min.X, bounds.Min.X + bounds.Dx()/2
	yStart, yEnd := bounds.Min.Y, bounds.Max.Y

	index  := 0 // pix index
	offset := 0 // increased for each triangle column
	for x := xStart; x < xEnd; x += 1 {
		for y := yStart + offset; y < yEnd - offset; y += 1 {
			img.SetAlpha(x, y, pix[index])
			index += 1
		}
		offset += 1
	}
	if index != len(pix) { panic("incorrect pix len") }
}

// Sets the colors of the triangle that goes from the right corners of
// the given image to its center, from bottom to top and right to left.
func setImgRightToCenterTriangle(img *image.Alpha, pix []color.Alpha) {
	bounds := img.Bounds()
	xStart, xEnd := bounds.Max.X - 1, bounds.Max.X - bounds.Dx()/2
	yStart, yEnd := bounds.Max.Y - 1, bounds.Min.Y

	index  := 0 // pix index
	offset := 0 // increased for each triangle column
	for x := xStart; x >= xEnd; x -= 1 {
		for y := yStart - offset; y >= yEnd + offset; y -= 1 {
			img.SetAlpha(x, y, pix[index])
			index += 1
		}
		offset += 1
	}
	if index != len(pix) { panic("incorrect pix len") }
}
