//go:build gtxt

package main

import "os"
import "image"
import "image/color"
import "image/png"
import "path/filepath"
import "log"
import "fmt"
import "math/rand"
import "time"
import "strings"

import "github.com/tinne26/etxt"

// Must be compiled with '-tags gtxt'

func main() {
	// we want random sentences in order to find the text size dynamically,
	// so we start declaring different text fragments to combine later
	who  := []string {
		"my doggy", "methuselah", "the king", "the queen", "mr. skywalker",
		"your little pony", "my banana", "gopher", "jigglypuff", "evil jin",
		"the genius programmer", "your boyfriend", "the last samurai",
		"the cute robot", "your ancestor's ghost",
	}
	what := []string {
		"climbs a tree", "writes a book", "stares at you", "commissions naughty art",
		"smiles", "takes scenery pics", "pays the bill", "practices times tables",
		"prays", "runs to take cover", "joins the chat", "downvotes your post",
		"discovers the moon", "poops", "questions your sense of humor",
		"re-opens the github issue", "talks to its clone", "arrives at the disco",
		"spies the neighbours", "solves the hardest equation", "discusses geopolitics",
		"gets mad at you for crossing the street",
	}
	how := []string {
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
	font, fontName, err := etxt.ParseFontFrom(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Font loaded: %s\n", fontName)

	// create cache
	cache := etxt.NewDefaultCache(1024*1024*1024) // 1GB cache

	// create and configure renderer
	renderer := etxt.NewStdRenderer()
	renderer.SetCacheHandler(cache.NewHandler())
	renderer.SetSizePx(16)
	renderer.SetFont(font)
	renderer.SetAlign(etxt.YCenter, etxt.XCenter)
	renderer.SetColor(color.RGBA{0, 0, 0, 255}) // black

	// generate the random sentences
	rand.Seed(time.Now().UnixNano())
	sentences := make([]string, 2 + rand.Intn(6))
	fmt.Printf("Generating %d sentences...\n", len(sentences))
	for i := 0; i < len(sentences); i++ {
		sentence := who[rand.Intn(len(who))] + " "
		sentence += what[rand.Intn(len(what))] + " "
		sentence += how[rand.Intn(len(how))]
		sentences[i] = sentence
	}
	fullText := strings.Join(sentences, "\n")

	// determine how much space should it take to draw the sentences,
	// plus a bit of vertical and horizontal padding
	rect := renderer.SelectionRect(fullText)
	w, h := rect.Width.Ceil() + 8, rect.Height.Ceil() + 8

	// create target image and fill it with white
	outImage := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h*4; i++ { outImage.Pix[i] = 255 }

	// set target and draw
	renderer.SetTarget(outImage)
	renderer.Draw(fullText, w/2, h/2)

	// store image as png
	filename, err := filepath.Abs("gtxt_rect_size.png")
	if err != nil { log.Fatal(err) }
	fmt.Printf("Output image: %s\n", filename)
	file, err := os.Create(filename)
	if err != nil { log.Fatal(err) }
	err = png.Encode(file, outImage)
	if err != nil { log.Fatal(err) }
	err = file.Close()
	if err != nil { log.Fatal(err) }
	fmt.Print("Program exited successfully.\n")
}
