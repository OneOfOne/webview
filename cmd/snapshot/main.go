package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/OneOfOne/webview"
)

func main() {
	if len(os.Args) < 2 {
		die("usage: snapshot url [snapshot.png]")
	}
	s := webview.DefaultSettings
	s.Offscreen = true // comment this to open an actual window
	s.Width, s.Height = 1920, 1080
	s.WebKit.EnableLocalFileAccess = true
	wv := webview.New("Snapshot Test", &s)
	defer wv.Close()

	wv.LoadURI(os.Args[1])
	img, err := wv.Snapshot()
	if err != nil {
		die("%s", err)
	}
	fn := "snapshot.png"
	if len(os.Args) == 3 {
		fn = os.Args[2]
	}
	f, err := os.Create(fn)
	if err != nil {
		die("%s", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(fn)
		die("%s", err)
	}

}

func die(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
	os.Exit(1)
}
