import pymupdf,json
d=pymupdf.open('D:/Estudo/Fatec/Provas/2026-2sem/Fatec_2026-2_Prova.pdf')
o='D:/Estudo/Fatec/Site/q/2026-2/'
m=json.load(open(o+'manifest.json',encoding='utf8'))
def f(i): return [c for c in m['contextos'] if c['id']==i][0]['imgs'][0]
# ctx 0: pág 3, 28..310
p=d[3];p.get_pixmap(dpi=120,clip=pymupdf.Rect(20,25,p.rect.width-20,310),colorspace=pymupdf.csGRAY).save(o+f(0),jpg_quality=62)
# ctx 1: pág 4, 172..367, apaga selo da questão 11 à esquerda
p=d[4];pix=p.get_pixmap(dpi=120,clip=pymupdf.Rect(20,172,p.rect.width-20,366),colorspace=pymupdf.csGRAY)
s=120/72;y0=int((359-172)*s)
pix.clear_with(255,pymupdf.IRect(0,y0,int(100*s),pix.height)); pix.clear_with(255,pymupdf.IRect(pix.width-40,y0,pix.width,pix.height))
pix.save(o+f(1),jpg_quality=62)
print(f(0),f(1),m['questoes'][-1]['imgs'],m['redacao'],[c for c in m['contextos'] if c['range'] is None])
