package webview

import (
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
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
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	go func() {
		http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
			os.Exit(0)
		})
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			u, _ := user.Current()
			tmpl.Execute(w, u)
		})
		log.Fatal(http.Serve(ln, nil))
	}()

	wv0 := New("Hello webkit2gtk", nil)
	t.Logf("loaded: %s", wv0.LoadURI("http://"+ln.Addr().String()))

	s := DefaultSettings
	s.Decorated, s.Fullscreen = false, true
	wv1 := New("Spinner", &s)

	wv1.LoadHTML(LoadingDoc)

	wv1.RunJavaScript(`document.body.id`, func(v JSValue, err error) {
		if err != nil {
			t.Errorf("js error: %v", err)
			return
		}
		t.Logf("js value (type=%s): %s", v.Type(), v.AsString())
	})
	wv1.RunJavaScript("document.body.requestFullscreen();", nil)
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
