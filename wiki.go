package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
)

var (
	addr = flag.Bool("addr", false, "find open address and print to final-port.txt")
)

type Page struct {
	Title    string
	Body     []byte
	HtmlBody template.HTML
}

var templates = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/view.html"))
var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var linksPattern = regexp.MustCompile("\\[[a-zA-Z0-9]+\\]")

func (p *Page) ParseWiki() []byte {
	return linksPattern.ReplaceAllFunc(p.Body, replaceWikiLinks)
}

func replaceWikiLinks(src []byte) []byte {
	stringSrc := string(src)
	linkName := stringSrc[1 : len(stringSrc)-1]
	return []byte("<a href='/view/" + linkName + "'>" + linkName + "</a>")
}

func getFilePath(title string) string {
	return "data/" + title + ".txt"
}

func (p *Page) save() error {
	return ioutil.WriteFile(getFilePath(p.Title), p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	body, err := ioutil.ReadFile(getFilePath(title))
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	if tmpl != "edit" {
		p.HtmlBody = template.HTML(p.ParseWiki())
	}
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
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.Handle("/", http.RedirectHandler("/view/FrontPage", 301))

	if *addr {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile("final-port.txt", []byte(l.Addr().String()), 0644)
		if err != nil {
			log.Fatal(err)
		}
		s := &http.Server{}
		s.Serve(l)
		return
	}

	http.ListenAndServe(":8080", nil)
}
