"""Baixa as provas oficiais do vestibular Fatec e gera os recortes de questões em conteudo/q.

Uso (na raiz do repositório):
    python -m pip install pymupdf
    python tools/baixar_provas.py

Os PDFs vêm do site oficial (vestibular.fatec.sp.gov.br, página "Provas e Gabaritos")
e ficam em provas_pdf/ (fora do git). O recorte usa tools/recorta.py e os ajustes de tools/overrides/.
"""
import os
import subprocess
import sys
import urllib.request

RAIZ = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
BASE = "https://fatweb.s3.amazonaws.com/vestibularfatec/gabarito/{cod}/Prova.pdf"

# código do arquivo no site oficial -> prova (ano-semestre)
PROVAS = {
    "202626241": "2026-2", "202619102": "2026-1", "202528719": "2025-2",
    "202426517": "2024-2", "202415903": "2024-1", "202327519": "2023-2",
    "202315287": "2023-1", "202228712": "2022-2", "202016520": "2020-1",
    "201928914": "2019-2", "201916513": "2019-1", "201827920": "2018-2",
    "201814793": "2018-1", "201729482": "2017-2", "201718392": "2017-1",
}


def main():
    pdfs = os.path.join(RAIZ, "provas_pdf")
    saida = os.path.join(RAIZ, "conteudo", "q")
    os.makedirs(pdfs, exist_ok=True)
    os.makedirs(saida, exist_ok=True)
    for cod, pid in sorted(PROVAS.items(), key=lambda kv: kv[1]):
        dest = os.path.join(pdfs, f"Fatec_{pid}_Prova.pdf")
        if not os.path.exists(dest):
            print("baixando", pid)
            req = urllib.request.Request(BASE.format(cod=cod), headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(req, timeout=60) as r, open(dest, "wb") as f:
                f.write(r.read())
        subprocess.run([sys.executable, "-E", "-X", "utf8", os.path.join(RAIZ, "tools", "recorta.py"), dest, saida, pid], check=True)


if __name__ == "__main__":
    main()
