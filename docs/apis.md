# APIs e fontes para o site (pesquisa de 2026-10-06)

Regra do site: sem rede em tempo de execução. Tudo abaixo só vale se baixado agora para JSON.

## Resumo da recomendação
- **Tirinhas/quadrinhos: não existe fonte viável.** Tirinhas atuais têm direito autoral. Não republique.
- **Textos em domínio público: Wikisource pt** (API MediaWiki, sem chave). Já baixei 6 trechos em `autores.json`.
- **Dicionário/sinônimos: Dicionário Aberto** (API sem chave, baixável). Licença precisa ser confirmada antes de usar (ver abaixo).
- **Resumos de autores: Wikipédia pt** (REST, CC BY-SA 4.0). Já baixados em `autores.json`.

## (a) Tirinhas e quadrinhos
| Fonte | O que dá | Chave | Licença | Viável? |
|---|---|---|---|---|
| xkcd (`https://xkcd.com/info.0.json`) | JSON com título, imagem, alt-text. **Só inglês** | não | CC BY-NC 2.5 (uso não comercial, com crédito) | Testado: responde. Técnico sim; pedagógico fraco (não é pt, não cai na prova) |
| GoComics (Garfield, Calvin etc.) | Não há API oficial; só wrappers que raspam o site | — | Direitos reservados aos autores/sindicatos. `gocomics.com` devolveu 403 ao curl | **Não.** Não republicar |
| Tirinhas brasileiras (Armandinho, Níquel Náusea, Laerte, Henfil, Cientirinhas...) | Nenhuma API pública encontrada | — | Direitos reservados (Cientirinhas é CC BY-NC-ND, ver abaixo) | **Não** |

**O que NÃO pode ser republicado:** imagem de qualquer tirinha, charge ou cartum das provas (Dilbert, Glasbergen, Calvin e Haroldo, Laerte, Henfil, Mike Keefe, Mike Thompson, Andy Singer, Sephko, Cientirinhas etc.), nem em JSON, nem como print. As imagens em `Site/q/` vêm da prova da Fatec, usadas para estudo; o aviso de licença visto na imagem do Cientirinhas #251 é "BY-NC-ND" (sem derivados e sem uso comercial). Para o site, o caminho é **linkar** a fonte e **descrever** a tirinha com palavras próprias, sem copiar a imagem.

Observação: não encontrei API de tirinhas em português; é ausência de resultado na busca, não prova de que não exista.

## (b) Textos em língua portuguesa
| Fonte | O que dá | Chave | Licença | Viável? |
|---|---|---|---|---|
| **Wikisource pt** `https://pt.wikisource.org/w/api.php` | Obras em domínio público (Machado, Bilac, Gonçalves Dias, Graciliano, Euclides, Pessanha) | não | Obra em domínio público; transcrição CC BY-SA | **Sim.** Testado (`action=parse`). Mandar User-Agent identificado |
| Domínio Público (MEC) `dominiopublico.gov.br` | Acervo de obras, PDF/texto | não há API | Cada obra tem sua condição; site responde (301) | Só download manual, sem API. Não usei |
| Project Gutenberg / `gutendex.com` | Catálogo, poucos livros em português | não | Domínio público (nos EUA) | **Não confirmei**: `gutendex.com` não respondeu (timeout) nos meus testes |

Domínio público no Brasil: autor falecido há mais de 70 anos. Entram Machado (1908), Bilac (1918), Gonçalves Dias (1864), Euclides (1909), Pessanha (1926), Graciliano (1953, passou em 2024). **Ficam de fora:** Drummond (1987), Manuel Bandeira (1968), Clarice, Vinicius, Gullar, Guimarães Rosa (1967), Carolina Maria de Jesus, Conceição Evaristo, letras de música. Wikisource pode ter versões com ortografia da época (ex.: "Vidas Sêcas", "Brazil").

## (c) Dicionário e sinônimos
| Fonte | O que dá | Chave | Licença | Viável? |
|---|---|---|---|---|
| **Dicionário Aberto** `https://api.dicionario-aberto.net/word/<palavra>` | Definições em XML (base: Novo Diccionário de Cândido de Figueiredo, 1913, modernizado). Sem sinônimos estruturados | não | Projeto "aberto"; **não achei a licença explícita** na busca | Testado: responde. Baixar só um conjunto pequeno e conferir a licença no site do projeto |
| Wiktionary pt (REST `/page/definition/<palavra>`) | Definições | não | CC BY-SA | Meu teste devolveu HTTP 501; não confirmado. Alternativa: API MediaWiki `action=parse` |
| Sinônimos | Nenhuma API livre confiável encontrada | — | — | **Não.** Se precisar, montar lista própria (autoria do site) |

## Furos
- Licença do Dicionário Aberto e disponibilidade do Gutenberg: não confirmadas.
- Busca por "API de tirinhas em português" foi superficial (2 buscas + testes diretos); pode existir algo que não achei.
- Não li termos de uso do Wikisource/Wikipédia além do padrão CC BY-SA; exigem atribuição (link da página já guardado no JSON).
