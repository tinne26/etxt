//go:build !gtxt

package etxt

import "os"
import "fmt"
import "image"
import "image/color"

import "github.com/hajimehoshi/ebiten/v2"

// --- basic helper functions ---

// variables for fillOver function
var vertices [4]ebiten.Vertex
var stdTriOpts ebiten.DrawTrianglesOptions
var mask1x1 *ebiten.Image
func init() {
	mask3x3 := ebiten.NewImage(3, 3)
	mask3x3.Fill(color.RGBA{255, 255, 255, 255})
	mask1x1 = mask3x3.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
	for i := 0; i < len(vertices); i++ {
		vertices[i].SrcX = 1.0
		vertices[i].SrcY = 1.0
	}
}

func fillOver(target Target, fillColor color.Color) {
	bounds := target.Bounds()
	if bounds.Empty() { return }

	minX, minY := float32(bounds.Min.X), float32(bounds.Min.Y)
	maxX, maxY := float32(bounds.Max.X), float32(bounds.Max.Y)
	fillOverF32(target, fillColor, minX, minY, maxX, maxY)
}

func fillOverF32(target Target, fillColor color.Color, minX, minY, maxX, maxY float32) {
	r, g, b, a := fillColor.RGBA()
	if a == 0 { return }
	fr, fg, fb, fa := float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535
	for i := 0; i < 4; i++ {
		vertices[i].ColorR = fr
		vertices[i].ColorG = fg
		vertices[i].ColorB = fb
		vertices[i].ColorA = fa
	}

	vertices[0].DstX = minX
	vertices[0].DstY = minY
	vertices[1].DstX = maxX
	vertices[1].DstY = minY
	vertices[2].DstX = maxX
	vertices[2].DstY = maxY
	vertices[3].DstX = minX
	vertices[3].DstY = maxY

	target.DrawTriangles(vertices[0 : 4], []uint16{0, 1, 2, 2, 3, 0}, mask1x1, &stdTriOpts)
}

// ---- shaders ----

var rectShader *ebiten.Shader
var rectShaderOpts ebiten.DrawTrianglesShaderOptions
var rectShaderSrc = []byte(`package main

var Rect vec4 // minX, minY, maxX, maxY

func Fragment(position vec4, _ vec2, color vec4) vec4 {
	dist := rectSDF(position.xy)
	dist = min( dist, 0)
	dist = min(-dist, 1)
	return color*dist
}

func rectSDF(position vec2) float {
	size := vec2(Rect[2] - Rect[0], Rect[3] - Rect[1])
	origPos := position - Rect.xy - size/2
	outDistXY := abs(origPos) - size/2
	outDist := length(max(outDistXY, 0.0))
	inDist := min(max(outDistXY.x, outDistXY.y), 0.0)
	return outDist + inDist
}
`)

func loadShader(target **ebiten.Shader, src []byte) {
	shader, err := ebiten.NewShader(src)
	if err != nil {
		fmt.Printf("Error while loading shader:\n>> %s\n", err.Error())
		os.Exit(1)
	}
	*target = shader
}

func drawSmoothRect(target *ebiten.Image, minX, minY, maxX, maxY float32, fillColor color.Color) {
	if rectShader == nil {
		loadShader(&rectShader, rectShaderSrc)
		rectShaderOpts.Uniforms = make(map[string]interface{}, 1)
		rectShaderOpts.Uniforms["Rect"] = []float32{ 0, 0, 0, 0 }
	}

	// set uniforms
	slice := rectShaderOpts.Uniforms["Rect"].([]float32)
	slice[0], slice[1] = minX, minY
	slice[2], slice[3] = maxX, maxY
	
	// set vertex colors
	r, g, b, a := fillColor.RGBA()
	if a == 0 { return }
	fr, fg, fb, fa := float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535
	for i := 0; i < 4; i++ {
		vertices[i].ColorR = fr
		vertices[i].ColorG = fg
		vertices[i].ColorB = fb
		vertices[i].ColorA = fa
	}

	vertices[0].DstX = minX - 1.0
	vertices[0].DstY = minY - 1.0
	vertices[1].DstX = maxX + 1.0
	vertices[1].DstY = minY - 1.0
	vertices[2].DstX = maxX + 1.0
	vertices[2].DstY = maxY + 1.0
	vertices[3].DstX = minX - 1.0
	vertices[3].DstY = maxY + 1.0

	target.DrawTrianglesShader(vertices[0 : 4], []uint16{0, 1, 2, 2, 3, 0}, rectShader, &rectShaderOpts)
}
