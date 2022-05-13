package main

import "os"
import "log"
import "strconv"
import "strings"
import "image"
import "image/color"
import "image/png"
import "math/rand"
import "time"

import "github.com/hajimehoshi/ebiten/v2"
import "github.com/hajimehoshi/ebiten/v2/ebitenutil"
import "github.com/tinne26/etxt/emask"
import "golang.org/x/image/math/fixed"

// Yeah, this has very little to do with etxt, but...
// One day I was testing the results of vector.Rasterizer against
// emask.EdgeMarkerRasterizer and I went for random tests. After
// having some trouble and exporting the images for visual comparison,
// I realized they were pretty cool. Imagine, writing tests leading
// to fancier results than when I try intentionally!
//
// And then I decided to build a bigger example to play with.
// Unlike other examples, the code is very raw, but feel free to
// have fun with it!

func init() {
	rand.Seed(time.Now().UnixNano())
}

var keys = []ebiten.Key {
	ebiten.KeySpace, ebiten.KeyArrowUp, ebiten.KeyArrowDown,
	ebiten.KeyE, ebiten.KeyN, ebiten.KeyD, ebiten.KeyH, ebiten.KeyF,
}

type Game struct {
	keyPressed map[ebiten.Key]bool
	rasterizer *emask.EdgeMarkerRasterizer
	shape emask.Shape
	size int
	margin int
	hideShortcuts bool
	segments int
	symmetryDir int // 0 = right, 1 = bottom right, 2 = down, ...,
	symmetryCount int // 0, 1, 2
	originalImg *image.Alpha
	symmetryImg *image.Alpha
	ebiImg *ebiten.Image
}

func (self *Game) Layout(w, h int) (int, int) { return w, h }
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
				if ebiten.IsKeyPressed(ebiten.KeyZ) {
					change := 5
					if slow { change = 1 }
					self.margin += change
				} else if ebiten.IsKeyPressed(ebiten.KeyS) {
					self.segments += 1
				} else {
					change := 20
					if slow { change = 1 }
					self.size += change
				}
			case ebiten.KeyArrowDown:
				slow := ebiten.IsKeyPressed(ebiten.KeyShiftLeft)
				if ebiten.IsKeyPressed(ebiten.KeyZ) {
					change := 5
					if slow { change = 1 }
					self.margin -= change
					if self.margin < 0 { self.margin = 0 }
				} else if ebiten.IsKeyPressed(ebiten.KeyS) {
					self.segments -= 1
					if self.segments < 3 { self.segments = 3 }
				} else {
					change := 20
					if slow { change = 1 }
					self.size -= change
					if self.size < 50 { self.size = 50 }
				}
			case ebiten.KeyN: // increase symmetry num
				self.symmetryCount += 1
				if self.symmetryCount == 2 { self.symmetryCount = 0 } // TODO: go up to == 3
				self.refreshSymmetry()
			case ebiten.KeyD:
				self.symmetryDir = (self.symmetryDir + 1)%8
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
				log.Printf("exported shape data")
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
	self.shape.MoveTo(-self.margin, -self.margin)
	self.shape.MoveTo(self.size + self.margin, self.size + self.margin)

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
	margin := bounds.Min.X
	w, h := bounds.Dx() + margin*2, bounds.Dy() + margin*2
	if w != h { panic("what?") }

	self.symmetryImg = image.NewAlpha(bounds)
	copy(self.symmetryImg.Pix, self.originalImg.Pix)

	switch self.symmetryCount {
	case 0:
		// nothing to do here
	case 1:
		// doubling the image
		switch self.symmetryDir {
		case 0: // right
			xStart, yStart, xEnd, yEnd := 0, 0, w/2, h
			pix := getImgQuad(self.originalImg, xStart, yStart, xEnd, yEnd)
			xStart, xEnd = w, w - w/2
			setImgQuad(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		case 2: // down
			xStart, yStart, xEnd, yEnd := 0, 0, w, h/2
			pix := getImgQuad(self.originalImg, xStart, yStart, xEnd, yEnd)
			yStart, yEnd = h, h - h/2
			setImgQuad(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		case 4: // left
			xStart, yStart, xEnd, yEnd := w/2, 0, w, h
			pix := getImgQuad(self.originalImg, xStart, yStart, xEnd, yEnd)
			xStart, xEnd = w - w/2, 0
			setImgQuad(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		case 6: // up
			xStart, yStart, xEnd, yEnd := 0, 0, w, h/2
			pix := getImgQuad(self.originalImg, xStart, yStart, xEnd, yEnd)
			yStart, yEnd = h, h - h/2
			setImgQuad(self.symmetryImg, xStart, yStart, xEnd, yEnd, pix)
		case 1: // diagonal right-down
			xStart, yStart := 0, 0
			pix := getImgHorzTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = w, h
			setImgVertTri(self.symmetryImg, xStart, yStart, w, pix)

			xStart, yStart = 0, 0
			pix = getImgVertTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = w, h
			setImgHorzTri(self.symmetryImg, xStart, yStart, w, pix)
		case 3: // diagonal down-left
			xStart, yStart := 0, 0
			pix := getImgHorzTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = 0, 0
			setImgVertTri(self.symmetryImg, xStart, yStart, w, pix)

			xStart, yStart = w, 0
			pix = getImgVertTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = 0, h
			setImgHorzTri(self.symmetryImg, xStart, yStart, w, pix)
		case 5: // diagonal left-up
			xStart, yStart := w, h
			pix := getImgHorzTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = 0, 0
			setImgVertTri(self.symmetryImg, xStart, yStart, w, pix)

			xStart, yStart = w, 0
			pix = getImgVertTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = w, 0
			setImgHorzTri(self.symmetryImg, xStart, yStart, w, pix)
		case 7: // diagonal up-right
			xStart, yStart := w, h
			pix := getImgHorzTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = w, h
			setImgVertTri(self.symmetryImg, xStart, yStart, w, pix)

			xStart, yStart = 0, 0
			pix = getImgVertTri(self.originalImg, xStart, yStart, w)
			xStart, yStart = 0, 0
			setImgHorzTri(self.symmetryImg, xStart, yStart, w, pix)
		}
	case 2:
		// double symmetry (x4)
		// TODO...
	}

	self.ebiImg = ebiten.NewImage(bounds.Dx(), bounds.Dy())
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
		content := "hide menu [H]\nexport [E]\ngenerate [SPACE]\nsize "
		content += strconv.Itoa(self.size) + " [UP/DOWN](+shift)\nmargin "
		content += strconv.Itoa(self.margin) + " [Z + UP/DOWN]\nsymmetry x"
		content += strconv.Itoa(self.symmetryCount) + " [N]\nsymmetry dir "
		content += strconv.Itoa(self.symmetryDir + 1) + "/8 [D]\nsegments "
		content += strconv.Itoa(self.segments) + " [S + UP/DOWN]"
		ebitenutil.DebugPrint(screen, content)
	}
}

func main() {
	ebiten.SetWindowTitle("rng shapes")
	ebiten.SetWindowResizable(true)
	ebiten.SetWindowSize(640, 480)
	game := &Game{
		rasterizer: emask.NewStdEdgeMarkerRasterizer(),
		keyPressed: make(map[ebiten.Key]bool),
		size: 476,
		segments: 16,
		margin: 2,
		symmetryCount: 1,
	}
	err := game.newImage()
	if err != nil { log.Fatal(err) }
	err = ebiten.RunGame(game)
	if err != nil { log.Fatal(err) }
}

// --- lots of helper functions for symmetries ---

func getImgQuad(img *image.Alpha, xStart, yStart, xEnd, yEnd int) []color.Alpha {
	result := make([]color.Alpha, 0, (xEnd - xStart)*(yEnd - yStart))
	for y := yStart; y != yEnd; y += 1 {
		for x := xStart; x != xEnd; x += 1 {
			result = append(result, img.AlphaAt(x, y))
		}
	}
	return result
}

func setImgQuad(img *image.Alpha, xStart, yStart, xEnd, yEnd int, pix []color.Alpha) {
	xChange, yChange := 1, 1
	if xEnd < xStart { xChange = -1 }
	if yEnd < yStart { yChange = -1 }
	i := 0
	for y := yStart; y != yEnd; y += yChange {
		for x := xStart; x != xEnd; x += xChange {
			img.SetAlpha(x, y, pix[i])
			i += 1
		}
	}
}

func getImgHorzTri(img *image.Alpha, xStart, yStart, size int) []color.Alpha {
	result := make([]color.Alpha, 0, (size*size)/4)
	xChange := 1
	xEnd := size
	if xStart != 0 {
		xChange = -1
		xEnd = 0
	}

	for rows := 0; rows != size/2; rows += 1 {
		y := rows
		if yStart != 0 { y = size - rows - 1 }
		for x := xStart + rows*xChange; x != xEnd - rows*xChange; x += xChange {
			result = append(result, img.AlphaAt(x, y))
		}
	}
	return result
}

func getImgVertTri(img *image.Alpha, xStart, yStart, size int) []color.Alpha {
	result := make([]color.Alpha, 0, (size*size)/4)
	yChange := 1
	yEnd := size
	if yStart != 0 {
		yChange = -1
		yEnd = 0
	}

	for cols := 0; cols != size/2; cols += 1 {
		x := cols
		if xStart != 0 { x = size - cols - 1 }
		for y := yStart + cols*yChange; y != yEnd - cols*yChange; y += yChange {
			result = append(result, img.AlphaAt(x, y))
		}
	}
	return result
}

func setImgHorzTri(img *image.Alpha, xStart, yStart, size int, pix []color.Alpha) {
	xChange := 1
	xEnd := size
	if xStart != 0 {
		xChange = -1
		xEnd = 0
	}

	i := 0
	for rows := 0; rows != size/2; rows += 1 {
		y := rows
		if yStart != 0 { y = size - rows - 1 }
		for x := xStart + rows*xChange; x != xEnd - rows*xChange; x += xChange {
			img.SetAlpha(x, y, pix[i])
			i += 1
		}
	}
}

func setImgVertTri(img *image.Alpha, xStart, yStart, size int, pix []color.Alpha) {
	yChange := 1
	yEnd := size
	if yStart != 0 {
		yChange = -1
		yEnd = 0
	}

	i := 0
	for cols := 0; cols != size/2; cols += 1 {
		x := cols
		if xStart != 0 { x = size - cols - 1 }
		for y := yStart + cols*yChange; y != yEnd - cols*yChange; y += yChange {
			img.SetAlpha(x, y, pix[i])
			i += 1
		}
	}
}
