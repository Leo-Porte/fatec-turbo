# Publicar numa VPS

Portal + Postgres + Caddy (HTTPS automático) num `docker compose`. Serve para poucas pessoas com login criado por você.

## 1. Máquina

Qualquer VPS Linux com 1 GB de RAM serve (Ubuntu 24.04, por exemplo). Libere as portas **80** e **443** no firewall do provedor.

Instale o Docker:

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # saia e entre de novo na sessão SSH
```

## 2. Código e configuração

```bash
git clone https://github.com/Leo-Porte/fatec-turbo.git
cd fatec-turbo
cp .env.example .env
nano .env
```

No `.env`:

- `DOMINIO`: seu domínio apontando para o IP da VPS. **Sem domínio**, use o IP com sslip.io, trocando os pontos por traços: VPS `203.0.113.10` → `203-0-113-10.sslip.io`. O Caddy tira o certificado HTTPS sozinho.
- `POSTGRES_PASSWORD`: gere com `openssl rand -base64 24`.

## 3. Subir

```bash
docker compose up -d --build
docker compose logs -f app      # deve mostrar "migration aplicada" e "com banco e login"
```

As migrations rodam sozinhas toda vez que o portal sobe.

## 4. Criar os logins

Não existe cadastro pelo site. Você cria cada pessoa:

```bash
docker compose exec app /fatec-turbo usuario criar leonardo Leonardo
docker compose exec app /fatec-turbo usuario criar amigo "Nome do Amigo"
```

O comando pede a senha (mínimo de 8 caracteres) sem mostrar na tela. Outros comandos:

```bash
docker compose exec app /fatec-turbo usuario listar
docker compose exec app /fatec-turbo usuario senha amigo      # troca a senha e encerra as sessões abertas
docker compose exec app /fatec-turbo usuario remover amigo    # apaga o usuário e todo o progresso dele
```

Mande para a pessoa: o endereço (`https://<DOMINIO>`), o login e a senha.

## 5. Atualizar

```bash
git pull
docker compose up -d --build
```

## 6. Backup

O banco é pequeno. Um dump por dia já basta:

```bash
docker compose exec -T db pg_dump -U fatec fatec | gzip > backup-$(date +%F).sql.gz
```

Para agendar, `crontab -e` e adicione (ajuste o caminho):

```
0 4 * * * cd /home/ubuntu/fatec-turbo && docker compose exec -T db pg_dump -U fatec fatec | gzip > backups/backup-$(date +\%F).sql.gz
```

Restaurar: `gunzip -c backup.sql.gz | docker compose exec -T db psql -U fatec fatec`.

## Consultas úteis

```bash
docker compose exec db psql -U fatec fatec
```

```sql
-- notas dos simulados
SELECT u.login, t.titulo, t.nota, t.fim FROM tentativas t JOIN usuarios u ON u.id = t.usuario_id ORDER BY t.fim DESC;
-- redações com nota registrada
SELECT u.login, r.prova, r.titulo, r.nota FROM redacoes r JOIN usuarios u ON u.id = r.usuario_id;
```

## Segurança, em resumo

- Senhas com bcrypt. Sessão em cookie `HttpOnly`, `Secure` e `SameSite=Lax`, válida por 60 dias.
- POST/PUT vindos de outro site são recusados (checagem de `Origin`).
- Limite de 8 tentativas de login a cada 10 minutos por IP.
- O Postgres não fica exposto: só o portal fala com ele, pela rede interna do compose.
