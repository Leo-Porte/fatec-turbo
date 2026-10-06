# Correções manuais dos recortes

`tools/recorta.py` faz o recorte automático, mas erra em páginas de duas colunas, textos de apoio
compartilhados e figuras perto do cabeçalho. Os recortes em `conteudo/q/` já vêm conferidos à mão,
questão por questão. **Eles são a fonte de verdade**: rodar `recorta.py` de novo sobrescreve as correções.

Estes arquivos registram como as correções foram feitas, para reaplicar se for preciso:

- `fix_colunas.py` + `<prova>_fix.json` (2017-1 a 2019-1): recorte por coluna e textos de apoio manuais.
- `colcut.py` + `<prova>.plan.json` e `2022-2.cc.json` (2019-2 a 2023-2): recorte por coluna, regrava recortes, manifest e redação.
- `fix.py` / `autoq.py` + `<prova>.fix.json` (2024-1 a 2026-1) e `2026-2.patch.py`: ajustes de bordas,
  textos de apoio de grupo e páginas da proposta de redação.

Os scripts foram escritos para os caminhos da máquina onde rodaram; confira os caminhos antes de usar.
