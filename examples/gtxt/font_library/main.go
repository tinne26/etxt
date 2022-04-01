package main

import "os"
import "path/filepath"
import "log"
import "fmt"

import "github.com/tinne26/etxt"

func main() {
	// get font directory path
	if len(os.Args) != 2 {
		msg := "Usage: expects one argument with the path to the font directory\n"
		fmt.Fprint(os.Stderr, msg)
		os.Exit(1)
	}

	// print given font directory
	fontDir, err := filepath.Abs(os.Args[1])
	if err != nil { log.Fatal(err) }
	fmt.Printf("Reading font directory: %s\n", fontDir)

	// create font library
	fontLib := etxt.NewFontLibrary()
	added, skipped, err := fontLib.ParseDirFonts(fontDir)
	if err != nil {
		log.Fatalf("Added %d fonts, skipped %d, failed with '%s'", added, skipped, err.Error())
	}
	fmt.Printf("Added %d fonts, skipped %d\n", added, skipped)

	// print, for each font parsed, its name, family and subfamily
	err = fontLib.EachFont(
		func(fontName string, font *etxt.Font) error {
			family, err := etxt.FontFamily(font)
			if err != nil {
				log.Printf("(failed to load family for font %s: %s)", fontName, err.Error())
				family = "unknown"
			}
			subfamily, err := etxt.FontSubfamily(font)
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
