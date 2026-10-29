# PRD 001 — Sincronizar nomes dos projetos no ApiManager

**Status:** revisão 2 implementada · **Script:** [scripts/sincronizar-nomes-projetos](../scripts/sincronizar-nomes-projetos/)

## Problema

Os nomes dos projetos são mantidos no Zoho CRM, no módulo `Projetos`. A configuração de cada projeto no ApiManager (`ProjectSetting`) tem um campo `ProjectName` que não acompanha o CRM e fica vazio ou desatualizado.

Além disso, o ApiManager tem configurações cujo `projectId` é o id do CRM seguido de um sufixo (`4918426002476447776-VOZ`, `4918426000054386272-IA-DEV`). A revisão 1 só atualiza ids exatos, então essas ficam sem nome.

## Objetivo

Um script que lista as configurações do ApiManager, encontra o projeto correspondente no CRM e grava o nome no `ProjectName`, incluindo as configurações com sufixo.

## Mudanças da revisão 2

| | Revisão 1 (atual) | Revisão 2 |
| --- | --- | --- |
| O que percorre | Projetos do CRM | Configurações do ApiManager |
| Ids com sufixo | Ignorados | Nome do CRM + sufixo entre parênteses |
| Configurações que já têm nome | Sobrescritas | Mantidas; sobrescritas só com `-forcar` |
| Projeto do CRM sem configuração | `PUT` com 404, contado como "sem configuração" | Não gera chamada (não está na lista do ApiManager) |
| Configuração sem projeto no CRM | Não detectada | Contada como "sem projeto no CRM" |

## Fontes

### Leitura: Zoho CRM

Função `standalone.listarProjetos` (código em `zoho-functions/CRM/Geral/Listar Projetos.deluge`).

```text
GET https://www.zohoapis.com/crm/v7/functions/listarprojetos/actions/execute?auth_type=apikey&zapikey=<ZOHO_ZAPIKEY>
```

O Zoho devolve o retorno da função como texto em `details.output`:

```json
{
  "code": "success",
  "details": { "output": "{\"data\":[{\"id\":\"4918426001337478951\",\"name\":\"Teste\"}],\"more_records\":false}" }
}
```

- `data[].id`: id do registro no CRM.
- `more_records`: `true` indica que a função atingiu o limite de 2.000 registros e há projetos que não vieram.

### Leitura: ApiManager (novo)

```text
GET <APIMANAGER_URL>/projectsetting?page=1&pageSize=200
access-token: <APIMANAGER_ACCESS_TOKEN>
```

```json
{
  "items": [
    { "projectId": "4918426000054386272-IA", "projectName": null, "modules": [] }
  ],
  "total": 86,
  "page": 1,
  "pageSize": 200
}
```

O script pede páginas de 200 (o máximo do ApiManager) até ter lido `total` itens ou receber uma página vazia.

### Escrita: ApiManager

```text
PUT <APIMANAGER_URL>/projectsetting/{projectId}/name
access-token: <APIMANAGER_ACCESS_TOKEN>
Content-Type: application/json

{ "projectName": "Nome do Projeto" }
```

`{projectId}` é o id completo da configuração, com sufixo quando houver.

#### Ambientes

O ApiManager tem dois ambientes, cada um com sua URL base:

| Ambiente | Variável | Valor |
| --- | --- | --- |
| teste | `APIMANAGER_URL_TESTE` | `https://localhost:7119` (ApiManager rodando localmente) |
| prod | `APIMANAGER_URL_PROD` | `https://apimanager.macrolabs.app` |

- As URLs ficam no `.env` (modelo no `.env.example`), não no código. O `access-token` (das rotas administrativas) é o mesmo nos dois ambientes.
- A opção `-ambiente teste|prod` escolhe qual URL usar, e ela vale para as duas rotas do ApiManager (listagem e `PUT`): o script lê e grava no mesmo ambiente. O padrão é `teste`, para que escrever em produção exija pedir `-ambiente prod` explicitamente.
- Se a variável do ambiente escolhido não estiver definida, o script falha antes de qualquer chamada.
- O script exibe no início o ambiente e a URL que vai usar.
- O ambiente de teste usa o certificado de desenvolvimento do .NET, que precisa estar confiável na máquina (`dotnet dev-certs https --trust`). O script não desativa a verificação de TLS.

## Regra do nome

O `projectId` da configuração é dividido no primeiro `-`:

- **id base**: a parte antes do primeiro `-`, que é o id do projeto no CRM;
- **sufixo**: o restante, que pode conter outros `-`.

| `projectId` | Nome no CRM | `projectName` gravado |
| --- | --- | --- |
| `4918426000052796600` | Oi | `Oi` |
| `4918426000052796600-DEV` | Oi | `Oi (DEV)` |
| `4918426000054386272-IA` | Vivo SVA | `Vivo SVA (IA)` |
| `4918426000054386272-IA-DEV` | Vivo SVA | `Vivo SVA (IA-DEV)` |

Só entram na regra ids cujo id base é numérico. Os demais (`abc`, `NEOENERGIA`, `PRJ-REFIDEL`, `Macro teste`…) são contados como "sem projeto no CRM".

## Requisitos

1. Ler a lista de projetos do CRM e a lista de configurações do ApiManager (todas as páginas). Falhar com mensagem clara se alguma chamada ou formato for inválido.
2. Para cada configuração, aplicar a "Regra do nome" e decidir:
   - já tem `projectName` preenchido e sem `-forcar`: **mantida**, sem chamada;
   - id base sem projeto no CRM: **sem projeto no CRM**, sem chamada, mesmo com `-forcar`;
   - caso contrário: chamar o `PUT` com o nome calculado.
3. Respostas fora de `2xx` no `PUT` são registradas como falha (com status e corpo) e o script segue para a próxima.
4. Ao final, exibir um resumo (atualizadas, mantidas, sem projeto no CRM, falhas). Sair com código diferente de zero se houver falha.
5. Avisar se `more_records` do CRM vier `true`.
6. Opção `-id` filtra pelo **id base**: processa a configuração exata e todas as com sufixo daquele projeto (teste com `4918426001337478951`, "Teste").
7. Opção `-dry-run` mostra, para cada configuração, a decisão e o nome que seria gravado, sem chamar o `PUT`.
8. Opção `-ambiente teste|prod` (padrão `teste`) para escolher a URL do ApiManager; ver "Ambientes".
9. Opção `-forcar` grava o nome calculado também nas configurações que já têm `projectName`, substituindo o nome atual. Exemplo: `4918426000105088330-VOZ` hoje se chama "Vivo Controle IA VOZ"; o projeto `4918426000105088330` no CRM se chama "Vivo Móvel", então com `-forcar` o nome passa a ser "Vivo Móvel (VOZ)". Sem `-forcar`, continua "Vivo Controle IA VOZ".

## Estado atual (produção, 29/10/2026)

- 86 configurações no ApiManager; 84 sem `projectName`.
- 49 com id exato no CRM e 25 com sufixo cujo id base está no CRM; descontando as 2 que já têm nome, **72 seriam atualizadas**.
- 12 sem projeto no CRM: `4918426000063655977`, `4918426000063655977-P`, `4918426002476447776`, `4918426002476447776-VOZ`, `6878169000000603013` e os ids não numéricos `abc`, `abcd`, `Macro teste`, `NEOENERGIA`, `PRJ-REFIDEL`, `teste123`, `TESTES`.
- 2 já têm nome: `4918426001337478951` ("Teste") e `4918426000105088330-VOZ` ("Vivo Controle IA VOZ").

## Configurações sem projeto no CRM

Configurações cujo id base não é encontrado no CRM **nunca têm o nome alterado**, nem com `-forcar`. Elas só aparecem na saída como "sem projeto no CRM". Exemplo: `4918426002476447776` e `4918426002476447776-VOZ` ficam como estão, porque o id `4918426002476447776` não está na lista do CRM.

## Segurança

- A chave do Zoho e o token do ApiManager vêm das variáveis de ambiente `ZOHO_ZAPIKEY` e `APIMANAGER_ACCESS_TOKEN`; nunca ficam no código nem no repositório.
- Variáveis não definidas no ambiente são lidas do `.env` na raiz do repositório (a primeira pasta acima do diretório atual que contém `.git`), no formato `CHAVE=valor`, com linhas vazias e `#` ignoradas. Variáveis já definidas no ambiente têm prioridade. O `.env` não é versionado; o modelo é o `.env.example`.
- Com `-ambiente prod`, o script escreve em produção. A execução completa em produção (sem `-id`) só acontece com autorização explícita.

## Fora do escopo

- Criar configuração no ApiManager para projetos que não têm.
- Remover ou renomear projetos no CRM.
- Paginação além do limite de 2.000 registros da função do CRM.
