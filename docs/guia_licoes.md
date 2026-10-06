# Guia para escrever lições do Fatec Turbo

O aluno estuda LENDO e fazendo questões, sem vídeo. Cada lição é curta (20–40 min de leitura + exercícios), direta, em português do Brasil, focada no que a prova da Fatec REALMENTE cobra. Explique como uma pessoa competente explicaria para outra: primeiro o que está acontecendo em linguagem normal, depois a regra ou a fórmula. Sem enrolação, sem "vale ressaltar".

## Material de base
- `docs/catalogo_licoes.md`: id, título, semana, requisitos e temas de cada lição. Use EXATAMENTE o id, semana e requisitos do catálogo.
- `dados/questoes.csv`: 832 questões reais classificadas (`prova;questao;disciplina;tema;subtema;conceito_chave;tipo;dificuldade;gabarito`). Filtre pelos temas da lição para ver como a Fatec cobra e o que mais cai.
- `D:/Estudo/Fatec/Analise/Topicos_de_estudo.md`: plano com o que decorar por tema (seção 4).
- Imagens das questões reais: `conteudo/q/<prova>/manifest.json` diz quais imagens compõem cada questão (`imgs`) e quais textos de apoio vêm antes (`ctx`, índices em `contextos`). Abra as imagens com Read antes de resolver.
- Modelos prontos: `dados/licoes/mat-porcentagem.json`, `dados/licoes/mat-estatistica.json` (com figura), `dados/licoes/red-estrutura.json`.

Todos os caminhos são relativos a `D:/Estudo/Fatec/portal/`.

## Esquema (um arquivo por lição: `dados/licoes/<id>.json`, UTF-8)
```json
{
 "id": "fis-cinematica",
 "disciplina": "Física",
 "tema": "Cinemática",
 "titulo": "Cinemática",
 "semana": 3,
 "requisitos": ["mat-unidades"],
 "minutos": 35,
 "por_que": "1–2 frases: quantas vezes caiu nas 15 provas e em que formato.",
 "secoes": [ {"titulo": "...", "texto": "..."} ],
 "decorar": ["fórmula ou fato curto", "..."],
 "pegadinhas": ["erro comum e como evitar", "..."],
 "resolvidas": [ {"prova": "2026-2", "n": 41, "resolucao": "passo a passo"} ],
 "exercicios": [ {"enunciado": "...", "alternativas": ["...","...","...","...","..."], "gabarito": "C", "resolucao": "...", "figura": null} ]
}
```
- `tema`: o tema principal do CSV (o primeiro listado no catálogo).
- `disciplina`: Matemática, Raciocínio Lógico, Física, Química, Biologia, História, Geografia, Português, Inglês ou Redação.
- Formatação permitida em `texto`, `resolucao` e `enunciado`: parágrafos separados por linha em branco (`\n\n`), listas com linhas começando em `- `, `**negrito**`, `` `fórmula` `` entre crases. Nada de HTML, LaTeX ou tabela em markdown (tabela vai como figura). Use Unicode: × ÷ ² ³ √ ≤ ≥ π Δ → ° ⁻ ₂.

## Regras de conteúdo
- **Comece pelo que mais cai.** As seções seguem a frequência do CSV, não a ordem de livro didático.
- **Se a lição depende de outra (requisitos), não reexplique a base**: cite em uma linha ("como vimos em Unidades…") e siga.
- **resolvidas**: 2 ou 3 questões REAIS (prova e número existentes no CSV com um dos temas da lição; prefira as mais recentes). Abra a imagem, resolva passo a passo e confira que a sua resposta bate com o gabarito oficial do CSV. Se não bater, ou se a imagem estiver incompleta, escolha outra. Não use questões anuladas.
- **exercicios**: 5 questões NOVAS por lição, no estilo Fatec (contexto do cotidiano, 5 alternativas, só uma certa), de fácil a média. Confira cada conta duas vezes; as alternativas erradas devem ser erros plausíveis. Varie a letra do gabarito (não repita a mesma letra em mais de 2 exercícios).
- **Fatos**: não invente datas, nomes, números ou citações. Em História, Geografia, Biologia e Literatura, só use o que você tem certeza; na dúvida, deixe de fora. Não copie trechos longos de obras com direito autoral.
- **Redação** (red-*): os exercícios são de múltipla escolha sobre reconhecer tese, argumento, repertório, coesão, proposta. Critérios oficiais estão em `dados/licoes/red-estrutura.json` (Portaria CEETEPS 5304/2026, Anexo I).

## Figuras (opcional, só quando ajudam: geometria, gráfico, tabela de dados, circuito simples, ciclo)
Desenhadas pelo próprio portal a partir destes dados. A figura tem que bater com o enunciado e com a resolução.

Geometria e funções:
```json
"figura": {"lib":"jsxgraph","bbox":[xmin,ymax,xmax,ymin],"eixos":false,
 "elementos":[
  {"t":"ponto","id":"A","xy":[0,0],"rotulo":"A"},
  {"t":"segmento","de":"A","ate":"B","rotulo":"5 cm"},
  {"t":"poligono","pts":["A","B","C"]},
  {"t":"circulo","centro":"A","raio":2},
  {"t":"funcao","expr":"x^2 - 2*x","dominio":[-1,3],"rotulo":"f"},
  {"t":"texto","xy":[1,1],"txt":"60°"},
  {"t":"angulo","pts":["B","A","C"],"rotulo":"α"}
 ]}
```
(pontos referenciados por id precisam estar declarados; `expr` aceita + − * / ^, parênteses, x, sqrt, sen, cos, tg, log, ln, abs, pi; `eixos: true` mostra eixos cartesianos.)

Gráfico de dados: `{"lib":"chart","tipo":"bar"|"line"|"pie","titulo":"...","rotulos":["2020","2021"],"series":[{"nome":"Vendas","dados":[10,12]}],"eixoY":"mil unidades"}`

Tabela: `{"lib":"tabela","cabecalho":["Item","Preço"],"linhas":[["Arroz","R$ 5,00"]]}`

## Antes de entregar
1. Valide cada arquivo: `python -E -X utf8 -c "import json;json.load(open('dados/licoes/<id>.json',encoding='utf-8'))"`.
2. Confira: 5 exercícios com 5 alternativas cada, gabarito em A–E, resolvidas com prova+n existentes no CSV.
3. Não escreva fora de `dados/licoes/` (temporários no seu próprio diretório temporário). Não mexa em lições de outros.
