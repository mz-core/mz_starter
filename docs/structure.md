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
- instalar dependencias do projeto, como `oxmysql`, `ox_lib`, `pma-voice` e os resources `mz_` usados no boot atual
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
- `baseevents`

`chat`, `yarn` e `webpack` pertencem ao `citizen/system_resources` dos artifacts atuais. O perfil FiveM/OneSync nao usa o `sessionmanager` legado.

Isso nao transforma o `mz_starter` em uma copia completa do `cfx-server-data`.
Se voce precisar de outros recursos stock do FiveM, eles continuam sendo uma decisao separada do setup.

## Estrutura esperada do ambiente
Hoje, a raiz deste repositorio funciona como o `server-data` do projeto.

A estrutura minima esperada e esta:

```text
server-data/
|-- server/citizen/system_resources/
|   |-- chat
|   |-- yarn
|   `-- webpack
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
|   |-- [maps]/
|   |   |-- bob74_ipl
|   |   `-- mz_cayo
|   |-- [standalone]/
|   |   `-- screenshot-basic
|   |-- [mz]/
|   |   |-- mz_notify
|   |   |-- mz_sync
|   |   |-- mz_core
|   |   |-- mz_economy
|   |   |-- mz_inventory
|   |   |-- mz_status
|   |   |-- mz_progress
|   |   |-- mz_medical
|   |   |-- mz_bank
|   |   |-- mz_settings
|   |   |-- mz_menu
|   |   |-- mz_interact
|   |   |-- mz_world_editor
|   |   |-- mz_phone
|   |   |-- mz_transport
|   |   |-- mz_org
|   |   |-- mz_interiors
|   |   |-- mz_houses
|   |   |-- mz_furniture
|   |   |-- mz_org_activities
|   |   |-- mz_computer
|   |   |-- mz_vehicles
|   |   |-- mz_target
|   |   |-- mz_radio
|   |   |-- mz_creator
|   |   |-- mz_multichar
|   |   |-- mz_clothing
|   |   |-- mz_fuel
|   |   |-- mz_garagem
|   |   |-- mz_hud
|   |   |-- mz_banguard
|   |   |-- mz_warns
|   |   |-- mz_admin
|   |   `-- mz_feedback
|   |-- [ox]/
|   |   |-- oxmysql
|   |   `-- ox_lib
|   |-- [som]/
|   |   `-- pma-voice
|   `-- [system]/
|       `-- baseevents
`-- tmp/
```

## Fronteira pratica entre os repositorios
Use esta regra simples:
- se a mudanca e de boot, install, configuracao ou documentacao operacional, ela tende a pertencer ao `mz_starter`
- se a mudanca e de logica do framework, ela tende a pertencer ao `mz_core`
- se a mudanca exige mais recursos stock alem dos que ja sao sincronizados aqui, trate isso como uma decisao de infraestrutura, nao de framework

## Resources MZ sem repositorio publico

O starter nao presume que uma URL com o mesmo nome do resource exista. Enquanto `mz_status`, `mz_medical`, `mz_world_editor`, `mz_police`, `mz_prison`, `mz_dispatch`, `mz_furniture`, `mz_computer`, `mz_transport`, `mz_multichar`, `mz_warns` e `mz_feedback` forem distribuicao local, o operador deve fornecer `MZ_RESOURCE_SOURCE_ROOT` apontando para a raiz que contem essas pastas.

O preflight verifica esse conjunto antes de baixar ou alterar dependencias. Depois da instalacao, `tools/check_structure.bat` exige `fxmanifest.lua` para todo resource listado no boot. Isso torna ausencia de `mz_warns`/`mz_computer` um erro observavel de instalacao, nao uma degradacao silenciosa em runtime.

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

`bob74_ipl` e `mz_cayo` pertencem ao grupo runtime `[maps]`, mas iniciam por
linhas `ensure` individuais em `cfg/resources.cfg` para garantir ordem
determinística: Bob, Settings (inicialização geral de mapas MP), Cayo. O
`mz_cayo` fonte permanece em sua própria
pasta no workspace e é copiado pelo instalador a partir de
`MZ_RESOURCE_SOURCE_ROOT`.

## screenshot-basic e fotos futuras

`screenshot-basic` fica em `resources/[standalone]/screenshot-basic` por ser dependencia de infraestrutura, nao regra de dominio.

Ele prepara uma fase futura do `mz_realestate` para fotos de imoveis:

- `/mre_photo_take listing_000001`
- salvar imagem no servidor/storage
- tabela futura `mz_realestate_listing_photos`
- galeria no detalhe do anuncio
- futuramente site/API publica

Tabela futura sugerida, ainda nao criada:

```text
mz_realestate_listing_photos
- id
- listing_code
- image_url
- caption
- sort_order
- uploaded_by
- created_at
```

Quando essa fase for implementada, evite trafegar base64 grande por eventos; prefira fluxo controlado de upload, arquivo ou endpoint.
