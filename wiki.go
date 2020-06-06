package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type Page struct {
	Title        string
	Body         []byte
	RenderedBody template.HTML
}

var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))
var validTitle = regexp.MustCompile("^/(view|edit|save)/([^/\\.]+)$")
var persistentStoragePath = "data/"

func (p *Page) save() error {
	filename := persistentStoragePath + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func processMath(bytes []byte) []byte {
	text := string(bytes)

	// Do display math first
	dmath := ""
	sections := strings.Split(text, "$$")

	for i := 0; i < len(sections); i++ {
		// The first element is always plain text
		if i%2 == 0 {
			dmath += sections[i]
		} else {
			dmath += "<span class=\"math display\">\\[" + sections[i] + "\\]</span>"
		}
	}

	return []byte(dmath)
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

	//withMath := processMath(p.Body)
	sanitized := bluemonday.UGCPolicy().SanitizeBytes(html.Bytes())
	p.RenderedBody = template.HTML(string(sanitized))
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
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	p.renderMarkup()
	renderTemplate(w, "view", p)
}

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
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))

	/*
		p1 := &Page{Title: "Example", Body: []byte("This is an example page.")}
		p1.save()
		p2, _ := loadPage("Example")
		fmt.Println(string(p2.Body))
	*/
}
