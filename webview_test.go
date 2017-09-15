package webview

import (
	"bytes"
	"html/template"
	"image/png"
	"log"
	"net"
	"net/http"
	"os/user"
	"testing"
)

var tmpl = template.Must(template.New("").Parse(`
<html>
	<head>
		<style>
		* { margin: 0; padding: 0; box-sizing: border-box; font-family: Helvetica, Arial, sans-serif; }
		body { color: #ffffff; background-color: #03a9f4; text-decoration: uppercase; font-size: 24px; }
		h1 { text-align: center; font-weight: normal}
		form { margin-left: auto; margin-right: auto; margin-top: 50px; width: 300px; }
		input[type="submit"] {
				border: 0 none;
				cursor: pointer;
				margin-top: 1em;
				background-color: #ffffff;
				color: #03a9f4;
				width: 100%;
				height: 2em;
				font-size: 24px;
				text-transform: uppercase;
		}
		</style>
	</head>
	<body>
	  <form action="/exit">
			<h1>Hello, {{ .Name }}!</h1>
			<input type="submit" value="Exit" />
		</form>
	</body>
</html>
`))

func init() {
	log.SetFlags(log.Lshortfile)
}

func TestUI(t *testing.T) {
	Debug = testing.Verbose()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	wv0 := New("Hello webkit2gtk", nil)

	go func() {
		http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
			wv0.Close()
		})
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			u, _ := user.Current()
			tmpl.Execute(w, u)
		})
		log.Fatal(http.Serve(ln, nil))
	}()

	t.Logf("loaded: %s", wv0.LoadURI("http://"+ln.Addr().String()))

	s := DefaultSettings
	//s.Decorated, s.Fullscreen = false, true
	wv1 := New("Spinner", &s)

	wv1.LoadHTML(LoadingDoc)

	v := wv1.RunJS(`const o = {v: [1,2,3], u: "s"}; o`)
	if err = v.Err(); err != nil {
		t.Errorf("js error: %v", err)
		// return
	}
	var data interface{}
	log.Println(v.AsObject(&data))
	t.Logf("js value (type=%s): %+v %s", v.Type(), data, v.AsJSON())

	wv1.OnMessage = func(js *JSValue, cb func(reply interface{}) *JSValue) {
		log.Printf("message from js: %s", js.val)
		if cb == nil {
			return
		}
		v := cb("hello from go")
		log.Printf("omg reply from javascript! %s", v.val)
	}
	wv1.RunJS(`postSystemMessage("hello from js", function(v) { console.log('js got:', v); return {reply_back_from_js: "woooo"}; });`)
	img, err := wv1.Snapshot()
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err = png.Encode(&buf, img); err != nil {
		t.Error(err)
		return
	}

	dimg, err := png.Decode(&buf)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%#+v", dimg.Bounds())
	<-wv0.Done()
	<-wv1.Done()
}

const LoadingDoc = `
<html>

<head>
	<style>
	body {
		background: #ccc;
	}
	div {
		border: 30px solid #f3f3f3;
		border-top: 30px solid #3498db;
		border-radius: 50%;
		width: 200px;
		height: 200px;
		animation: spin 2s linear infinite;
	}

	.middle {
		position: absolute;
		top:0;
		bottom: 0;
		left: 0;
		right: 0;

		margin: auto;
	}
	@keyframes spin {
		0% {
			transform: rotate(0deg);
		}
		100% {
			transform: rotate(360deg);
		}
	}
	</style>
</head>
<body id="this-is-the-body">
	<div class="middle"></div>
</body>
</html>
`
