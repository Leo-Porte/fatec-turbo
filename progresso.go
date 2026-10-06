package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

// O progresso do aluno é o mesmo documento JSON que o app.js mantém no navegador
// ({v, rotina, lidas, ex, testes, sims, reds}). O servidor guarda uma cópia por usuário,
// com versão para detectar edição simultânea em dois aparelhos.

func carregaProgresso(ctx context.Context, uid int64) (json.RawMessage, int, error) {
	var d []byte
	var v int
	err := DB.QueryRowContext(ctx, `SELECT dados, versao FROM progresso WHERE usuario_id=$1`, uid).Scan(&d, &v)
	if errors.Is(err, sql.ErrNoRows) {
		return json.RawMessage(`null`), 0, nil
	}
	return d, v, err
}

type pedidoProgresso struct {
	Base  int                        `json:"base"`
	Dados map[string]json.RawMessage `json:"dados"`
}

// apiProgresso: PUT/POST {base, dados}. Se ninguém gravou desde a versão base, substitui (apagar funciona).
// Se outro aparelho gravou no meio, mescla por chave e devolve o resultado com mesclado=true.
func apiProgresso(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	if DB == nil || u == nil {
		http.Error(w, "sem banco", http.StatusNotFound)
		return
	}
	var p pedidoProgresso
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&p); err != nil || p.Dados == nil {
		http.Error(w, "pedido inválido", 400)
		return
	}
	ctx := r.Context()
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		http.Error(w, "erro interno", 500)
		return
	}
	defer tx.Rollback()
	var atual []byte
	var versao int
	err = tx.QueryRowContext(ctx, `SELECT dados, versao FROM progresso WHERE usuario_id=$1 FOR UPDATE`, u.ID).Scan(&atual, &versao)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "erro interno", 500)
		return
	}
	final := p.Dados
	mesclado := false
	if atual != nil && p.Base != versao {
		var srv map[string]json.RawMessage
		if json.Unmarshal(atual, &srv) == nil {
			final = mesclaProgresso(srv, p.Dados)
			mesclado = true
		}
	}
	b, _ := json.Marshal(final)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO progresso(usuario_id, dados, versao, atualizado_em) VALUES($1,$2,1,now())
		ON CONFLICT (usuario_id) DO UPDATE SET dados=EXCLUDED.dados, versao=progresso.versao+1, atualizado_em=now()`, u.ID, b); err != nil {
		http.Error(w, "erro interno", 500)
		return
	}
	if err := derivaTabelas(ctx, tx, u.ID, final, !mesclado); err != nil {
		log.Printf("derivar tabelas de %s: %v", u.Login, err)
		http.Error(w, "erro interno", 500)
		return
	}
	var nova int
	tx.QueryRowContext(ctx, `SELECT versao FROM progresso WHERE usuario_id=$1`, u.ID).Scan(&nova)
	if err := tx.Commit(); err != nil {
		http.Error(w, "erro interno", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	out := map[string]any{"versao": nova, "mesclado": mesclado}
	if mesclado {
		out["dados"] = json.RawMessage(b)
	}
	json.NewEncoder(w).Encode(out)
}

// mesclaProgresso junta dois estados sem perder trabalho: chaves de um ou de outro entram;
// em conflito no mesmo simulado vence o finalizado (ou o com mais tempo); na mesma redação, a mais recente.
func mesclaProgresso(srv, cli map[string]json.RawMessage) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for k, v := range srv {
		out[k] = v
	}
	for k, v := range cli {
		if k == "v" {
			out[k] = v
			continue
		}
		var a, b map[string]json.RawMessage
		if json.Unmarshal(out[k], &a) != nil || json.Unmarshal(v, &b) != nil || a == nil {
			out[k] = v
			continue
		}
		for id, item := range b {
			velho, existe := a[id]
			if !existe {
				a[id] = item
				continue
			}
			switch k {
			case "sims":
				if pesoSim(item) >= pesoSim(velho) {
					a[id] = item
				}
			case "reds":
				if campoNum(item, "atualizado") >= campoNum(velho, "atualizado") {
					a[id] = item
				}
			case "ex":
				var x, y map[string]json.RawMessage
				if json.Unmarshal(velho, &x) == nil && json.Unmarshal(item, &y) == nil && x != nil {
					for i, l := range y {
						x[i] = l
					}
					a[id], _ = json.Marshal(x)
				} else {
					a[id] = item
				}
			default:
				a[id] = item
			}
		}
		out[k], _ = json.Marshal(a)
	}
	return out
}

func campoNum(raw json.RawMessage, campo string) float64 {
	var m map[string]any
	json.Unmarshal(raw, &m)
	f, _ := m[campo].(float64)
	return f
}

func pesoSim(raw json.RawMessage) float64 {
	var m map[string]any
	json.Unmarshal(raw, &m)
	p := 0.0
	if m["status"] == "finalizado" {
		p = 1e12
	}
	s, _ := m["segundos"].(float64)
	return p + s
}

func msParaTempo(v any) any {
	f, ok := v.(float64)
	if !ok || f <= 0 {
		return nil
	}
	return time.UnixMilli(int64(f))
}

// derivaTabelas mantém tentativas e redacoes em linha com o progresso, para consultas.
// Com substituir=true, o que sumiu do progresso (resultado apagado) também some das tabelas.
func derivaTabelas(ctx context.Context, tx *sql.Tx, uid int64, dados map[string]json.RawMessage, substituir bool) error {
	var sims map[string]map[string]any
	json.Unmarshal(dados["sims"], &sims)
	var testes map[string]map[string]any
	json.Unmarshal(dados["testes"], &testes)
	var reds map[string]map[string]any
	json.Unmarshal(dados["reds"], &reds)
	if substituir {
		if _, err := tx.ExecContext(ctx, `DELETE FROM tentativas WHERE usuario_id=$1`, uid); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM redacoes WHERE usuario_id=$1`, uid); err != nil {
			return err
		}
	}
	const up = `INSERT INTO tentativas(usuario_id,id,tipo,prova,titulo,inicio,fim,segundos,acertos,total,nota,respostas,resultado)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (usuario_id,id) DO UPDATE SET tipo=EXCLUDED.tipo, prova=EXCLUDED.prova, titulo=EXCLUDED.titulo, inicio=EXCLUDED.inicio,
		fim=EXCLUDED.fim, segundos=EXCLUDED.segundos, acertos=EXCLUDED.acertos, total=EXCLUDED.total, nota=EXCLUDED.nota,
		respostas=EXCLUDED.respostas, resultado=EXCLUDED.resultado`
	for id, s := range sims {
		if s["status"] != "finalizado" {
			continue
		}
		res, _ := s["resultado"].(map[string]any)
		resp, _ := json.Marshal(s["respostas"])
		rj, _ := json.Marshal(res)
		tipo, _ := s["tipo"].(string)
		prova, _ := s["prova"].(string)
		titulo, _ := s["titulo"].(string)
		seg, _ := s["segundos"].(float64)
		ac, _ := res["acertos"].(float64)
		tot, _ := res["total"].(float64)
		nota, _ := res["nota"].(float64)
		if _, err := tx.ExecContext(ctx, up, uid, id, tipo, prova, titulo, msParaTempo(s["inicio"]), msParaTempo(s["fim"]),
			int(seg), int(ac), int(tot), int(nota), resp, rj); err != nil {
			return err
		}
	}
	for id, t := range testes {
		if t["corrigido"] == nil {
			continue
		}
		resp, _ := json.Marshal(t["resp"])
		ac, _ := t["acertos"].(float64)
		tot, _ := t["total"].(float64)
		nota := 0
		if tot > 0 {
			nota = int(ac*100/tot + .5)
		}
		titulo := id
		if tt, ok := C.PorTeste[id]; ok {
			titulo = tt.Titulo
		}
		if _, err := tx.ExecContext(ctx, up, uid, "teste-"+id, "teste", "", titulo, nil, msParaTempo(t["corrigido"]),
			0, int(ac), int(tot), nota, resp, []byte(`{}`)); err != nil {
			return err
		}
	}
	for prova, r := range reds {
		titulo, _ := r["titulo"].(string)
		texto, _ := r["texto"].(string)
		var nota any
		if n, ok := r["nota"].(float64); ok {
			nota = int(n)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO redacoes(usuario_id,prova,titulo,texto,nota,pdf_em,atualizado_em) VALUES($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (usuario_id,prova) DO UPDATE SET titulo=EXCLUDED.titulo, texto=EXCLUDED.texto, nota=EXCLUDED.nota,
			pdf_em=EXCLUDED.pdf_em, atualizado_em=EXCLUDED.atualizado_em`,
			uid, prova, titulo, texto, nota, msParaTempo(r["pdfEm"]), msParaTempo(r["atualizado"])); err != nil {
			return err
		}
	}
	return nil
}
