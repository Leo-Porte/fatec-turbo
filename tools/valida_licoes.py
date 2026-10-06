"""Confere as lições de dados/licoes contra o catálogo e o CSV de questões.

Uso (na raiz do repositório): python tools/valida_licoes.py [id ...]
"""
import csv
import glob
import json
import os
import re
import sys
from collections import Counter

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def catalogo():
    cat = {}
    for linha in open(os.path.join(RAIZ, "docs", "catalogo_licoes.md"), encoding="utf-8"):
        m = re.match(r"\|\s*(\d+)\s*\|\s*([a-z0-9-]+)\s*\|[^|]*\|\s*([^|]*)\|", linha)
        if m:
            req = [r.strip() for r in m.group(3).split(",") if r.strip() and r.strip() != "—"]
            cat[m.group(2)] = (int(m.group(1)), req)
    return cat


def main():
    cat = catalogo()
    q = {}
    with open(os.path.join(RAIZ, "dados", "questoes.csv"), encoding="utf-8-sig") as fh:
        for r in csv.DictReader(fh, delimiter=";"):
            q[(r["prova"], r["questao"])] = r
    ids = sys.argv[1:] or [os.path.basename(f)[:-5] for f in sorted(glob.glob(os.path.join(RAIZ, "dados", "licoes", "*.json")))]
    problemas = 0
    for i in ids:
        f = os.path.join(RAIZ, "dados", "licoes", i + ".json")
        erros = []
        try:
            l = json.load(open(f, encoding="utf-8"))
        except Exception as e:
            print(f"{i}: JSON INVÁLIDO {e}")
            problemas += 1
            continue
        if l.get("id") != i:
            erros.append(f"id {l.get('id')} difere do arquivo")
        if i in cat:
            s, req = cat[i]
            if l.get("semana") != s:
                erros.append(f"semana {l.get('semana')} != catálogo {s}")
            if sorted(l.get("requisitos") or []) != sorted(req):
                erros.append(f"requisitos {l.get('requisitos')} != catálogo {req}")
        else:
            erros.append("fora do catálogo")
        for k in ("titulo", "disciplina", "tema", "secoes", "decorar", "exercicios"):
            if not l.get(k):
                erros.append(f"sem {k}")
        exs = l.get("exercicios") or []
        if len(exs) != 5:
            erros.append(f"{len(exs)} exercícios (esperado 5)")
        for n, e in enumerate(exs, 1):
            if len(e.get("alternativas") or []) != 5:
                erros.append(f"ex{n}: {len(e.get('alternativas') or [])} alternativas")
            if e.get("gabarito") not in list("ABCDE"):
                erros.append(f"ex{n}: gabarito {e.get('gabarito')!r}")
            if not e.get("resolucao"):
                erros.append(f"ex{n}: sem resolução")
        gabs = Counter(e.get("gabarito") for e in exs)
        if gabs and max(gabs.values()) > 2:
            erros.append(f"gabaritos concentrados {dict(gabs)}")
        for r in l.get("resolvidas") or []:
            c = q.get((r.get("prova"), str(r.get("n"))))
            if not c:
                erros.append(f"resolvida {r.get('prova')} q{r.get('n')} não existe no CSV")
            elif "ANUL" in c["gabarito"].upper():
                erros.append(f"resolvida {r.get('prova')} q{r.get('n')} é anulada")
            elif not os.path.exists(os.path.join(RAIZ, "conteudo", "q", r["prova"], "manifest.json")):
                erros.append(f"prova {r['prova']} sem recortes")
        res = len(l.get("resolvidas") or [])
        if not i.startswith("red-") and res < 2:
            erros.append(f"só {res} resolvidas")
        print(f"{i}: {'ok' if not erros else '; '.join(erros)}")
        problemas += bool(erros)
    print(f"\n{len(ids)} lições, {problemas} com problema")
    return 1 if problemas else 0


if __name__ == "__main__":
    sys.exit(main())
