"""Recorta cada questão das provas Fatec em imagens.

Uso: python -E -X utf8 recorta.py <pdf> <saida_dir> <prova_id>
Ajustes manuais por prova em overrides/<prova_id>.json (todas as chaves opcionais):
  extra_cuts: [[pagina, y, "c"|"q", numero_ou_null], ...]  pontos de corte extras (y em pontos PDF, página 0-based)
  manual: {"NN": [[pagina, y0, y1], ...]}   recorte manual da questão NN (substitui o automático)
  manual_ctx: {"NN": [[pagina, y0, y1], ...]}  texto de apoio manual exibido antes da questão NN
  red_pages: [pagina, ...]  páginas da proposta de redação
  end_page: pagina  última página com questões
Gera <saida_dir>/<prova_id>/qNN-k.jpg, ctx-NN-k.jpg, red-k.jpg e <saida_dir>/<prova_id>/manifest.json
"""
import json, os, re, sys
import pymupdf

pdf, out, pid = sys.argv[1], sys.argv[2], sys.argv[3]
_ovf = os.path.join(os.path.dirname(os.path.abspath(__file__)), "overrides", f"{pid}.json")
ov = json.load(open(_ovf, encoding="utf-8")) if os.path.exists(_ovf) else {}
DPI = 120
CTX_START = re.compile(r"^(Leia|Considere|Observe|Analise|Examine|Texto|Para responder|Atente|Veja|Responda|Imagine|Suponha)", re.I)
HDR = re.compile(r"^Quest[ãa]o\s*(\d{1,2})\b", re.I)
GROUP = re.compile(r"quest[õo]es\s+(?:de\s+)?(\d{1,2})\s*(?:a|e|até)\s*(\d{1,2})", re.I)

doc = pymupdf.open(pdf)
od = os.path.join(out, pid)
os.makedirs(od, exist_ok=True)

def page_bounds(p):
    """Área útil: abaixo do cabeçalho e acima do rodapé."""
    H = p.rect.height
    top, bot = 0, H
    for b in p.get_text("blocks"):
        t = b[4].strip()
        if re.search(r"VESTIBULAR|Fatec\s*$", t) or re.fullmatch(r"\d{1,2}", t):
            if b[1] > H * 0.85: bot = min(bot, b[1] - 4)
            elif b[3] < H * 0.12: top = max(top, b[3] + 2)
    return top, bot

# 1. Linha do tempo de marcadores em ordem de leitura (página, y)
events = []  # (page, y, kind, num, text)
first_q_page = None
red_pages = []
for pi, p in enumerate(doc):
    blocks = sorted(p.get_text("blocks"), key=lambda b: (b[1], b[0]))
    full = p.get_text("text")
    if re.search(r"PROPOSTA DE REDA|^\s*REDAÇÃO\s*$|Instruções para a Redação|TEMA DA REDAÇÃO", full, re.M | re.I) and not re.search(r"RASCUNHO", full[:400], re.I):
        red_pages.append(pi)
    seen_e = False
    for b in blocks:
        t = b[4].strip()
        m = HDR.match(t)
        if m:
            events.append((pi, b[1], "q", int(m.group(1)), t))
            first_q_page = pi if first_q_page is None else first_q_page
            seen_e = False
            continue
        if re.search(r"(^|\n)\s*\(E\)", t):
            seen_e = True
            events.append((pi, b[3], "e", None, t))
            continue
        if CTX_START.match(t):
            events.append((pi, b[1], "c", None, t))

# Rótulo "Questão" e número em blocos separados
if not any(e[2] == "q" for e in events):
    pass
for pi, p in enumerate(doc):
    words = p.get_text("words")
    for i, w in enumerate(words[:-1]):
        if re.fullmatch(r"Quest[ãa]o", w[4], re.I) and re.fullmatch(r"\d{2}", words[i + 1][4]):
            n = int(words[i + 1][4])
            if not any(e[2] == "q" and e[3] == n for e in events):
                events.append((pi, w[1] - 2, "q", n, f"Questão {n:02d}"))
events.sort(key=lambda e: (e[0], e[1]))

# 2. Pontos de corte: cabeçalho de questão e início de contexto após o (E) da anterior
cuts = []  # (page, y, kind, num)
after_e = True
for pi, y, k, n, t in events:
    if k == "q":
        cuts.append((pi, y - 3, "q", n, t)); after_e = False
    elif k == "e":
        after_e = True
    elif k == "c" and after_e:
        cuts.append((pi, y - 3, "c", None, t)); after_e = False
for c in ov.get("extra_cuts", []):  # [page, y, "c"|"q", num]
    cuts.append((c[0], c[1], c[2], c[3], "override"))
cuts.sort(key=lambda c: (c[0], c[1]))
last_q_page = max((c[0] for c in cuts if c[2] == "q"), default=len(doc) - 1)
end_page = ov.get("end_page", min(red_pages) if red_pages and min(red_pages) > last_q_page else last_q_page)

def crop(pi, y0, y1, name):
    p = doc[pi]
    top, bot = page_bounds(p)
    y0, y1 = max(y0, top), min(y1, bot)
    if y1 - y0 < 12:
        return None
    clip = pymupdf.Rect(20, y0, p.rect.width - 20, y1)
    pix = p.get_pixmap(dpi=DPI, clip=clip, colorspace=pymupdf.csGRAY)
    # descarta faixa em branco (sobra de topo/rodapé de página)
    if pix.is_unicolor or (pix.height < 45 and min(pix.samples) > 200):
        return None
    fn = f"{name}.jpg"
    pix.save(os.path.join(od, fn), jpg_quality=62)
    return fn

def segment(start, end, base):
    """Recorta de start=(page,y) até end=(page,y) atravessando páginas."""
    files, k = [], 0
    for pi in range(start[0], end[0] + 1):
        y0 = start[1] if pi == start[0] else 0
        y1 = end[1] if pi == end[0] else 1e9
        if pi > end_page and pi != start[0]:
            break
        f = crop(pi, y0, y1, f"{base}-{k}")
        if f: files.append(f); k += 1
    return files

# 3. Gerar recortes
qs, ctxs, pending_ctx = {}, [], None
for i, c in enumerate(cuts):
    nxt = cuts[i + 1] if i + 1 < len(cuts) else (end_page, 1e9, "end", None, "")
    if nxt[0] > end_page:
        nxt = (end_page, 1e9, "end", None, "")
    if c[2] == "c":
        idx = len(ctxs)
        files = segment((c[0], c[1]), (nxt[0], nxt[1]), f"ctx-{idx:02d}")
        g = GROUP.search(c[4] + " " + " ".join(e[4] for e in events if e[0] == c[0] and abs(e[1] - c[1]) < 30))
        ctxs.append({"id": idx, "imgs": files, "range": [int(g.group(1)), int(g.group(2))] if g else None})
        pending_ctx = idx
    else:
        n = c[3]
        files = segment((c[0], c[1]), (nxt[0], nxt[1]), f"q{n:02d}")
        q = {"n": n, "imgs": files, "ctx": []}
        if pending_ctx is not None:
            q["ctx"].append(pending_ctx)
            pending_ctx = None
        qs[n] = q

# recortes manuais
for nn, regs in ov.get("manual", {}).items():
    n = int(nn); files = []
    for k, (pi, y0, y1) in enumerate(regs):
        f = crop(pi, y0, y1, f"q{n:02d}-m{k}")
        if f: files.append(f)
    qs.setdefault(n, {"n": n, "imgs": [], "ctx": []})["imgs"] = files
for nn, regs in ov.get("manual_ctx", {}).items():
    n = int(nn); idx = len(ctxs); files = []
    for k, (pi, y0, y1) in enumerate(regs):
        f = crop(pi, y0, y1, f"ctx-m{n:02d}-{k}")
        if f: files.append(f)
    ctxs.append({"id": idx, "imgs": files, "range": None})
    if n in qs: qs[n]["ctx"] = [idx]

# contextos de grupo valem para todas as questões do intervalo
for cx in ctxs:
    if cx["range"]:
        a, b = cx["range"]
        for n in range(a, b + 1):
            if n in qs and cx["id"] not in qs[n]["ctx"]:
                qs[n]["ctx"].insert(0, cx["id"])

# 4. Redação: páginas da proposta inteiras
red = []
for k, pi in enumerate(ov.get("red_pages", [p for p in red_pages if p > last_q_page or p == last_q_page])):
    p = doc[pi]
    top, bot = page_bounds(p)
    pix = p.get_pixmap(dpi=DPI, clip=pymupdf.Rect(20, top, p.rect.width - 20, bot), colorspace=pymupdf.csGRAY)
    fn = f"red-{k}.jpg"; pix.save(os.path.join(od, fn), jpg_quality=62); red.append(fn)

man = {"prova": pid, "questoes": [qs[n] for n in sorted(qs)], "contextos": ctxs, "redacao": red,
       "n_questoes": len(qs), "faltando": [n for n in range(1, max(qs) + 1) if n not in qs] if qs else []}
json.dump(man, open(os.path.join(od, "manifest.json"), "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print(pid, "questoes", len(qs), "contextos", len(ctxs), "redacao", len(red), "faltando", man["faltando"], "red_pages", red_pages)
