package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Questao é uma questão real de prova: imagens recortadas do PDF oficial + classificação.
type Questao struct {
	N     int        `json:"n"`
	Imgs  []string   `json:"imgs"`  // caminhos relativos a conteudo/q/<prova>/
	Ctx   [][]string `json:"ctx"`   // textos de apoio, cada um com suas imagens
	Gab   string     `json:"gab"`   // letra oficial ou "ANULADA"
	Disc  string     `json:"disc"`
	Tema  string     `json:"tema"`
	Sub   string     `json:"sub"`
	Prova string     `json:"prova"`
}

func (q Questao) Anulada() bool { return strings.Contains(strings.ToUpper(q.Gab), "ANUL") }

type Prova struct {
	ID       string     `json:"id"`
	Questoes []*Questao `json:"questoes"`
	Redacao  []string   `json:"redacao"` // imagens da proposta
	TemaRed  string     `json:"tema_red"`
}

func (p *Prova) Nome() string { return NomeProva(p.ID) }

func NomeProva(id string) string { return strings.Replace(id, "-", "/", 1) }

// Figura é desenhada no servidor como SVG (ver figura.go).
type Figura struct {
	Lib       string          `json:"lib"` // jsxgraph | chart | tabela
	BBox      []float64       `json:"bbox,omitempty"`
	Eixos     bool            `json:"eixos,omitempty"`
	Altura    int             `json:"altura,omitempty"`
	Elementos []Elemento      `json:"elementos,omitempty"`
	Tipo      string          `json:"tipo,omitempty"`
	Titulo    string          `json:"titulo,omitempty"`
	Rotulos   []string        `json:"rotulos,omitempty"`
	Series    []Serie         `json:"series,omitempty"`
	EixoY     string          `json:"eixoY,omitempty"`
	Cabecalho []string        `json:"cabecalho,omitempty"`
	Linhas    [][]string      `json:"linhas,omitempty"`
	Extra     json.RawMessage `json:"-"`
}

type Elemento struct {
	T       string    `json:"t"`
	ID      string    `json:"id,omitempty"`
	XY      []float64 `json:"xy,omitempty"`
	Rotulo  string    `json:"rotulo,omitempty"`
	De      string    `json:"de,omitempty"`
	Ate     string    `json:"ate,omitempty"`
	Pts     []string  `json:"pts,omitempty"`
	Centro  string    `json:"centro,omitempty"`
	Raio    float64   `json:"raio,omitempty"`
	Expr    string    `json:"expr,omitempty"`
	Dominio []float64 `json:"dominio,omitempty"`
	Txt     string    `json:"txt,omitempty"`
}

type Serie struct {
	Nome  string    `json:"nome"`
	Dados []float64 `json:"dados"`
}

// Exercicio é uma questão nova, escrita por nós, com figura opcional gerada em código.
type Exercicio struct {
	Enunciado    string   `json:"enunciado"`
	Alternativas []string `json:"alternativas"`
	Gabarito     string   `json:"gabarito"`
	Resolucao    string   `json:"resolucao"`
	Figura       *Figura  `json:"figura,omitempty"`
	Tema         string   `json:"tema,omitempty"`
}

type Resolvida struct {
	Prova     string `json:"prova"`
	N         int    `json:"n"`
	Resolucao string `json:"resolucao"`
}

type Secao struct {
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

type Licao struct {
	ID         string      `json:"id"`
	Disciplina string      `json:"disciplina"`
	Tema       string      `json:"tema"`
	Titulo     string      `json:"titulo"`
	Semana     int         `json:"semana"`
	Minutos    int         `json:"minutos"`
	PorQue     string      `json:"por_que"`
	Secoes     []Secao     `json:"secoes"`
	Decorar    []string    `json:"decorar"`
	Pegadinhas []string    `json:"pegadinhas"`
	Resolvidas []Resolvida `json:"resolvidas"`
	Exercicios []Exercicio `json:"exercicios"`
}

// Teste é um conjunto de questões novas publicado por nós (dados/testes/*.json).
type Teste struct {
	ID         string      `json:"id"`
	Titulo     string      `json:"titulo"`
	Semana     int         `json:"semana"`
	Descricao  string      `json:"descricao"`
	Publicado  string      `json:"publicado"`
	Exercicios []Exercicio `json:"exercicios"`
}

type Aparicao struct {
	Prova    string `json:"prova"`
	Questoes []int  `json:"questoes"`
	Obra     string `json:"obra"`
}

type Autor struct {
	Nome        string     `json:"nome"`
	Tipo        string     `json:"tipo"`
	Disciplina  string     `json:"disciplina"`
	Aparicoes   []Aparicao `json:"aparicoes"`
	Resumo      string     `json:"resumo"`
	FonteResumo string     `json:"fonte_resumo"`
	Licenca     string     `json:"licenca"`
	Dica        string     `json:"dica"`
	Trecho      string     `json:"trecho"`
	FonteTrecho string     `json:"fonte_trecho"`
}

type Autores struct {
	Autores  []Autor `json:"autores"`
	Veiculos []struct {
		Nome      string     `json:"nome"`
		Aparicoes []Aparicao `json:"aparicoes"`
	} `json:"veiculos"`
}

type Semana struct {
	S    int
	Foco string
	Sim  string // prova do simulado do fim de semana
	Nota string
}

var InicioPlano = time.Date(2026, 10, 5, 0, 0, 0, 0, time.Local)
var DiaProva = time.Date(2026, 12, 13, 13, 0, 0, 0, time.Local)

var Plano = []Semana{
	{1, "Matemática básica e Raciocínio Lógico", "2026-2", "Simulado diagnóstico: faça sem estudar antes. Ele mostra de onde você parte."},
	{2, "Matemática básica e Raciocínio Lógico", "2026-1", ""},
	{3, "Física e Química", "2025-2", ""},
	{4, "Física e Química", "2024-2", ""},
	{5, "Biologia, História e Geografia (+ 15 min de notícias por dia)", "2024-1", ""},
	{6, "Biologia, História e Geografia (+ 15 min de notícias por dia)", "2023-2", ""},
	{7, "Português e Inglês", "2023-1", ""},
	{8, "Português e Inglês + revisão do caderno de erros", "2022-2", ""},
	{9, "Revisão: lições já lidas e caderno de erros", "2020-1", "Simulado no horário real: domingo, 13h, 5 horas seguidas."},
	{10, "Revisão leve: fórmulas, negações lógicas, tabela de doenças", "", "Sábado 12/12 é de descanso. Prova domingo 13/12, 13h."},
}

// Provas reservadas para treinos rápidos (as recentes ficam para os simulados de fim de semana).
var ProvasTreino = map[string]bool{"2020-1": true, "2019-2": true, "2019-1": true, "2018-2": true, "2018-1": true, "2017-2": true, "2017-1": true}

var DiscOrdem = []string{"Português", "Matemática", "Multidisciplinar", "Raciocínio Lógico", "Física", "Química", "Biologia", "História", "Geografia", "Inglês"}

type Conteudo struct {
	Provas    []*Prova
	PorProva  map[string]*Prova
	Licoes    []*Licao
	PorLicao  map[string]*Licao
	Testes    []*Teste
	PorTeste  map[string]*Teste
	Autores   Autores
	FS        fs.FS // raiz com dados/ e conteudo/
	SemProvas bool
}

func SemanaAtual(agora time.Time) int {
	d := int(agora.Sub(InicioPlano).Hours() / 24)
	s := d/7 + 1
	if s < 1 {
		s = 1
	}
	if s > 10 {
		s = 10
	}
	return s
}

func DatasSemana(s int) string {
	a := InicioPlano.AddDate(0, 0, (s-1)*7)
	b := a.AddDate(0, 0, 6)
	return a.Format("02/01") + " a " + b.Format("02/01")
}

// CarregaConteudo lê dados/ (nosso material) e conteudo/q (recortes das provas, gerados por tools/recorta.py).
func CarregaConteudo(dados fs.FS) (*Conteudo, error) {
	c := &Conteudo{PorProva: map[string]*Prova{}, PorLicao: map[string]*Licao{}, PorTeste: map[string]*Teste{}, FS: dados}

	// Classificação + gabarito (nosso CSV)
	type cls struct{ gab, disc, tema, sub string }
	clsMap := map[string]cls{}
	temaRed := map[string]string{}
	if f, err := dados.Open("dados/questoes.csv"); err == nil {
		r := csv.NewReader(f)
		r.Comma = ';'
		r.FieldsPerRecord = -1
		rows, err := r.ReadAll()
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("questoes.csv: %w", err)
		}
		for i, row := range rows {
			if i == 0 || len(row) < 9 {
				continue
			}
			prova, q := strings.TrimPrefix(row[0], string(rune(0xFEFF))), row[1]
			if q == "RED" {
				temaRed[prova] = row[4]
				continue
			}
			clsMap[prova+"|"+q] = cls{row[8], row[2], row[3], row[4]}
		}
	}

	// Recortes gerados por tools/recorta.py
	manifests, _ := fs.Glob(dados, "conteudo/q/*/manifest.json")
	for _, m := range manifests {
		var man struct {
			Prova    string `json:"prova"`
			Questoes []struct {
				N    int      `json:"n"`
				Imgs []string `json:"imgs"`
				Ctx  []int    `json:"ctx"`
			} `json:"questoes"`
			Contextos []struct {
				ID   int      `json:"id"`
				Imgs []string `json:"imgs"`
			} `json:"contextos"`
			Redacao []string `json:"redacao"`
		}
		b, err := fs.ReadFile(dados, m)
		if err != nil || json.Unmarshal(b, &man) != nil {
			return nil, fmt.Errorf("manifest inválido: %s", m)
		}
		ctx := map[int][]string{}
		for _, cx := range man.Contextos {
			ctx[cx.ID] = cx.Imgs
		}
		p := &Prova{ID: man.Prova, Redacao: man.Redacao, TemaRed: temaRed[man.Prova]}
		for _, q := range man.Questoes {
			k := clsMap[man.Prova+"|"+strconv.Itoa(q.N)]
			qq := &Questao{N: q.N, Imgs: q.Imgs, Gab: k.gab, Disc: k.disc, Tema: k.tema, Sub: k.sub, Prova: man.Prova}
			for _, ci := range q.Ctx {
				qq.Ctx = append(qq.Ctx, ctx[ci])
			}
			p.Questoes = append(p.Questoes, qq)
		}
		sort.Slice(p.Questoes, func(i, j int) bool { return p.Questoes[i].N < p.Questoes[j].N })
		c.Provas = append(c.Provas, p)
		c.PorProva[p.ID] = p
	}
	sort.Slice(c.Provas, func(i, j int) bool { return c.Provas[i].ID > c.Provas[j].ID })
	c.SemProvas = len(c.Provas) == 0

	// Lições e testes
	licoes, _ := fs.Glob(dados, "dados/licoes/*.json")
	for _, f := range licoes {
		var l Licao
		if err := lerJSON(dados, f, &l); err != nil {
			return nil, err
		}
		c.Licoes = append(c.Licoes, &l)
		c.PorLicao[l.ID] = &l
	}
	sort.Slice(c.Licoes, func(i, j int) bool {
		if c.Licoes[i].Semana != c.Licoes[j].Semana {
			return c.Licoes[i].Semana < c.Licoes[j].Semana
		}
		return c.Licoes[i].ID < c.Licoes[j].ID
	})
	testes, _ := fs.Glob(dados, "dados/testes/*.json")
	for _, f := range testes {
		var t Teste
		if err := lerJSON(dados, f, &t); err != nil {
			return nil, err
		}
		c.Testes = append(c.Testes, &t)
		c.PorTeste[t.ID] = &t
	}
	sort.Slice(c.Testes, func(i, j int) bool { return c.Testes[i].Semana > c.Testes[j].Semana })
	if _, err := fs.Stat(dados, "dados/autores.json"); err == nil {
		if err := lerJSON(dados, "dados/autores.json", &c.Autores); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func lerJSON(fsys fs.FS, path string, v any) error {
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
