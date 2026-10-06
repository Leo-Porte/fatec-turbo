package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Revisão da semana: a correção já foi feita por script (gabarito oficial). Aqui o portal só junta o que o aluno
// fez na semana num relatório para a IA resumir pontos fracos e montar o plano da semana seguinte.

// temaParaLicao liga cada tema do CSV à lição que o cobre, lendo a coluna "Temas do CSV" do catálogo.
var temaParaLicao = map[string]string{}

func carregaCatalogo(fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "docs/catalogo_licoes.md")
	if err != nil {
		return
	}
	reLinha := regexp.MustCompile(`^\|\s*\d+\s*\|\s*([a-z0-9-]+)\s*\|[^|]*\|[^|]*\|\s*([^|]*)\|`)
	reTema := regexp.MustCompile(`\s*\(\d+\)\s*$`)
	for _, l := range strings.Split(string(b), "\n") {
		m := reLinha.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		for _, t := range strings.Split(m[2], ",") {
			t = strings.TrimSpace(reTema.ReplaceAllString(t, ""))
			t = strings.TrimSpace(regexp.MustCompile(`\s*\((Português|Inglês)\)$`).ReplaceAllString(t, ""))
			if t != "" && temaParaLicao[t] == "" {
				temaParaLicao[t] = m[1]
			}
		}
	}
}

func licaoDoTema(tema string) string {
	if id := temaParaLicao[tema]; id != "" {
		if l, ok := C.PorLicao[id]; ok {
			return l.Titulo
		}
	}
	for _, l := range C.Licoes {
		if l.Tema == tema {
			return l.Titulo
		}
	}
	return ""
}

func pagRevisao(w http.ResponseWriter, r *http.Request) {
	s := SemanaAtual(time.Now())
	ini := InicioPlano.AddDate(0, 0, (s-1)*7)
	render(w, r, "revisao", "Revisão da semana", "desempenho", map[string]any{
		"Semana": s, "Datas": DatasSemana(s), "Inicio": ini.UnixMilli(), "Fim": ini.AddDate(0, 0, 7).UnixMilli(),
		"Proxima": ini.AddDate(0, 0, 7).Format("02/01"), "Prompt": PromptRevisaoSemanal(s),
	})
}

type simSemana struct {
	Titulo    string            `json:"titulo"`
	Tipo      string            `json:"tipo"`
	Fim       int64             `json:"fim"`
	Segundos  float64           `json:"segundos"`
	Acertos   int               `json:"acertos"`
	Total     int               `json:"total"`
	Nota      int               `json:"nota"`
	PorDisc   map[string][2]int `json:"porDisc"`
	PorTema   map[string][2]int `json:"porTema"`
	Erros     []string          `json:"erros"` // chaves "prova|n"
	Respostas map[string]string `json:"respostas"`
}

type pedidoRevisao struct {
	Semana int                          `json:"semana"`
	Sims   []simSemana                  `json:"sims"`
	Lidas  []string                     `json:"lidas"`
	Ex     map[string]map[string]string `json:"ex"` // lição -> índice do exercício -> letra marcada
	Testes []struct {
		ID      string `json:"id"`
		Acertos int    `json:"acertos"`
		Total   int    `json:"total"`
	} `json:"testes"`
	Reds []struct {
		Prova  string `json:"prova"`
		Titulo string `json:"titulo"`
		Texto  string `json:"texto"`
		Nota   *int   `json:"nota"`
	} `json:"reds"`
	ResumoAnterior string `json:"resumoAnterior"`
}

func postRevisaoSemana(w http.ResponseWriter, r *http.Request) {
	var p pedidoRevisao
	if err := json.Unmarshal([]byte(r.FormValue("dados")), &p); err != nil || p.Semana < 1 || p.Semana > len(Plano) {
		http.Error(w, "pedido inválido", 400)
		return
	}
	var b strings.Builder
	pr := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	pr("# Revisão da semana %d (%s) · Fatec Turbo\n\n", p.Semana, DatasSemana(p.Semana))
	pr("Aluno se preparando para o vestibular Fatec de %s. Foco planejado da semana: %s.\n", DiaProva.Format("02/01/2006"), Plano[p.Semana-1].Foco)
	pr("Todas as notas abaixo já foram corrigidas automaticamente pelo gabarito oficial.\n\n")

	// simulados e treinos
	pr("## Simulados e treinos da semana\n\n")
	if len(p.Sims) == 0 {
		pr("Nenhum simulado ou treino finalizado nesta semana.\n\n")
	} else {
		pr("| Data | Tipo | Atividade | Acertos | Nota (0–100) | Tempo |\n|---|---|---|---|---|---|\n")
		for _, s := range p.Sims {
			pr("| %s | %s | %s | %d/%d | %d | %s |\n", time.UnixMilli(s.Fim).Format("02/01 15:04"), s.Tipo, s.Titulo, s.Acertos, s.Total, s.Nota, (time.Duration(s.Segundos) * time.Second).Round(time.Minute))
		}
		pr("\n")
	}

	disc, temas := map[string][2]int{}, map[string][2]int{}
	for _, s := range p.Sims {
		for k, v := range s.PorDisc {
			d := disc[k]
			disc[k] = [2]int{d[0] + v[0], d[1] + v[1]}
		}
		for k, v := range s.PorTema {
			t := temas[k]
			temas[k] = [2]int{t[0] + v[0], t[1] + v[1]}
		}
	}
	if len(disc) > 0 {
		pr("## Acerto por disciplina\n\n| Disciplina | Acertos | %% |\n|---|---|---|\n")
		for _, d := range DiscOrdem {
			if v, ok := disc[d]; ok && v[1] > 0 {
				pr("| %s | %d/%d | %d%% |\n", d, v[0], v[1], v[0]*100/v[1])
			}
		}
		pr("\n")
	}

	// temas com erro, do que mais pesou para o que menos pesou, com o conceito de cada questão errada
	errosPorTema := map[string][]string{}
	for _, s := range p.Sims {
		for _, k := range s.Erros {
			q := achaQuestao(k)
			if q == nil || q.Anulada() {
				continue
			}
			marcou := s.Respostas[k]
			if marcou == "" {
				marcou = "em branco"
			}
			errosPorTema[q.Tema] = append(errosPorTema[q.Tema], fmt.Sprintf("Fatec %s Q%d (%s): marcou %s, gabarito %s. Conceito cobrado: %s", NomeProva(q.Prova), q.N, q.Sub, marcou, q.Gab, q.Conc))
		}
	}
	nomes := make([]string, 0, len(errosPorTema))
	for t := range errosPorTema {
		nomes = append(nomes, t)
	}
	sort.Slice(nomes, func(i, j int) bool { return len(errosPorTema[nomes[i]]) > len(errosPorTema[nomes[j]]) })
	if len(nomes) > 0 {
		pr("## Temas com erro (do que mais errou para o que menos errou)\n\n")
		for _, t := range nomes {
			v := temas[t]
			pr("### %s: %d erro(s), acerto %d/%d", t, len(errosPorTema[t]), v[0], v[1])
			if l := licaoDoTema(t); l != "" {
				pr(" · lição no portal: \"%s\"", l)
			}
			pr("\n\n")
			for _, e := range errosPorTema[t] {
				pr("- %s\n", e)
			}
			pr("\n")
		}
	}

	// lições e exercícios
	pr("## Lições lidas na semana\n\n")
	if len(p.Lidas) == 0 {
		pr("Nenhuma lição marcada como lida nesta semana.\n\n")
	} else {
		for _, id := range p.Lidas {
			t := id
			if l, ok := C.PorLicao[id]; ok {
				t = l.Titulo + " (" + l.Disciplina + ")"
			}
			ex := ""
			if resp, ok := p.Ex[id]; ok && len(resp) > 0 {
				if l, ok := C.PorLicao[id]; ok {
					ac := 0
					for i, e := range l.Exercicios {
						if resp[strconv.Itoa(i)] == e.Gabarito {
							ac++
						}
					}
					ex = fmt.Sprintf(": exercícios %d/%d certos (%d respondidos)", ac, len(l.Exercicios), len(resp))
				}
			}
			pr("- %s%s\n", t, ex)
		}
		pr("\n")
	}
	if len(p.Testes) > 0 {
		pr("## Testes da semana\n\n")
		for _, t := range p.Testes {
			nome := t.ID
			if tt, ok := C.PorTeste[t.ID]; ok {
				nome = tt.Titulo
			}
			pr("- %s: %d/%d\n", nome, t.Acertos, t.Total)
		}
		pr("\n")
	}

	// redações
	if len(p.Reds) > 0 {
		pr("## Redações da semana\n\nTranscritas como o aluno escreveu à mão, com os erros mantidos.\n\n")
		for _, rd := range p.Reds {
			tema := rd.Prova
			if pv, ok := C.PorProva[rd.Prova]; ok {
				tema = pv.TemaRed
			}
			nota := "sem nota registrada"
			if rd.Nota != nil {
				nota = fmt.Sprintf("nota registrada: %d", *rd.Nota)
			}
			pr("### Fatec %s · %s (%s)\n\nTema: %s\n\n```\n%s\n```\n\n", NomeProva(rd.Prova), rd.Titulo, nota, tema, strings.TrimSpace(rd.Texto))
		}
	}

	// o que vem na próxima semana
	if p.Semana < len(Plano) {
		prox := Plano[p.Semana]
		pr("## Próxima semana no cronograma (semana %d, %s)\n\nFoco: %s.", prox.S, DatasSemana(prox.S), prox.Foco)
		if prox.Sim != "" {
			pr(" Simulado do fim de semana: Fatec %s.", NomeProva(prox.Sim))
		}
		pr("\n\nLições disponíveis no portal para essa semana:\n\n")
		for _, l := range C.Licoes {
			if l.Semana == prox.S {
				pr("- %s (%s, %d min)\n", l.Titulo, l.Disciplina, l.Minutos)
			}
		}
		pr("\n")
	}
	if strings.TrimSpace(p.ResumoAnterior) != "" {
		pr("## Plano que a IA deu na semana anterior\n\n%s\n", strings.TrimSpace(p.ResumoAnterior))
	}
	baixar(w, "text/markdown; charset=utf-8", fmt.Sprintf("revisao-semana-%d.md", p.Semana), []byte(b.String()))
}
