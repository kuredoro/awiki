package main

import (
    "os"
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


var templates = template.Must(template.ParseFiles("tmpl/front.html", "tmpl/view.html"))
var nonIDchars = regexp.MustCompile("[^a-zA-Z0-9-]+")
var persistentStoragePath = "data"

type ToCEntry struct {
    redirTo string
    title string
    level int
    sub []*ToCEntry
}

func GenMdHeader(raw ToCEntry) (title, redirTo string) {
    // This is approximate id repr that md render outputs
    title = raw.title
    redirTo = strings.ToLower(title)
    redirTo = strings.ReplaceAll(redirTo, " ", "-")
    redirTo = nonIDchars.ReplaceAllString(redirTo, "")
    redirTo = "#" + redirTo
    return
}

type ToCSettings struct {
    itemStart, itemEnd string
    listStart, listEnd string
}

type FrontPage struct {
    Index template.HTML
}

type Page struct {
	Title        string
	Body         []byte
	RenderedBody template.HTML
    AsideToC     template.HTML
}

func (p *Page) save() error {
	filename := persistentStoragePath + "/" + p.Title + ".txt"
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

func extractHeaders(md []byte, strGen func(ToCEntry) (string, string)) []ToCEntry {

    var heads []ToCEntry
    idCounter := make(map[string]int)
    minLevel := math.MaxInt32

    scanner := bufio.NewScanner(bytes.NewReader(md))
    for scanner.Scan() {
        line := scanner.Text()
        line = strings.TrimSpace(line)

        var head ToCEntry
        for _, r := range line {
            if r != '#' {
                break
            }

            head.level++
        }

        if head.level == 0 {
            continue
        }

        head.title = strings.TrimSpace(line[head.level:])
        head.title, head.redirTo = strGen(head)

        dupId := idCounter[head.redirTo]
        idCounter[head.redirTo]++
        if dupId != 0 {
            head.redirTo += "-" + fmt.Sprint(dupId)
        }

        heads = append(heads, head)
        
        if minLevel > head.level {
            minLevel = head.level
        }
    }

    return heads
}

func inflateToC(entries []ToCEntry) *ToCEntry {

    var toc ToCEntry
    s := make([]*ToCEntry, 1, 7)
    s[0] = &toc

    for _, h := range entries {
        sLast := len(s)-1
        for ; sLast >= 0; sLast-- {
            if s[sLast].level < h.level {
                break
            }
        }
        s = s[:sLast+1]

        entry := &ToCEntry{
            redirTo: h.redirTo,
            title: h.title,
            level: h.level,
        }

        s[sLast].sub = append(s[sLast].sub, entry)
        s = append(s, entry)
    }

    return &toc
}

func (p *Page) generateToC() {

    headers := extractHeaders(p.Body, GenMdHeader)
    toc := inflateToC(headers)

    if len(toc.sub) == 0 {
        p.AsideToC = template.HTML("")
        return
    }

    asideToC := ToCSettings{
        itemStart: "<li>",
        itemEnd:   "</li>",
        listStart: "<ol>",
        listEnd:   "</ol>",
    }

    var str strings.Builder
    str.WriteString("<aside id=\"toc-aside\" class=\"\">\n<h2>Table of Contents</h2>\n")
    asideToC.marshall(&str, toc.sub)
    str.WriteString("</aside>\n")

    p.AsideToC = template.HTML(str.String())
}


func (s *ToCSettings) marshall(str *strings.Builder, x interface{}) {

    switch val := x.(type) {
    case *ToCEntry:
        str.WriteString(s.itemStart)
        str.WriteString(fmt.Sprintf("\n<a href=\"%s\">%s</a>\n", val.redirTo, val.title))

        if len(val.sub) > 0 {
            s.marshall(str, val.sub)
        }

        str.WriteString(s.itemEnd)
        str.WriteRune('\n')
        
    case []*ToCEntry:
        str.WriteString(s.listStart)
        str.WriteRune('\n')
        for _, p := range val {
            s.marshall(str, p)
        }
        str.WriteString(s.listEnd)
        str.WriteRune('\n')
    }
}


func loadPage(path string) (*Page, error) {
	filename := persistentStoragePath + "/" + path + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

    for slashPos := len(path)-1; slashPos >= 0; slashPos-- {
        if path[slashPos] == '/' {
            path = path[slashPos+1:]
            break
        }
    }

	return &Page{Title: path, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
    title := strings.TrimPrefix(r.URL.Path, "/")

	p, err := loadPage(title)
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p.renderMarkup()
    p.generateToC()
	renderTemplate(w, "view", p)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Path) != 1 {
        viewHandler(w, r)
        return
	}

    headers, err := treeDir(persistentStoragePath, 0)
    if err != nil {
        http.Error(w, fmt.Sprintf("could not generate index for pages, %v", err), http.StatusInternalServerError)
    }

    if len(headers) > 0 {
        headers[0].level = 1
    }

    var page FrontPage
    page.Index = template.HTML(renderIndex(headers))

    err = templates.ExecuteTemplate(w, "front.html", page)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

func treeDir(path string, level int) ([]*ToCEntry, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("couldn't open path %q", path)
    }

    fileInfo, err := f.Readdir(-1)
    if err != nil {
        return nil, fmt.Errorf("couldn't list directory %q", path)
    }
    f.Close()

    var dirName string
    for slashPos := len(path)-1; slashPos >= 0; slashPos-- {
        if path[slashPos] == '/' {
            dirName = path[slashPos+1:]
            break
        }
    }

    locDir := strings.TrimPrefix(path, persistentStoragePath)

    var dirs []string
    var heads []*ToCEntry
    heads = append(heads, &ToCEntry{title: dirName, level: level})

    for _, file := range fileInfo {
        if file.IsDir() {
            dirs = append(dirs, file.Name())
            continue
        }

        entry := ToCEntry{level: 0}
        entry.title = strings.TrimSuffix(file.Name(), ".txt")
        entry.redirTo = locDir + "/" + entry.title
        heads = append(heads, &entry)
    }

    for _, dir := range dirs {
        subHeads, err := treeDir(path + "/" + dir, level+1)
        if err != nil {
            return nil, err // Info is already incoded in the path passed to tree
        }

        heads = append(heads, subHeads...)
    }

    return heads, nil
}

func renderIndex(heads []*ToCEntry) string {
    var str strings.Builder

    for _, head := range heads {
        if 0 < head.level && head.level < 7 {
            str.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", head.level+1, head.title, head.level+1))
        }

        if head.level == 0 {
            str.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a><br />\n", head.redirTo, head.title))
        }
    }

    return str.String()
}

func main() {
	http.HandleFunc("/", rootHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    log.Printf("Deployed at http://localhost%s\n", deployPort)
	log.Fatal(http.ListenAndServe(deployPort, nil))
}
