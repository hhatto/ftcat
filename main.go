package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/antage/eventsource"
	"github.com/codegangsta/cli"
	"github.com/fsnotify/fsnotify"
	scrapbox "github.com/hhatto/go-scrapbox-parser"
	"github.com/hhatto/gorst"
	"github.com/shurcooL/github_flavored_markdown"
	"github.com/skratchdot/open-golang/open"
)

const VERSION string = "0.1"
const indexTemplateHTML = `<!doctype html>
<html>
<head>
  <meta charset='utf-8'/>
  <title>{{.Filename}} - {{.Dirname}}</title>
  <link rel="stylesheet" type="text/css" href="/static/github.min.css" />
  <link rel="stylesheet" type="text/css" href="/static/scrapbox.css" />
  <script>
    function startup() {
        var eventList = document.getElementById("eventList");
        var evtSource = new EventSource("/events");

        // Start listening on the event source
        evtSource.addEventListener("cre", function(e) {
			eventList.removeChild(eventList.childNodes[0]);
            var newElement = document.createElement("div");
			newElement.innerHTML = e.data;
            eventList.appendChild(newElement);
        }, false);
    }
  </script>
</head>
<body onload="startup()">
<div class="container markdown-body">
  <div id="eventList">
  </div>
</div>
</body>
</html>
`

var targetFileName string
var gChan chan string

func getContentString(filename string) (output string, err error) {
	ext := filepath.Ext(filename)
	var input []byte
	if ext == ".md" {
		if input, err = ioutil.ReadFile(targetFileName); err != nil {
			log.Println("indexHandler: ", err)
			return "", err
		}
		o := github_flavored_markdown.Markdown(input)
		outputBuffer := bytes.NewBuffer(o)
		output = outputBuffer.String()
	} else if ext == ".sb" || ext == ".scrapbox" {
		// scrapbox
		p := scrapbox.NewParser()
		input, err := os.Open(filename)
		if err != nil {
			log.Printf("%v", err)
			return "", err
		}
		defer input.Close()
		o := p.ToHTML(input)
		outputBuffer := bytes.NewBuffer(o)
		output = outputBuffer.String()
	} else {
		// rst
		input, err := os.Open(filename)
		if err != nil {
			log.Printf("%v", err)
			return "", err
		}
		defer input.Close()
		var buf bytes.Buffer
		p := rst.NewParser(nil)
		html := rst.ToHTML(&buf)
		p.ReStructuredText(input, html)
		output = buf.String()
	}
	return output, err
}

func fileWatcher(ch chan string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(filepath.Dir(targetFileName))
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Name != targetFileName {
				continue
			}
			if event.Op&fsnotify.Write != fsnotify.Write &&
				event.Op&fsnotify.Create != fsnotify.Create {
				continue
			}
			log.Println("modified file:", event, event.Name)
			output, _ := getContentString(event.Name)
			ch <- output
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.New("index").Parse(indexTemplateHTML)
	if err != nil {
		log.Println("indexHandler: ", err)
		return
	}

	output, err := getContentString(targetFileName)
	if err != nil {
		return
	}
	var indexObj struct {
		Filename string
		Dirname  string
	}
	indexObj.Filename = filepath.Base(targetFileName)
	indexObj.Dirname = filepath.Dir(targetFileName)
	err = t.Execute(w, indexObj)
	if err != nil {
		log.Println("indexHandler: ", err)
	}

	gChan <- output
}

func execCmd(c *cli.Context) {
	if len(c.Args()) < 1 {
		fmt.Println("usage: ftcat FILE")
		return
	}

	ch := make(chan string)
	gChan = make(chan string)
	targetFileName = c.Args()[0]

	/* open webbrowser */
	open.Start("http://0.0.0.0:8089")

	go fileWatcher(ch)

	/* for static files */
	staticFilePath := path.Join(os.Getenv("GOPATH"), "src/github.com/hhatto/ftcat/static")
	fs := http.FileServer(http.Dir(staticFilePath))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	/* index */
	http.HandleFunc("/", indexHandler)

	/* server sent events */
	es := eventsource.New(nil, nil)
	defer es.Close()
	http.Handle("/events", es)

	/* message broker */
	go func() {
		id := 1
		for {
			select {
			case n := <-ch:
				es.SendEventMessage(n, "cre", strconv.Itoa(id))
				id++
			case n := <-gChan:
				time.Sleep(300 * time.Millisecond)
				es.SendEventMessage(n, "cre", strconv.Itoa(id))
				id++
			}
		}
	}()

	log.Fatal(http.ListenAndServe(":8089", nil))
}

func main() {
	app := cli.NewApp()
	app.Name = "baa"
	app.Version = VERSION
	app.Usage = "markdown live previewer"
	app.Action = execCmd

	app.Run(os.Args)
}
