"""fix.py <prova> <spec.json>  - recorta regioes explicitas (pagina,x0,y0,x1,y1[,[brancos]]) e atualiza manifest.
spec: {"q":{"NN":[regs]}, "ctx":[{"id":N|null,"name":s,"regs":[..],"range":[a,b]|null}], "qctx":{"NN":[id|name,..]}, "red":[regs]}"""
import json,sys,os,glob,pymupdf
pid,specf=sys.argv[1],sys.argv[2]
spec=json.load(open(specf,encoding='utf8'))
d=pymupdf.open(f'D:/Estudo/Fatec/Provas/{pid[:6]}sem/Fatec_{pid}_Prova.pdf')
o=f'D:/Estudo/Fatec/Site/q/{pid}/'
m=json.load(open(o+'manifest.json',encoding='utf8'))
S=120/72
def crop(reg,fn):
    pg,x0,y0,x1,y1=reg[:5]
    pix=d[pg].get_pixmap(dpi=120,clip=pymupdf.Rect(x0,y0,x1,y1),colorspace=pymupdf.csGRAY)
    for w in (reg[5] if len(reg)>5 else []):
        a=max(0,int((w[0]-x0)*S));b=max(0,int((w[1]-y0)*S));c=min(pix.width,int((w[2]-x0)*S)+1);e=min(pix.height,int((w[3]-y0)*S)+1)
        pix.set_rect(pymupdf.IRect(pix.x+a,pix.y+b,pix.x+c,pix.y+e),(255,))
    pix.save(o+fn,jpg_quality=62);return fn
def rm(pat):
    for f in glob.glob(o+pat): os.remove(f)
names={}
for c in spec.get('ctx',[]):
    if c.get('id') is None:
        c['id']=max([x['id'] for x in m['contextos']]+[-1])+1
        m['contextos'].append({'id':c['id'],'imgs':[],'range':None})
    names[c['name']]=c['id']
    ent=[x for x in m['contextos'] if x['id']==c['id']][0]
    rm(f"ctx-{c['id']:02d}-*.jpg")
    ent['imgs']=[crop(r,f"ctx-{c['id']:02d}-{k}.jpg") for k,r in enumerate(c['regs'])]
    ent['range']=c.get('range')
for nn,regs in spec.get('q',{}).items():
    n=int(nn);rm(f"q{n:02d}-*.jpg")
    [q]=[x for x in m['questoes'] if x['n']==n]
    q['imgs']=[crop(r,f"q{n:02d}-{k}.jpg") for k,r in enumerate(regs)]
for nn,l in spec.get('qctx',{}).items():
    [q]=[x for x in m['questoes'] if x['n']==int(nn)]
    q['ctx']=[names.get(i,i) for i in l]
for nn,fl in spec.get('dropimg',{}).items():
    [q]=[x for x in m['questoes'] if x['n']==int(nn)]
    for f in fl:
        if f in q['imgs']: q['imgs'].remove(f)
        if os.path.exists(o+f): os.remove(o+f)
if 'red' in spec:
    rm('red-*.jpg');m['redacao']=[crop(r,f'red-{k}.jpg') for k,r in enumerate(spec['red'])]
json.dump(m,open(o+'manifest.json','w',encoding='utf8'),ensure_ascii=False,indent=1)
print('ok')
