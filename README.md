# scripts-trabalho

Scripts utilitários para o dia a dia de trabalho nos projetos da Macro.

## Estrutura

```
prds/
  NNN-<nome-do-script>.md   # PRD de desenvolvimento de cada script
scripts/
  <nome-do-script>/
    README.md      # o que faz, linguagem, dependências e como executar
    <arquivos do script>
```

Os scripts podem ser em linguagens diferentes, por isso cada um fica isolado em sua própria pasta dentro de [scripts/](scripts/), com seus arquivos e dependências. Os requisitos de cada script ficam em [prds/](prds/).

## Scripts

| Script | Linguagem | O que faz |
| --- | --- | --- |
| [configurar-abandono-projetos](scripts/configurar-abandono-projetos/) | Go | Grava a configuração de abandono (`/recovery`) do ApiManager nos projetos listados em JSON que existem no ApiManager. |
| [sincronizar-nomes-projetos](scripts/sincronizar-nomes-projetos/) | Go | Grava no `ProjectName` das configurações do ApiManager o nome do projeto no Zoho CRM (com sufixo, quando houver). |

## Uso

Consulte o `README.md` da pasta do script desejado.

## Agentes

As instruções para agentes de IA ficam em [AGENTS.md](AGENTS.md). O [CLAUDE.md](CLAUDE.md) apenas importa esse arquivo.
