# WhatsApp Gateway API — Documento de Arquitetura

## 1. Visão Geral

API não oficial de WhatsApp com foco em disparo em massa e alta disponibilidade. O diferencial é o conceito de **Instance Group** — uma camada de orquestração que gerencia múltiplos números de WhatsApp como um pool unificado, com estratégias configuráveis de rotação e failover.

### Princípios

- Uma instância = uma sessão WhatsApp = um número
- O Instance Group é a abstração que o cliente consome
- O cliente nunca precisa saber qual número está ativo
- Nenhuma instância deve operar além do seu limite seguro
- Se um número morre, o sistema se autocura

---

## 2. Stack Tecnológico

| Camada | Tecnologia | Justificativa |
|---|---|---|
| **Linguagem** | Go (1.22+) | Concorrência nativa via goroutines, baixo consumo de memória, ideal para centenas de WebSockets simultâneos |
| **Protocolo WhatsApp** | whatsmeow | Lib Go madura, usada no mautrix-whatsapp bridge, implementa Signal Protocol + Noise Pipes |
| **API HTTP** | Fiber ou Echo | Frameworks Go de alta performance, suporte a middleware, WebSocket, e rate limiting nativo |
| **Fila de mensagens** | Redis Streams / NATS JetStream | Filas de disparo por instância, redistribuição em failover, persistência de mensagens pendentes |
| **Rate Limiting** | Redis (Token Bucket) | Controle de budget por instância, por grupo e global |
| **Banco de dados** | PostgreSQL | Persistência de estado, credenciais, logs, métricas, configuração de grupos |
| **Cache** | Redis | Estado de instâncias, sessões, contadores de budget |
| **Monitoramento** | Prometheus + Grafana | Métricas de envio, taxa de entrega, saúde das instâncias |
| **Logs** | Zap (Go) + Loki | Logs estruturados com correlação por grupo/instância |
| **Deploy** | Docker + Docker Compose | Containers isolados, fácil de escalar horizontalmente |

### Dependências Externas

- **Redis 7+** — filas, cache, rate limiting, pub/sub para eventos
- **PostgreSQL 16+** — dados persistentes
- **MinIO / S3** — armazenamento de mídia (imagens, vídeos, documentos)

---

## 3. Arquitetura

### 3.1 Visão Macro

```
                    ┌─────────────────────────┐
                    │      API Gateway         │
                    │   (Auth + Rate Limit)    │
                    └────────────┬────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │    Group Orchestrator    │
                    │  (Strategy Engine)       │
                    └────────────┬────────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                   │
     ┌────────▼───────┐ ┌───────▼────────┐ ┌───────▼────────┐
     │  Instance A     │ │  Instance B     │ │  Instance C     │
     │  +55 11 9xxxx   │ │  +55 11 8xxxx   │ │  +55 21 9xxxx   │
     │  status: active │ │  status: resting│ │  status: warming│
     └────────┬───────┘ └───────┬────────┘ └───────┬────────┘
              │                  │                   │
              └──────────────────┼──────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │   WhatsApp Servers       │
                    │   (web.whatsapp.com)     │
                    └─────────────────────────┘
```

### 3.2 Componentes Internos

**API Gateway** — ponto de entrada. Valida API key, aplica rate limit global, roteia para o serviço correto.

**Group Orchestrator** — cérebro do sistema. Decide qual instância usar baseado na estratégia configurada (failover, rotation, hybrid). Gerencia o ciclo de vida das instâncias dentro do grupo.

**Instance Manager** — gerencia instâncias individuais. Cuida da conexão/desconexão, armazenamento de credenciais, QR code, status da sessão.

**Dispatcher** — consome a fila de mensagens e envia pela instância designada. Respeita rate limits, delays humanizados, e budget.

**Health Monitor** — goroutine dedicada por instância. Verifica heartbeat do WebSocket, taxa de entrega, e sinais de degradação.

**Queue Manager** — distribui mensagens entre as filas das instâncias. Em caso de failover, redistribui mensagens pendentes.

**Webhook Emitter** — unifica eventos de todas as instâncias do grupo e dispara para o webhook configurado pelo cliente.

**Media Handler** — upload/download de mídia, compressão, cache em S3/MinIO.

---

## 4. Modelo de Dados

### 4.1 Entidades Principais

```sql
-- Tenant / Cliente da API
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    api_key     VARCHAR(128) UNIQUE NOT NULL,
    plan        VARCHAR(50) DEFAULT 'basic',
    max_groups  INT DEFAULT 5,
    max_instances INT DEFAULT 20,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    is_active   BOOLEAN DEFAULT TRUE
);

-- Instance Group
CREATE TABLE instance_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id),
    name        VARCHAR(255) NOT NULL,
    strategy    VARCHAR(20) NOT NULL CHECK (strategy IN ('failover', 'rotation', 'hybrid')),
    config      JSONB NOT NULL DEFAULT '{}',
    webhook_url TEXT,
    webhook_events TEXT[] DEFAULT '{}',
    is_active   BOOLEAN DEFAULT TRUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Instância (sessão WhatsApp)
CREATE TABLE instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID REFERENCES instance_groups(id),
    phone_number    VARCHAR(20),
    display_name    VARCHAR(255),
    status          VARCHAR(20) DEFAULT 'disconnected'
                    CHECK (status IN (
                        'disconnected', 'connecting', 'available',
                        'resting', 'warming', 'suspect', 'banned'
                    )),
    priority        INT DEFAULT 0,
    daily_budget    INT DEFAULT 200,
    hourly_budget   INT DEFAULT 30,
    warmup_days     INT DEFAULT 0,
    messages_today  INT DEFAULT 0,
    messages_hour   INT DEFAULT 0,
    delivery_rate   FLOAT DEFAULT 1.0,
    last_active_at  TIMESTAMPTZ,
    banned_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Credenciais de sessão (criptografadas)
CREATE TABLE session_credentials (
    instance_id     UUID PRIMARY KEY REFERENCES instances(id),
    creds_encrypted BYTEA NOT NULL,
    iv              BYTEA NOT NULL,
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- Log de mensagens
CREATE TABLE message_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id        UUID REFERENCES instance_groups(id),
    instance_id     UUID REFERENCES instances(id),
    recipient       VARCHAR(20) NOT NULL,
    message_type    VARCHAR(20) NOT NULL,
    content_hash    VARCHAR(64),
    status          VARCHAR(20) DEFAULT 'queued'
                    CHECK (status IN ('queued', 'sent', 'delivered', 'read', 'failed')),
    error_code      VARCHAR(50),
    queued_at       TIMESTAMPTZ DEFAULT NOW(),
    sent_at         TIMESTAMPTZ,
    delivered_at    TIMESTAMPTZ,
    read_at         TIMESTAMPTZ
);

-- Métricas diárias por instância
CREATE TABLE instance_metrics (
    instance_id     UUID REFERENCES instances(id),
    date            DATE NOT NULL,
    messages_sent   INT DEFAULT 0,
    messages_delivered INT DEFAULT 0,
    messages_failed INT DEFAULT 0,
    delivery_rate   FLOAT DEFAULT 0,
    avg_delivery_time_ms INT DEFAULT 0,
    PRIMARY KEY (instance_id, date)
);
```

---

## 5. API — Endpoints

### 5.1 Autenticação

Toda request precisa do header `X-API-Key`. O gateway valida contra a tabela `tenants`.

### 5.2 Instance Groups

```
POST   /api/v1/groups                    — Criar grupo
GET    /api/v1/groups                    — Listar grupos do tenant
GET    /api/v1/groups/:id                — Detalhes do grupo
PATCH  /api/v1/groups/:id                — Atualizar configuração
DELETE /api/v1/groups/:id                — Remover grupo (desconecta todas)
GET    /api/v1/groups/:id/status         — Status consolidado (instâncias, filas)
```

**Criação do grupo — payload:**

```json
{
  "name": "campanha-black-friday",
  "strategy": "rotation",
  "config": {
    "daily_budget_per_instance": 200,
    "hourly_budget_per_instance": 30,
    "min_delay_ms": 2000,
    "max_delay_ms": 7000,
    "delivery_rate_threshold": 0.85,
    "warmup_enabled": true,
    "warmup_days": 14,
    "warmup_initial_budget": 20,
    "cooldown_hours": 8
  },
  "webhook_url": "https://meusite.com/webhook",
  "webhook_events": ["message.sent", "message.delivered", "message.failed", "instance.status_changed", "instance.banned"]
}
```

**Criação do grupo com estratégia failover:**

```json
{
  "name": "atendimento-suporte",
  "strategy": "failover",
  "config": {
    "health_check_interval_s": 30,
    "failover_timeout_s": 10,
    "auto_promote": true,
    "max_retry_before_failover": 3
  },
  "webhook_url": "https://meusite.com/webhook",
  "webhook_events": ["message.received", "instance.status_changed"]
}
```

**Criação do grupo com estratégia híbrida:**

```json
{
  "name": "disparo-agressivo",
  "strategy": "hybrid",
  "config": {
    "daily_budget_per_instance": 300,
    "hourly_budget_per_instance": 50,
    "min_delay_ms": 1500,
    "max_delay_ms": 5000,
    "delivery_rate_threshold": 0.80,
    "warmup_enabled": true,
    "warmup_days": 7,
    "warmup_initial_budget": 30,
    "cooldown_hours": 6,
    "failover_timeout_s": 5,
    "auto_promote": true
  }
}
```

### 5.3 Instances

```
POST   /api/v1/groups/:id/instances              — Adicionar instância ao grupo
GET    /api/v1/groups/:id/instances              — Listar instâncias do grupo
DELETE /api/v1/groups/:id/instances/:instanceId  — Remover instância
GET    /api/v1/instances/:id/qrcode              — Obter QR Code para conectar
POST   /api/v1/instances/:id/pair                — Conectar via pairing code
GET    /api/v1/instances/:id/status              — Status da instância
POST   /api/v1/instances/:id/disconnect          — Desconectar sessão
POST   /api/v1/instances/:id/restart             — Reconectar sessão
PATCH  /api/v1/instances/:id/priority            — Alterar prioridade na fila
```

### 5.4 Mensagens (via grupo)

```
POST   /api/v1/groups/:id/messages/send          — Envio único
POST   /api/v1/groups/:id/messages/broadcast      — Disparo em massa
GET    /api/v1/groups/:id/messages/:msgId/status  — Status de uma mensagem
POST   /api/v1/groups/:id/messages/broadcast/cancel — Cancelar disparo em andamento
GET    /api/v1/groups/:id/messages/stats          — Estatísticas de envio
```

**Envio único:**

```json
{
  "to": "5511999999999",
  "type": "text",
  "content": {
    "body": "Olá, tudo bem?"
  }
}
```

**Disparo em massa:**

```json
{
  "recipients": ["5511999999999", "5511888888888"],
  "type": "text",
  "content": {
    "body": "Promoção especial para você! {{nome}}"
  },
  "variables": {
    "5511999999999": { "nome": "João" },
    "5511888888888": { "nome": "Maria" }
  },
  "options": {
    "shuffle_recipients": true,
    "vary_content": true,
    "schedule_at": "2025-01-15T10:00:00Z"
  }
}
```

**Tipos de mensagem suportados:**

- `text` — texto simples
- `image` — imagem com caption opcional
- `video` — vídeo com caption opcional
- `audio` — áudio (PTT ou arquivo)
- `document` — documento com filename
- `location` — latitude/longitude
- `contact` — vCard
- `sticker` — sticker WebP
- `reaction` — reação a mensagem
- `poll` — enquete

### 5.5 Contatos e Grupos WhatsApp

```
GET    /api/v1/groups/:id/contacts               — Listar contatos
POST   /api/v1/groups/:id/contacts/check          — Verificar se números têm WhatsApp
GET    /api/v1/groups/:id/whatsapp-groups         — Listar grupos de WhatsApp
POST   /api/v1/groups/:id/whatsapp-groups/:wgId/send — Enviar para grupo WhatsApp
```

### 5.6 Webhooks

```
PUT    /api/v1/groups/:id/webhook                 — Configurar webhook
GET    /api/v1/groups/:id/webhook                 — Ver configuração atual
POST   /api/v1/groups/:id/webhook/test            — Testar webhook
GET    /api/v1/groups/:id/webhook/logs            — Logs de entrega do webhook
```

**Payload de evento do webhook:**

```json
{
  "event": "message.delivered",
  "group_id": "uuid",
  "instance_id": "uuid",
  "phone_number": "+5511999999999",
  "timestamp": "2025-01-15T10:05:32Z",
  "data": {
    "message_id": "uuid",
    "recipient": "5511888888888",
    "delivered_at": "2025-01-15T10:05:32Z"
  }
}
```

---

## 6. Estratégias de Orquestração

### 6.1 Failover

```
                ┌──── health check (30s) ────┐
                │                             │
                ▼                             │
  ┌─────────────────────┐                    │
  │   Instance A (active)│──── TIMEOUT ──────┤
  └──────────┬──────────┘                    │
             │ ban detectado                  │
             ▼                                │
  ┌─────────────────────┐                    │
  │   Instance B         │◄── auto promote   │
  │   (standby → active) │                    │
  └──────────┬──────────┘                    │
             │                                │
             └────── health check continua ──┘
```

**Regras:**

- Apenas uma instância fica `active`, as demais ficam `standby`
- O health check roda a cada N segundos (configurável)
- Se a instância ativa não responde em `failover_timeout_s`, promove a próxima
- A ordem de promoção segue o campo `priority` da instância
- Se `auto_promote` está desligado, emite evento e espera ação manual

### 6.2 Rotation

```
  ┌─────────────────────────────────────────┐
  │           Queue Manager                  │
  │  [msg1] [msg2] [msg3] [msg4] [msg5]     │
  └─────────┬───────────┬───────────┬───────┘
            │           │           │
    ┌───────▼──┐  ┌─────▼────┐  ┌──▼────────┐
    │ Inst A    │  │ Inst B    │  │ Inst C     │
    │ budget:   │  │ budget:   │  │ budget:    │
    │ 180/200   │  │ 200/200   │  │ 45/60      │
    │ available │  │ available │  │ warming    │
    └──────────┘  └──────────┘  └───────────┘
```

**Regras:**

- Todas as instâncias `available` recebem mensagens
- O dispatcher seleciona a instância com **maior budget restante**
- Quando budget diário zera → status muda para `resting`
- Quando budget horário zera → pausa temporária (sem mudar status)
- Números em `warming` recebem carga reduzida conforme o warmup schedule
- A cada novo dia (meia-noite UTC ou configurável), budgets são resetados e instâncias `resting` voltam para `available`

**Warmup Schedule:**

| Dia | Budget (% do total) |
|-----|---------------------|
| 1-3 | 10% |
| 4-7 | 25% |
| 8-10 | 50% |
| 11-14 | 75% |
| 15+ | 100% |

### 6.3 Hybrid (Rotation + Failover)

Combina as duas estratégias. Opera em rotação normalmente, mas quando uma instância é banida:

1. Remove imediatamente do pool de rotação
2. Redistribui a fila pendente entre as instâncias restantes
3. Emite evento `instance.banned`
4. Recalcula budgets (carga distribuída entre menos instâncias)
5. Se o número de instâncias `available` cai abaixo de um mínimo, emite alerta crítico

---

## 7. Segurança

### 7.1 Autenticação e Autorização

- **API Key por tenant** — hash bcrypt armazenado no banco, validado no gateway
- **Scoped API Keys** — possibilidade de criar keys com permissões específicas (só leitura, só envio, admin)
- **IP Allowlist** — opcional, restringe quais IPs podem usar a API key
- **Rate limit por tenant** — independente do rate limit por instância, limita requests/minuto do cliente na API

### 7.2 Criptografia

- **Credenciais de sessão** — criptografadas com AES-256-GCM antes de salvar no banco. A chave de criptografia fica em variável de ambiente, nunca no banco.
- **TLS em tudo** — API servida via HTTPS, conexões ao banco e Redis com TLS habilitado
- **Webhook signature** — cada payload de webhook é assinado com HMAC-SHA256 usando um secret por grupo. O header `X-Webhook-Signature` permite ao cliente validar autenticidade.
- **Mídia** — arquivos armazenados no S3/MinIO com URLs pré-assinadas de curta duração (15 min)

### 7.3 Isolamento

- **Multi-tenant rígido** — queries sempre filtram por `tenant_id`, não é possível um tenant acessar dados de outro
- **Instâncias isoladas** — cada instância roda sua goroutine com contexto próprio, crash de uma não afeta outras
- **Secrets separados** — cada grupo pode ter seu próprio webhook secret

### 7.4 Proteção contra abuso

- **Rate limiting em cascata:** global → tenant → grupo → instância
- **Validação de conteúdo** — bloqueia payloads maiores que o limite, valida tipos de arquivo
- **Blacklist de números** — lista de números que não devem ser contatados (opt-out / compliance)
- **Audit log** — toda ação administrativa é registrada com timestamp, IP, e tenant

---

## 8. Anti-Ban — Estratégias

Esta é a camada mais crítica do sistema. O WhatsApp bane agressivamente números que apresentam comportamento automatizado.

### 8.1 Delay Humanizado

Nunca usar delay fixo. O intervalo entre mensagens deve simular comportamento humano:

```
delay = base_delay + random(0, jitter)

// Exemplo:
// base_delay = 3000ms
// jitter = 4000ms
// resultado: delay entre 3s e 7s, distribuição uniforme
```

Adicionar delays extras:

- **Entre blocos de mensagens** — a cada 20-30 mensagens, pausa de 30-120 segundos
- **Variação por horário** — de madrugada, delays maiores (humano não dispara 300 msgs às 3h da manhã)
- **Typing simulation** — antes de enviar texto, emitir "composing" event com duração proporcional ao tamanho da mensagem

### 8.2 Variação de Conteúdo

Mensagens idênticas em massa são o maior gatilho de ban. Estratégias:

- **Spintax** — `{Olá|Oi|E aí}, {{nome}}! {Tudo bem|Como vai}?` gera combinações diferentes
- **Variáveis por destinatário** — personalização com dados do contato
- **Ordem aleatória de destinatários** — nunca enviar na ordem sequencial da lista
- **Variação de mídia** — se enviando imagem, alterar levemente metadata (não o conteúdo visual) para gerar hashes diferentes

### 8.3 Warmup de Números Novos

Número novo que começa disparando em massa é banido em horas. O warmup gradual é obrigatório:

- **Semana 1** — apenas mensagens individuais para contatos que respondem, máximo 20/dia
- **Semana 2** — aumentar para 50/dia, incluir alguns grupos
- **Semana 3** — 100/dia, incluir envio de mídia
- **Semana 4+** — budget completo

O sistema deve automatizar isso: ao adicionar uma instância ao grupo, ela entra em status `warming` e o dispatcher respeita o budget reduzido automaticamente.

### 8.4 Monitoramento Preditivo

Em vez de reagir ao ban, detectar sinais antecipados:

| Sinal | Ação |
|---|---|
| Taxa de entrega cai abaixo de 90% | Reduzir velocidade de envio pela metade |
| Taxa de entrega cai abaixo de 80% | Pausar envio, marcar como `suspect` |
| WebSocket desconecta e não reconecta em 30s | Failover imediato |
| Mensagens ficam em `sent` sem virar `delivered` por 5+ min | Sinal de shadow ban, pausar |
| Número de `read receipts` cai drasticamente | Possível restrição, monitorar |

### 8.5 Horário de Operação

Configurar janelas de envio que simulam horário comercial:

```json
{
  "operating_hours": {
    "timezone": "America/Sao_Paulo",
    "windows": [
      { "start": "08:00", "end": "12:00" },
      { "start": "13:30", "end": "18:00" }
    ],
    "days": ["mon", "tue", "wed", "thu", "fri"]
  }
}
```

Mensagens fora da janela são enfileiradas para o próximo período.

---

## 9. Otimizações de Performance

### 9.1 Connection Pool

Cada instância mantém um único WebSocket, mas o sistema precisa gerenciar eficientemente:

- **Goroutine por instância** — lightweight, Go gerencia scheduling
- **Context com cancel** — cada instância tem seu `context.Context` derivado. Cancel do contexto faz cleanup limpo
- **Graceful shutdown** — ao desligar o servidor, salva estado das filas no Redis antes de fechar conexões

### 9.2 Batch Processing

Para disparo em massa:

- **Pré-processamento em batch** — validar todos os números, resolver variáveis, gerar conteúdo variado, tudo antes de enfileirar
- **Enfileiramento em batch** — usar `XADD` do Redis Streams em pipeline, não um por um
- **Confirmação assíncrona** — o endpoint de broadcast retorna imediatamente com um `broadcast_id`, o cliente consulta progresso via polling ou webhook

### 9.3 Cache Inteligente

- **Status de instância** — cacheado no Redis, atualizado por evento (não por polling)
- **Budget counters** — Redis `INCR` atômico, sem necessidade de query SQL
- **Contatos verificados** — cache de quais números têm WhatsApp (TTL de 24h)
- **Media dedup** — se o mesmo arquivo é enviado para múltiplos destinatários, upload uma vez e reusa a media key

### 9.4 Compressão de Filas

Para broadcasts com milhares de destinatários:

- Não criar uma entrada na fila por destinatário. Criar um `broadcast job` com a lista completa
- O dispatcher puxa do job e processa em chunks
- Permite cancelamento eficiente (marca o job como cancelado, dispatcher para de consumir)

### 9.5 Database

- **Particionamento de `message_logs`** — particionar por mês. Tabela de logs cresce rápido com disparo em massa
- **Índices parciais** — ex: índice em `status = 'queued'` para queries do dispatcher
- **Connection pooling** — PgBouncer na frente do PostgreSQL
- **Read replicas** — queries de métricas e histórico vão para réplica, escrita vai para o primário

---

## 10. Observabilidade

### 10.1 Métricas (Prometheus)

```
# Mensagens
wpp_messages_sent_total{group_id, instance_id, status}
wpp_messages_delivery_duration_seconds{group_id, instance_id}
wpp_messages_in_queue{group_id}

# Instâncias
wpp_instance_status{group_id, instance_id, status}
wpp_instance_budget_remaining{group_id, instance_id}
wpp_instance_delivery_rate{group_id, instance_id}
wpp_instance_websocket_latency_ms{instance_id}

# Grupos
wpp_group_active_instances{group_id}
wpp_group_failovers_total{group_id}
wpp_group_messages_per_second{group_id}

# Sistema
wpp_goroutines_active
wpp_redis_connection_pool_usage
wpp_pg_connection_pool_usage
```

### 10.2 Alertas

| Alerta | Condição | Severidade |
|---|---|---|
| Instância banida | `status = banned` | Critical |
| Grupo sem instâncias ativas | `available_count = 0` | Critical |
| Taxa de entrega baixa | `delivery_rate < 0.85` por 5 min | Warning |
| Fila crescendo | `queue_size` aumentando por 10+ min | Warning |
| Shadow ban suspeito | `sent` sem `delivered` por 5+ min | Warning |
| Budget do grupo esgotado | Todas instâncias `resting` | Info |
| WebSocket instável | Reconexão > 3x em 5 min | Warning |
| Redis latência alta | p99 > 50ms | Warning |

### 10.3 Dashboard

Dashboard Grafana com painéis:

- **Overview** — total de mensagens/dia, taxa de entrega global, instâncias ativas
- **Por Grupo** — status de cada instância, budget restante, fila pendente
- **Disparo em andamento** — progresso do broadcast, estimativa de conclusão
- **Health** — latência do WebSocket, reconexões, erros
- **Anti-ban** — gráfico de taxa de entrega por instância ao longo do tempo

---

## 11. Deploy e Infraestrutura

### 11.1 Docker Compose (desenvolvimento / instância única)

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:pass@postgres:5432/wppgateway
      - REDIS_URL=redis://redis:6379
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - LOG_LEVEL=info
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    volumes:
      - pgdata:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: wppgateway
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redisdata:/data
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafanadata:/var/lib/grafana
    restart: unless-stopped

volumes:
  pgdata:
  redisdata:
  grafanadata:
```

### 11.2 Produção — Considerações

- **Horizontal scaling** — múltiplos containers da API atrás de um load balancer. Cada container gerencia um subset das instâncias.
- **Instance affinity** — uma instância (WebSocket) roda em apenas um container. Usar Redis para coordenação de quem gerencia qual instância (distributed lock).
- **Backup** — pg_dump diário + WAL archiving para point-in-time recovery. Redis AOF para persistência das filas.
- **Blue-green deploy** — ao atualizar, subir nova versão e migrar instâncias gradualmente (sem derrubar todas as conexões de uma vez).

---

## 12. Estrutura de Diretórios do Projeto

```
wpp-gateway/
├── cmd/
│   └── server/
│       └── main.go                  # Entrypoint
├── internal/
│   ├── api/
│   │   ├── handler/                 # HTTP handlers
│   │   │   ├── group.go
│   │   │   ├── instance.go
│   │   │   ├── message.go
│   │   │   └── webhook.go
│   │   ├── middleware/              # Auth, rate limit, logging
│   │   │   ├── auth.go
│   │   │   ├── ratelimit.go
│   │   │   └── logger.go
│   │   └── router.go               # Roteamento
│   ├── orchestrator/
│   │   ├── group.go                 # Group Orchestrator
│   │   ├── strategy_failover.go     # Lógica de failover
│   │   ├── strategy_rotation.go     # Lógica de rotação
│   │   ├── strategy_hybrid.go       # Lógica híbrida
│   │   └── dispatcher.go           # Queue consumer + sender
│   ├── instance/
│   │   ├── manager.go               # Lifecycle management
│   │   ├── connection.go            # WhatsApp connection (whatsmeow)
│   │   ├── health.go                # Health monitoring
│   │   └── session.go               # Credential management
│   ├── queue/
│   │   ├── redis_stream.go          # Redis Streams implementation
│   │   └── manager.go               # Queue distribution logic
│   ├── webhook/
│   │   ├── emitter.go               # Event emission
│   │   └── signer.go                # HMAC signing
│   ├── antiban/
│   │   ├── delay.go                 # Humanized delays
│   │   ├── spintax.go               # Content variation
│   │   ├── warmup.go                # Warmup schedule
│   │   └── monitor.go              # Predictive monitoring
│   ├── media/
│   │   ├── handler.go               # Upload/download
│   │   └── s3.go                    # S3/MinIO integration
│   ├── store/
│   │   ├── postgres/                # SQL queries e repos
│   │   │   ├── tenant.go
│   │   │   ├── group.go
│   │   │   ├── instance.go
│   │   │   └── message.go
│   │   └── redis/                   # Cache e counters
│   │       ├── budget.go
│   │       ├── cache.go
│   │       └── lock.go
│   └── config/
│       └── config.go                # Configuração via env vars
├── pkg/
│   ├── crypto/                      # AES encryption helpers
│   ├── validator/                   # Input validation
│   └── logger/                      # Structured logging setup
├── migrations/                      # SQL migrations
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml
├── grafana/
│   └── dashboards/                  # Dashboard JSON exports
├── prometheus/
│   └── prometheus.yml
├── go.mod
├── go.sum
└── README.md
```

---

## 13. Roadmap de Desenvolvimento

### Fase 1 — Core (semanas 1-4)
- Instância única: conectar, enviar/receber mensagens, webhook
- CRUD de instâncias e tenants
- Persistência de sessão (reconnect sem novo QR)

### Fase 2 — Instance Groups (semanas 5-8)
- Implementar conceito de grupo
- Estratégia failover
- Health monitoring
- Dashboard básico

### Fase 3 — Rotation + Broadcast (semanas 9-12)
- Estratégia rotation
- Estratégia hybrid
- Disparo em massa com filas
- Spintax e variação de conteúdo
- Warmup automático

### Fase 4 — Anti-ban + Observabilidade (semanas 13-16)
- Delay humanizado completo
- Monitoramento preditivo
- Métricas Prometheus + Grafana
- Alertas automatizados
- Audit log

### Fase 5 — Produção (semanas 17-20)
- Horizontal scaling
- Testes de carga
- Documentação da API (OpenAPI/Swagger)
- SDK cliente (Go, Node, Python)
- Painel administrativo web

---

## 14. Pontos de Atenção

**Legais** — APIs não oficiais violam os Termos de Serviço do WhatsApp. Números podem ser banidos permanentemente. Para uso comercial legítimo, considere a API oficial (WhatsApp Business Cloud API).

**Resiliência** — o protocolo do WhatsApp muda sem aviso. A lib whatsmeow pode quebrar a qualquer atualização do WhatsApp. Manter dependências atualizadas e ter alertas para falhas de protocolo.

**Escala** — cada instância consome ~5-10MB de RAM. Com 100 instâncias, o servidor precisa de pelo menos 2GB só para conexões. Planejar capacity de acordo.

**Backup de números** — números banidos não voltam. Ter estoque de chips/números e processo de warmup contínuo para repor perdas.

**Compliance** — implementar opt-out (LGPD/GDPR), blacklist de números, e limites de envio por destinatário para evitar spam abusivo.
