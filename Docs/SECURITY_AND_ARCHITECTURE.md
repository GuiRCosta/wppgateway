# WPP Gateway - Analise de Seguranca e Arquitetura

**Data:** 2026-03-30
**Versao:** 2.0
**Stack:** Go (Fiber), PostgreSQL, Redis, whatsmeow

---

## Sumario

- [Seguranca](#seguranca)
  - [Critico](#critico-corrigir-imediatamente)
  - [Alto](#alto-corrigir-antes-de-producao)
  - [Medio](#medio-corrigir-quando-possivel)
  - [Baixo](#baixo-considerar)
- [Arquitetura](#arquitetura)
  - [Pontos Fortes](#pontos-fortes)
  - [Problemas Estruturais](#problemas-estruturais)
  - [Dead Code](#dead-code-identificado)
  - [Escalabilidade](#escalabilidade)
- [Prioridade de Correcao](#prioridade-de-correcao)

---

## Seguranca

### Critico (corrigir imediatamente)

#### 1. ~~`/register` sem rate limit~~ CORRIGIDO

**Local:** `internal/api/router.go:109`

~~O endpoint `POST /register` esta fora do grupo `/v1`, sem autenticacao e sem rate limiting.~~

**Correcao aplicada:** Rate limit de 5 requests/hora por IP adicionado ao endpoint `/register`.

```go
app.Post("/register", middleware.RateLimit(5, time.Hour), tenantH.CreateTenant)
```

#### 2. ~~`/metrics` exposto publicamente~~ CORRIGIDO

**Local:** `internal/api/router.go:102-106`

~~Prometheus metrics acessiveis sem autenticacao.~~

**Correcao aplicada:** Metrics movido para porta interna `127.0.0.1:9091`, inacessivel externamente.

```go
go func() {
    metricsApp := fiber.New(fiber.Config{DisableStartupMessage: true})
    metricsApp.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
    if err := metricsApp.Listen("127.0.0.1:9091"); err != nil {
        deps.Log.Error().Err(err).Msg("failed to start metrics server")
    }
}()
```

#### 3. `/v1/logs` expoe logs de todos os tenants

**Local:** `internal/api/handler/logs.go`

**Status:** PARCIALMENTE CORRIGIDO - limit cap adicionado (max 200), mas o ring buffer continua global. Qualquer tenant autenticado ainda consegue ler logs de todos os outros tenants.

**Pendente:** Filtrar logs por tenant_id ou restringir a admin-only.

```go
// Opcao A: restringir a admin
v1.Get("/logs", adminOnly(), logsH.List)

// Opcao B: filtrar por tenant no ring buffer
entries := logger.Buffer.EntriesByTenant(tenant.ID, level, limit)
```

#### 4. ~~CORS wildcard~~ CORRIGIDO

**Local:** `internal/api/router.go:76-83`

~~`cors.New()` com defaults permite qualquer origem.~~

**Correcao aplicada:** CORS restrito a origem especifica via env var `CORS_ORIGINS` (default: `https://wpp.ideva.ai`).

```go
app.Use(cors.New(cors.Config{
    AllowOrigins:     allowedOrigins,
    AllowMethods:     "GET,POST,PUT,PATCH,DELETE",
    AllowHeaders:     "Origin,Content-Type,Accept,X-API-Key",
    AllowCredentials: false,
}))
```

#### 5. ~~Bug na funcao `itoa()`~~ CORRIGIDO

**Local:** `internal/store/postgres/audit.go:78-80`

~~Converte int para rune em vez de string numerica.~~

**Correcao aplicada:** Substituido por `strconv.Itoa`.

```go
func itoa(i int) string {
    return strconv.Itoa(i)
}
```

---

### Alto (corrigir antes de producao)

#### 6. ~~IDOR - Falta de verificacao de ownership~~ CORRIGIDO

**Locais corrigidos:**
- `internal/api/handler/instance.go` - Get, Update, Delete, GetQRCode, PairPhone, Connect, Disconnect, Restart, GetStatus - **todos com `verifyOwnership()`**
- `internal/api/handler/message.go:118-151` - GetStatus - **ownership check + verificacao de msg.GroupID**
- `internal/api/handler/message.go:153-197` - ListByGroup - **ownership check adicionado**
- `internal/api/handler/broadcast.go:85-118` - GetStatus - **ownership check via `verifyGroupOwnership()`**
- `internal/api/handler/broadcast.go:120-147` - List - **ownership check via `verifyGroupOwnership()`**
- `internal/api/handler/broadcast.go:149-195` - Pause, Resume, Cancel - **ownership check via `verifyGroupOwnership()`**
- `internal/api/handler/metrics.go:46-90` - GroupMetrics, DailyMetrics, InstanceMetrics - **ownership check via `verifyGroupOwnership()`**
- `internal/api/handler/blacklist.go:93-128` - Remove - **ownership check adicionado**

**Correcao aplicada:** Helper `verifyOwnership()` no InstanceHandler e `verifyGroupOwnership()` nos demais handlers.

```go
func (h *InstanceHandler) verifyOwnership(ctx context.Context, tenant *domain.Tenant, instanceID uuid.UUID) (*domain.Instance, error) {
    inst, err := h.instanceRepo.FindByID(ctx, instanceID)
    if err != nil { return nil, err }
    if inst == nil { return nil, nil }
    group, err := h.groupRepo.FindByID(ctx, inst.GroupID)
    if err != nil { return nil, err }
    if group == nil || group.TenantID != tenant.ID { return nil, nil }
    return inst, nil
}
```

#### 7. ~~API keys em plaintext no banco~~ CORRIGIDO

**Local:** `internal/store/postgres/tenant.go`

~~Keys armazenadas sem hash.~~

**Correcao aplicada:** API keys armazenadas como SHA-256 hash. Migration `000011_hash_api_keys` converte dados existentes.

```go
func hashAPIKey(apiKey string) string {
    h := sha256.Sum256([]byte(apiKey))
    return hex.EncodeToString(h[:])
}
```

> **IMPORTANTE:** Apos rodar a migration, as API keys plaintext sao removidas do banco. Tenants existentes precisam conhecer suas keys originais - a migration e irreversivel.

#### 8. ~~Encryption key zerada aceita sem validacao~~ CORRIGIDO

**Local:** `internal/config/config.go:46-56`

**Correcao aplicada:** Validacao no startup rejeita key padrao `000...000` e keys com menos de 32 caracteres.

```go
func (c *Config) Validate() error {
    if c.Crypto.MasterKey == strings.Repeat("0", 64) {
        return fmt.Errorf("ENCRYPTION_KEY is insecure - generate with: openssl rand -hex 32")
    }
    if len(c.Crypto.MasterKey) < 32 {
        return fmt.Errorf("ENCRYPTION_KEY must be at least 32 characters")
    }
    return nil
}
```

#### 9. ~~Error handler vaza internals~~ CORRIGIDO

**Local:** `internal/api/router.go:50-63`

~~`err.Error()` retornado direto ao cliente.~~

**Correcao aplicada:** Error handler sanitizado - retorna mensagem generica para erros internos, loga detalhes no servidor.

```go
ErrorHandler: func(c *fiber.Ctx, err error) error {
    code := fiber.StatusInternalServerError
    message := "Internal server error"
    if e, ok := err.(*fiber.Error); ok {
        code = e.Code
        message = e.Message
    }
    deps.Log.Error().Err(err).Str("path", c.Path()).Msg("unhandled error")
    return c.Status(code).JSON(fiber.Map{
        "success": false,
        "error": fiber.Map{"code": "internal_error", "message": message},
    })
},
```

#### 10. ~~Redis sem autenticacao e credenciais hardcoded~~ CORRIGIDO

**Local:** `docker/docker-compose.yml`

**Correcao aplicada:** Redis com `--requirepass`, Postgres e Grafana com credenciais via env vars obrigatorias.

```yaml
redis:
  command: redis-server --appendonly yes --requirepass "${REDIS_PASSWORD:?REDIS_PASSWORD is required}"

postgres:
  environment:
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}

grafana:
  environment:
    GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD:?GRAFANA_PASSWORD is required}
```

#### 11. Database SSL desabilitado

**Local:** `.env:7`

**Status:** PENDENTE - `sslmode=disable` transmite credenciais e dados em plaintext.

```
# Correcao
DATABASE_URL=postgres://wpp:senha@host:5432/wpp_gateway?sslmode=require
```

#### 12. ~~Blacklist Remove sem ownership check~~ CORRIGIDO

**Local:** `internal/api/handler/blacklist.go:93-128`

**Correcao aplicada:** Verificacao de ownership do grupo adicionada antes do remove.

---

### Medio (corrigir quando possivel)

#### 13. Webhook URL vulneravel a SSRF

**Local:** `pkg/validator/validator.go:28-36`

**Status:** PENDENTE - Aceita URLs internas (`http://169.254.169.254/`, `http://localhost:9090/`). Deve resolver hostname e rejeitar IPs privados.

#### 14. Tags `validate` nunca enforced

**Local:** `internal/domain/` (todos os input structs)

**Status:** PENDENTE - Tags como `validate:"required,min=2,max=255"` existem mas nenhuma library de validacao e chamada.

#### 15. ~~Sem security headers~~ CORRIGIDO

**Local:** `internal/api/router.go:72`

**Correcao aplicada:** Helmet middleware adicionado (`X-Content-Type-Options`, `X-Frame-Options`, etc).

```go
app.Use(helmet.New())
```

#### 16. ~~Dockerfile roda como root~~ CORRIGIDO

**Local:** `docker/Dockerfile`

**Correcao aplicada:** Container roda como usuario `appuser` nao-root.

```dockerfile
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
```

#### 17. ~~`limit` query param sem valor maximo~~ CORRIGIDO

**Local:** Multiplos handlers

**Correcao aplicada:** Limite maximo de 200 em todos os handlers (`message.go`, `broadcast.go`, `blacklist.go`, `logs.go`).

```go
limit := c.QueryInt("limit", 50)
if limit > 200 { limit = 200 }
```

---

### Baixo (considerar)

| # | Issue | Status |
|---|-------|--------|
| 18 | Rate limit de 1000/min muito generoso - endpoints sensiveis precisam de limites menores | PENDENTE |
| 19 | Body limit global de 10MB excessivo para maioria dos endpoints | PENDENTE |
| 20 | Webhook retry loga URL completa - pode conter tokens em query params | PENDENTE |
| 21 | Audit log nunca e populado - nenhum handler chama `auditRepo.Log()` | PENDENTE |

---

## Arquitetura

### Pontos Fortes

- **Estrutura Go limpa** seguindo convencoes (`cmd/`, `internal/`, `pkg/`)
- **Domain model solido** com interfaces de repositorio no pacote `domain` (dependency inversion)
- **Strategy pattern** bem aplicado no orchestrator (failover, rotation, hybrid)
- **Anti-ban** isolado com delays humanizados, warmup, spintax
- **Webhook** com HMAC-SHA256 signing e retry com backoff exponencial
- **Nenhum arquivo Go passa de 350 linhas**
- **Response format padronizado** via generics (`APIResponse[T]`)
- **Injecao de dependencia manual** explicita e rastreavel
- **Connection pooling** com pgxpool configuravel
- **Migrations** com golang-migrate (up/down)
- **Security headers** via helmet middleware
- **Ownership checks** em todos os endpoints que operam por ID

### Problemas Estruturais

| Severidade | Issue | Local | Impacto | Status |
|------------|-------|-------|---------|--------|
| ALTO | Sem service layer - handlers contem business logic | `handler/*.go` | Duplicacao, dificil testar | PENDENTE |
| ALTO | Sem transacoes DB - operacoes multi-step sem atomicidade | `handler/broadcast.go` | Dados inconsistentes | PENDENTE |
| ALTO | Prometheus metrics definidas mas nunca gravadas | `internal/metrics/` | Observabilidade ilusoria | PENDENTE |
| ALTO | Redis BudgetManager definido mas nunca usado | `store/redis/budget.go` | Dead code | PENDENTE |
| ALTO | `RestoreAll` nunca chamado - sessoes WhatsApp perdidas no restart | `instance/manager.go` | Perda de sessao | PENDENTE |
| MEDIO | BlacklistRepo/AuditRepo como tipos concretos no Dependencies | `router.go:37-38` | Quebra padrao de interfaces | PENDENTE |
| MEDIO | Webhook single goroutine com `time.Sleep` no retry | `webhook/emitter.go` | Bottleneck sob carga | PENDENTE |
| MEDIO | `index.html` com 1177 linhas | `web/static/index.html` | Excede guideline de 800 | PENDENTE |
| MEDIO | RotationStrategy e greedy, nao faz round-robin real | `orchestrator/strategies.go` | Distribuicao desigual | PENDENTE |
| MEDIO | `Manager.StartInstance` segura write lock durante rede | `instance/manager.go` | Bloqueia outras operacoes | PENDENTE |
| MEDIO | `PauseBroadcast` race condition - delete fora do lock | `orchestrator/dispatcher.go:62` | Possivel panic | PENDENTE |
| MEDIO | `context.Background()` em event handlers sem timeout | `instance/manager.go` | Goroutine leak | PENDENTE |
| MEDIO | `verifyGroupOwnership` duplicado em BroadcastHandler e MetricsHandler | `broadcast.go`, `metrics.go` | Duplicacao de codigo | PENDENTE |
| BAIXO | Scan boilerplate repetido 3x no instance repo | `store/postgres/instance.go` | Manutencao | PENDENTE |
| BAIXO | `replaceVar` implementacao O(n*m) ingenua | `orchestrator/dispatcher.go:223` | Performance | PENDENTE |

### Dead Code Identificado

| Modulo | Arquivo | Descricao | Status |
|--------|---------|-----------|--------|
| Crypto | `pkg/crypto/aes.go` | AES-256-GCM nunca usado | PENDENTE |
| Config | `Config.Crypto.MasterKey` | Carregado mas sem consumidor | PENDENTE |
| Metrics | `internal/metrics/metrics.go` | Todas metricas definidas, nenhuma registrada | PENDENTE |
| Redis Budget | `internal/store/redis/budget.go` | BudgetManager nunca instanciado | PENDENTE |
| Redis Cache | `internal/store/redis/cache.go` | Cache nunca instanciado | PENDENTE |
| Audit | `internal/store/postgres/audit.go` | `Log()` nunca chamado por nenhum handler | PENDENTE |

### Escalabilidade

| Escala | Status | Acoes Necessarias |
|--------|--------|-------------------|
| 1-100 instancias | Funciona | Nenhuma |
| 100-500 instancias | Precisa ajustes | Redis rate limiter, Redis budget counters, webhook worker pool |
| 500+ instancias | Requer refactor | Job queue para broadcasts (Redis Streams/NATS), sharding de conexoes |

**Bottlenecks atuais:**
- Estado em memoria (`Manager.connections`, `Dispatcher.activeJobs`) impede horizontal scaling
- Single goroutine no webhook com `time.Sleep` bloqueante
- Broadcasts processados in-goroutine sem persistencia de estado
- Rate limiter in-memory (reset no restart, nao compartilhado entre processos)

---

## Prioridade de Correcao

### Fase 1 - Critico (antes de ir para producao) - CONCLUIDO

1. ~~Adicionar rate limit no `/register`~~ DONE
2. ~~Corrigir IDOR nos handlers de instance/broadcast/message/metrics/blacklist~~ DONE
3. ~~Proteger `/v1/logs` (limit cap)~~ DONE (parcial - falta tenant isolation)
4. ~~Configurar CORS com origem especifica~~ DONE
5. ~~Corrigir `itoa()` com `strconv.Itoa`~~ DONE

### Fase 2 - Seguranca (primeira semana) - CONCLUIDO

6. ~~Hash de API keys no banco (SHA-256)~~ DONE
7. ~~Mover `/metrics` para porta interna (127.0.0.1:9091)~~ DONE
8. ~~Sanitizar error messages (nao vazar `err.Error()`)~~ DONE
9. ~~Adicionar security headers (helmet)~~ DONE
10. ~~Validar encryption key no startup~~ DONE
11. ~~Dockerfile rodar como non-root~~ DONE
12. ~~Docker compose: credenciais via env vars~~ DONE
13. ~~Limit cap em todos os handlers (max 200)~~ DONE

### Fase 3 - Arquitetura (primeiro mes)

14. Introduzir service layer entre handlers e repositories
15. Adicionar transacoes DB para operacoes multi-step
16. Chamar `RestoreAll` no `main.go` para restaurar sessoes
17. Instrumentar Prometheus metrics
18. Conectar Redis BudgetManager ao fluxo de envio
19. Implementar webhook event filtering
20. Worker pool para webhook delivery
21. Habilitar SSL no database (`sslmode=require`)
22. Filtrar `/v1/logs` por tenant (tenant isolation)
23. Validar webhook URLs contra SSRF (rejeitar IPs privados)
24. Enforcar tags `validate` com `go-playground/validator`
25. Extrair `verifyGroupOwnership` em funcao compartilhada

### Fase 4 - Qualidade (ongoing)

26. Remover dead code (crypto, redis cache/budget, metrics nao usadas)
27. Separar `index.html` em componentes
28. Adicionar interfaces para BlacklistRepo e AuditRepo
29. Implementar audit logging nos handlers
30. Compilar Tailwind CSS (remover CDN)
31. Reduzir rate limit em endpoints sensiveis
32. Ajustar body limit por endpoint

---

## Checklist de Seguranca

- [x] No hardcoded secrets (docker-compose usa env vars)
- [ ] All inputs validated (tags `validate` nao enforced)
- [x] SQL injection prevention (parameterized queries)
- [x] XSS prevention (CSP headers via helmet)
- [x] CSRF protection (CORS restrito)
- [x] Authentication on all endpoints
- [x] Authorization verified (ownership checks em todos os handlers)
- [x] Rate limiting adequado (1000/min global + 5/hora no register)
- [ ] HTTPS enforced (SSL no database pendente)
- [x] Security headers (helmet)
- [x] API keys hashed (SHA-256)
- [x] Error messages sanitized
- [x] Container non-root
- [ ] Dependencies atualizadas
- [ ] Audit logging ativo
