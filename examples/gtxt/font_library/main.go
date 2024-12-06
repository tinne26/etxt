package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/tinne26/etxt/font"
	"golang.org/x/image/font/sfnt"
)

// Must be compiled with '-tags gtxt'.
// This example expects a path to a font directory as the first
// argument, reads the fonts in it and prints their names to the
// terminal.

func main() {
	// get font directory path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font directory\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// print given font directory
	fontDir, err := filepath.Abs(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Reading font directory: %s\n", fontDir)

	// create font library
	fontLib := font.NewLibrary()
	added, skipped, err := fontLib.ParseAllFromPath(fontDir)
	if err != nil {
		log.Fatalf("Added %d fonts, skipped %d, failed with '%s'", added, skipped, err.Error())
	}
	fmt.Printf("Added %d fonts, skipped %d\n", added, skipped)

	// print, for each font parsed, its name, family and subfamily
	err = fontLib.EachFont(
		func(fontName string, sfntFont *sfnt.Font) error {
			family, err := font.GetFamily(sfntFont)
			if err != nil {
				log.Printf("(failed to load family for font %s: %s)", fontName, err.Error())
				family = "unknown"
			}
			subfamily, err := font.GetSubfamily(sfntFont)
			if err != nil {
				log.Printf("(failed to load subfamily for font %s: %s)", fontName, err.Error())
				subfamily = "unknown"
			}
			fmt.Printf("* %s (%s | %s)\n", fontName, family, subfamily)
			return nil
		})
	if err != nil {
		log.Fatal("FontLibrary.EachFont error!: " + err.Error())
	}
	fmt.Print("Program exited successfully.\n")
}
