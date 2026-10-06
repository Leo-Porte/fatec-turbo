"""Segmentação melhorada (colunas, banners, trim). Uso: python fix.py <prova> [spec.json]
spec: {"streams":{"pag":[[x0,x1,y0,y1],...]}, "red":[pags], "end_page":n,
       "q":{"NN":{"r":[[pag,x0,x1,y0,y1],...],"ctx":["A","B"]}},
       "ctx":{"A":[[pag,x0,x1,y0,y1],...]},
       "attach":{"NN":["A"]}, "noctx":[NN], "drop_ctx_text":[...], "extra_ev":[[stream_idx,y,"q"|"c"|"b",num,"x"]]}
"""
import json, os, re, sys, pymupdf
pid = sys.argv[1]
ROOT = 'D:/Estudo/Fatec/Site'
specf = sys.argv[2] if len(sys.argv) > 2 else f'{ROOT}/overrides/{pid}_fix.json'
spec = json.load(open(specf, encoding='utf8')) if os.path.exists(specf) else {}
pdf = f'D:/Estudo/Fatec/Provas/{pid}sem/Fatec_{pid}_Prova.pdf'
doc = pymupdf.open(pdf)
od = f'{ROOT}/q/{pid}'
os.makedirs(od, exist_ok=True)
DPI = 120
CTX = re.compile(r"^(Leia|Considere|Observe|Analise|Examine|Texto|Para responder|Atente|Veja|Responda|Imagine|Suponha)", re.I)
HDR = re.compile(r"^Quest[ãa]o\s*(\d{1,2})\b", re.I)
GRP = re.compile(r"(?:quest[õo]es|n[úu]meros)\s+(?:de\s+(?:n[úu]meros\s+)?)?(\d{1,2})\s*(?:a|e|até)\s*(\d{1,2})", re.I)
BAN = re.compile(r"^(?=.*[A-ZÀ-Ü]{5})[A-ZÀ-Ü][A-ZÀ-Ü \-]{4,}$")


def bounds(p):
    H = p.rect.height
    top, bot = 0, H
    for b in p.get_text("blocks"):
        t = b[4].strip()
        if re.search(r"VESTIBULAR|Fatec\s*$", t) or re.fullmatch(r"\d{1,2}", t):
            if b[1] > H * .85:
                bot = min(bot, b[1] - 4)
            elif b[3] < H * .12:
                top = max(top, b[3] + 2)
    return top, bot


PB = [bounds(p) for p in doc]


def items(pi):
    p = doc[pi]
    out = []
    for b in p.get_text("blocks"):
        if b[4].strip() or b[6] == 1:
            out.append(pymupdf.Rect(b[:4]))
    for ii in p.get_image_info():
        out.append(pymupdf.Rect(ii['bbox']))
    for d in p.get_drawings():
        r = d['rect']
        if r.width > p.rect.width * .97 and r.height > 100:
            continue
        if r.width < 1 and r.height < 1:
            continue
        out.append(pymupdf.Rect(r))
    return out


IT = {}


def content(pi, x0, x1, y0, y1):
    if pi not in IT:
        IT[pi] = items(pi)
    top, bot = PB[pi]
    y0 = max(y0, top)
    y1 = min(y1, bot)
    ys = []
    for r in IT[pi]:
        cx = (r.x0 + r.x1) / 2
        if x0 - 2 <= cx <= x1 + 2 and r.y1 > y0 + 1 and r.y0 < y1:
            ys.append((max(r.y0, y0), min(r.y1, y1)))
    if not ys:
        return None
    return min(a for a, _ in ys), max(b for _, b in ys)


def crop(pi, x0, x1, y0, y1, name, trim=True):
    p = doc[pi]
    if trim:
        c = content(pi, x0, x1, y0, y1)
        if not c:
            return None
        y0 = max(y0, c[0] - 3)
        y1 = min(y1, c[1] + 4)
    if y1 - y0 < 10:
        return None
    pix = p.get_pixmap(dpi=DPI, clip=pymupdf.Rect(x0, y0, x1, y1), colorspace=pymupdf.csGRAY)
    if pix.is_unicolor:
        return None
    fn = name + '.jpg'
    pix.save(f'{od}/{fn}', jpg_quality=62)
    return fn


W = doc[0].rect.width
H = doc[0].rect.height
red = spec.get('red')
red_auto = []
for pi, p in enumerate(doc):
    full = p.get_text("text")
    if re.search(r"PROPOSTA DE REDA|Instruções para a Redação|TEMA DA REDAÇÃO|^\s*REDAÇÃO\s*$", full, re.M | re.I) and not re.search(r"RASCUNHO", full[:400]):
        red_auto.append(pi)
if red is None:
    red = red_auto
first_red = min(red) if red else len(doc)
endp = spec.get('end_page', first_red - 1)
streams = []
for pi in range(0, endp + 1):
    for r in spec.get('streams', {}).get(str(pi), [[20, W - 20, 0, H]]):
        streams.append((pi, *r))

ev = []
after_e = True
for si, (pi, x0, x1, y0, y1) in enumerate(streams):
    p = doc[pi]
    bl = [b for b in p.get_text("blocks") if x0 - 2 <= (b[0] + b[2]) / 2 <= x1 + 2 and y0 - 2 <= b[1] < y1]
    bl.sort(key=lambda b: (b[1], b[0]))
    for b in bl:
        t = b[4].strip().replace('\n', ' ')
        if b[6] == 1:
            continue
        m = HDR.match(t)
        if m and len(re.findall(r'Quest[ãa]o\s*\d', t)) > 1:
            continue
        if m:
            ev.append((si, b[1] - 5, 'q', int(m.group(1)), t))
            after_e = False
            continue
        if re.search(r"\(E\)", t):
            after_e = True
            continue
        if BAN.match(t) and t != 'REDAÇÃO' and len(t.split()) <= 2 and len(t) < 40 and b[3] - b[1] < 30:
            ev.append((si, b[1] - 8, 'b', None, t))
            continue
        if CTX.match(t) and (after_e or re.search(r"quest[õo]es|para responder", t, re.I)):
            ev.append((si, b[1] - 4, 'c', None, t))
            after_e = False
for si, (pi, x0, x1, y0, y1) in enumerate(streams):
    ws = doc[pi].get_text("words")
    for i, w in enumerate(ws[:-1]):
        if re.fullmatch(r"Quest[ãa]o", w[4], re.I) and re.fullmatch(r"\d{2}", ws[i + 1][4]) and x0 - 2 <= w[0] <= x1 and y0 - 2 <= w[1] < y1:
            n = int(ws[i + 1][4])
            if not any(e[2] == 'q' and e[3] == n for e in ev):
                ev.append((si, w[1] - 5, 'q', n, f'Questão {n:02d}'))
for c in spec.get('extra_ev', []):
    ev.append((c[0], c[1], c[2], c[3], c[4] if len(c) > 4 else 'extra'))
ev = [e for e in ev if not (e[2] == 'c' and any(s in e[4] for s in spec.get('drop_ctx_text', [])))]
ev = [(e[0], e[1] + spec.get('qdy', {}).get(str(e[3]), 0), *e[2:]) if e[2] == 'q' else e for e in ev]
ev.sort(key=lambda e: (e[0], e[1]))
_m = []
for e in ev:
    if e[2] == 'c' and any(q[2] == 'q' and q[0] == e[0] and 0 <= e[1] - q[1] < 60 for q in ev):
        continue
    if False:
        pass

    if e[2] == 'c' and _m and _m[-1][2] == 'c' and _m[-1][0] == e[0] and e[1] - _m[-1][1] < 40:
        continue
    _m.append(e)
ev = _m


def has_e(pi, x0, x1, a, b):
    for bl in doc[pi].get_text("blocks"):
        if x0 - 2 <= (bl[0] + bl[2]) / 2 <= x1 + 2 and a - 2 <= bl[1] < b and re.search(r"\(E\)", bl[4]):
            return True
    return False


def seg(start, end, base, stop_e=False):
    files = []
    k = 0
    for si in range(start[0], (end[0] if end else len(streams) - 1) + 1):
        pi, x0, x1, y0, y1 = streams[si]
        a = start[1] if si == start[0] else y0
        b = end[1] if (end and si == end[0]) else y1
        f = crop(pi, x0, x1, a, b, f'{base}-{k}')
        if f:
            files.append(f)
            k += 1
        if stop_e and has_e(pi, x0, x1, a, b):
            break
    return files


qs = {}
ctxs = []
named = {}
for f in os.listdir(od):
    if f.endswith('.jpg'):
        os.remove(f'{od}/{f}')
pending = None
for i, e in enumerate(ev):
    if e[2] == 'b':
        continue
    nxt = (ev[i + 1][0], ev[i + 1][1]) if i + 1 < len(ev) else None
    if e[2] == 'c':
        idx = len(ctxs)
        files = seg((e[0], e[1]), nxt, f'ctx-{idx:02d}')
        pi_, x0_, x1_, y0_, y1_ = streams[e[0]]
        near = ' '.join(b[4].replace(chr(10), ' ') for b in doc[pi_].get_text("blocks") if e[1] - 5 <= b[1] <= e[1] + 70 and x0_ - 2 <= (b[0] + b[2]) / 2 <= x1_ + 2)
        g = GRP.search(near)
        ctxs.append({'id': idx, 'imgs': files, 'range': [int(g.group(1)), int(g.group(2))] if g else None})
        pending = idx
    else:
        n = e[3]
        files = seg((e[0], e[1]), nxt, f'q{n:02d}', True)
        q = {'n': n, 'imgs': files, 'ctx': []}
        if pending is not None:
            q['ctx'].append(pending)
            pending = None
        qs[n] = q
for nm, regs in spec.get('ctx', {}).items():
    idx = len(ctxs)
    files = []
    for k, (pi, x0, x1, y0, y1) in enumerate(regs):
        f = crop(pi, x0, x1, y0, y1, f'ctx-{nm}-{k}')
        if f:
            files.append(f)
    ctxs.append({'id': idx, 'imgs': files, 'range': None})
    named[nm] = idx
for nn, d in spec.get('q', {}).items():
    n = int(nn)
    files = []
    for k, (pi, x0, x1, y0, y1) in enumerate(d['r']):
        f = crop(pi, x0, x1, y0, y1, f'q{n:02d}-m{k}', not d.get('notrim'))
        if f:
            files.append(f)
    q = qs.setdefault(n, {'n': n, 'imgs': [], 'ctx': []})
    q['imgs'] = files
    if 'ctx' in d:
        q['ctx'] = [named[c] if isinstance(c, str) else c for c in d['ctx']]
for nn in spec.get('noctx', []):
    qs[nn]['ctx'] = []
for nn, l in spec.get('attach', {}).items():
    for c in l:
        c = named[c] if isinstance(c, str) else c
        if c not in qs[int(nn)]['ctx']:
            qs[int(nn)]['ctx'].append(c)
for cx in ctxs:
    if cx['range'] and cx['id'] not in spec.get('norange', []):
        a, b = cx['range']
        for n in range(a, b + 1):
            if n in qs and cx['id'] not in qs[n]['ctx'] and n not in spec.get('noctx', []):
                qs[n]['ctx'].append(cx['id'])
for q in qs.values():
    q['ctx'] = sorted(set(q['ctx']))
rd = []
for k, pi in enumerate(red):
    top, bot = PB[pi]
    pix = doc[pi].get_pixmap(dpi=DPI, clip=pymupdf.Rect(20, top, W - 20, bot), colorspace=pymupdf.csGRAY)
    fn = f'red-{k}.jpg'
    pix.save(f'{od}/{fn}', jpg_quality=62)
    rd.append(fn)
man = {'prova': pid, 'questoes': [qs[n] for n in sorted(qs)], 'contextos': ctxs, 'redacao': rd,
       'n_questoes': len(qs), 'faltando': [n for n in range(1, max(qs) + 1) if n not in qs]}
json.dump(man, open(f'{od}/manifest.json', 'w', encoding='utf8'), ensure_ascii=False, indent=1)
print(pid, 'q', len(qs), 'ctx', len(ctxs), 'red', rd, 'falt', man['faltando'])
