"""Recortador column-aware (substitui a saida de recorta.py para uma prova).
Uso: python -E -X utf8 colcut.py <prova_id> [--dry]
Rode DEPOIS de recorta.py: sobrescreve q/<prova>/q*.jpg, ctx-*.jpg e manifest.json (mantem 'redacao').
Questao = do cabecalho 'Questao NN' ate o bloco da alternativa (E). Contexto = de 'Leia/Observe...' ate o proximo cabecalho.
Paginas de 2 colunas sao lidas coluna esquerda -> direita (recorta.py corta sempre largura total).
Ajustes opcionais em <prova_id>.cc.json ao lado: {"drop_blocks": [[pg, y], ...]}  ignora bloco que comeca em (pg, ~y)
"""
import json, os, re, sys, glob
import pymupdf

pid = sys.argv[1]; dry = "--dry" in sys.argv
HERE = os.path.dirname(os.path.abspath(__file__))
SITE = os.path.dirname(os.path.dirname(HERE))
pdf = f"D:/Estudo/Fatec/Provas/{pid}sem/Fatec_{pid}_Prova.pdf"
od = os.path.join(SITE, "q", pid)
cfgf = os.path.join(HERE, f"{pid}.cc.json")
cfg = json.load(open(cfgf, encoding="utf-8")) if os.path.exists(cfgf) else {}
DPI = 120
HDR = re.compile(r"^Quest[ãa]o\s*(\d{1,2})\b", re.I)
CTX = re.compile(r"^(Leia|Considere|Observe|Analise|Examine|Texto|Para responder|Atente|Veja|Responda|Imagine|Suponha)", re.I)
GROUP = re.compile(r"quest[õo]es\s+(?:de\s+)?(\d{1,2})\s*(?:a|e|até)\s*(\d{1,2})", re.I)
doc = pymupdf.open(pdf)


def footer_blocks(p):
    """blocos de rodape (VESTIBULAR ... Fatec + numero da pagina colado/proximo)"""
    H = p.rect.height
    bl = p.get_text("blocks")
    ref = [b for b in bl if re.search(r"VESTIBULAR|Fatec\s*$", b[4].strip()) and b[1] > H * 0.85]
    out = list(ref)
    for b in bl:
        if re.fullmatch(r"\d{1,2}", b[4].strip()) and b[1] > H * 0.85 and any(abs(b[1] - r[1]) < 20 for r in ref):
            out.append(b)
    return out


def bounds(p):
    H = p.rect.height; top, bot = 0, H
    for b in p.get_text("blocks"):
        t = b[4].strip()
        if re.search(r"VESTIBULAR|Fatec\s*$", t) or re.fullmatch(r"\d{1,2}", t):
            if b[1] > H * 0.85:
                if t in [x[4].strip() for x in footer_blocks(p)] or re.search(r"VESTIBULAR|Fatec\s*$", t): bot = min(bot, b[1] - 1)
            elif b[3] < H * 0.12: top = max(top, b[3] + 2)
    return top, bot


def bot_for(p, x0, x1):
    """limite inferior do recorte: acima do rodape, mas so se o rodape cruza a faixa x0..x1"""
    H = p.rect.height; bot = H - 6
    for b in p.get_text("blocks"):
        t = b[4].strip()
        if (re.search(r"VESTIBULAR|Fatec\s*$", t) or re.fullmatch(r"\d{1,2}", t)) and b[1] > H * 0.85 and b[2] > x0 and b[0] < x1:
            bot = min(bot, b[1] - 1)
    return bot


def is_banner(t, b):
    s = t.strip()
    return s == s.upper() and len(s) < 35 and not re.search(r"\d", s) and (b[2] - b[0]) > 200 and "\n" not in s


def elements(p, top, bot):
    """(x0,y0,x1,y1,texto,tipo) de texto, imagens e desenhos da pagina"""
    W, H = p.rect.width, p.rect.height
    els = [(b[0], b[1], b[2], b[3], b[4], "t") for b in p.get_text("blocks")]
    # bloco com cabecalho fundido a outro texto: quebra por linhas (dict)
    dblocks = [bb for bb in p.get_text("dict")["blocks"] if bb.get("type") == 0]
    fixed = []
    for e in els:
        txt = e[4].strip()
        if re.match(r"^Quest[ãa]o\s*\d{1,2}\s*$", txt, re.I) or not re.search(r"Quest[ãa]o\s*\d{1,2}", txt):
            fixed.append(e); continue
        bb = next((x for x in dblocks if abs(x["bbox"][0] - e[0]) < 1.5 and abs(x["bbox"][1] - e[1]) < 1.5), None)
        if bb is None: fixed.append(e); continue
        lines = [("".join(sp["text"] for sp in ln["spans"]), ln["bbox"]) for ln in bb["lines"]]
        seg = []
        def flush():
            if seg:
                fixed.append((min(l[1][0] for l in seg), min(l[1][1] for l in seg), max(l[1][2] for l in seg), max(l[1][3] for l in seg), "\n".join(l[0] for l in seg), "t"))
                seg.clear()
        i = 0
        while i < len(lines):
            t0 = lines[i][0].strip()
            m2 = re.fullmatch(r"Quest[ãa]o\s*(\d{1,2})?", t0, re.I)
            if m2 and (m2.group(1) or (i + 1 < len(lines) and re.fullmatch(r"\d{1,2}", lines[i + 1][0].strip()))):
                flush()
                if m2.group(1): hl, n = [lines[i]], m2.group(1); i += 1
                else: hl, n = [lines[i], lines[i + 1]], lines[i + 1][0].strip(); i += 2
                fixed.append((min(l[1][0] for l in hl), min(l[1][1] for l in hl), max(l[1][2] for l in hl), max(l[1][3] for l in hl), f"Questão {n}", "t"))
            else:
                seg.append(lines[i]); i += 1
        flush()
    els = fixed
    hdrs = [e for e in els if HDR.match(e[4].strip())]
    for im in p.get_image_info():
        x0, y0, x1, y1 = im["bbox"]
        if (x1 - x0) > W - 10 and (y1 - y0) > H - 100: continue
        if (x1 - x0) < 4 or (y1 - y0) < 4: continue
        els.append((x0, y0, x1, y1, "<img>", "i"))
    bars = [d["rect"] for d in p.get_drawings() if d["rect"].width > 300 and 8 <= d["rect"].height <= 40]
    for i, e in enumerate(els):
        if e[5] == "t" and "\n" not in e[4].strip() and e[4].strip().isupper() and 6 <= len(e[4].strip()) < 35 and (e[2] - e[0]) > 80 and not re.search(r"\d", e[4]):
            cx, cy = (e[0] + e[2]) / 2, (e[1] + e[3]) / 2
            if any(r.x0 <= cx <= r.x1 and r.y0 <= cy <= r.y1 for r in bars):
                els[i] = (e[0], e[1], e[2], e[3], e[4], "b")
    for d in p.get_drawings():
        r = d["rect"]
        if any(br.contains(r) for br in bars): continue
        if r.width < 1 and r.height < 1: continue
        if r.height > 250 and r.width < 4: continue
        if r.width > W - 30 and r.height > 100: continue
        if any(r.y0 >= h[1] - 6 and r.y1 <= h[3] + 6 for h in hdrs): continue
        els.append((r.x0, r.y0, r.x1, r.y1, "<dr>", "d"))
    return [e for e in els if e[3] > top and e[1] < bot]


RED = []


def streams():
    for pi, p in enumerate(doc):
        W = p.rect.width; top, bot = bounds(p)
        bl = elements(p, top, bot)
        red = [b for b in bl if b[5] in ("t", "b") and re.fullmatch(r"REDA\S+", b[4].strip()) and b[4].strip().isupper()]
        ry = min((b[1] for b in red), default=None)
        if red and not RED: RED.append((pi, ry))
        if ry is not None: bl = [b for b in bl if b[1] < ry - 1]
        # zonas de 2 colunas: divisor vertical (inteiro ou parcial) + "splits" do .cc.json
        zones = []
        for d in p.get_drawings():
            r = d["rect"]
            if r.height > 150 and r.width < 4 and abs((r.x0 + r.x1) / 2 - W / 2) < 40:
                z = ((r.x0 + r.x1) / 2, r.y0 - 25, r.y1 + 1)
                if not any(abs(z[0] - o[0]) < 3 and abs(z[1] - o[1]) < 6 for o in zones): zones.append(z)
        for sp in cfg.get("splits", []):
            if sp[0] == pi: zones.append((sp[1], sp[2], sp[3] if len(sp) > 3 else bot))
        zones.sort(key=lambda z: z[1])
        key = lambda b: (b[1], b[0])
        used = set()
        pos = top
        for (dx, zy0, zy1) in zones:
            cand = [e for e in bl if id(e) not in used and e[1] >= zy0 - 2 and e[1] <= zy1 and (e[2] - e[0]) < 380]
            wide = [e for e in bl if id(e) not in used and e[1] >= zy0 - 2 and e[1] <= zy1 and (e[2] - e[0]) >= 380]
            crossing = [e for e in cand + wide if e[5] == "t" and e[0] < dx - 5 and e[2] > dx + 5]
            nonc = [e for e in cand if e not in crossing and not (e[5] == "t" and e[0] < dx - 5 and e[2] > dx + 5)]
            ymin = min([e[1] for e in nonc if e[5] == "t"], default=zy1)
            crossing = [e for e in crossing if e[3] <= ymin + 4]
            inz = [e for e in cand if e not in crossing]
            above = [e for e in bl if id(e) not in used and id(e) not in {id(x) for x in inz} and (e[1] < zy0 - 2 or e in crossing)]
            for e in inz + above: used.add(id(e))
            if above:
                zy0 = max([zy0] + [e[3] + 6 for e in above if e[3] > zy0 and e[5] == "t"])
                yield pi, 20, W - 20, pos, zy0, sorted(above, key=key)
            L = [e for e in inz if (e[0] + e[2]) / 2 < dx]
            R = [e for e in inz if (e[0] + e[2]) / 2 >= dx]
            yield pi, 20, dx - 5, zy0, zy1, sorted(L, key=key)
            yield pi, dx + 5, W - 20, zy0, zy1, sorted(R, key=key)
            pos = zy1
        rest = [e for e in bl if id(e) not in used]
        if rest or not zones: yield pi, 20, W - 20, pos, bot, sorted(rest, key=key)
        if red: return


items = []
cur = None
dropped = []
banners = []
for pi, x0, x1, top, bot, blocks in streams():
    if cur is not None and (cur["kind"] == "c" or not cur["seenE"]):
        cur["regs"].append([pi, top, top, x0, x1])
    elif cur is not None:
        cur = None
    for b in blocks:
        t = b[4].strip(); by0, by1 = b[1], b[3]; typ = b[5]
        if any(d[0] == pi and abs(d[1] - by0) < 3 for d in cfg.get("drop_blocks", [])):
            continue
        m = HDR.match(t) if typ == "t" else None
        if m:
            cur = {"kind": "q", "n": int(m.group(1)), "regs": [[pi, by0 - 1, by1, x0, x1]], "seenE": False, "text": t}
            items.append(cur); continue
        if typ == "b" or (typ == "t" and is_banner(t, b)):
            banners.append((pi, x0, by0)); cur = None; continue
        if typ == "t" and CTX.match(t) and (cur is None or (cur["kind"] == "q" and cur["seenE"])):
            cur = {"kind": "c", "n": None, "regs": [[pi, by0 - 3, by1, x0, x1]], "seenE": False, "text": t}
            items.append(cur); continue
        if cur is None:
            continue
        r = cur["regs"][-1]
        if cur["kind"] == "q" and cur["seenE"]:
            if by0 < r[2] + 9 and not (typ == "t" and HDR.match(t)):
                r[2] = max(r[2], by1); cur["eY1"] = max(cur["eY1"], by1)
            else: dropped.append((pi, round(by0), t[:50]))
            continue
        r[2] = max(r[2], by1)
        cur["text"] += "\n" + t
        if cur["kind"] == "q" and re.search(r"(^|\n)\s*\(E\)", t):
            cur["seenE"] = True; cur["eY1"] = by1

for it in items:
    for r in it["regs"]:
        top, _ = bounds(doc[r[0]]); bot = doc[r[0]].rect.height - 4
        r[2] = min(r[2] + (10 if it["kind"] == "c" else 5), bot)
        for (bp, bx, by) in banners:
            if bp == r[0] and bx == r[3] and r[1] < by < r[2]: r[2] = by - 4
        r[1] = max(r[1], top)
for i, it in enumerate(items):
    for r in it["regs"]:
        for jt in items[i + 1:]:
            for s in jt["regs"]:
                if s[0] == r[0] and s[3] == r[3] and r[1] < s[1] < r[2]:
                    r[2] = s[1] - 1
lastq = max(i for i, it in enumerate(items) if it["kind"] == "q")
items = items[: lastq + 1]
plan = []
for it in items:
    g = GROUP.search(it["text"]) if it["kind"] == "c" else None
    plan.append({"kind": it["kind"], "n": it["n"], "regs": [[round(v, 1) if isinstance(v, float) else v for v in r] for r in it["regs"]],
                 "range": [int(g.group(1)), int(g.group(2))] if g else None, "seenE": it["seenE"]})
json.dump(plan, open(os.path.join(HERE, f"{pid}.plan.json"), "w"), indent=0)
print("blocos descartados depois do (E):", dropped)
print("questoes sem (E):", [i["n"] for i in plan if i["kind"] == "q" and not i["seenE"]])
print("n questoes:", sum(1 for i in plan if i["kind"] == "q"), "ctx:", sum(1 for i in plan if i["kind"] == "c"))
if dry: sys.exit()


def footers(p):
    return [b[:4] for b in footer_blocks(p)]


def crop(reg, name):
    from PIL import Image, ImageDraw
    pi, y0, y1, x0, x1 = reg
    if y1 - y0 < 12: return None
    p = doc[pi]
    if y1 - y0 < 60 and not p.get_text("words", clip=pymupdf.Rect(x0, y0, x1, y1)) and not [i for i in p.get_image_info() if pymupdf.Rect(i["bbox"]).intersects(pymupdf.Rect(x0, y0, x1, y1)) and i["bbox"][3] - i["bbox"][1] > 12]:
        return None
    pix = p.get_pixmap(dpi=DPI, clip=pymupdf.Rect(x0, y0, x1, y1), colorspace=pymupdf.csGRAY)
    if pix.is_unicolor: return None
    im = Image.frombytes("L", (pix.width, pix.height), pix.samples)
    k = DPI / 72
    dr = ImageDraw.Draw(im)
    for f in footers(p):   # apaga o rodape (a numeracao/rodape da pagina nao faz parte da questao)
        dr.rectangle([(f[0] - 6 - x0) * k, (f[1] - 2 - y0) * k, (f[2] + 40 - x0) * k, (f[3] + 8 - y0) * k], fill=255)
    px = im.load(); w, h = im.size
    rows = [any(px[x, r] < 235 for x in range(w)) for r in range(h)]
    if not any(rows): return None
    a = rows.index(True); b = h - rows[::-1].index(True)
    if b - a < 24 and (y1 - y0) < 60: return None
    im = im.crop((0, max(0, a - 6), w, min(h, b + 6)))
    im.save(os.path.join(od, name), quality=62); return name


for f in glob.glob(os.path.join(od, "q*.jpg")) + glob.glob(os.path.join(od, "ctx-*.jpg")):
    os.remove(f)
qs, ctxs, pend = {}, [], None
for it in plan:
    if it["kind"] == "c":
        idx = len(ctxs)
        files = [f for f in (crop(r, f"ctx-{idx:02d}-{k}.jpg") for k, r in enumerate(it["regs"])) if f]
        ctxs.append({"id": idx, "imgs": files, "range": it["range"]}); pend = idx
    else:
        n = it["n"]
        files = [f for f in (crop(r, f"q{n:02d}-{k}.jpg") for k, r in enumerate(it["regs"])) if f]
        q = {"n": n, "imgs": files, "ctx": []}
        if pend is not None: q["ctx"].append(pend); pend = None
        qs[n] = q
for cx in ctxs:
    if cx["range"]:
        a, b = cx["range"]
        for n in range(a, b + 1):
            if n in qs and cx["id"] not in qs[n]["ctx"]: qs[n]["ctx"].insert(0, cx["id"])
old = json.load(open(os.path.join(od, "manifest.json"), encoding="utf-8"))
redf = []
for f in glob.glob(os.path.join(od, "red-*.jpg")):
    os.remove(f)
regs = cfg.get("red")
if regs is None and not RED:
    lastp = max(r[0] for it in plan for r in it["regs"])
    for pi in range(lastp, len(doc)):
        if re.search(r"PROPOSTA DE REDA|TEMA DA REDA|Instruções para a Reda", doc[pi].get_text("text")):
            RED.append((pi, 0)); break
if regs is None and RED:
    rp, ry = RED[0]; regs = []
    for pi in range(rp, len(doc)):
        txt = doc[pi].get_text("text")
        if pi > rp and re.search(r"RASCUNHO", txt[:300]): break
        top, bot = bounds(doc[pi]); W = doc[pi].rect.width
        regs.append([pi, (max(top, ry - 4) if pi == rp and ry > 100 else top), bot, 20, W - 20])
for k, r in enumerate(regs or []):
    f = crop(r, f"red-{k}.jpg")
    if f: redf.append(f)
if not redf: redf = old.get("redacao", [])
man = {"prova": pid, "questoes": [qs[n] for n in sorted(qs)], "contextos": ctxs, "redacao": redf,
       "n_questoes": len(qs), "faltando": [n for n in range(1, max(qs) + 1) if n not in qs]}
json.dump(man, open(os.path.join(od, "manifest.json"), "w", encoding="utf-8"), ensure_ascii=False, indent=1)
print("ok", len(qs), "faltando", man["faltando"])
