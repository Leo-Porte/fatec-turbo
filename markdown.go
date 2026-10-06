package main

import (
	"html"
	"html/template"
	"regexp"
	"strings"
)

var (
	reNegrito = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reCodigo  = regexp.MustCompile("`([^`]+)`")
	reLista   = regexp.MustCompile(`^\s*- `)
)

// MarkdownInline: **negrito** e `fórmula`, com o resto escapado.
func MarkdownInline(s string) template.HTML {
	t := html.EscapeString(s)
	t = reNegrito.ReplaceAllString(t, "<strong>$1</strong>")
	t = reCodigo.ReplaceAllString(t, "<code>$1</code>")
	return template.HTML(t)
}

// Markdown mínimo das lições: parágrafos (linha em branco), listas "- ", **negrito**, `fórmula`.
func Markdown(s string) template.HTML {
	var out strings.Builder
	for _, bloco := range regexp.MustCompile(`\n{2,}`).Split(strings.ReplaceAll(s, "\r", ""), -1) {
		bloco = strings.Trim(bloco, "\n")
		if bloco == "" {
			continue
		}
		var par []string
		var lista []string
		flushPar := func() {
			if len(par) > 0 {
				out.WriteString("<p>" + strings.Join(par, "<br>") + "</p>")
				par = nil
			}
		}
		flushLista := func() {
			if len(lista) > 0 {
				out.WriteString("<ul><li>" + strings.Join(lista, "</li><li>") + "</li></ul>")
				lista = nil
			}
		}
		for _, l := range strings.Split(bloco, "\n") {
			if reLista.MatchString(l) {
				flushPar()
				lista = append(lista, string(MarkdownInline(reLista.ReplaceAllString(l, ""))))
			} else {
				flushLista()
				par = append(par, string(MarkdownInline(l)))
			}
		}
		flushPar()
		flushLista()
	}
	return template.HTML(out.String())
}
