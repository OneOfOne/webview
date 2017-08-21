package main

import (
	"log"

	"github.com/OneOfOne/webview"
)

func main() {
	s := webview.DefaultSettings
	s.Offscreen = true
	wv := webview.New("Test UI 0", &s)
	wv.OnPageLoad = func(_ string) {
		wv.RunJavaScript(`const vs = ["test"]; for(const v of vs) console.log(v); vs[0];`, func(v webview.JSValue, err error) {
			log.Printf("%s: %v", v.Type(), v.AsString())
			wv.Close()
		})
	}
	wv.LoadHTML("<html><body>hi</body></html>")
	<-wv.Done()
}
