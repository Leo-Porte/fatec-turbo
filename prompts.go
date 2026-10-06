package main

import (
	"fmt"
	"strings"
)

// Prompts para o aluno colar, junto com o arquivo gerado, em qualquer IA.
// A ideia: o portal não depende de uma IA específica; quem usa escolhe a sua.

func PromptRevisaoProva(titulo string, total int) string {
	return strings.TrimSpace(fmt.Sprintf(`
Você é um professor de cursinho especialista no vestibular da Fatec (São Paulo). Anexei um PDF chamado "%s" com %d questões de múltipla escolha. A primeira página traz uma tabela com a resposta que eu marquei e o gabarito oficial de cada questão; depois vêm as questões, como imagens da prova original.

Faça, nesta ordem:

1. Para CADA questão, resolva você mesmo ANTES de olhar o gabarito. Depois:
   a) Valide a questão: ela está completa e legível? O gabarito oficial confere com a sua resolução? Se não conferir, diga qual alternativa você acha correta e por quê. Questão marcada como ANULADA conta como acerto para todo mundo.
   b) Diga se eu acertei ou errei.
   c) Explique em 2 a 4 linhas o raciocínio e o conceito que a questão cobra. Se eu errei, diga qual foi o erro provável.
2. No fim, calcule a minha porcentagem de acerto (acertos ÷ total × 100, com uma casa decimal) e monte uma tabela de acertos por disciplina.
3. Liste os 5 temas que eu mais preciso revisar, do mais urgente para o menos, com o que estudar em cada um.

Regras: responda em português do Brasil. Não invente: se não conseguir ler uma questão, diga isso em vez de chutar. Se discordar do gabarito oficial, deixe claro que é a sua opinião e mantenha a contagem pelo gabarito oficial.`, titulo, total))
}

func PromptRevisaoTeste(titulo string, total int) string {
	return strings.TrimSpace(fmt.Sprintf(`
Você é um professor de cursinho especialista no vestibular da Fatec (São Paulo). Anexei um arquivo chamado "%s" com %d questões NOVAS no estilo da Fatec, escritas para estudo. Para cada questão o arquivo traz o enunciado, as alternativas, a figura (quando existe, descrita com os dados exatos), a resposta que eu marquei e o gabarito proposto pelo autor.

Faça, nesta ordem:

1. Para CADA questão, resolva você mesmo ANTES de olhar o gabarito proposto. Depois:
   a) VALIDE a questão (obrigatório): o enunciado é coerente e tem todos os dados? Existe exatamente UMA alternativa correta? O gabarito proposto está certo? A figura bate com o enunciado? Classifique como "válida" ou "com problema" e explique o problema, se houver.
   b) Diga se eu acertei ou errei, usando a SUA resolução quando a questão tiver problema no gabarito.
   c) Explique em 2 a 4 linhas o raciocínio e o conceito cobrado.
2. No fim, calcule a minha porcentagem de acerto (acertos ÷ total de questões válidas × 100, com uma casa decimal) e liste as questões com problema, para o autor corrigir.
3. Diga em 3 linhas o que eu devo revisar.

Regras: responda em português do Brasil e não invente dados que não estão no arquivo.`, titulo, total))
}

func PromptRedacao(tema string) string {
	return strings.TrimSpace(fmt.Sprintf(`
Você é um corretor experiente de redação do vestibular da Fatec (São Paulo). Anexei um PDF com a minha redação, transcrita EXATAMENTE como escrevi à mão, com os meus erros de ortografia, acentuação e pontuação mantidos de propósito. Não corrija o texto antes de avaliar: avalie o que está escrito.

Tema proposto: "%s"
Gênero: a Fatec aceita dissertativo-argumentativo ou narrativo. Diga qual eu usei e avalie por ele.

Critérios oficiais (Portaria CEETEPS nº 5304/2026, Anexo I, item Redação):
- Nota de 0 a 100, em número inteiro, com DOIS critérios de igual peso, que se condicionam mutuamente:
  1) Correção gramatical: modalidade culta, adequação vocabular, concordância, colocação, regência, preposições e conjunções, grafia, paragrafação e pontuação.
  2) Apresentação e desenvolvimento do conteúdo: adequação ao tema e ao gênero, coesão e unidade do texto e, no dissertativo, argumentação coerente e posicionamento claro.
- Nota ZERO se: fugir ao tema e/ou ao gênero; tiver 5 linhas ou menos (sem contar o título); identificar o candidato; não for texto articulado verbalmente; estiver em outra língua; for só cópia da coletânea ou de outras partes da prova.
- Nota limitada a 50 se houver desvio de tema ("hipertrofia do exemplo") ou tangenciamento (tema abordado de forma superficial).
- Texto que defenda ideias que violem direitos humanos pode ter a nota reduzida ou zerada. A falta de título diminui a nota.

Faça, nesta ordem:
1. Verifique, uma a uma, as condições de nota zero e de teto 50, e diga se o texto passa.
2. Dê a nota final de 0 a 100 (número inteiro) e a nota de cada critério (0 a 50 cada), explicando cada uma em 3 a 4 linhas.
3. Liste TODOS os erros de norma culta que encontrar, no formato: linha aproximada → trecho como está → forma correta → regra.
4. Aponte 3 pontos fortes e 3 melhorias concretas para a próxima redação.
5. Reescreva só a introdução, mostrando como ela ficaria mais forte, sem mudar a minha tese.

Responda em português do Brasil. Seja exigente como a banca, mas específico: cada crítica precisa apontar o trecho.`, tema))
}
