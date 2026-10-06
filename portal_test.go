package main

import (
	"math"
	"strings"
	"testing"
)

func TestParseExpr(t *testing.T) {
	casos := []struct {
		expr string
		x, y float64
	}{
		{"x^2 - 2*x", 3, 3},
		{"2x + 1", 2, 5},
		{"-x^2", 3, -9},
		{"sqrt(x)", 9, 3},
		{"3(x+1)", 1, 6},
		{"x/4", 2, 0.5},
		{"2^x", 3, 8},
		{"1,5x", 2, 3},
	}
	for _, c := range casos {
		f, err := ParseExpr(c.expr)
		if err != nil {
			t.Fatalf("%q: %v", c.expr, err)
		}
		if got := f(c.x); math.Abs(got-c.y) > 1e-9 {
			t.Errorf("%q em x=%v: got %v, want %v", c.expr, c.x, got, c.y)
		}
	}
	if _, err := ParseExpr("x +"); err == nil {
		t.Error("esperava erro para expressão incompleta")
	}
}

// O celular insere aspas curvas, travessão e reticências; emoji pode aparecer. Nada disso pode derrubar o PDF.
func TestPDFRedacaoCaracteresEspeciais(t *testing.T) {
	texto := "“Futebol” – paixão… nacional 😀\nSegunda linha com ç, ã, é e uma palavraenormesemespacoquepassadalarguradalinhaporquealguemdigitoutudojuntoassimmesmo."
	b, n, err := PDFRedacao("2026-2", "Tema", "Título “curvo”", texto)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "%PDF") {
		t.Fatal("não gerou PDF")
	}
	if n < 3 {
		t.Errorf("esperava pelo menos 3 linhas, veio %d", n)
	}
}

func TestRenderFigura(t *testing.T) {
	geo := &Figura{Lib: "jsxgraph", BBox: []float64{-1, 5, 5, -1}, Eixos: true, Elementos: []Elemento{
		{T: "ponto", ID: "A", XY: []float64{0, 0}, Rotulo: "A"},
		{T: "ponto", ID: "B", XY: []float64{4, 0}, Rotulo: "B"},
		{T: "ponto", ID: "C", XY: []float64{0, 3}, Rotulo: "C"},
		{T: "poligono", Pts: []string{"A", "B", "C"}},
		{T: "segmento", De: "B", Ate: "C", Rotulo: "5"},
		{T: "angulo", Pts: []string{"B", "A", "C"}, Rotulo: "90°"},
		{T: "funcao", Expr: "x^2/4", Dominio: []float64{0, 4}},
	}}
	s := string(RenderFigura(geo))
	for _, want := range []string{"<svg", "<polygon", "<path", ">90°<"} {
		if !strings.Contains(s, want) {
			t.Errorf("figura geométrica sem %q", want)
		}
	}
	bar := &Figura{Lib: "chart", Tipo: "bar", Rotulos: []string{"a", "b"}, Series: []Serie{{Nome: "s", Dados: []float64{3, 5}}}}
	if !strings.Contains(string(RenderFigura(bar)), "<rect") {
		t.Error("gráfico de barras sem retângulos")
	}
	if RenderFigura(&Figura{Lib: "chart"}) != "" {
		t.Error("gráfico sem séries deveria sair vazio")
	}
}
