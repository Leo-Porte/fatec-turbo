# Fatec Turbo

Portal de estudo para o vestibular da Fatec, feito de leitura, simulados e redação. Sem vídeo, sem conta, sem servidor de dados: um programa só, que roda no seu computador e abre no navegador.

> Se você descobriu esse repo, me agradeça no futuro.

## O que tem

- **Rotina de 10 semanas** até a prova (13/12/2026), com o que estudar a cada noite e o simulado de cada fim de semana.
- **Lições** em texto: o que a Fatec cobra em cada tema, o que decorar, questões reais resolvidas e exercícios novos.
- **Simulados** com as 15 provas oficiais de 2017 a 2026 (832 questões), cronômetro de 5 horas e correção por disciplina e por tema.
- **Treinos rápidos** de 10 questões por disciplina ou tema, usando as provas de 2017 a 2020.
- **Caderno de erros** montado sozinho a partir do que você errou.
- **Testes da semana** com questões novas; figuras e gráficos são desenhados pelo próprio portal (SVG gerado no servidor, nenhuma biblioteca no navegador).
- **Redação**: você escreve no papel, transcreve com os erros do jeito que escreveu, e o portal gera a folha de redação em PDF e um prompt para mandar para a IA que quiser corrigir.
- **Revisão por IA, de qualquer fornecedor**: depois de um simulado ou teste, baixe o arquivo com as questões e suas respostas, copie o prompt e mande para a IA que preferir. O prompt obriga a IA a validar cada questão e calcular a sua porcentagem de acerto.
- **Autores** que caíram nas provas, com resumo e dicas.

O progresso fica no `localStorage` do navegador. Para trocar de aparelho, use **Seus dados → Exportar/Importar**.

## Rodar

Baixe/compile o binário e execute. Ele sobe em `http://127.0.0.1:8027` e abre o navegador.

```
fatec-turbo.exe            # Windows
./fatec-turbo -addr 0.0.0.0:8027 -abrir=false   # servidor, por exemplo
```

## Compilar

Precisa de Go 1.25+ (ou Docker).

```
go build -o fatec-turbo .
# Windows a partir de qualquer sistema:
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o fatec-turbo.exe .
# Sem Go instalado, com Docker:
docker run --rm -v "$PWD:/src" -w /src golang:1.25-alpine sh -c 'GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o fatec-turbo.exe .'
```

Tudo vai embutido no binário: `dados/` (nosso material), `conteudo/` (recortes das provas) e `web/` (páginas e estilos).

## Estrutura

```
main.go        rotas e páginas
conteudo.go    carga de provas, lições, testes e autores; plano de 10 semanas
figura.go      figuras das questões novas (geometria, funções, gráficos) em SVG
pdf.go         folha de redação e PDF de revisão
prompts.go     prompts para revisão e correção por IA
markdown.go    markdown mínimo das lições
dados/         questoes.csv (classificação e gabarito), licoes/, testes/, autores.json
conteudo/q/    recortes das questões, um diretório por prova (gerados por tools/)
web/           templates HTML, app.css, app.js
tools/         baixar_provas.py, recorta.py e ajustes manuais de recorte
```

### Publicar um teste novo

Crie `dados/testes/<id>.json`:

```json
{"id": "s3-fisica", "titulo": "Cinemática e energia", "semana": 3, "descricao": "...", "publicado": "2026-10-19",
 "exercicios": [{"enunciado": "...", "alternativas": ["...","...","...","...","..."], "gabarito": "B", "resolucao": "...",
   "figura": {"lib": "jsxgraph", "bbox": [-1, 6, 8, -1], "eixos": true,
              "elementos": [{"t": "funcao", "expr": "x^2/4", "dominio": [0, 4]}]}}]}
```

`figura.lib` aceita `jsxgraph` (pontos, segmentos, polígonos, círculos, funções, ângulos, textos), `chart` (barras, linhas, pizza) e `tabela`. Recompile para publicar.

### Regenerar os recortes das provas

```
python -m pip install pymupdf
python tools/baixar_provas.py
```

## Licença e material de terceiros

O código e o material de estudo escrito para este projeto (lições, testes, classificação das questões) estão sob a licença Apache 2.0 (arquivo `LICENSE`).

**As provas, os gabaritos e os recortes em `conteudo/` não estão cobertos por essa licença.** Eles são do Centro Paula Souza / Fatec, publicados no site oficial do vestibular (vestibular.fatec.sp.gov.br), e incluem textos, tirinhas e imagens de terceiros, que pertencem aos seus autores. Estão aqui só para estudo. Os resumos de autores vêm da Wikipédia (CC BY-SA), com link para a fonte.
