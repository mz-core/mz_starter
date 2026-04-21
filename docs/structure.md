# Estrutura do ambiente

## Objetivo deste documento
Este documento define a fronteira entre `mz_starter`, `mz_core` e os recursos oficiais usados no boot.

Ele existe para evitar duas ambiguidades:
- tratar o `mz_starter` como se fosse o framework
- assumir que o `mz_starter` sozinho entrega toda a base operacional do FiveM

## Visao geral
O ambiente atual tem tres camadas com responsabilidades diferentes:
- recursos oficiais do FiveM usados no boot
- `mz_starter`: boot, configuracao e instalacao do projeto
- `mz_core`: logica do framework

## O que pertence ao `mz_starter`
Este repositorio e responsavel por:
- definir o boot via `server.cfg`
- separar a configuracao em arquivos dentro de `cfg/`
- documentar o setup operacional
- instalar dependencias do projeto, como `oxmysql`, `ox_lib`, `mz_core`, `mz_notify`, `mz_sync` e `pma-voice`
- sincronizar os recursos oficiais minimos para o boot atual

O `mz_starter` deve continuar enxuto e focado em ambiente.

## O que pertence ao `mz_core`
O `mz_core` e o recurso do framework.

E nele que devem ficar:
- logica de player e sessao
- orgs, grades e permissoes de dominio
- contas, inventario e veiculos
- regras internas e evolucao funcional do framework

Se uma mudanca for de regra de negocio ou de comportamento do framework, ela pertence ao `mz_core`, nao ao `mz_starter`.

## O que continua fora do escopo
O instalador sincroniza apenas os recursos oficiais necessarios para o boot atual:
- `mapmanager`
- `spawnmanager`
- `sessionmanager`

Isso nao transforma o `mz_starter` em uma copia completa do `cfx-server-data`.
Se voce precisar de outros recursos stock do FiveM, eles continuam sendo uma decisao separada do setup.

## Estrutura esperada do ambiente
Hoje, a raiz deste repositorio funciona como o `server-data` do projeto.

A estrutura minima esperada e esta:

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
|-- resources/
|   |-- [managers]/
|   |   |-- mapmanager
|   |   `-- spawnmanager
|   |-- [mz]/
|   |   |-- mz_core
|   |   |-- mz_notify
|   |   `-- mz_sync
|   |-- [ox]/
|   |   |-- oxmysql
|   |   `-- ox_lib
|   |-- [som]/
|   |   `-- pma-voice
|   `-- [system]/
|       `-- sessionmanager
`-- tmp/
```

## Fronteira pratica entre os repositorios
Use esta regra simples:
- se a mudanca e de boot, install, configuracao ou documentacao operacional, ela tende a pertencer ao `mz_starter`
- se a mudanca e de logica do framework, ela tende a pertencer ao `mz_core`
- se a mudanca exige mais recursos stock alem dos que ja sao sincronizados aqui, trate isso como uma decisao de infraestrutura, nao de framework

## O que nao deve crescer dentro do `mz_starter`
Para manter a fronteira limpa, este repositorio nao deve absorver:
- logica de dominio ja existente no `mz_core`
- regras de gameplay do framework
- duplicacao de permissoes, sessao, inventario, contas ou veiculos
- codigo que exista apenas para compensar falta de definicao entre starter e core

## Checklist de consistencia
Antes de mexer no projeto, vale validar:
- isto e boot/configuracao do ambiente ou e logica do framework?
- esta dependencia e do projeto ou e uma necessidade extra da infraestrutura?
- esta mudanca deixaria o `mz_starter` mais perto de um framework?

Se a resposta para a ultima pergunta for "sim", o mais provavel e que a mudanca esteja indo para o repositorio errado.
