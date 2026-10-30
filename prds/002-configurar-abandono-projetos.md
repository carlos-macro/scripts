# PRD 002 — Configurar abandono dos projetos no ApiManager

**Status:** revisão 2 implementada (novo formato da listagem) · **Script:** [scripts/configurar-abandono-projetos](../scripts/configurar-abandono-projetos/)

## Problema

A recuperação de abandono de cada projeto é configurada no ApiManager (`ProjectSetting`, rota `/recovery`). As configurações dos projetos estão em [recovery-settings-projetos.json](../scripts/configurar-abandono-projetos/recovery-settings-projetos.json), com 37 entradas (36 projetos e o projeto de teste), mas ainda precisam ser gravadas no ApiManager, uma a uma.

## Objetivo

Um script em Go, no mesmo padrão do [sincronizar-nomes-projetos](../scripts/sincronizar-nomes-projetos/), que lê a lista de projetos do arquivo JSON, confere quais têm configuração no ApiManager e grava a configuração de abandono só nesses.

## Fontes

### Entrada: arquivo JSON

Padrão: `recovery-settings-projetos.json`, na pasta do script (versionado junto com ele).

```json
{
  "projetos": [
    {
      "projectId": "4918426000326286577",
      "body": { "enable": true, "minutes": 10, "ignoreChatbot": false, "chatbotMinutes": 20, "maxHours": 24 }
    }
  ]
}
```

- `projectId`: id completo da configuração no ApiManager, com sufixo quando houver (`4918426000427538811-IA`).
- `body`: enviado ao `PUT` como está. `enable`, `minutes`, `ignoreChatbot` e `maxHours` são obrigatórios; `chatbotMinutes` é obrigatório no arquivo, mas pode ser `null`.
- Só `projectId` e `body` são lidos; qualquer outro campo da entrada é ignorado.

### Leitura: ApiManager

A mesma listagem do PRD 001, para saber quais configurações existem e se já têm abandono configurado:

```text
GET <APIMANAGER_URL>/projectsetting?page=1&pageSize=200
access-token: <APIMANAGER_ACCESS_TOKEN>
```

```json
{
  "items": [
    {
      "projectId": "4918426000047567003",
      "projectName": "Vivo",
      "modules": ["TelecomModule", "TelecomPipeline"],
      "updateByTimeEnabled": false,
      "intergrallEnabled": false,
      "recoveryEnabled": false,
      "recoverySettings": null
    },
    {
      "projectId": "4918426002284882870",
      "projectName": "Tim Ultrafibra Ativo",
      "modules": ["TelecomModule", "CrossSellModule"],
      "updateByTimeEnabled": false,
      "intergrallEnabled": false,
      "recoveryEnabled": true,
      "recoverySettings": { "enabled": true, "minutes": 1, "ignoreChatbot": false, "chatbotMinutes": 1, "maxHours": 4 }
    }
  ],
  "total": 86,
  "page": 1,
  "pageSize": 200
}
```

- O script pede páginas de 200 até ter lido `total` itens ou receber uma página vazia.
- De cada item usa só `projectId`, `recoveryEnabled` e `recoverySettings`; os demais campos são ignorados. Sem abandono configurado, vêm `false` e `null`.
- `recoverySettings`, quando preenchido, traz `enabled`, `minutes`, `ignoreChatbot`, `chatbotMinutes` e `maxHours`.
- Se algum item vier sem `recoveryEnabled` ou sem a chave `recoverySettings` (ela pode ser `null`, mas precisa existir), o script falha antes de qualquer `PUT`: sem esses campos não dá para saber o que seria sobrescrito.

### Escrita: ApiManager

```text
PUT <APIMANAGER_URL>/projectsetting/{projectId}/recovery
access-token: <APIMANAGER_ACCESS_TOKEN>
Content-Type: application/json

{ "enable": true, "minutes": 1, "ignoreChatbot": false, "chatbotMinutes": 1, "maxHours": 4 }
```

`{projectId}` é o id da entrada do arquivo, exatamente como está (com sufixo, quando houver).

### Ambientes

Iguais aos do PRD 001, com as mesmas variáveis do `.env` da raiz:

| Ambiente | Variável | Valor |
| --- | --- | --- |
| teste (local) | `APIMANAGER_URL_TESTE` | `https://localhost:7119` |
| prod | `APIMANAGER_URL_PROD` | `https://apimanager.macrolabs.app` |

- `-ambiente teste|prod` escolhe a URL, que vale para a listagem e para o `PUT`. O padrão é `teste`; escrever em produção exige `-ambiente prod` explícito.
- Se a variável do ambiente escolhido não estiver definida, o script falha antes de qualquer chamada.
- O script exibe no início o ambiente, a URL e o arquivo que vai usar.
- O ambiente local usa o certificado de desenvolvimento do .NET (`dotnet dev-certs https --trust`). O script não desativa a verificação de TLS.

## Requisitos

1. Ler e validar o arquivo inteiro antes de qualquer chamada. Falhar com mensagem clara (indicando a entrada) se o JSON for inválido, se faltar `projectId` ou algum campo do `body`, ou se houver `projectId` repetido.
2. Ler a lista de configurações do ApiManager (todas as páginas). Falhar com mensagem clara se a chamada ou o formato for inválido.
3. Para cada entrada do arquivo, na ordem do arquivo:
   - `projectId` sem configuração no ApiManager (comparação exata, com sufixo): **sem configuração**, sem chamada, mesmo com `-forcar`;
   - abandono já configurado (`recoveryEnabled` `true` ou `recoverySettings` diferente de `null`) e sem `-forcar`: **mantida**, sem chamada;
   - caso contrário: chamar o `PUT` com o `body` da entrada.
4. Respostas fora de `2xx` no `PUT` são registradas como falha (com status e corpo) e o script segue para a próxima.
5. Ao final, exibir um resumo (atualizadas, mantidas, sem configuração, falhas). Sair com código diferente de zero se houver falha.
6. Opção `-id` processa só a entrada com aquele `projectId` exato. Se o id não estiver no arquivo, o script falha antes de chamar o `PUT`.
7. Opção `-dry-run` mostra, para cada entrada, a decisão e o `body` que seria enviado, sem chamar o `PUT`.
8. Opção `-ambiente teste|prod` (padrão `teste`); ver "Ambientes".
9. Opção `-arquivo <caminho>` para usar outro arquivo. Padrão: `recovery-settings-projetos.json` (relativo à pasta de execução, que é a pasta do script no `go run .`).
10. Opção `-forcar` grava o `body` também nas configurações que já têm abandono configurado, substituindo a configuração atual. Nas linhas de configurações que já tinham abandono (mantidas ou sobrescritas), o script mostra o valor atual ao lado do `body`: `recoverySettings` resumido (`enabled`, `minutes`, `ignoreChatbot`, `chatbotMinutes`, `maxHours`) ou, se vier `null` com `recoveryEnabled` `true`, só `recoveryEnabled=true`.

## Projeto de teste

O projeto `4918426001337478951` ("Teste") é usado para validar o script antes da execução completa, como no PRD 001. Ele é a primeira entrada do arquivo:

```json
{
  "projectId": "4918426001337478951",
  "body": { "enable": true, "minutes": 3, "ignoreChatbot": false, "chatbotMinutes": null, "maxHours": 24 }
}
```

- Os testes manuais começam sempre com `-id 4918426001337478951` (primeiro com `-dry-run`), em `teste` e depois em `prod`.
- Como a entrada fica no arquivo, a execução completa também grava no projeto de teste.

## Saída

Uma linha por entrada, com o `projectId` e o `body` resumido (`enable=true minutes=20 ignoreChatbot=false chatbotMinutes=- maxHours=24`):

- `[ok]`: configuração gravada (com `-dry-run`, `[atualizaria]`);
- `[mantida]`: já tinha abandono configurado e `-forcar` não foi usado (mostra o valor atual);
- `[sem configuração]`: o `projectId` não existe no ApiManager do ambiente escolhido;
- `[falha]`: erro na gravação, com status e corpo.

## Implementação

- Go, só biblioteca padrão, módulo próprio na pasta do script.
- O carregamento do `.env` (`env.go`) é copiado do `sincronizar-nomes-projetos`, porque cada script é isolado na sua pasta. O comportamento é o mesmo: variáveis do ambiente têm prioridade e as ausentes são lidas do `.env` na raiz do repositório.
- Testes automatizados para a validação do arquivo e para a decisão de cada entrada (sem configuração, mantida, atualizada, com e sem `-forcar`).

## Segurança

- O token vem de `APIMANAGER_ACCESS_TOKEN` (o mesmo do PRD 001, já no `.env`); nunca fica no código nem no repositório. O token do exemplo do `curl` não vai para nenhum arquivo versionado.
- Não há variável nova: o `.env.example` não muda.
- Com `-ambiente prod`, o script escreve em produção. A execução completa em produção (sem `-id`) só acontece com autorização explícita.

## Fora do escopo

- Criar configuração no ApiManager para projetos que não têm.
- Desfazer a gravação: o script não guarda backup da configuração anterior (o valor atual só aparece na saída).
- Consultar o Zoho CRM: o arquivo já traz os `projectId` completos.
