package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

const deployPort = ":8080"


var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))
var validTitle = regexp.MustCompile("^/(view|edit|save)/([^/\\.]+)$")
var nonIDchars = regexp.MustCompile("[^a-zA-Z0-9-]+")
var persistentStoragePath = "data/"

type MarkupHeader struct {
    id string
    title string
    level int
}

type ToCEntry struct {
    id string
    title string
    level int
    sub []*ToCEntry
}

type Page struct {
	Title        string
	Body         []byte
	RenderedBody template.HTML
    AsideToC     template.HTML
}

func (p *Page) save() error {
	filename := persistentStoragePath + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func (p *Page) renderMarkup() {
	md := goldmark.New(
		goldmark.WithExtensions(mathjax.MathJax),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	var html bytes.Buffer

	if err := md.Convert(p.Body, &html); err != nil {
		fmt.Print(err)
	}

	sanitized := bluemonday.UGCPolicy().SanitizeBytes(html.Bytes())
	p.RenderedBody = template.HTML(string(sanitized))
}

func (p *Page) generateToC() {
    var heads []MarkupHeader
    idCounter := make(map[string]int)
    minLevel := math.MaxInt32

    scanner := bufio.NewScanner(bytes.NewReader(p.Body))
    for scanner.Scan() {
        line := scanner.Text()
        line = strings.TrimSpace(line)

        var head MarkupHeader
        for _, r := range line {
            if r != '#' {
                break
            }

            head.level++
        }

        if head.level == 0 {
            continue
        }

        // This is approximate id repr that md render outputs
        head.title = strings.TrimSpace(line[head.level:])
        head.id = strings.ToLower(head.title)
        head.id = strings.ReplaceAll(head.id, " ", "-")
        head.id = nonIDchars.ReplaceAllString(head.id, "")

        dupId := idCounter[head.id]
        idCounter[head.id]++
        if dupId != 0 {
            head.id += "-" + fmt.Sprint(dupId)
        }

        heads = append(heads, head)
        
        if minLevel > head.level {
            minLevel = head.level
        }

        log.Printf("h%d id=%q", head.level, head.id)
    }

    var toc ToCEntry
    s := make([]*ToCEntry, 1, 7)
    s[0] = &toc

    for _, h := range heads {
        sLast := len(s)-1
        for ; sLast >= 0; sLast-- {
            if s[sLast].level < h.level {
                break
            }
        }
        s = s[:sLast+1]

        entry := &ToCEntry{
            id: h.id,
            title: h.title,
            level: h.level,
        }

        s[sLast].sub = append(s[sLast].sub, entry)
        s = append(s, entry)
    }

    var str strings.Builder
    marshallToC(&str, toc.sub)

    p.AsideToC = template.HTML(str.String())
}


func marshallToC(str *strings.Builder, x interface{}) {

    switch val := x.(type) {
    case *ToCEntry:
        str.WriteString(fmt.Sprintf("<li>\n<a href=\"#%s\">%s</a>\n", val.id, val.title))

        if len(val.sub) > 0 {
            marshallToC(str, val.sub)
        }

        str.WriteString("</li>\n")
        
    case []*ToCEntry:
        str.WriteString("<ol>\n")
        for _, p := range val {
            marshallToC(str, p)
        }
        str.WriteString("</ol>\n")

    default:
        log.Printf("got unknown type %T", val)
    }
}


func loadPage(title string) (*Page, error) {
	filename := persistentStoragePath + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.renderMarkup()
    p.generateToC()
	renderTemplate(w, "view", p)
}

/*
func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)

	if err != nil {
		p = &Page{Title: title}
	}

	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}
*/

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) == 1 {
		http.Redirect(w, r, "/view/FrontPage", http.StatusFound)
	}
}

func makeHandler(fn func(w http.ResponseWriter, r *http.Request, title string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		match := validTitle.FindStringSubmatch(r.URL.Path)
		if match == nil {
			http.NotFound(w, r)
			return
		}

		fn(w, r, match[2])
	}
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/view/", makeHandler(viewHandler))
	//http.HandleFunc("/edit/", makeHandler(editHandler))
	//http.HandleFunc("/save/", makeHandler(saveHandler))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    log.Printf("Deployed at http://localhost%s\n", deployPort)
	log.Fatal(http.ListenAndServe(deployPort, nil))
}
