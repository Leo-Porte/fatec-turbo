package main

import (
	"bytes"
	"fmt"
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
