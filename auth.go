package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const (
	cookieSessao  = "ft_sessao"
	duracaoSessao = 60 * 24 * time.Hour
)

type Usuario struct {
	ID    int64
	Login string
	Nome  string
}

type ctxChave struct{}

func usuarioDe(r *http.Request) *Usuario {
	u, _ := r.Context().Value(ctxChave{}).(*Usuario)
	return u
}

func hashToken(t string) string {
	s := sha256.Sum256([]byte(t))
	return hex.EncodeToString(s[:])
}

// comSessao identifica o usuário pelo cookie e exige login em tudo, menos /entrar e arquivos estáticos.
// Sem banco (modo local) não faz nada.
func comSessao(cookieSeguro bool, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if DB == nil {
			h.ServeHTTP(w, r)
			return
		}
		livre := r.URL.Path == "/entrar" || strings.HasPrefix(r.URL.Path, "/static/")
		if c, err := r.Cookie(cookieSessao); err == nil && c.Value != "" {
			var u Usuario
			err := DB.QueryRowContext(r.Context(), `
				SELECT u.id, u.login, u.nome FROM sessoes s JOIN usuarios u ON u.id = s.usuario_id
				WHERE s.token_hash = $1 AND s.expira_em > now()`, hashToken(c.Value)).Scan(&u.ID, &u.Login, &u.Nome)
			if err == nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxChave{}, &u))
				h.ServeHTTP(w, r)
				return
			}
		}
		if livre {
			h.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") || r.Method != http.MethodGet {
			http.Error(w, "sessão expirada: entre de novo", http.StatusUnauthorized)
			return
		}
		http.Redirect(w, r, "/entrar?volta="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
	})
}

// mesmaOrigem barra POST/PUT vindos de outro site (CSRF). O cookie já é SameSite=Lax; isto é a segunda trava.
func mesmaOrigem(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if o := r.Header.Get("Origin"); o != "" {
				if u, err := url.Parse(o); err != nil || u.Host != r.Host {
					http.Error(w, "origem não permitida", http.StatusForbidden)
					return
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

// limite simples de tentativas de login por IP: 8 por 10 minutos.
var tentativas = struct {
	sync.Mutex
	m map[string][]time.Time
}{m: map[string][]time.Time{}}

func ipDe(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" && confiaProxy { // só atrás do Caddy; direto na internet seria forjável
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	h, _, _ := net.SplitHostPort(r.RemoteAddr)
	return h
}

func podeTentar(ip string) bool {
	tentativas.Lock()
	defer tentativas.Unlock()
	agora := time.Now()
	var recentes []time.Time
	for _, t := range tentativas.m[ip] {
		if agora.Sub(t) < 10*time.Minute {
			recentes = append(recentes, t)
		}
	}
	if len(recentes) >= 8 {
		tentativas.m[ip] = recentes
		return false
	}
	tentativas.m[ip] = append(recentes, agora)
	return true
}

func pagEntrar(w http.ResponseWriter, r *http.Request) {
	if DB == nil || usuarioDe(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	render(w, r, "entrar", "Entrar", "", map[string]any{"Volta": r.URL.Query().Get("volta")})
}

func postEntrar(cookieSeguro bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		volta := r.FormValue("volta")
		if !strings.HasPrefix(volta, "/") || strings.HasPrefix(volta, "//") {
			volta = "/"
		}
		erro := func(msg string) {
			w.WriteHeader(http.StatusUnauthorized)
			render(w, r, "entrar", "Entrar", "", map[string]any{"Volta": volta, "Erro": msg, "Login": r.FormValue("login")})
		}
		if !podeTentar(ipDe(r)) {
			erro("Muitas tentativas. Espere 10 minutos e tente de novo.")
			return
		}
		login := strings.ToLower(strings.TrimSpace(r.FormValue("login")))
		var id int64
		var hash string
		err := DB.QueryRowContext(r.Context(), `SELECT id, senha_hash FROM usuarios WHERE login=$1`, login).Scan(&id, &hash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.FormValue("senha"))) != nil {
			erro("Login ou senha incorretos.")
			return
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			http.Error(w, "erro interno", 500)
			return
		}
		token := base64.RawURLEncoding.EncodeToString(b)
		if _, err := DB.ExecContext(r.Context(), `INSERT INTO sessoes(token_hash, usuario_id, expira_em) VALUES($1,$2,$3)`,
			hashToken(token), id, time.Now().Add(duracaoSessao)); err != nil {
			http.Error(w, "erro interno", 500)
			return
		}
		DB.ExecContext(r.Context(), `UPDATE usuarios SET ultimo_acesso=now() WHERE id=$1`, id)
		DB.ExecContext(r.Context(), `DELETE FROM sessoes WHERE expira_em < now()`)
		http.SetCookie(w, &http.Cookie{Name: cookieSessao, Value: token, Path: "/", HttpOnly: true, Secure: cookieSeguro,
			SameSite: http.SameSiteLaxMode, MaxAge: int(duracaoSessao.Seconds())})
		http.Redirect(w, r, volta, http.StatusSeeOther)
	}
}

func postSair(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieSessao); err == nil && DB != nil {
		DB.ExecContext(r.Context(), `DELETE FROM sessoes WHERE token_hash=$1`, hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: cookieSessao, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/entrar", http.StatusSeeOther)
}

// ---------- comandos de administração ----------
//   fatec-turbo usuario criar <login> [nome...]
//   fatec-turbo usuario senha <login>
//   fatec-turbo usuario listar
//   fatec-turbo usuario remover <login>

func cmdUsuario(db *sql.DB, args []string) error {
	uso := errors.New("uso: usuario criar <login> [nome] | usuario senha <login> | usuario listar | usuario remover <login>")
	if len(args) == 0 {
		return uso
	}
	ctx := context.Background()
	switch args[0] {
	case "listar":
		rows, err := db.QueryContext(ctx, `SELECT login, nome, criado_em, ultimo_acesso FROM usuarios ORDER BY login`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var l, n string
			var c time.Time
			var u sql.NullTime
			rows.Scan(&l, &n, &c, &u)
			ult := "nunca"
			if u.Valid {
				ult = u.Time.Format("02/01/2006 15:04")
			}
			fmt.Printf("%-20s %-25s criado %s  último acesso %s\n", l, n, c.Format("02/01/2006"), ult)
		}
		return rows.Err()
	case "criar", "senha":
		if len(args) < 2 {
			return uso
		}
		login := strings.ToLower(strings.TrimSpace(args[1]))
		if login == "" || strings.ContainsAny(login, " \t") {
			return errors.New("login sem espaços")
		}
		senha, err := lerSenha()
		if err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(senha), 12)
		if err != nil {
			return err
		}
		if args[0] == "criar" {
			nome := strings.Join(args[2:], " ")
			if nome == "" {
				nome = args[1]
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO usuarios(login, nome, senha_hash) VALUES($1,$2,$3)`, login, nome, string(hash)); err != nil {
				return fmt.Errorf("não criei (login já existe?): %w", err)
			}
			fmt.Println("usuário criado:", login)
			return nil
		}
		res, err := db.ExecContext(ctx, `UPDATE usuarios SET senha_hash=$2 WHERE login=$1`, login, string(hash))
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errors.New("login não existe")
		}
		db.ExecContext(ctx, `DELETE FROM sessoes WHERE usuario_id=(SELECT id FROM usuarios WHERE login=$1)`, login)
		fmt.Println("senha trocada (sessões abertas foram encerradas):", login)
		return nil
	case "remover":
		if len(args) < 2 {
			return uso
		}
		res, err := db.ExecContext(ctx, `DELETE FROM usuarios WHERE login=$1`, strings.ToLower(args[1]))
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return errors.New("login não existe")
		}
		fmt.Println("usuário removido, com todo o progresso:", args[1])
		return nil
	}
	return uso
}

// lerSenha pede a senha sem eco no terminal; sem terminal (pipe), lê uma linha da entrada padrão.
func lerSenha() (string, error) {
	if s := os.Getenv("FT_SENHA"); s != "" {
		return checaSenha(s)
	}
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Print("Senha: ")
		a, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		fmt.Print("Repita: ")
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		if string(a) != string(b) {
			return "", errors.New("as senhas não conferem")
		}
		return checaSenha(string(a))
	}
	l, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && l == "" {
		return "", errors.New("informe a senha pela entrada padrão ou FT_SENHA")
	}
	return checaSenha(strings.TrimRight(l, "\r\n"))
}

func checaSenha(s string) (string, error) {
	if len([]rune(s)) < 8 {
		return "", errors.New("senha precisa de pelo menos 8 caracteres")
	}
	return s, nil
}
