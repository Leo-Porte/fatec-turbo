package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"io/fs"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

const (
	pdfMargem  = 15.0
	pdfLargura = 210.0 - 2*pdfMargem
)

func novoPDF() (*fpdf.Fpdf, func(string) string) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMargem, pdfMargem, pdfMargem)
	pdf.SetAutoPageBreak(true, pdfMargem)
	pdf.SetCreator("fatec-turbo", true)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // cp1252: cobre os acentos do português
	return pdf, tr
}

// PDFRedacao gera a "folha de redação" virtual: título + texto em linhas numeradas, como na prova.
func PDFRedacao(provaID, tema, titulo, texto string) ([]byte, int, error) {
	pdf, tr := novoPDF()
	pdf.SetTitle(tr("Redação Fatec "+NomeProva(provaID)), false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 13)
	pdf.CellFormat(0, 7, tr("Vestibular Fatec · Simulado de redação · prova "+NomeProva(provaID)), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.MultiCell(0, 5, tr("Tema: "+tema), "", "L", false)
	pdf.SetFont("Helvetica", "I", 8.5)
	pdf.SetTextColor(90, 90, 90)
	pdf.MultiCell(0, 4.2, tr("Transcrição fiel do texto escrito à mão: erros de ortografia, acentuação e pontuação foram mantidos de propósito. Gerado em "+time.Now().Format("02/01/2006 15:04")+"."), "", "L", false)
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(3)

	// Título
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetDrawColor(150, 150, 150)
	pdf.CellFormat(14, 8, tr("Título"), "B", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, tr(titulo), "B", 1, "L", false, 0, "")
	pdf.Ln(1)

	// Linhas numeradas
	pdf.SetFont("Times", "", 12)
	larg := pdfLargura - 10
	var linhas []string
	for _, par := range strings.Split(strings.ReplaceAll(texto, "\r", ""), "\n") {
		if strings.TrimSpace(par) == "" {
			continue
		}
		linhas = append(linhas, quebraLinhas(pdf, tr, strings.TrimRight(par, " "), larg)...)
	}
	total := len(linhas)
	minimo := 30
	if total > minimo {
		minimo = total
	}
	alt := 8.2
	for i := 0; i < minimo; i++ {
		if pdf.GetY()+alt > 297-pdfMargem {
			pdf.AddPage()
		}
		y := pdf.GetY()
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetTextColor(130, 130, 130)
		pdf.CellFormat(10, alt, fmt.Sprintf("%02d", i+1), "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Times", "", 12)
		txt := ""
		if i < total {
			txt = linhas[i]
		}
		pdf.CellFormat(larg, alt, " "+txt, "", 0, "L", false, 0, "")
		pdf.Line(pdfMargem+10, y+alt, pdfMargem+pdfLargura, y+alt)
		pdf.SetY(y + alt)
	}
	pdf.Ln(3)
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(90, 90, 90)
	pdf.MultiCell(0, 4.2, tr(fmt.Sprintf("Linhas usadas: %d (nesta folha, cerca de 95 caracteres por linha; na folha manuscrita a contagem muda).", total)), "", "L", false)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), total, nil
}

// quebraLinhas divide um parágrafo em linhas que cabem em larg (mm), medindo o texto já convertido
// para cp1252. Não usa fpdf.SplitText, que entra em pânico com caracteres acima de U+00FF
// (aspas curvas, travessão, reticências que o celular insere sozinho).
func quebraLinhas(pdf *fpdf.Fpdf, tr func(string) string, par string, larg float64) []string {
	var out []string
	atual := ""
	for _, p := range strings.Fields(par) {
		w := tr(p)
		cand := w
		if atual != "" {
			cand = atual + " " + w
		}
		if pdf.GetStringWidth(cand) <= larg || atual == "" {
			atual = cand
			// palavra sozinha maior que a linha: corta em pedaços
			for pdf.GetStringWidth(atual) > larg && len(atual) > 1 {
				i := len(atual) - 1
				for i > 1 && pdf.GetStringWidth(atual[:i]) > larg {
					i--
				}
				out = append(out, atual[:i])
				atual = atual[i:]
			}
			continue
		}
		out = append(out, atual)
		atual = w
	}
	if atual != "" {
		out = append(out, atual)
	}
	return out
}

type ItemRevisao struct {
	Q        *Questao
	Resposta string
}

// PDFRevisaoProva junta as questões (imagens da prova), a resposta do aluno e o gabarito oficial.
func PDFRevisaoProva(fsys fs.FS, titulo string, itens []ItemRevisao) ([]byte, error) {
	pdf, tr := novoPDF()
	pdf.SetTitle(tr(titulo), false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 14)
	pdf.MultiCell(0, 7, tr(titulo), "", "L", false)
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 4.5, tr(fmt.Sprintf("%d questões. Tabela: questão, prova de origem, disciplina, resposta marcada pelo aluno e gabarito oficial. Questões anuladas contam como acerto. Gerado em %s.", len(itens), time.Now().Format("02/01/2006 15:04"))), "", "L", false)
	pdf.Ln(2)

	// tabela de respostas em 3 colunas
	cols := 3
	lin := (len(itens) + cols - 1) / cols
	cw := []float64{9, 15, 18, 10, 10}
	larg := 0.0
	for _, w := range cw {
		larg += w
	}
	gap := (pdfLargura - larg*float64(cols)) / float64(cols-1)
	y0 := pdf.GetY()
	pdf.SetFont("Helvetica", "B", 7.5)
	for c := 0; c < cols; c++ {
		pdf.SetXY(pdfMargem+float64(c)*(larg+gap), y0)
		for i, h := range []string{"Q", "Prova", "Disc.", "Marcou", "Gab."} {
			pdf.CellFormat(cw[i], 5, tr(h), "B", 0, "C", false, 0, "")
		}
	}
	pdf.SetFont("Helvetica", "", 7.5)
	for i, it := range itens {
		c, r := i/lin, i%lin
		pdf.SetXY(pdfMargem+float64(c)*(larg+gap), y0+5+float64(r)*4.6)
		resp := it.Resposta
		if resp == "" {
			resp = "branco"
		}
		disc := it.Q.Disc
		if len([]rune(disc)) > 10 {
			disc = string([]rune(disc)[:9]) + "."
		}
		for k, v := range []string{fmt.Sprint(i + 1), NomeProva(it.Q.Prova) + " Q" + fmt.Sprint(it.Q.N), disc, resp, it.Q.Gab} {
			if k == 1 {
				pdf.SetFont("Helvetica", "", 6.5)
			}
			pdf.CellFormat(cw[k], 4.6, tr(v), "", 0, "C", false, 0, "")
			pdf.SetFont("Helvetica", "", 7.5)
		}
	}
	pdf.SetY(y0 + 5 + float64(lin)*4.6 + 4)

	for i, it := range itens {
		pdf.AddPage()
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(0, 6, tr(fmt.Sprintf("Questão %d de %d · Fatec %s, questão %d · %s", i+1, len(itens), NomeProva(it.Q.Prova), it.Q.N, it.Q.Disc)), "", 1, "L", false, 0, "")
		pdf.Ln(1)
		var imgs []string
		for _, cx := range it.Q.Ctx {
			imgs = append(imgs, cx...)
		}
		imgs = append(imgs, it.Q.Imgs...)
		for _, f := range imgs {
			if err := pdfImagem(pdf, fsys, "conteudo/q/"+it.Q.Prova+"/"+f); err != nil {
				return nil, err
			}
		}
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pdfImagem(pdf *fpdf.Fpdf, fsys fs.FS, path string) error {
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return err
	}
	w := pdfLargura
	h := w * float64(cfg.Height) / float64(cfg.Width)
	maxH := 297 - 2*pdfMargem - 10
	if h > maxH {
		h = maxH
		w = h * float64(cfg.Width) / float64(cfg.Height)
	}
	if pdf.GetY()+h > 297-pdfMargem {
		pdf.AddPage()
	}
	opt := fpdf.ImageOptions{ImageType: "JPG", ReadDpi: false}
	pdf.RegisterImageOptionsReader(path, opt, bytes.NewReader(b))
	pdf.ImageOptions(path, pdfMargem, pdf.GetY(), w, h, false, opt, 0, "")
	pdf.SetY(pdf.GetY() + h + 2)
	return nil
}
