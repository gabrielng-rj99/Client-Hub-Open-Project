# Project Summary — Client Hub Open Project

Resumo único e consolidado de requisitos, estado atual e próximos passos.

## Visão geral
- [x] Sistema de gestão de clientes, contratos e financeiro para pequenas empresas, MEIs e freelancers.
- [x] Licença: **AGPL-3.0**.

## Stack
- [x] **Backend:** Go 1.25.6, PostgreSQL 16, API REST com JWT (access + refresh)
- [x] **Frontend:** React 19.2.4 (Vite 7.3.1), React Router 7.13.0, CSS Modules
- [x] **Testes:** Go tests, suíte Python API/segurança (Python 3.13.9, pytest 9.0.2), Vitest 4.0.18, Playwright 1.58.2
- [x] **Lint:** ESLint 9.39.3

## Escopo Beta (MVP) — Core deve funcionar 100%
### 1) Autenticação
- [x] Login, refresh token, logout
- [x] Bloqueio por tentativas falhas
- [x] Políticas de senha por role (via UI)

### 2) Usuários (Admin+)
- [x] CRUD, bloqueio/desbloqueio
- [x] Políticas de sessão por role (via UI)

### 3) Clientes
- [x] CRUD, arquivamento (soft delete), tags
- [x] Validação CPF/CNPJ
- [x] Status automático com base em contratos ativos

### 4) Contratos
- [x] CRUD, categorias/subcategorias
- [x] Datas início/fim, status automático
- [x] Unarchive via `archived_at` com audit logging

### 5) Financeiro
- [x] Parcelas, recorrências, vencimentos
- [x] Marcar pago/pendente
- [x] Dashboard financeiro, atrasos, próximos vencimentos

### 6) Permissões
- [x] Roles: root, admin, user, viewer
- [x] Permissões granulares por recurso
- [x] Validação no banco (não apenas claims)

### 7) Auditoria
- [x] Log completo de operações
- [x] IP, UA, método, path, old/new value
- [x] Apenas root acessa

### 8) Configurações
- [x] Tema global e por usuário
- [x] Segurança e políticas via UI
- [x] Configuração de dashboard

## Requisitos não-funcionais (Beta)
### Segurança (obrigatório)
- [x] bcrypt, JWT com expiração
- [x] Rate limit, brute force protection
- [x] Security headers (CSP, HSTS, X-Frame-Options etc.)
- [x] Validação de input e sanitização
- [x] Auditoria de operações sensíveis

### Disponibilidade
- [x] `/health` e modo manutenção via `/api/initialize/`

### Performance (pendente)
- [ ] Definir SLAs e testes de carga

## Status atual (consolidado)
### Concluído
- [x] Centralização de runtime em `app/` (deploy/dev/monolith/docker)
- [x] `tests/pytest.ini` e `tests/run_all_tests.sh` alinhados
- [x] README com heading `## License`
- [x] Ajuste de testes AGPL para aceitar build tags Go
- [x] Backend Go tests ok (inclui testes de archive/unarchive)
- [x] Suíte Python de segurança operacional (14 arquivos: auth, bypass, XSS, SQLi, JWT, roles, permissions, financeiro, contratos, clientes, settings, appearance, initialization, block)
- [x] Seed do ambiente de testes, auth flows e resets robustos no `conftest.py`
- [x] Build frontend direcionado para `app/monolith/frontend`
- [x] Preflight de Docker (buildx + permissões de volume)
- [x] Test scripts usam `TEST_ROOT_PASSWORD`

### Bloqueios conhecidos
- [ ] Permissões em `app/docker/data/postgres` exigem `chown` com privilégio
- [ ] Docker buildx não habilitado em alguns ambientes

## Testes — última execução registrada
- [x] AGPL compliance: **PASS**
- [x] Docker build: **PASS**
- [x] Go tests: **PASS**
- [x] Python segurança: **operacional**

## Próximos passos recomendados
- [ ] Executar suíte Python completa com `TEST_API_URL=http://localhost:63000/api`.
- [ ] Resolver buildx e permissões de volume nos ambientes alvo.
- [ ] Reexecutar suíte após ajustes de ambiente.
- [ ] Ajustar botões (UI) para interagir com novo aparato de segurança e permissões refatorado.

## Pós-Beta (futuro)
- [ ] Webhooks, API pública, exportações
- [ ] Relatórios financeiros avançados
- [ ] Temas adicionais e dashboards analíticos

## Contato & contribuição
- [ ] Repositório/Issues: **a definir**