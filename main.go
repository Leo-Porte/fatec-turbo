// Fatec Turbo: portal de estudo para o vestibular Fatec, feito de leitura, simulados e redação.
// Um binário só: conteúdo, páginas e estilos vão embutidos. O progresso de cada pessoa fica no navegador.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // fuso de São Paulo mesmo em contêiner sem zoneinfo
)

//go:embed dados conteudo web
var embedded embed.FS

var (
	C    *Conteudo
	tpls map[string]*template.Template
)

var (
	ImgBase     = "/q/" // prefixo das imagens das questões; pode apontar para um CDN (ex.: jsDelivr)
	confiaProxy bool
)

func main() {
	// subcomandos de administração: fatec-turbo usuario ... | fatec-turbo migrar
	if len(os.Args) > 1 && (os.Args[1] == "usuario" || os.Args[1] == "migrar") {
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			log.Fatal("defina DATABASE_URL")
		}
		db, err := conectaBanco(dsn)
		if err != nil {
			log.Fatal(err)
		}
		if err := migra(db); err != nil {
			log.Fatal(err)
		}
		if os.Args[1] == "usuario" {
			C = &Conteudo{PorTeste: map[string]*Teste{}}
			if err := cmdUsuario(db, os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		return
	}

	addr := flag.String("addr", envOu("FT_ADDR", "127.0.0.1:8027"), "endereço do servidor")
	abrir := flag.Bool("abrir", os.Getenv("DATABASE_URL") == "", "abrir o navegador ao iniciar")
	dsn := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres (ex.: postgres://u:s@host/db). Vazio = modo local, sem login")
	seguro := flag.Bool("cookie-seguro", os.Getenv("FT_COOKIE_SEGURO") == "1", "cookie de sessão só por HTTPS")
	flag.BoolVar(&confiaProxy, "proxy", os.Getenv("FT_PROXY") == "1", "confiar em X-Forwarded-For (atrás do Caddy)")
	img := flag.String("imagens", os.Getenv("FT_IMAGENS_URL"), "URL base das imagens das questões (vazio = servir do próprio binário)")
	flag.Parse()
	if *img != "" {
		ImgBase = strings.TrimRight(*img, "/") + "/"
	}

	var err error
	C, err = CarregaConteudo(embedded)
	if err != nil {
		log.Fatal(err)
	}
	if err := carregaTemplates(); err != nil {
		log.Fatal(err)
	}
	if *dsn != "" {
		if DB, err = conectaBanco(*dsn); err != nil {
			log.Fatal(err)
		}
		if err := migra(DB); err != nil {
			log.Fatal(err)
		}
	}

	mux := http.NewServeMux()
	static, _ := fs.Sub(embedded, "web/static")
	mux.Handle("GET /static/", cache(http.StripPrefix("/static/", http.FileServerFS(static))))
	qfs, _ := fs.Sub(embedded, "conteudo/q")
	mux.Handle("GET /q/", cache(http.StripPrefix("/q/", http.FileServerFS(qfs))))
	mux.HandleFunc("GET /saude", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	mux.HandleFunc("GET /{$}", pagInicio)
	mux.HandleFunc("GET /licoes", pagLicoes)
	mux.HandleFunc("GET /licoes/{id}", pagLicao)
	mux.HandleFunc("GET /simulados", pagSimulados)
	mux.HandleFunc("GET /simulado/{prova}", pagSimulado)
	mux.HandleFunc("GET /treino", pagTreino)
	mux.HandleFunc("GET /testes", pagTestes)
	mux.HandleFunc("GET /testes/{id}", pagTeste)
	mux.HandleFunc("GET /redacao", pagRedacoes)
	mux.HandleFunc("GET /redacao/{prova}", pagRedacao)
	mux.HandleFunc("GET /autores", pagAutores)
	mux.HandleFunc("GET /desempenho", pagSimples("desempenho", "Desempenho"))
	mux.HandleFunc("GET /erros", pagSimples("erros", "Caderno de erros"))
	mux.HandleFunc("GET /dados", pagSimples("dados", "Seus dados"))
	mux.HandleFunc("GET /entrar", pagEntrar)
	mux.HandleFunc("POST /entrar", postEntrar(*seguro))
	mux.HandleFunc("POST /sair", postSair)

	mux.HandleFunc("POST /api/corrigir", apiCorrigir)
	mux.HandleFunc("GET /api/questoes", apiQuestoes)
	mux.HandleFunc("PUT /api/progresso", apiProgresso)
	mux.HandleFunc("POST /api/progresso", apiProgresso) // sendBeacon ao fechar a página
	mux.HandleFunc("POST /revisao/prova.pdf", postRevisaoProva)
	mux.HandleFunc("POST /revisao/teste.md", postRevisaoTeste)
	mux.HandleFunc("POST /redacao.pdf", postRedacaoPDF)

	modo := "local, sem login"
	if DB != nil {
		modo = "com banco e login"
	}
	url := "http://" + *addr
	log.Printf("Fatec Turbo no ar: %s  (%s; %d provas, %d lições, %d testes)", url, modo, len(C.Provas), len(C.Licoes), len(C.Testes))
	if C.SemProvas {
		log.Printf("Aviso: nenhuma prova em conteudo/q. Rode tools/baixar_provas.py antes de compilar.")
	}
	if *abrir {
		go func() { time.Sleep(400 * time.Millisecond); abreNavegador(url) }()
	}
	srv := &http.Server{Addr: *addr, Handler: logReq(mesmaOrigem(comSessao(*seguro, mux))),
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func envOu(k, padrao string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return padrao
}

func abreNavegador(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func cache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		h.ServeHTTP(w, r)
	})
}

func logReq(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		h.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/q/") && !strings.HasPrefix(r.URL.Path, "/static/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(t).Round(time.Millisecond))
		}
	})
}

// ---------- templates ----------

func carregaTemplates() error {
	funcs := template.FuncMap{
		"md":        Markdown,
		"mdInline":  MarkdownInline,
		"figura":    RenderFigura,
		"nomeProva": NomeProva,
		"datas":     DatasSemana,
		"letra":     func(i int) string { return string(rune('A' + i)) },
		"add":       func(a, b int) int { return a + b },
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"join":        strings.Join,
		"dict2":       func(i int, e Exercicio) map[string]any { return map[string]any{"I": i, "E": e} },
		"promptTeste": PromptRevisaoTeste,
		"img":         func(prova, f string) string { return ImgBase + prova + "/" + f },
	}
	base, err := template.New("base").Funcs(funcs).ParseFS(embedded, "web/templates/base.html")
	if err != nil {
		return err
	}
	pags, _ := fs.Glob(embedded, "web/templates/p_*.html")
	tpls = map[string]*template.Template{}
	for _, p := range pags {
		t, err := template.Must(base.Clone()).ParseFS(embedded, p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		nome := strings.TrimSuffix(strings.TrimPrefix(p, "web/templates/p_"), ".html")
		tpls[nome] = t
	}
	return nil
}

type Pagina struct {
	Titulo    string
	Aba       string
	Dias      int
	D         any
	Usuario   *Usuario
	Servidor  bool        // progresso sincronizado com o banco
	Progresso template.JS // estado salvo do usuário (JSON), quando há banco
	Versao    int
}

func render(w http.ResponseWriter, r *http.Request, nome, titulo, aba string, d any) {
	t, ok := tpls[nome]
	if !ok {
		http.Error(w, "página não encontrada: "+nome, 500)
		return
	}
	dias := int(time.Until(DiaProva).Hours()/24) + 1
	if dias < 0 {
		dias = 0
	}
	pg := Pagina{Titulo: titulo, Aba: aba, Dias: dias, D: d, Progresso: "null"}
	if u := usuarioDe(r); u != nil && DB != nil {
		pg.Usuario, pg.Servidor = u, true
		if dados, v, err := carregaProgresso(r.Context(), u.ID); err == nil {
			// escapa < > & dentro do JSON: uma redação com "</script>" não pode quebrar a página
			var buf bytes.Buffer
			json.HTMLEscape(&buf, dados)
			pg.Progresso, pg.Versao = template.JS(buf.String()), v
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	if err := t.ExecuteTemplate(w, "base", pg); err != nil {
		log.Printf("template %s: %v", nome, err)
	}
}

func pagSimples(nome, titulo string) http.HandlerFunc {
	aba := map[string]string{"desempenho": "desempenho", "erros": "estudar", "dados": ""}[nome]
	return func(w http.ResponseWriter, r *http.Request) { render(w, r, nome, titulo, aba, nil) }
}

// ---------- páginas ----------

type Tarefa struct {
	ID, Txt, Link string
	Auto          string // chave para marcar sozinho pelo JS: "licao:<id>", "sim:<prova>", "red:<prova>"
}

func tarefasSemana(s int) []Tarefa {
	w := Plano[s-1]
	var out []Tarefa
	tem := false
	for _, l := range C.Licoes {
		if l.Semana == s {
			tem = true
			out = append(out, Tarefa{fmt.Sprintf("s%d-lic-%s", s, l.ID), "Ler a lição “" + l.Titulo + "” e fazer os exercícios", "/licoes/" + l.ID, "licao:" + l.ID})
		}
	}
	for _, t := range C.Testes {
		if t.Semana == s {
			out = append(out, Tarefa{fmt.Sprintf("s%d-teste-%s", s, t.ID), "Fazer o teste “" + t.Titulo + "”", "/testes/" + t.ID, "teste:" + t.ID})
		}
	}
	if !tem && s <= 8 {
		out = append(out, Tarefa{fmt.Sprintf("s%d-foco", s), "Noites: " + w.Foco + ". Enquanto as lições desta semana não entram, faça treinos rápidos do bloco.", "/simulados#treino", ""})
	}
	if s <= 9 {
		out = append(out, Tarefa{fmt.Sprintf("s%d-treinos", s), "2 treinos rápidos (10 questões) nos temas com mais erro", "/simulados#treino", ""})
	}
	if w.Sim != "" {
		out = append(out, Tarefa{fmt.Sprintf("s%d-sim", s), "Fim de semana: simulado da prova " + NomeProva(w.Sim) + " (5 horas)", "/simulado/" + w.Sim, "sim:" + w.Sim})
		out = append(out, Tarefa{fmt.Sprintf("s%d-red", s), "Fim de semana: redação da prova " + NomeProva(w.Sim) + " no papel, com 1 hora; depois transcrever aqui", "/redacao/" + w.Sim, "red:" + w.Sim})
	}
	out = append(out, Tarefa{fmt.Sprintf("s%d-erros", s), "Revisar o caderno de erros", "/erros", ""})
	return out
}

func pagInicio(w http.ResponseWriter, r *http.Request) {
	atual := SemanaAtual(time.Now())
	s := atual
	if v, err := strconv.Atoi(r.URL.Query().Get("s")); err == nil && v >= 1 && v <= len(Plano) {
		s = v
	}
	type sem struct {
		S         int
		Datas     string
		Atual, Ve bool
	}
	var ss []sem
	for _, p := range Plano {
		ss = append(ss, sem{p.S, DatasSemana(p.S), p.S == atual, p.S == s})
	}
	render(w, r, "inicio", "Início", "inicio", map[string]any{
		"Semana": Plano[s-1], "Datas": DatasSemana(s), "Tarefas": tarefasSemana(s), "Semanas": ss, "EAtual": s == atual,
		"TotalLicoes": len(C.Licoes), "SemProvas": C.SemProvas,
	})
}

func pagLicoes(w http.ResponseWriter, r *http.Request) {
	type grupo struct {
		S      int
		Datas  string
		Licoes []*Licao
	}
	var gs []grupo
	for _, l := range C.Licoes {
		if len(gs) == 0 || gs[len(gs)-1].S != l.Semana {
			gs = append(gs, grupo{l.Semana, DatasSemana(max(l.Semana, 1)), nil})
		}
		gs[len(gs)-1].Licoes = append(gs[len(gs)-1].Licoes, l)
	}
	render(w, r, "licoes", "Lições", "estudar", map[string]any{"Grupos": gs, "Testes": C.Testes, "Nomes": nomesLicoes()})
}

func pagLicao(w http.ResponseWriter, r *http.Request) {
	l, ok := C.PorLicao[r.PathValue("id")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	type res struct {
		R Resolvida
		Q *Questao
	}
	var rs []res
	for _, x := range l.Resolvidas {
		if p, ok := C.PorProva[x.Prova]; ok {
			for _, q := range p.Questoes {
				if q.N == x.N {
					rs = append(rs, res{x, q})
				}
			}
		}
	}
	gab := make([]string, len(l.Exercicios))
	for i, e := range l.Exercicios {
		gab[i] = e.Gabarito
	}
	render(w, r, "licao", l.Titulo, "estudar", map[string]any{"L": l, "Resolvidas": rs, "Gab": gab, "Nomes": nomesLicoes()})
}

func nomesLicoes() map[string]string {
	m := map[string]string{}
	for _, l := range C.Licoes {
		m[l.ID] = l.Titulo
	}
	return m
}

func pagSimulados(w http.ResponseWriter, r *http.Request) {
	s := SemanaAtual(time.Now())
	type pv struct {
		P      *Prova
		Semana int
		Datas  string
		Atual  bool
		Treino bool
	}
	var ps []pv
	for _, p := range C.Provas {
		x := pv{P: p, Treino: ProvasTreino[p.ID]}
		for _, w := range Plano {
			if w.Sim == p.ID {
				x.Semana, x.Datas, x.Atual = w.S, DatasSemana(w.S), w.S == s
			}
		}
		ps = append(ps, x)
	}
	// temas disponíveis para treino, por disciplina
	temas := map[string]map[string]int{}
	for _, p := range C.Provas {
		if !ProvasTreino[p.ID] {
			continue
		}
		for _, q := range p.Questoes {
			if q.Anulada() {
				continue
			}
			if temas[q.Disc] == nil {
				temas[q.Disc] = map[string]int{}
			}
			temas[q.Disc][q.Tema]++
		}
	}
	render(w, r, "simulados", "Simulados", "simulados", map[string]any{"Provas": ps, "Disc": DiscOrdem, "Temas": temas})
}

// Dados que o executor de simulado recebe: sem gabarito (a correção é no servidor).
type QView struct {
	K    string     `json:"k"` // chave da resposta
	P    string     `json:"p"`
	N    int        `json:"n"`
	Disc string     `json:"d"`
	Imgs []string   `json:"i"`
	Ctx  [][]string `json:"c"`
}

func qview(q *Questao, chave string) QView {
	pref := ImgBase + q.Prova + "/"
	v := QView{K: chave, P: q.Prova, N: q.N, Disc: q.Disc}
	for _, f := range q.Imgs {
		v.Imgs = append(v.Imgs, pref+f)
	}
	for _, cx := range q.Ctx {
		var l []string
		for _, f := range cx {
			l = append(l, pref+f)
		}
		v.Ctx = append(v.Ctx, l)
	}
	return v
}

func pagSimulado(w http.ResponseWriter, r *http.Request) {
	p, ok := C.PorProva[r.PathValue("prova")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	var qs []QView
	for _, q := range p.Questoes {
		qs = append(qs, qview(q, strconv.Itoa(q.N)))
	}
	render(w, r, "executor", "Simulado Fatec "+p.Nome(), "simulados", map[string]any{
		"Cfg": map[string]any{"id": "p" + p.ID, "tipo": "prova", "prova": p.ID, "titulo": "Simulado Fatec " + p.Nome(), "limite": 5 * 3600, "questoes": qs,
			"prompt": PromptRevisaoProva("Revisão do simulado Fatec "+p.Nome(), len(qs))},
	})
}

func pagTreino(w http.ResponseWriter, r *http.Request) {
	disc, tema := r.URL.Query().Get("disc"), r.URL.Query().Get("tema")
	var pool []*Questao
	for _, p := range C.Provas {
		if !ProvasTreino[p.ID] {
			continue
		}
		for _, q := range p.Questoes {
			if q.Anulada() || (disc != "" && q.Disc != disc) || (tema != "" && q.Tema != tema) {
				continue
			}
			pool = append(pool, q)
		}
	}
	if len(pool) < 5 && tema != "" { // tema raro: completa com as demais provas
		for _, p := range C.Provas {
			if ProvasTreino[p.ID] {
				continue
			}
			for _, q := range p.Questoes {
				if q.Tema == tema && !q.Anulada() {
					pool = append(pool, q)
				}
			}
		}
	}
	if len(pool) == 0 {
		http.Redirect(w, r, "/simulados", http.StatusSeeOther)
		return
	}
	// o navegador manda as questões já vistas em ?visto=p|n,p|n para priorizar inéditas
	visto := map[string]bool{}
	for _, k := range strings.Split(r.URL.Query().Get("visto"), ",") {
		visto[k] = true
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	sort.SliceStable(pool, func(i, j int) bool {
		return !visto[pool[i].Prova+"|"+strconv.Itoa(pool[i].N)] && visto[pool[j].Prova+"|"+strconv.Itoa(pool[j].N)]
	})
	if len(pool) > 10 {
		pool = pool[:10]
	}
	var qs []QView
	for _, q := range pool {
		qs = append(qs, qview(q, q.Prova+"|"+strconv.Itoa(q.N)))
	}
	nome := tema
	if nome == "" {
		nome = disc
	}
	if nome == "" {
		nome = "misto"
	}
	render(w, r, "executor", "Treino: "+nome, "simulados", map[string]any{
		"Cfg": map[string]any{"id": fmt.Sprintf("t%d", time.Now().UnixMilli()), "tipo": "treino", "titulo": "Treino: " + nome, "filtro": map[string]string{"disc": disc, "tema": tema}, "limite": len(qs) * 180, "questoes": qs,
			"prompt": PromptRevisaoProva("Revisão do treino "+nome, len(qs))},
	})
}

func pagTestes(w http.ResponseWriter, r *http.Request) {
	render(w, r, "testes", "Testes", "estudar", map[string]any{"Testes": C.Testes})
}

func pagTeste(w http.ResponseWriter, r *http.Request) {
	t, ok := C.PorTeste[r.PathValue("id")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	gab := make([]string, len(t.Exercicios))
	for i, e := range t.Exercicios {
		gab[i] = e.Gabarito
	}
	render(w, r, "teste", t.Titulo, "estudar", map[string]any{"T": t, "Gab": gab})
}

func pagRedacoes(w http.ResponseWriter, r *http.Request) {
	s := SemanaAtual(time.Now())
	render(w, r, "redacoes", "Redação", "redacao", map[string]any{"Provas": C.Provas, "Atual": Plano[s-1].Sim})
}

func pagRedacao(w http.ResponseWriter, r *http.Request) {
	p, ok := C.PorProva[r.PathValue("prova")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	var imgs []string
	for _, f := range p.Redacao {
		imgs = append(imgs, ImgBase+p.ID+"/"+f)
	}
	render(w, r, "redacao", "Redação "+p.Nome(), "redacao", map[string]any{"P": p, "Imgs": imgs, "Prompt": PromptRedacao(p.TemaRed)})
}

func pagAutores(w http.ResponseWriter, r *http.Request) {
	as := append([]Autor(nil), C.Autores.Autores...)
	sort.Slice(as, func(i, j int) bool {
		if len(as[i].Aparicoes) != len(as[j].Aparicoes) {
			return len(as[i].Aparicoes) > len(as[j].Aparicoes)
		}
		return as[i].Nome < as[j].Nome
	})
	render(w, r, "autores", "Autores", "estudar", map[string]any{"Autores": as, "Veiculos": C.Autores.Veiculos})
}

// ---------- API ----------

type pedidoCorrecao struct {
	Itens     []string          `json:"itens"` // chaves "prova|n"
	Respostas map[string]string `json:"respostas"`
}

type ResultadoItem struct {
	K    string `json:"k"`
	Gab  string `json:"gab"`
	Ok   bool   `json:"ok"`
	Disc string `json:"disc"`
	Tema string `json:"tema"`
	Sub  string `json:"sub"`
}

func achaQuestao(chave string) *Questao {
	p, n, ok := strings.Cut(chave, "|")
	if !ok {
		return nil
	}
	pr, ok := C.PorProva[p]
	if !ok {
		return nil
	}
	nn, _ := strconv.Atoi(n)
	for _, q := range pr.Questoes {
		if q.N == nn {
			return q
		}
	}
	return nil
}

func apiCorrigir(w http.ResponseWriter, r *http.Request) {
	var p pedidoCorrecao
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&p); err != nil {
		http.Error(w, "pedido inválido", 400)
		return
	}
	out := struct {
		Acertos int               `json:"acertos"`
		Total   int               `json:"total"`
		Nota    int               `json:"nota"`
		PorDisc map[string][2]int `json:"porDisc"`
		PorTema map[string][2]int `json:"porTema"`
		Itens   []ResultadoItem   `json:"itens"`
	}{PorDisc: map[string][2]int{}, PorTema: map[string][2]int{}}
	for _, k := range p.Itens {
		q := achaQuestao(k)
		if q == nil {
			continue
		}
		ok := q.Anulada() || p.Respostas[k] == q.Gab
		out.Total++
		if ok {
			out.Acertos++
		}
		d, t := out.PorDisc[q.Disc], out.PorTema[q.Tema]
		d[1]++
		t[1]++
		if ok {
			d[0]++
			t[0]++
		}
		out.PorDisc[q.Disc], out.PorTema[q.Tema] = d, t
		out.Itens = append(out.Itens, ResultadoItem{k, q.Gab, ok, q.Disc, q.Tema, q.Sub})
	}
	if out.Total > 0 {
		out.Nota = (out.Acertos*100 + out.Total/2) / out.Total
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// apiQuestoes devolve imagens e classificação de questões pedidas (?k=prova|n,prova|n), para o caderno de erros.
func apiQuestoes(w http.ResponseWriter, r *http.Request) {
	type item struct {
		QView
		Gab  string `json:"gab"`
		Tema string `json:"tema"`
		Sub  string `json:"sub"`
	}
	var out []item
	for _, k := range strings.Split(r.URL.Query().Get("k"), ",") {
		if q := achaQuestao(k); q != nil {
			out = append(out, item{qview(q, k), q.Gab, q.Tema, q.Sub})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func postRevisaoProva(w http.ResponseWriter, r *http.Request) {
	var p struct {
		Titulo    string            `json:"titulo"`
		Itens     []string          `json:"itens"`
		Respostas map[string]string `json:"respostas"`
	}
	if err := json.Unmarshal([]byte(r.FormValue("dados")), &p); err != nil {
		http.Error(w, "pedido inválido", 400)
		return
	}
	var itens []ItemRevisao
	for _, k := range p.Itens {
		if q := achaQuestao(k); q != nil {
			itens = append(itens, ItemRevisao{q, p.Respostas[k]})
		}
	}
	if len(itens) == 0 {
		http.Error(w, "nenhuma questão", 400)
		return
	}
	b, err := PDFRevisaoProva(embedded, p.Titulo, itens)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	baixar(w, "application/pdf", nomeArquivo(p.Titulo)+".pdf", b)
}

func postRevisaoTeste(w http.ResponseWriter, r *http.Request) {
	t, ok := C.PorTeste[r.FormValue("teste")]
	var exs []Exercicio
	titulo := ""
	if ok {
		exs, titulo = t.Exercicios, t.Titulo
	} else if l, ok := C.PorLicao[r.FormValue("licao")]; ok {
		exs, titulo = l.Exercicios, "Exercícios: "+l.Titulo
	} else {
		http.NotFound(w, r)
		return
	}
	var resp map[string]string
	_ = json.Unmarshal([]byte(r.FormValue("respostas")), &resp)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n%d questões novas no estilo Fatec. Gerado em %s.\n\n", titulo, len(exs), time.Now().Format("02/01/2006 15:04"))
	for i, e := range exs {
		fmt.Fprintf(&b, "---\n\n## Questão %d\n\n%s\n\n", i+1, e.Enunciado)
		if e.Figura != nil {
			fj, _ := json.MarshalIndent(e.Figura, "", "  ")
			fmt.Fprintf(&b, "Figura (dados exatos usados para desenhar):\n\n```json\n%s\n```\n\n", fj)
		}
		for j, a := range e.Alternativas {
			fmt.Fprintf(&b, "(%c) %s\n", 'A'+j, a)
		}
		marcou := resp[strconv.Itoa(i)]
		if marcou == "" {
			marcou = "em branco"
		}
		fmt.Fprintf(&b, "\n- Resposta do aluno: **%s**\n- Gabarito proposto pelo autor: **%s**\n\n", marcou, e.Gabarito)
	}
	baixar(w, "text/markdown; charset=utf-8", nomeArquivo(titulo)+".md", []byte(b.String()))
}

func postRedacaoPDF(w http.ResponseWriter, r *http.Request) {
	p, ok := C.PorProva[r.FormValue("prova")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	titulo, texto := strings.TrimSpace(r.FormValue("titulo")), r.FormValue("texto")
	b, _, err := PDFRedacao(p.ID, p.TemaRed, titulo, texto)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	baixar(w, "application/pdf", "redacao-fatec-"+p.ID+".pdf", b)
}

func baixar(w http.ResponseWriter, tipo, nome string, b []byte) {
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Content-Disposition", `attachment; filename="`+nome+`"`)
	w.Write(b)
}

func nomeArquivo(s string) string {
	s = strings.ToLower(s)
	rep := strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c", "/", "-", ":", "", " ", "-")
	s = rep.Replace(s)
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
		}
	}
	if b.Len() == 0 {
		return "fatec"
	}
	return b.String()
}
