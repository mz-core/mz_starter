# Estrutura do ambiente

## Objetivo deste documento
Este documento define a fronteira entre `mz_starter`, `mz_core` e `cfx-server-data`.

Ele existe para evitar duas ambiguidades:
- tratar o `mz_starter` como se fosse o framework
- assumir que o `mz_starter` sozinho entrega toda a base operacional do FiveM

## Visão geral
O ambiente atual tem três camadas com responsabilidades diferentes:
- `cfx-server-data`: base stock do servidor FiveM
- `mz_starter`: boot, configuração e instalação do projeto
- `mz_core`: lógica do framework

## O que pertence ao `mz_starter`
Este repositório é responsável por:
- definir o boot via `server.cfg`
- separar a configuração em arquivos dentro de `cfg/`
- documentar o setup operacional
- instalar dependências do projeto, como `oxmysql`, `ox_lib` e `mz_core`

O `mz_starter` deve continuar enxuto e focado em ambiente.

## O que pertence ao `mz_core`
O `mz_core` é o recurso do framework.

É nele que devem ficar:
- lógica de player e sessão
- orgs, grades e permissões de domínio
- contas, inventário e veículos
- regras internas e evolução funcional do framework

Se uma mudança for de regra de negócio ou de comportamento do framework, ela pertence ao `mz_core`, não ao `mz_starter`.

## O que pertence ao `cfx-server-data`
O `cfx-server-data` representa a base stock do servidor FiveM.

No contexto atual deste projeto, ele continua sendo a origem esperada dos recursos stock usados no boot, como:
- `mapmanager`
- `spawnmanager`

O `mz_starter` não distribui esses recursos e o instalador atual também não os baixa.

## Estrutura esperada do ambiente
Hoje, a raiz deste repositório funciona como o `server-data` do projeto.

A estrutura mínima esperada é esta:

```text
server-data/
|-- server.cfg
|-- cfg/
|   |-- base.cfg
|   |-- database.cfg
|   |-- endpoints.cfg
|   |-- onesync.cfg
|   |-- permissions.cfg
|   |-- resources.cfg
|   `-- tags.cfg
`-- resources/
    |-- [ox]/
    |   |-- oxmysql
    |   `-- ox_lib
    |-- [mz]/
    |   `-- mz_core
    `-- [system]/
        |-- mapmanager
        `-- spawnmanager
```

O nome da pasta que contém os recursos stock pode variar no ambiente.
O ponto importante é este: os recursos stock exigidos no boot precisam existir fora do escopo do `mz_starter`.

## Fronteira prática entre os repositórios
Use esta regra simples:
- se a mudança é de boot, install, configuração ou documentação operacional, ela tende a pertencer ao `mz_starter`
- se a mudança é de lógica do framework, ela tende a pertencer ao `mz_core`
- se a mudança é sobre recursos stock do FiveM, ela não pertence a nenhum dos dois repositórios

## O que não deve crescer dentro do `mz_starter`
Para manter a fronteira limpa, este repositório não deve absorver:
- lógica de domínio já existente no `mz_core`
- regras de gameplay do framework
- duplicação de permissões, sessão, inventário, contas ou veículos
- código que exista apenas para compensar falta de definição entre starter e core

## Checklist de consistência
Antes de mexer no projeto, vale validar:
- isto é boot/configuração do ambiente ou é lógica do framework?
- esta dependência é do projeto ou é da base stock do FiveM?
- esta mudança deixaria o `mz_starter` mais perto de um framework?

Se a resposta para a última pergunta for "sim", o mais provável é que a mudança esteja indo para o repositório errado.
