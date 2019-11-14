package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type Link struct {
	URL  string
	Time time.Time
}

func ParseLink(yyyymm string) *Link {
	t, err := time.Parse("200601", yyyymm)
	if err != nil {
		panic(err.Error())
	}
	return &Link{
		URL:  fmt.Sprintf("%s.html", yyyymm),
		Time: t,
	}
}

type Image struct {
	Filename, SmallFile string
	Size, SmallSize     int64
}

func NewImage(path, filename string) Image {
	fi, err := os.Stat(filepath.Join(path, filename))
	if err != nil {
		log.Fatal(err)
	}
	small := strings.Replace(filename, ".gif", "_sm.gif", 1)
	sfi, err := os.Stat(filepath.Join(path, small))
	if err != nil {
		log.Fatal(err)
	}
	return Image{
		Filename:  filename,
		SmallFile: small,
		Size:      fi.Size() / 1024,
		SmallSize: sfi.Size()/ 1024,
	}
}

type Page struct {
	Title    string
	Images   []Image
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
		{{range .Images}}
			<div class="image-block">
				<a href="{{.SmallFile}}"><img src="{{ .SmallFile }}" alt="{{.SmallFile}}"></a><br/>
				<span class="alt"><a href="{{.Filename}}">{{.Filename}}</a> size:{{.Size}}k</span><br/>
				<span class="alt"><a href="{{.SmallFile}}">{{.SmallFile}}</a> size:{{.SmallSize}}k</span><br/>
			</div>
		{{end}}
		<p>
			{{if .Previous }}<a href="{{.Previous.URL}}"><button>&larr; {{.Previous.Time.Format "Jan 2006"}}</button></a>{{end}}
			{{if .Next }}<a href="{{.Next.URL}}"><button>{{.Next.Time.Format "Jan 2006"}} &rarr;</button></a>{{end}}
		</p>
	</body>
</html>`

func existingFiles(dir string) (map[string]bool, error) {
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	files, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	found := make(map[string]bool)
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".gif") {
			continue
		}
		hash, err := fileHash(filepath.Join(dir, filename))
		if err != nil {
			return nil, err
		}
		found[hash] = true
	}
	return found, nil
}

// return the sha1 of the bytes of a file
func fileHash(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func NewGifName(sourcefile string) (string, error) {
	fi, err := os.Stat(sourcefile)
	if err != nil {
		return "", err
	}
	modifyTime := fi.ModTime()
	stat := fi.Sys().(*syscall.Stat_t)
	ctime := time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
	if modifyTime.Before(ctime) {
		ctime = modifyTime
	}
	return ctime.Format("20060102_150405_") + randStr(6) + ".gif", nil
}

var cleaner = strings.NewReplacer("-", "", "_", "")

func randStr(size int) string {
	var randomness string
	for i := 0; i < 5; i++ {
		b := make([]byte, size)
		_, err := rand.Read(b)
		if err != nil {
			log.Panicf("ERROR: failed to read %d random bytes for shortid: %s", size, err)
		}
		randomness = cleaner.Replace(base64.RawURLEncoding.EncodeToString(b))
		if len(randomness) >= 6 {
			randomness = randomness[:6]
		}
		if len(randomness) == 6 {
			break
		}
	}
	if len(randomness) != 6 {
		log.Panic("ERROR: failed generating shortid")
	}
	return randomness
}

// Copy the src file to dst. Any existing file will be overwritten and will not
// copy file attributes.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func finder(newGifs chan <- string, existingHashes map[string]bool) {
	defer close(newGifs)
	// find Gifs
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	
	for _, baseDir := range []string {
		filepath.Join(u.HomeDir, "Library/Messages/Attachments"),
		"/private/var/folders",
		} {
			filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("%s",err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			log.Printf("%s", path)
			filename := filepath.Base(path)
			ok, _ := filepath.Match("output*.GIF", filename)
			if !ok {
				ok, _ = filepath.Match("Motion-Still*.gif", filename)
			}
			if !ok {
				ok, _ = filepath.Match("Motion-Still*.GIF", filename)
			}
			if ok {
				hash, err := fileHash(path)
				if err != nil {
					return err
				}
				if existingHashes[hash] {
					log.Printf("Already have %s - %s", path, hash)
					return nil
				}
				newGifs <- path
			}
			return nil
		})
	}
}

func main() {
	targetDir := flag.String("dir", ".", "target directory")
	flag.Parse()

	if *targetDir == "" {
		log.Fatal("missing --dir")
	}

	t, err := template.New("webpage").Parse(tpl)
	if err != nil {
		log.Fatal(err)
	}

	existingHashes, err := existingFiles(*targetDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%d existing *.gif files in %s", len(existingHashes), *targetDir)

	
	newGifs := make(chan string)
	go finder(newGifs, existingHashes)
	
	// move new files
	found := 0
	for newGif := range newGifs {
		found++
		newGifFilename, err := NewGifName(newGif)
		if err != nil {
			log.Fatal(err)
		}
		newGifFilename = filepath.Join(*targetDir, newGifFilename)
		log.Printf("copying %s to %s", newGif, newGifFilename)
		err = Copy(newGif, newGifFilename)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("found %d new files", found)

	d, err := os.Open(*targetDir)
	if err != nil {
		log.Fatal(err)
	}
	files, err := d.Readdirnames(-1)
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)

	// generate missing optimized gif's using gifsicle
	// https://github.com/kornelski/giflossy
	// https://kornel.ski/lossygif
	allFiles := make(map[string]bool)
	for _, filename := range files {
		allFiles[filename] = true
	}
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".gif") {
			continue
		}
		if strings.HasSuffix(filename, "_sm.gif") {
			continue
		}
		small := strings.Replace(filename, ".gif", "_sm.gif", 1)
		if allFiles[small] {
			continue
		}
		log.Printf("optimizing %s", filename)
		cmd := exec.Command("gifsicle", "-O3", "--lossy=30", "-o", filepath.Join(*targetDir, small), filepath.Join(*targetDir, filename))
		cmd.Stderr = ioutil.Discard
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}

	data := make(map[string][]Image)
	var grouped []string
	for _, filename := range files {
		if !strings.HasSuffix(filename, ".gif") || strings.HasSuffix(filename, "_sm.gif") {
			continue
		}
		yearmonth := filename[:6]
		if _, ok := data[yearmonth]; !ok {
			grouped = append(grouped, yearmonth)
		}
		data[yearmonth] = append(data[yearmonth], NewImage(*targetDir, filename))
	}

	for i, yyyymm := range grouped {
		ts, err := time.Parse("200601", yyyymm)
		if err != nil {
			log.Fatal(err)
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
		of, err := os.Create(filepath.Join(*targetDir, htmlname))
		if err != nil {
			log.Fatal(err)
		}
		err = t.Execute(of, page)
		if err != nil {
			log.Fatal(err)
		}
		of.Close()
		if i == len(grouped)-1 {
			log.Printf("symlinking index.html to %s", htmlname)
			os.Remove(filepath.Join(*targetDir, "index.html"))
			err = os.Symlink(htmlname, filepath.Join(*targetDir, "index.html"))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
