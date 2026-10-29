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
| [sincronizar-nomes-projetos](scripts/sincronizar-nomes-projetos/) | Go | Copia os nomes dos projetos do Zoho CRM para o `ProjectName` do ApiManager. |

## Uso

Consulte o `README.md` da pasta do script desejado.

## Agentes

As instruções para agentes de IA ficam em [AGENTS.md](AGENTS.md). O [CLAUDE.md](CLAUDE.md) apenas importa esse arquivo.
