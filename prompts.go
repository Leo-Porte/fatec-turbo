package main

import (
	"fmt"
	"strings"
)

// Prompts para o aluno colar, junto com o arquivo gerado, em qualquer IA.
// A ideia: o portal não depende de uma IA específica; quem usa escolhe a sua.

// PromptRevisaoSemanal: a IA não corrige nada (isso já foi feito pelo gabarito). Ela resume pontos fracos e planeja.
func PromptRevisaoSemanal(semana int) string {
	return strings.TrimSpace(fmt.Sprintf(`
Você é um tutor do vestibular da Fatec (São Paulo). Anexei o relatório da minha semana %d de estudo, gerado pelo meu portal. As notas e os erros JÁ foram corrigidos pelo gabarito oficial: não refaça a correção nem resolva as questões de novo.

Com base só no relatório, faça, nesta ordem:

1. Resumo da semana em até 5 linhas: o que melhorou, o que piorou, se eu cumpri o foco planejado.
2. Meus 3 a 5 pontos fracos mais importantes, do mais urgente para o menos. Para cada um: o tema, o padrão de erro que você enxerga nas questões erradas (use os "conceitos cobrados" listados) e o que exatamente eu preciso saber de cor.
3. Se houver redação: os 3 problemas que mais tiram nota, com um trecho do meu texto como exemplo de cada.
4. Plano para a próxima semana, em tópicos curtos: o que estudar em cada noite (cerca de 1 hora) e no fim de semana, usando os nomes das lições do portal que aparecem no relatório. Priorize os pontos fracos sem abandonar o foco da semana do cronograma.
5. Se o relatório trouxer o plano da semana anterior, diga em 2 linhas se eu segui e o que ficou pendente.

Regras: responda em português do Brasil, direto e curto (no máximo uma página). Não invente notas, questões ou dados que não estejam no relatório. Se houver pouca atividade na semana, diga isso e foque no plano.`, semana))
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
