//go:build gtxt

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/font"
)

// Must be compiled with '-tags gtxt'

func main() {
	// we want random sentences in order to find the text size dynamically,
	// so we start declaring different text fragments to combine later
	who := []string{
		"my doggy", "methuselah", "the king", "the queen", "mr. skywalker",
		"your little pony", "my banana", "gopher", "jigglypuff", "evil jin",
		"the genius programmer", "your boyfriend", "the last samurai",
		"the cute robot", "your ancestor's ghost",
	}
	what := []string{
		"climbs a tree", "writes a book", "stares at you", "commissions naughty art",
		"smiles", "takes scenery pics", "pays the bill", "practices times tables",
		"prays", "runs to take cover", "joins the chat", "downvotes your post",
		"discovers the moon", "poops", "questions your sense of humor",
		"re-opens the github issue", "talks to its clone", "arrives at the disco",
		"spies the neighbours", "solves the hardest equation", "discusses geopolitics",
		"gets mad at you for crossing the street",
	}
	how := []string{
		"while dancing", "in style", "while undressing", "while getting high",
		"maniacally", "early in the morning", "right at the last moment",
		"as the world ends", "without much fuss", "bare-chested", "periodically",
		"every day", "with the gang", "without using the hands",
		"with the eyes closed", "bored as hell", "while remembering the past",
	}

	// get font path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font to be used\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// parse font
	sfntFont, fontName, err := font.ParseFromPath(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Font loaded: %s\n", fontName)

	// create and configure renderer
	renderer := etxt.NewRenderer()
	renderer.Utils().SetCache8MiB()
	renderer.SetSize(16)
	renderer.SetFont(sfntFont)
	renderer.SetAlign(etxt.Center)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// generate the random sentences
	rand.Seed(time.Now().UnixNano())
	sentences := make([]string, 2+rand.Intn(6))
	fmt.Printf("Generating %d sentences...\n", len(sentences))
	for i := 0; i < len(sentences); i++ {
		sentence := who[rand.Intn(len(who))] + " "
		sentence += what[rand.Intn(len(what))] + " "
		sentence += how[rand.Intn(len(how))]
		sentences[i] = sentence
	}
	fullText := strings.Join(sentences, "\n")

	// determine how much space should it take to draw the
	// sentences, plus a bit of vertical and horizontal padding
	w, h := renderer.Measure(fullText).PadInts(8, 6).IntSize()

	// create target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h*4; i++ {
		outImage.Pix[i] = 255
	}

	// draw the sentences
	renderer.Draw(outImage, fullText, w/2, h/2)

	// store image as png
	filename, err := filepath.Abs("gtxt_measure.png")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	err = png.Encode(file, outImage)
	if err != nil {
		log.Fatal(err)
	}
	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Program exited successfully.\n")
}
