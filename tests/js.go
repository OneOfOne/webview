package main

import (
	"log"

	"github.com/OneOfOne/webview"
)

func main() {
	s := webview.DefaultSettings
	s.Offscreen = true // comment this to open an actual window
	s.WebKit.EnableLocalFileAccess = true
	wv := webview.New("Test UI 0", &s)

	wv.LoadHTML("<html><body>hi</body></html>")
	v := wv.RunJS(`const vs = ["test"]; for(const v of vs) console.log(v); vs[0];`)
	if v.Err() != nil {
		panic(v.Err())
	}
	log.Printf("%s: %v", v.Type(), v.AsString())
	wv.Close()
	<-wv.Done()
}
