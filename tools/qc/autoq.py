import pymupdf,re,sys,json
# autoq.py prova "pg:Q:f" (ultima da pagina; fim real + mascara rodape) ou "pg:Q:N" (termina no cabecalho da questao N, mesma pagina)
pid=sys.argv[1]
d=pymupdf.open(f'D:/Estudo/Fatec/Provas/{pid[:6]}sem/Fatec_{pid}_Prova.pdf')
def hdr(p,n):
    ws=p.get_text("words")
    for i,w in enumerate(ws[:-1]):
        if w[4].startswith("Quest") and ws[i+1][4]==f"{n:02d}": return w[1]
spec={}
for it in sys.argv[2].split(','):
    pg,n,m=it.split(':');pg=int(pg);n=int(n);p=d[pg];bl=p.get_text("blocks")
    hy=hdr(p,n)-1.0
    if m=='f':
        foot=[b for b in bl if re.search(r"VESTIBULAR",b[4])]
        fx=foot[0][:4]
        body=[b for b in bl if b[3]>hy+4 and not re.search(r"VESTIBULAR",b[4]) and not re.fullmatch(r"\d{1,2}",b[4].strip())]
        y1=max(b[3] for b in body)+2
        fw=[w for w in p.get_text('words') if w[2]>fx[0]-4 and w[1]<fx[1]-2 and w[3]>hy and not re.search('VESTIB|Fatec',w[4])]
        my=(max(w[3] for w in fw)+0.8) if fw else 696
        mask=[[fx[0]-4,min(my,fx[1]-2),p.rect.width,p.rect.height]]
    elif m[0]=='y': y1=float(m[1:]);mask=[]
    else: y1=hdr(p,int(m))-0.8;mask=[]
    spec[str(n)]=[[pg,20,round(hy,1),p.rect.width-20,round(y1,1),mask]]
print(json.dumps(spec))
