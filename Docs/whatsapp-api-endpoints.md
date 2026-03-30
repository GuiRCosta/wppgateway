# WhatsApp Gateway API — Referência Completa de Endpoints

> Base URL: `https://api.seudominio.com/v1`
> Autenticação: Header `X-API-Key: {sua_api_key}`
> Content-Type: `application/json`

---

## 1. Autenticação e Tenant

### API Keys

```
POST   /auth/keys                        — Gerar nova API key (scoped)
GET    /auth/keys                        — Listar API keys ativas
DELETE /auth/keys/:keyId                 — Revogar API key
PATCH  /auth/keys/:keyId                 — Atualizar permissões/IP allowlist
```

**Criar API key com escopo:**

```json
POST /auth/keys
{
  "name": "key-disparo-marketing",
  "scopes": ["messages:write", "groups:read", "instances:read"],
  "allowed_ips": ["189.10.20.30", "189.10.20.31"],
  "expires_at": "2026-12-31T23:59:59Z"
}
```

**Escopos disponíveis:**

| Escopo | Descrição |
|---|---|
| `*` | Acesso total (admin) |
| `groups:read` | Ler grupos e status |
| `groups:write` | Criar/editar/deletar grupos |
| `instances:read` | Ler instâncias e status |
| `instances:write` | Criar/conectar/desconectar instâncias |
| `messages:read` | Consultar mensagens e status |
| `messages:write` | Enviar mensagens |
| `contacts:read` | Listar/verificar contatos |
| `webhooks:write` | Configurar webhooks |
| `metrics:read` | Acessar métricas e relatórios |

### Tenant (conta)

```
GET    /account                          — Dados da conta
PATCH  /account                          — Atualizar dados
GET    /account/usage                    — Uso atual (instâncias, mensagens, limites)
```

---

## 2. Instance Groups

```
POST   /groups                           — Criar grupo
GET    /groups                           — Listar grupos
GET    /groups/:groupId                  — Detalhes do grupo
PATCH  /groups/:groupId                  — Atualizar configuração/estratégia
DELETE /groups/:groupId                  — Deletar grupo (desconecta tudo)
GET    /groups/:groupId/status           — Status consolidado em tempo real
GET    /groups/:groupId/metrics          — Métricas do grupo (envios, entregas, bans)
POST   /groups/:groupId/pause            — Pausar todos os disparos do grupo
POST   /groups/:groupId/resume           — Retomar disparos
```

---

## 3. Instâncias

### Lifecycle

```
POST   /groups/:groupId/instances                     — Adicionar instância ao grupo
GET    /groups/:groupId/instances                     — Listar instâncias do grupo
GET    /instances/:instanceId                          — Detalhes da instância
DELETE /groups/:groupId/instances/:instanceId          — Remover instância do grupo
PATCH  /instances/:instanceId                          — Atualizar config (priority, budget)
```

### Conexão

```
GET    /instances/:instanceId/qrcode                   — Obter QR code (base64 + raw)
GET    /instances/:instanceId/qrcode/image             — QR code como imagem PNG
POST   /instances/:instanceId/pair                     — Conectar via pairing code (8 dígitos)
POST   /instances/:instanceId/connect                  — Reconectar sessão existente
POST   /instances/:instanceId/disconnect               — Desconectar (mantém credenciais)
POST   /instances/:instanceId/logout                   — Logout completo (apaga credenciais)
POST   /instances/:instanceId/restart                  — Desconectar e reconectar
GET    /instances/:instanceId/connection-state          — Estado da conexão WebSocket
```

### Perfil da instância

```
GET    /instances/:instanceId/profile                  — Dados do perfil (nome, foto, status)
PATCH  /instances/:instanceId/profile/name             — Alterar nome de exibição
PATCH  /instances/:instanceId/profile/status           — Alterar recado/status
PATCH  /instances/:instanceId/profile/photo            — Alterar foto de perfil
DELETE /instances/:instanceId/profile/photo             — Remover foto de perfil
```

### Presence (presença online)

```
POST   /instances/:instanceId/presence/available       — Ficar "online"
POST   /instances/:instanceId/presence/unavailable     — Ficar "offline"
```

---

## 4. Mensagens — Envio

Todos os endpoints de envio funcionam **via grupo** (o orchestrator escolhe a instância) ou **via instância direta** (bypass do grupo).

### Via grupo (recomendado)

```
POST   /groups/:groupId/messages/send                  — Envio único
POST   /groups/:groupId/messages/broadcast             — Disparo em massa
GET    /groups/:groupId/messages/broadcast/:broadcastId — Status do broadcast
POST   /groups/:groupId/messages/broadcast/:broadcastId/cancel — Cancelar broadcast
POST   /groups/:groupId/messages/broadcast/:broadcastId/pause  — Pausar broadcast
POST   /groups/:groupId/messages/broadcast/:broadcastId/resume — Retomar broadcast
```

### Via instância direta (para casos específicos)

```
POST   /instances/:instanceId/messages/send            — Envio por instância específica
```

### 4.1 Texto

```json
POST /groups/:groupId/messages/send
{
  "to": "5511999999999",
  "type": "text",
  "content": {
    "body": "Olá, tudo bem?",
    "preview_url": true
  }
}
```

### 4.2 Imagem

```json
{
  "to": "5511999999999",
  "type": "image",
  "content": {
    "url": "https://exemplo.com/foto.jpg",
    "caption": "Confira nossa promoção!",
    "mime_type": "image/jpeg"
  }
}
```

**Ou com upload base64:**

```json
{
  "to": "5511999999999",
  "type": "image",
  "content": {
    "base64": "/9j/4AAQSkZJRgABAQ...",
    "caption": "Confira nossa promoção!",
    "mime_type": "image/jpeg",
    "filename": "promo.jpg"
  }
}
```

### 4.3 Vídeo

```json
{
  "to": "5511999999999",
  "type": "video",
  "content": {
    "url": "https://exemplo.com/video.mp4",
    "caption": "Veja o vídeo!",
    "mime_type": "video/mp4"
  }
}
```

### 4.4 Áudio

```json
{
  "to": "5511999999999",
  "type": "audio",
  "content": {
    "url": "https://exemplo.com/audio.ogg",
    "mime_type": "audio/ogg; codecs=opus",
    "ptt": true
  }
}
```

> `ptt: true` = envia como mensagem de voz (push-to-talk), com o ícone de microfone. `ptt: false` = envia como arquivo de áudio.

### 4.5 Documento

```json
{
  "to": "5511999999999",
  "type": "document",
  "content": {
    "url": "https://exemplo.com/contrato.pdf",
    "mime_type": "application/pdf",
    "filename": "Contrato_2025.pdf",
    "caption": "Segue o contrato para assinatura"
  }
}
```

### 4.6 Sticker

```json
{
  "to": "5511999999999",
  "type": "sticker",
  "content": {
    "url": "https://exemplo.com/sticker.webp",
    "mime_type": "image/webp"
  }
}
```

> Stickers devem ser WebP, 512x512px, máximo 100KB (estático) ou 500KB (animado).

### 4.7 Localização

```json
{
  "to": "5511999999999",
  "type": "location",
  "content": {
    "latitude": -23.5505,
    "longitude": -46.6333,
    "name": "Escritório São Paulo",
    "address": "Av. Paulista, 1000 - São Paulo, SP"
  }
}
```

### 4.8 Contato (vCard)

```json
{
  "to": "5511999999999",
  "type": "contact",
  "content": {
    "contacts": [
      {
        "name": {
          "formatted_name": "João Silva",
          "first_name": "João",
          "last_name": "Silva"
        },
        "phones": [
          {
            "phone": "+5511999999999",
            "type": "CELL"
          }
        ],
        "emails": [
          {
            "email": "joao@email.com",
            "type": "WORK"
          }
        ],
        "org": {
          "company": "Empresa XYZ"
        }
      }
    ]
  }
}
```

### 4.9 Reação

```json
{
  "to": "5511999999999",
  "type": "reaction",
  "content": {
    "message_id": "ABCDEF123456",
    "emoji": "👍"
  }
}
```

> Envie `emoji: ""` para remover uma reação.

### 4.10 Enquete / Poll

```json
{
  "to": "5511999999999",
  "type": "poll",
  "content": {
    "title": "Qual horário prefere?",
    "options": ["Manhã (8h-12h)", "Tarde (13h-18h)", "Noite (19h-22h)"],
    "max_selections": 1
  }
}
```

### 4.11 Link com Preview

```json
{
  "to": "5511999999999",
  "type": "text",
  "content": {
    "body": "Confira nosso site: https://meusite.com.br",
    "preview_url": true
  }
}
```

---

## 5. Mensagens Interativas — Botões e Listas

### 5.1 Botões (até 3)

```json
{
  "to": "5511999999999",
  "type": "button",
  "content": {
    "body": "Escolha uma opção para continuar:",
    "footer": "Responda clicando no botão",
    "buttons": [
      { "id": "btn_sim", "text": "✅ Sim, quero!" },
      { "id": "btn_nao", "text": "❌ Não, obrigado" },
      { "id": "btn_info", "text": "ℹ️ Mais informações" }
    ]
  }
}
```

### 5.2 Botões com Imagem/Vídeo/Documento (header de mídia)

```json
{
  "to": "5511999999999",
  "type": "button",
  "content": {
    "header": {
      "type": "image",
      "url": "https://exemplo.com/produto.jpg"
    },
    "body": "Produto XYZ — R$ 99,90. Deseja comprar?",
    "footer": "Oferta válida até 30/01",
    "buttons": [
      { "id": "btn_comprar", "text": "Comprar agora" },
      { "id": "btn_carrinho", "text": "Adicionar ao carrinho" }
    ]
  }
}
```

### 5.3 Lista (até 10 itens, divididos em seções)

```json
{
  "to": "5511999999999",
  "type": "list",
  "content": {
    "body": "Confira nosso cardápio do dia:",
    "footer": "Selecione um item para fazer o pedido",
    "button_text": "Ver cardápio",
    "sections": [
      {
        "title": "🍕 Pizzas",
        "items": [
          {
            "id": "pizza_marg",
            "title": "Margherita",
            "description": "Molho, mussarela e manjericão — R$ 39,90"
          },
          {
            "id": "pizza_pepperoni",
            "title": "Pepperoni",
            "description": "Molho, mussarela e pepperoni — R$ 44,90"
          }
        ]
      },
      {
        "title": "🥤 Bebidas",
        "items": [
          {
            "id": "bebida_refri",
            "title": "Refrigerante 2L",
            "description": "Coca-Cola, Guaraná ou Fanta — R$ 12,90"
          },
          {
            "id": "bebida_suco",
            "title": "Suco Natural",
            "description": "Laranja, limão ou maracujá — R$ 9,90"
          }
        ]
      }
    ]
  }
}
```

### 5.4 Botões de URL e Chamada (Call to Action)

```json
{
  "to": "5511999999999",
  "type": "cta_button",
  "content": {
    "body": "Acesse nosso site para finalizar a compra:",
    "buttons": [
      {
        "type": "url",
        "text": "Abrir site",
        "url": "https://meusite.com.br/checkout"
      },
      {
        "type": "phone",
        "text": "Ligar para suporte",
        "phone": "+5511999999999"
      }
    ]
  }
}
```

### 5.5 Template com Variáveis (mensagens modelo)

```json
{
  "to": "5511999999999",
  "type": "template",
  "content": {
    "name": "confirmacao_pedido",
    "language": "pt_BR",
    "components": [
      {
        "type": "body",
        "parameters": [
          { "type": "text", "text": "João" },
          { "type": "text", "text": "#12345" },
          { "type": "text", "text": "R$ 159,90" }
        ]
      },
      {
        "type": "button",
        "sub_type": "url",
        "index": 0,
        "parameters": [
          { "type": "text", "text": "12345" }
        ]
      }
    ]
  }
}
```

### 5.6 Product Message (catálogo)

```json
{
  "to": "5511999999999",
  "type": "product",
  "content": {
    "catalog_id": "123456789",
    "product_id": "SKU-001",
    "body": "Confira este produto!"
  }
}
```

### 5.7 Product List (vários produtos)

```json
{
  "to": "5511999999999",
  "type": "product_list",
  "content": {
    "catalog_id": "123456789",
    "header": "Nossos mais vendidos",
    "body": "Escolha seus produtos favoritos:",
    "sections": [
      {
        "title": "Camisetas",
        "product_ids": ["SKU-001", "SKU-002", "SKU-003"]
      },
      {
        "title": "Calças",
        "product_ids": ["SKU-010", "SKU-011"]
      }
    ]
  }
}
```

---

## 6. Disparo em Massa (Broadcast)

### 6.1 Broadcast simples

```json
POST /groups/:groupId/messages/broadcast
{
  "recipients": [
    "5511999999999",
    "5511888888888",
    "5521777777777"
  ],
  "type": "text",
  "content": {
    "body": "Olá {{nome}}, aproveite {{desconto}} de desconto! {{link}}"
  },
  "variables": {
    "5511999999999": { "nome": "João", "desconto": "20%", "link": "https://site.com/j" },
    "5511888888888": { "nome": "Maria", "desconto": "15%", "link": "https://site.com/m" },
    "5521777777777": { "nome": "Carlos", "desconto": "25%", "link": "https://site.com/c" }
  },
  "options": {
    "shuffle_recipients": true,
    "vary_content": true,
    "schedule_at": null,
    "respect_operating_hours": true,
    "skip_invalid_numbers": true,
    "skip_non_whatsapp": true
  }
}
```

**Resposta:**

```json
{
  "broadcast_id": "bc_abc123",
  "status": "processing",
  "total_recipients": 3,
  "estimated_duration_minutes": 1,
  "created_at": "2025-01-15T10:00:00Z"
}
```

### 6.2 Broadcast com mídia

```json
{
  "recipients": ["5511999999999", "5511888888888"],
  "type": "image",
  "content": {
    "url": "https://exemplo.com/promo.jpg",
    "caption": "{{nome}}, essa oferta é para você! 🔥"
  },
  "variables": {
    "5511999999999": { "nome": "João" },
    "5511888888888": { "nome": "Maria" }
  }
}
```

### 6.3 Broadcast com Spintax

```json
{
  "recipients": ["5511999999999", "5511888888888"],
  "type": "text",
  "content": {
    "body": "{Olá|Oi|E aí|Fala}, {{nome}}! {Tudo bem|Tudo certo|Como vai}? {Queria te contar|Sabia que|Olha só}: {temos|estamos com|preparamos} {uma promoção|um desconto|uma oferta} {incrível|especial|imperdível} {pra você|exclusiva|só hoje}! {Confere lá|Dá uma olhada|Acessa}: {{link}}"
  },
  "variables": {
    "5511999999999": { "nome": "João", "link": "https://site.com/j" },
    "5511888888888": { "nome": "Maria", "link": "https://site.com/m" }
  },
  "options": {
    "spintax": true
  }
}
```

> Com spintax, cada destinatário recebe uma combinação diferente, gerando milhares de variações únicas.

### 6.4 Broadcast com CSV/lista externa

```json
{
  "recipients_url": "https://meusite.com/api/lista-contatos.csv",
  "type": "text",
  "content": {
    "body": "Olá {{nome}}, sua fatura de {{valor}} vence em {{vencimento}}."
  },
  "options": {
    "csv_phone_column": "telefone",
    "csv_delimiter": ";",
    "skip_header": true
  }
}
```

### 6.5 Gerenciamento de broadcast

```
GET    /groups/:groupId/messages/broadcast                     — Listar broadcasts
GET    /groups/:groupId/messages/broadcast/:broadcastId        — Detalhes e progresso
POST   /groups/:groupId/messages/broadcast/:broadcastId/pause  — Pausar
POST   /groups/:groupId/messages/broadcast/:broadcastId/resume — Retomar
POST   /groups/:groupId/messages/broadcast/:broadcastId/cancel — Cancelar
GET    /groups/:groupId/messages/broadcast/:broadcastId/report — Relatório final
```

**Resposta do status do broadcast:**

```json
GET /groups/:groupId/messages/broadcast/bc_abc123
{
  "broadcast_id": "bc_abc123",
  "status": "in_progress",
  "progress": {
    "total": 1000,
    "sent": 450,
    "delivered": 380,
    "read": 120,
    "failed": 12,
    "pending": 538
  },
  "instances_used": [
    { "id": "inst_1", "phone": "+5511999...", "sent": 230 },
    { "id": "inst_2", "phone": "+5511888...", "sent": 220 }
  ],
  "started_at": "2025-01-15T10:00:00Z",
  "estimated_completion": "2025-01-15T11:30:00Z"
}
```

---

## 7. Mensagens — Consulta e Status

```
GET    /groups/:groupId/messages                         — Listar mensagens do grupo (paginado)
GET    /groups/:groupId/messages/:messageId              — Detalhes de uma mensagem
GET    /groups/:groupId/messages/:messageId/status       — Status de entrega
DELETE /groups/:groupId/messages/:messageId              — Apagar mensagem para todos
```

**Filtros disponíveis na listagem:**

```
GET /groups/:groupId/messages?status=delivered&from=2025-01-01&to=2025-01-15&recipient=5511999999999&type=text&instance_id=xxx&limit=50&offset=0
```

---

## 8. Chat Management

```
GET    /instances/:instanceId/chats                      — Listar todas as conversas
GET    /instances/:instanceId/chats/:chatId              — Detalhes da conversa
GET    /instances/:instanceId/chats/:chatId/messages     — Histórico de mensagens do chat
POST   /instances/:instanceId/chats/:chatId/archive      — Arquivar conversa
POST   /instances/:instanceId/chats/:chatId/unarchive    — Desarquivar
POST   /instances/:instanceId/chats/:chatId/pin          — Fixar conversa
POST   /instances/:instanceId/chats/:chatId/unpin        — Desafixar
POST   /instances/:instanceId/chats/:chatId/mute         — Silenciar
POST   /instances/:instanceId/chats/:chatId/unmute       — Tirar silêncio
POST   /instances/:instanceId/chats/:chatId/read         — Marcar como lido
POST   /instances/:instanceId/chats/:chatId/unread       — Marcar como não lido
DELETE /instances/:instanceId/chats/:chatId              — Apagar conversa
POST   /instances/:instanceId/chats/:chatId/typing       — Simular "digitando..."
POST   /instances/:instanceId/chats/:chatId/recording    — Simular "gravando áudio..."
```

**Typing/Recording:**

```json
POST /instances/:instanceId/chats/:chatId/typing
{
  "duration_ms": 3000
}
```

---

## 9. Contatos

```
GET    /instances/:instanceId/contacts                   — Listar todos os contatos
GET    /instances/:instanceId/contacts/:contactId        — Detalhes do contato
GET    /instances/:instanceId/contacts/:contactId/photo  — Foto do contato
POST   /instances/:instanceId/contacts/:contactId/block  — Bloquear contato
POST   /instances/:instanceId/contacts/:contactId/unblock — Desbloquear
GET    /instances/:instanceId/contacts/blocked            — Listar bloqueados
```

### Verificação de números

```json
POST /groups/:groupId/contacts/check
{
  "numbers": [
    "5511999999999",
    "5511888888888",
    "5511777777777"
  ]
}
```

**Resposta:**

```json
{
  "results": [
    { "number": "5511999999999", "exists": true, "jid": "5511999999999@s.whatsapp.net" },
    { "number": "5511888888888", "exists": true, "jid": "5511888888888@s.whatsapp.net" },
    { "number": "5511777777777", "exists": false, "jid": null }
  ],
  "valid_count": 2,
  "invalid_count": 1
}
```

### Busca de perfil

```json
POST /groups/:groupId/contacts/profile
{
  "numbers": ["5511999999999"]
}
```

**Resposta:**

```json
{
  "results": [
    {
      "number": "5511999999999",
      "name": "João Silva",
      "status": "Disponível",
      "photo_url": "https://...",
      "is_business": false
    }
  ]
}
```

---

## 10. Grupos de WhatsApp

### Gerenciamento

```
GET    /instances/:instanceId/wa-groups                          — Listar grupos que participa
POST   /instances/:instanceId/wa-groups                          — Criar grupo
GET    /instances/:instanceId/wa-groups/:waGroupId               — Detalhes do grupo
GET    /instances/:instanceId/wa-groups/:waGroupId/photo         — Foto do grupo
PATCH  /instances/:instanceId/wa-groups/:waGroupId/name          — Alterar nome
PATCH  /instances/:instanceId/wa-groups/:waGroupId/description   — Alterar descrição
PATCH  /instances/:instanceId/wa-groups/:waGroupId/photo         — Alterar foto
PATCH  /instances/:instanceId/wa-groups/:waGroupId/settings      — Alterar configurações
DELETE /instances/:instanceId/wa-groups/:waGroupId/leave         — Sair do grupo
```

**Criar grupo:**

```json
POST /instances/:instanceId/wa-groups
{
  "name": "Promoções 2025",
  "description": "Grupo de ofertas exclusivas",
  "participants": [
    "5511999999999",
    "5511888888888"
  ]
}
```

**Alterar configurações:**

```json
PATCH /instances/:instanceId/wa-groups/:waGroupId/settings
{
  "announce": true,
  "restrict": true,
  "ephemeral_duration": 86400
}
```

> `announce: true` = apenas admins enviam mensagens. `restrict: true` = apenas admins editam dados do grupo. `ephemeral_duration` = mensagens temporárias (em segundos: 86400 = 24h, 604800 = 7 dias).

### Participantes

```
GET    /instances/:instanceId/wa-groups/:waGroupId/participants                  — Listar membros
POST   /instances/:instanceId/wa-groups/:waGroupId/participants/add              — Adicionar membros
POST   /instances/:instanceId/wa-groups/:waGroupId/participants/remove           — Remover membros
POST   /instances/:instanceId/wa-groups/:waGroupId/participants/promote          — Promover a admin
POST   /instances/:instanceId/wa-groups/:waGroupId/participants/demote           — Remover admin
```

**Adicionar membros:**

```json
POST /instances/:instanceId/wa-groups/:waGroupId/participants/add
{
  "participants": ["5511999999999", "5511888888888"]
}
```

### Convite

```
GET    /instances/:instanceId/wa-groups/:waGroupId/invite-code   — Obter link de convite
POST   /instances/:instanceId/wa-groups/:waGroupId/invite-code   — Revogar e gerar novo link
POST   /instances/:instanceId/wa-groups/join                     — Entrar via link de convite
```

### Envio para grupos WhatsApp

```json
POST /groups/:groupId/messages/send
{
  "to": "120363012345678901@g.us",
  "type": "text",
  "content": {
    "body": "Mensagem para o grupo!"
  }
}
```

> Grupos WhatsApp usam JID no formato `XXXXXXXXX@g.us`

### Menção em grupo

```json
{
  "to": "120363012345678901@g.us",
  "type": "text",
  "content": {
    "body": "Olá @5511999999999, tudo bem?",
    "mentions": ["5511999999999"]
  }
}
```

---

## 11. Status / Stories

```
POST   /instances/:instanceId/status/text                — Postar status de texto
POST   /instances/:instanceId/status/image               — Postar status com imagem
POST   /instances/:instanceId/status/video               — Postar status com vídeo
POST   /instances/:instanceId/status/audio               — Postar status com áudio
GET    /instances/:instanceId/status                      — Ver status dos contatos
DELETE /instances/:instanceId/status/:statusId            — Apagar meu status
```

**Postar status de texto:**

```json
POST /instances/:instanceId/status/text
{
  "content": {
    "body": "Promoção relâmpago! 50% OFF só hoje 🔥",
    "background_color": "#FF5733",
    "font": 2
  },
  "privacy": {
    "type": "contacts",
    "whitelist": ["5511999999999", "5511888888888"]
  }
}
```

**Postar status com imagem:**

```json
POST /instances/:instanceId/status/image
{
  "content": {
    "url": "https://exemplo.com/promo.jpg",
    "caption": "Confira nossa nova coleção!"
  },
  "privacy": {
    "type": "all"
  }
}
```

> `privacy.type`: `all` (todos os contatos), `contacts` (contatos selecionados via whitelist), `blacklist` (todos exceto os listados).

---

## 12. Labels / Etiquetas

> Disponível apenas para contas WhatsApp Business.

```
GET    /instances/:instanceId/labels                             — Listar etiquetas
POST   /instances/:instanceId/labels                             — Criar etiqueta
PATCH  /instances/:instanceId/labels/:labelId                    — Editar etiqueta
DELETE /instances/:instanceId/labels/:labelId                    — Deletar etiqueta
POST   /instances/:instanceId/chats/:chatId/labels/:labelId      — Adicionar etiqueta ao chat
DELETE /instances/:instanceId/chats/:chatId/labels/:labelId      — Remover etiqueta do chat
GET    /instances/:instanceId/labels/:labelId/chats              — Listar chats com etiqueta
```

---

## 13. Catálogo / Business

> Disponível apenas para contas WhatsApp Business.

```
GET    /instances/:instanceId/catalog                            — Listar produtos do catálogo
GET    /instances/:instanceId/catalog/:productId                 — Detalhes do produto
POST   /instances/:instanceId/catalog                            — Adicionar produto
PATCH  /instances/:instanceId/catalog/:productId                 — Editar produto
DELETE /instances/:instanceId/catalog/:productId                 — Remover produto
GET    /instances/:instanceId/business-profile                   — Perfil comercial
PATCH  /instances/:instanceId/business-profile                   — Editar perfil comercial
```

---

## 14. Mídia

```
POST   /media/upload                                             — Upload de arquivo
GET    /media/:mediaId                                           — Download de mídia
GET    /media/:mediaId/info                                      — Metadados (tipo, tamanho, hash)
DELETE /media/:mediaId                                           — Deletar mídia do storage
```

**Upload:**

```
POST /media/upload
Content-Type: multipart/form-data

file: (binary)
```

**Resposta:**

```json
{
  "media_id": "media_abc123",
  "url": "https://api.seudominio.com/v1/media/media_abc123",
  "mime_type": "image/jpeg",
  "size_bytes": 245760,
  "sha256": "abc123def456...",
  "expires_at": "2025-01-16T10:00:00Z"
}
```

> Use o `media_id` ou a `url` retornada nos endpoints de envio de mensagem.

---

## 15. Webhooks

### Configuração

```
PUT    /groups/:groupId/webhook                          — Configurar webhook do grupo
GET    /groups/:groupId/webhook                          — Ver configuração
DELETE /groups/:groupId/webhook                          — Remover webhook
POST   /groups/:groupId/webhook/test                     — Enviar evento de teste
GET    /groups/:groupId/webhook/logs                     — Logs de entrega (últimas 24h)
POST   /groups/:groupId/webhook/retry/:logId             — Reenviar evento falhado
```

**Configuração:**

```json
PUT /groups/:groupId/webhook
{
  "url": "https://meusite.com/webhook/whatsapp",
  "secret": "meu_secret_para_hmac",
  "events": [
    "message.received",
    "message.sent",
    "message.delivered",
    "message.read",
    "message.failed",
    "message.reaction",
    "instance.connected",
    "instance.disconnected",
    "instance.banned",
    "instance.status_changed",
    "group.participant_added",
    "group.participant_removed",
    "broadcast.completed",
    "broadcast.failed",
    "contact.updated",
    "presence.update",
    "call.received"
  ],
  "retry_policy": {
    "max_retries": 5,
    "retry_intervals_s": [5, 30, 120, 600, 3600]
  },
  "headers": {
    "X-Custom-Header": "valor-customizado"
  }
}
```

### Eventos de webhook — payloads

**Mensagem recebida:**

```json
{
  "event": "message.received",
  "group_id": "grp_xxx",
  "instance_id": "inst_xxx",
  "phone_number": "+5511999999999",
  "timestamp": "2025-01-15T10:05:00Z",
  "data": {
    "message_id": "MSG_ABC123",
    "from": "5511888888888",
    "from_name": "João Silva",
    "chat_id": "5511888888888@s.whatsapp.net",
    "is_group": false,
    "type": "text",
    "content": {
      "body": "Olá, preciso de ajuda!"
    },
    "timestamp": 1705312900,
    "is_forwarded": false,
    "is_frequently_forwarded": false
  }
}
```

**Mensagem recebida em grupo:**

```json
{
  "event": "message.received",
  "group_id": "grp_xxx",
  "instance_id": "inst_xxx",
  "timestamp": "2025-01-15T10:05:00Z",
  "data": {
    "message_id": "MSG_DEF456",
    "from": "5511888888888",
    "from_name": "João Silva",
    "chat_id": "120363012345678901@g.us",
    "chat_name": "Grupo Promoções",
    "is_group": true,
    "type": "image",
    "content": {
      "url": "https://mmg.whatsapp.net/...",
      "caption": "Olha isso!",
      "mime_type": "image/jpeg"
    }
  }
}
```

**Resposta a botão:**

```json
{
  "event": "message.received",
  "data": {
    "type": "button_response",
    "content": {
      "selected_button_id": "btn_sim",
      "selected_button_text": "✅ Sim, quero!",
      "original_message_id": "MSG_ORIGINAL"
    }
  }
}
```

**Resposta a lista:**

```json
{
  "event": "message.received",
  "data": {
    "type": "list_response",
    "content": {
      "selected_item_id": "pizza_marg",
      "selected_item_title": "Margherita",
      "selected_item_description": "Molho, mussarela e manjericão — R$ 39,90",
      "original_message_id": "MSG_ORIGINAL"
    }
  }
}
```

**Resposta a enquete:**

```json
{
  "event": "message.received",
  "data": {
    "type": "poll_response",
    "content": {
      "poll_message_id": "MSG_POLL",
      "selected_options": ["Manhã (8h-12h)"],
      "voter": "5511888888888"
    }
  }
}
```

**Chamada recebida:**

```json
{
  "event": "call.received",
  "data": {
    "call_id": "CALL_123",
    "from": "5511888888888",
    "type": "voice",
    "status": "ringing",
    "timestamp": 1705312900
  }
}
```

**Instância banida:**

```json
{
  "event": "instance.banned",
  "group_id": "grp_xxx",
  "instance_id": "inst_xxx",
  "timestamp": "2025-01-15T10:05:00Z",
  "data": {
    "phone_number": "+5511999999999",
    "reason": "connection_lost_permanent",
    "messages_redistributed": 45,
    "new_active_instance": "inst_yyy"
  }
}
```

**Verificação de assinatura do webhook (exemplo do lado do cliente):**

```
Header: X-Webhook-Signature: sha256=a1b2c3d4e5...

// Verificação:
expected = HMAC-SHA256(secret, raw_body)
valid = constant_time_compare(expected, received_signature)
```

---

## 16. Blacklist / Opt-out

```
GET    /groups/:groupId/blacklist                        — Listar números bloqueados
POST   /groups/:groupId/blacklist                        — Adicionar números à blacklist
DELETE /groups/:groupId/blacklist/:number                 — Remover da blacklist
POST   /groups/:groupId/blacklist/import                 — Importar lista (CSV)
GET    /groups/:groupId/blacklist/export                  — Exportar lista (CSV)
```

**Adicionar à blacklist:**

```json
POST /groups/:groupId/blacklist
{
  "numbers": ["5511999999999", "5511888888888"],
  "reason": "opt_out"
}
```

> Números na blacklist são automaticamente excluídos de qualquer broadcast. Compliance com LGPD/GDPR.

---

## 17. Métricas e Relatórios

```
GET    /groups/:groupId/metrics                          — Métricas gerais do grupo
GET    /groups/:groupId/metrics/daily                    — Métricas dia a dia
GET    /groups/:groupId/metrics/instances                — Métricas por instância
GET    /groups/:groupId/metrics/delivery-rate            — Taxa de entrega ao longo do tempo
GET    /groups/:groupId/metrics/broadcast/:broadcastId   — Relatório de um broadcast
GET    /account/metrics                                  — Métricas globais da conta
```

**Resposta de métricas diárias:**

```json
GET /groups/:groupId/metrics/daily?from=2025-01-01&to=2025-01-15
{
  "period": { "from": "2025-01-01", "to": "2025-01-15" },
  "daily": [
    {
      "date": "2025-01-01",
      "sent": 1200,
      "delivered": 1150,
      "read": 800,
      "failed": 12,
      "delivery_rate": 0.958,
      "read_rate": 0.695,
      "instances_used": 3,
      "instances_banned": 0
    }
  ],
  "totals": {
    "sent": 15000,
    "delivered": 14200,
    "read": 9800,
    "failed": 150,
    "avg_delivery_rate": 0.946,
    "avg_read_rate": 0.690
  }
}
```

---

## 18. Agendamento

```
POST   /groups/:groupId/schedules                        — Criar envio agendado
GET    /groups/:groupId/schedules                        — Listar agendamentos
GET    /groups/:groupId/schedules/:scheduleId            — Detalhes do agendamento
PATCH  /groups/:groupId/schedules/:scheduleId            — Editar agendamento
DELETE /groups/:groupId/schedules/:scheduleId            — Cancelar agendamento
```

**Agendamento único:**

```json
POST /groups/:groupId/schedules
{
  "type": "broadcast",
  "schedule_at": "2025-01-20T10:00:00-03:00",
  "payload": {
    "recipients": ["5511999999999"],
    "type": "text",
    "content": {
      "body": "Lembrete: sua consulta é amanhã às 14h!"
    }
  }
}
```

**Agendamento recorrente:**

```json
{
  "type": "broadcast",
  "schedule_at": "2025-01-20T10:00:00-03:00",
  "recurrence": {
    "type": "weekly",
    "days": ["mon", "wed", "fri"],
    "ends_at": "2025-06-30T23:59:59-03:00"
  },
  "payload": {
    "recipients_url": "https://meusite.com/api/clientes-ativos.csv",
    "type": "text",
    "content": {
      "body": "{{nome}}, confira nossas novidades da semana!"
    }
  }
}
```

---

## 19. Logs e Auditoria

```
GET    /account/audit-log                                — Log de ações administrativas
GET    /groups/:groupId/logs                             — Logs de atividade do grupo
GET    /instances/:instanceId/logs                       — Logs de atividade da instância
```

**Filtros:**

```
GET /account/audit-log?action=instance.created&from=2025-01-01&actor=key_xxx&limit=100
```

**Ações registradas:**

- `tenant.key_created`, `tenant.key_revoked`
- `group.created`, `group.deleted`, `group.strategy_changed`
- `instance.created`, `instance.connected`, `instance.disconnected`, `instance.banned`
- `broadcast.created`, `broadcast.cancelled`
- `blacklist.updated`
- `webhook.configured`

---

## 20. Health e Sistema

```
GET    /health                                           — Health check da API
GET    /health/detailed                                  — Status de cada dependência
GET    /system/info                                      — Versão, uptime, estatísticas
```

**Health detalhado:**

```json
GET /health/detailed
{
  "status": "healthy",
  "uptime_seconds": 864000,
  "version": "1.4.2",
  "dependencies": {
    "postgresql": { "status": "up", "latency_ms": 2 },
    "redis": { "status": "up", "latency_ms": 1 },
    "s3": { "status": "up", "latency_ms": 15 }
  },
  "instances": {
    "total": 25,
    "connected": 22,
    "resting": 2,
    "banned": 1
  },
  "queues": {
    "pending_messages": 340,
    "active_broadcasts": 2
  }
}
```

---

## Códigos de Erro

| Código | Nome | Descrição |
|---|---|---|
| 400 | `bad_request` | Payload inválido ou campo obrigatório faltando |
| 401 | `unauthorized` | API key ausente ou inválida |
| 403 | `forbidden` | API key sem permissão para este recurso |
| 404 | `not_found` | Recurso não encontrado |
| 409 | `conflict` | Instância já existe no grupo / número já conectado |
| 422 | `unprocessable` | Número inválido, mídia inválida, etc |
| 429 | `rate_limited` | Limite de requests excedido |
| 500 | `internal_error` | Erro interno do servidor |
| 502 | `whatsapp_error` | Erro na comunicação com servidores do WhatsApp |
| 503 | `no_instance_available` | Nenhuma instância disponível no grupo para envio |

**Formato de erro:**

```json
{
  "error": {
    "code": "no_instance_available",
    "message": "Nenhuma instância disponível no grupo. Todas estão em descanso ou banidas.",
    "details": {
      "group_id": "grp_xxx",
      "instances": {
        "resting": 2,
        "banned": 1,
        "available": 0
      }
    }
  }
}
```

---

## Paginação

Todos os endpoints de listagem usam cursor-based pagination:

```
GET /groups/:groupId/messages?limit=50&cursor=eyJpZCI6ImFiYzEyMyJ9

{
  "data": [...],
  "pagination": {
    "limit": 50,
    "has_more": true,
    "next_cursor": "eyJpZCI6ImRlZjQ1NiJ9",
    "total_count": 1500
  }
}
```

---

## Rate Limits

| Recurso | Limite | Janela |
|---|---|---|
| Requests gerais | 1000/min | Por API key |
| Envio de mensagens | 100/min | Por grupo |
| Broadcast | 10/min | Por grupo |
| Upload de mídia | 50/min | Por tenant |
| Verificação de números | 500/min | Por grupo |
| Webhook logs | 60/min | Por grupo |

Headers de rate limit em toda resposta:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1705313000
```
