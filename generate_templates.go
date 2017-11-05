package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

type Link struct {
	URL string
	Time time.Time
}

func ParseLink(yyyymm string) *Link {
	t, err := time.Parse("200601", yyyymm)
	if err != nil {
		panic(err.Error())
	}
	return &Link{
		URL: fmt.Sprintf("%s.html", yyyymm),
		Time: t,
	}
}

type Page struct {
	Title    string
	Images   []string
	Previous *Link
	Next     *Link
}


const tpl = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
		<meta name="viewport" content="initial-scale=1.0, width=480">
		<style>
			.image-block {
				margin-top:20px;
			}
			.image-block > img {
				margin:0 auto;
			}
			button {
				font-size:20px;
				min-width:100px;
			    font-family: Helvetica;
			}
			h1 {
				display: inline;
				font-family: Helvetica;
			    font-size: 24px;
			    padding: 0 1em;				
			}
			.alt {
				font-size:10px;
				color:#666;
			}
		</style>
	</head>
	<body>
		
		<div>
			{{if .Previous }}<a href="{{.Previous.URL}}"><button>&larr; {{.Previous.Time.Format "Jan 2006"}}</button></a>{{end}}
			<h1>{{.Title}}</h1>
			{{if .Next }}<a href="{{.Next.URL}}"><button>{{.Next.Time.Format "Jan 2006"}} &rarr;</button></a>{{end}}
		</div>
		{{range .Images}}<div class="image-block"><a href="{{.}}"><img src="{{ . }}" alt="{{.}}"></a><br/><span class="alt">{{.}}</span></div>{{end}}
		<p>
			{{if .Previous }}<a href="{{.Previous.URL}}"><button>&larr; {{.Previous.Time.Format "Jan 2006"}}</button></a>{{end}}
			{{if .Next }}<a href="{{.Next.URL}}"><button>{{.Next.Time.Format "Jan 2006"}} &rarr;</button></a>{{end}}
		</p>
	</body>
</html>`

func main() {
	flag.Parse()

	t, err := template.New("webpage").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	d, err := os.Open(".")
	if err != nil {
		log.Fatal(err)
	}
	files, err := d.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)
	data := make(map[string][]string)
	var grouped []string
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".gif") {
			continue
		}
		yearmonth := filename[:6]
		if _, ok := data[yearmonth]; !ok {
			grouped = append(grouped, yearmonth)
		}
		data[yearmonth] = append(data[yearmonth], filename)
	}

	for i, yyyymm := range grouped {
		ts, err := time.Parse("200601", yyyymm)
		if err != nil {
			panic(err.Error())
		}
		htmlname := fmt.Sprintf("%s.html", yyyymm)
		page := Page{
			Title:  ts.Format("Jan 2006"),
			Images: data[yyyymm],
		}
		if i != 0 {
			page.Previous = ParseLink(grouped[i-1])
		}
		if i < len(grouped)-1 {
			page.Next = ParseLink(grouped[i+1])
		}
		log.Printf("creating %s for %d images", htmlname, len(page.Images))
		of, err := os.Create(htmlname)
		if err != nil {
			log.Fatal(err)
		}
		err = t.Execute(of, page)
		if err != nil {
			log.Fatal(err)
		}
		of.Close()
		if i == len(grouped)-1 {
			os.Symlink(htmlname, "index.html")
		}
	}
}
