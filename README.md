# Fatec Turbo

Portal de estudo para o vestibular da Fatec, feito de leitura, simulados e redação. Sem vídeo. Um programa só, em dois modos:

- **Local**: roda no seu computador, sem conta; o progresso fica no navegador.
- **Servidor**: com Postgres e login criado pelo administrador, o progresso de cada aluno fica salvo no banco e aparece em qualquer aparelho. Passo a passo em [`docs/deploy.md`](docs/deploy.md).

> Se você descobriu esse repo, me agradeça no futuro.

## O que tem

- **Rotina de 10 semanas** até a prova (13/12/2026), com o que estudar a cada noite e o simulado de cada fim de semana. Todas as semanas abertas.
- **Lições** em texto: o que a Fatec cobra em cada tema, o que decorar, questões reais resolvidas e exercícios novos. Seguem uma ordem de pré-requisitos ([`docs/catalogo_licoes.md`](docs/catalogo_licoes.md)): uma lição libera quando as anteriores são marcadas como lidas.
- **Simulados** com as 15 provas oficiais de 2017 a 2026 (832 questões), cronômetro de 5 horas e correção por disciplina e por tema.
- **Treinos rápidos** de 10 questões por disciplina ou tema, usando as provas de 2017 a 2020.
- **Caderno de erros** montado sozinho a partir do que você errou.
- **Testes da semana** com questões novas; figuras e gráficos são desenhados pelo próprio portal (SVG gerado no servidor, nenhuma biblioteca no navegador).
- **Redação**: você escreve no papel, transcreve com os erros do jeito que escreveu, e o portal gera a folha de redação em PDF e um prompt para mandar para a IA que quiser corrigir.
- **Correção por script**: simulados, treinos, testes e exercícios são corrigidos na hora pelo gabarito, sem IA.
- **Revisão da semana com IA, de qualquer fornecedor**: uma vez por semana o portal monta um relatório (notas, temas com erro e o conceito que faltou em cada questão, lições lidas, redações). Você manda para a IA que preferir com o prompt pronto; ela resume os pontos fracos e monta o plano da semana seguinte. A resposta fica salva no histórico e entra no relatório seguinte.
- **Autores** que caíram nas provas, com resumo e dicas.

No modo local o progresso fica no `localStorage` do navegador; para trocar de aparelho, use **Seus dados → Exportar/Importar**.

## Rodar

Modo local: compile e execute. Sobe em `http://127.0.0.1:8027` e abre o navegador.

```
fatec-turbo.exe
```

Modo servidor (com banco e login): veja [`docs/deploy.md`](docs/deploy.md). Em resumo, `docker compose up -d --build` e
`docker compose exec app /fatec-turbo usuario criar <login> <nome>`.

| Variável / flag | Para quê |
|---|---|
| `DATABASE_URL` / `-db` | Postgres. Vazio = modo local, sem login |
| `FT_ADDR` / `-addr` | endereço (padrão `127.0.0.1:8027`) |
| `FT_COOKIE_SEGURO=1` / `-cookie-seguro` | cookie de sessão só por HTTPS |
| `FT_PROXY=1` / `-proxy` | confiar no IP vindo do Caddy (limite de tentativas de login) |
| `FT_IMAGENS_URL` / `-imagens` | servir as imagens das questões por um CDN, ex.: `https://cdn.jsdelivr.net/gh/Leo-Porte/fatec-turbo@main/conteudo/q` |

As migrations (`migrations/*.sql`) vão embutidas no binário e rodam sozinhas ao subir.

## Compilar

Precisa de Go 1.26+ (ou Docker).

```
go build -o fatec-turbo .
# Windows a partir de qualquer sistema:
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o fatec-turbo.exe .
# Sem Go instalado, com Docker:
docker run --rm -v "$PWD:/src" -w /src golang:1.26-alpine sh -c 'GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o fatec-turbo.exe .'
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
