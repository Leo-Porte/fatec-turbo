-- Usuários criados pelo administrador (sem cadastro aberto).
CREATE TABLE usuarios (
    id            BIGSERIAL PRIMARY KEY,
    login         TEXT NOT NULL UNIQUE,
    nome          TEXT NOT NULL DEFAULT '',
    senha_hash    TEXT NOT NULL,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    ultimo_acesso TIMESTAMPTZ
);

-- Sessão: guardamos só o hash (sha256) do token que vai no cookie.
CREATE TABLE sessoes (
    token_hash  TEXT PRIMARY KEY,
    usuario_id  BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expira_em   TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessoes_usuario ON sessoes(usuario_id);

-- Estado completo do aluno, no mesmo formato que o navegador usa (app.js).
CREATE TABLE progresso (
    usuario_id    BIGINT PRIMARY KEY REFERENCES usuarios(id) ON DELETE CASCADE,
    dados         JSONB NOT NULL,
    versao        INTEGER NOT NULL DEFAULT 1,
    atualizado_em TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Simulados, treinos e testes finalizados, derivados do progresso, para consulta.
CREATE TABLE tentativas (
    usuario_id BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    id         TEXT NOT NULL,
    tipo       TEXT NOT NULL,              -- prova | treino | teste
    prova      TEXT NOT NULL DEFAULT '',
    titulo     TEXT NOT NULL DEFAULT '',
    inicio     TIMESTAMPTZ,
    fim        TIMESTAMPTZ,
    segundos   INTEGER NOT NULL DEFAULT 0,
    acertos    INTEGER NOT NULL DEFAULT 0,
    total      INTEGER NOT NULL DEFAULT 0,
    nota       INTEGER NOT NULL DEFAULT 0,
    respostas  JSONB NOT NULL DEFAULT '{}',
    resultado  JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (usuario_id, id)
);
CREATE INDEX tentativas_fim ON tentativas(usuario_id, fim);

-- Redações transcritas, derivadas do progresso.
CREATE TABLE redacoes (
    usuario_id    BIGINT NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    prova         TEXT NOT NULL,
    titulo        TEXT NOT NULL DEFAULT '',
    texto         TEXT NOT NULL DEFAULT '',
    nota          INTEGER,
    pdf_em        TIMESTAMPTZ,
    atualizado_em TIMESTAMPTZ,
    PRIMARY KEY (usuario_id, prova)
);
