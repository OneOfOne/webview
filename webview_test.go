package gtkwebview

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
	StartGUI()
	defer DestoryGUI()
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

	wv0 := New("Test UI 0", nil)
	wv1 := New("Test UI 1", nil)

	wv0.LoadURI("http://google.com/ncr")

	wv1.LoadHTML(LoadingDoc)

	select {
	case <-wv0.Done():
	}
	select {
	case <-wv1.Done():
	}
	// select {
	// case <-wv1.Done():
	// }
	// select {
	// case <-wv1.Done():
	// }
}

const LoadingDoc = `
<html>

<head>
	<style>
	div {
		border: 16px solid #f3f3f3;
		border-top: 16px solid #3498db;
		border-radius: 50%;
		width: 120px;
		height: 120px;
		animation: spin 2s linear infinite;
		margin: 0 auto;
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
<body>
	<div></div>
</body>
</html>
`
