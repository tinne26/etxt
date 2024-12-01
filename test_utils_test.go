package etxt

import "os"
import "fmt"
import "image"
import "image/png"

func doesNotPanic(function func()) (didNotPanic bool) {
	didNotPanic = true
	defer func() { didNotPanic = (recover() == nil) }()
	function()
	return
}

func debugExport(name string, img image.Image) {
	file, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = png.Encode(file, img)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = file.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
