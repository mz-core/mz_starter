# Instalacao

## Objetivo deste guia
Este documento descreve o contrato operacional atual do setup do `mz_starter`.

O foco aqui e deixar claro:
- o que precisa existir antes de rodar o instalador
- o que o instalador realmente faz
- o que continua fora do escopo deste repositorio

## Starter 2.0: instalacao completa pela interface

O executavel `mz-starter` abre a interface local. Em Instalacao completa do MZ Core, informe um destino novo ou vazio, uma pasta de distribuicao local autorizada para os resources sem release publica e, opcionalmente, artifacts existentes. Clique em Verificar componentes para ver os 51 componentes, releases, dependencias, bloqueios e avisos antes de instalar. O botao INSTALAR SERVIDOR so e habilitado sem bloqueios e com a confirmacao marcada.

A instalacao completa escreve `server.cfg`, `logo.png`, `cfg/`, `resources/` e os resources resolvidos em um staging privado. Nada e publicado se qualquer componente obrigatorio falhar; o destino precisa continuar vazio e nao pode estar dentro de um servidor existente. Resources publicos com SHA fixado usam a URL direta do GitHub Release; os demais usam pacotes locais autorizados, releases privados com token ou os snapshots oficiais fixados das dependencias externas.

Os artifacts sao separados do server-data. O modo existente valida o executavel; o modo de download oficial so esta disponivel no Windows e registra o SHA-256 obtido. Sem artifacts, o resultado e `completed_files_only`: os arquivos estao completos, mas o servidor nao esta pronto para iniciar. Licenca, banco, endpoints e permissoes nunca sao preenchidos automaticamente.

O modo Instalacao parcial experimental continua disponivel e instala apenas um resource publico com confirmacao explicita. Os scripts `tools/install_deps.*` permanecem como contrato legado para atualizacao de uma arvore ja existente.

## Papel de cada parte no setup

### `mz_starter`
Responsavel pela camada de boot do projeto:
- `server.cfg`
- arquivos em `cfg/`
- scripts de instalacao de dependencias do projeto

### `mz_core`
Responsavel pela logica do framework:
- dominio de player e sessao
- orgs, contas, inventario, veiculos e logs
- regras e fluxos internos do core

### `cfx-server-data`
Responsavel pela base stock do servidor FiveM.

No estado atual deste projeto, o instalador usa o repositorio oficial `citizenfx/cfx-server-data` para sincronizar apenas os recursos stock necessarios ao boot atual:
- `mapmanager`
- `spawnmanager`
- `baseevents`

Os artifacts atuais do FXServer fornecem `chat`, `yarn` e `webpack` em `citizen/system_resources`. O perfil FiveM/OneSync do projeto nao requer o `sessionmanager` legado.

O restante da base stock continua fora do escopo do `mz_starter`.

## Pre-requisitos
Antes de rodar o instalador, o ambiente precisa ter:
- FXServer artifacts ja preparados
- Git disponivel
- acesso a internet para baixar dependencias
- `curl` e `unzip` disponiveis no Linux
- uma distribuicao MZ completa para os resources ainda sem repositorio publico

## Fluxo recomendado de instalacao
1. Prepare o ambiente base do FiveM com seus artifacts.
2. Coloque o `mz_starter` como raiz do `server-data` do projeto.
3. Ajuste as configuracoes locais:
   - `cfg/database.cfg`
   - `cfg/base.cfg`
   - `cfg/permissions.cfg`, se necessario
   - consulte `cfg/fivem_options.example.cfg` para exemplos opcionais do FXServer
4. Defina `MZ_RESOURCE_SOURCE_ROOT` quando resources distribution-only ainda nao estiverem nos destinos runtime (`resources/[mz]` e `resources/[maps]/mz_cayo`).
5. Rode o instalador de dependencias do projeto.
6. Rode `tools/check_structure.bat` no Windows ou `bash tools/check_structure.sh` no Linux e corrija qualquer `[MISSING]`.
7. Inicie o servidor usando o `server.cfg` deste repositorio.

`cfg/fivem_options.example.cfg` e um catalogo de referencia, nao e executado pelo
`server.cfg` e todas as linhas de configuracao estao comentadas. Copie apenas as
opcoes desejadas para os arquivos ativos, sem duplicar valores ja configurados.

## Como rodar o instalador

### Windows
1. Abra a pasta do `mz_starter`.
2. Aponte a raiz da distribuicao completa e rode o instalador:

```bat
set "MZ_RESOURCE_SOURCE_ROOT=D:\git-hub\repo_oficial"
tools\install_deps.bat
tools\check_structure.bat
```

### Linux
1. Abra o terminal na pasta do `mz_starter`.
2. Aponte a raiz da distribuicao completa e rode o instalador:

```sh
export MZ_RESOURCE_SOURCE_ROOT=/opt/mz-framework
bash tools/install_deps.sh
bash tools/check_structure.sh
```

`MZ_RESOURCE_SOURCE_ROOT` deve conter as pastas `mz_*` e `mz_cayo` diretamente abaixo dela. O instalador usa essa origem somente quando o target ainda nao existe, exclui metadados Git/Graphify e nunca substitui uma pasta local ja instalada. Em staging que ja possua `mz_police`, publique a revisao pelo processo normal de deploy antes do gate; rodar novamente o instalador nao atualiza essa copia.

## Grupo `[maps]` e migração do Bob74

O layout de runtime usa:

```text
resources/
`-- [maps]/
    |-- bob74_ipl/
    `-- mz_cayo/
```

O boot lista os resources explicitamente e nesta ordem:

```cfg
ensure bob74_ipl
ensure mz_settings
ensure mz_cayo
```

`mz_settings` inicializa os mapas MP gerais e é uma dependência de
`mz_cayo`. Nao use `ensure [maps]`: a ordem de descoberta dentro do grupo nao
garante que Bob inicie antes de `mz_cayo`.

Ao executar o instalador completo:

- um Bob ausente e clonado diretamente para `resources/[maps]/bob74_ipl`;
- uma única cópia em `resources/[standalone]/bob74_ipl` é movida para `[maps]`
  somente quando é um clone Git limpo no commit pinado;
- alterações locais, commit divergente, ausência de `.git`, destino já ocupado,
  caminho inesperado ou cópias duplicadas interrompem o instalador sem
  sobrescrever/remover conteúdo;
- nesses bloqueios, faça backup e valide manualmente a única cópia antes de
  movê-la para `resources/[maps]/bob74_ipl`; não mantenha ambas ativas;
- `mz_cayo` é copiado da pasta-fonte `$MZ_RESOURCE_SOURCE_ROOT/mz_cayo` para
  `resources/[maps]/mz_cayo` somente se não houver uma instalação local;
  arquivos preexistentes são preservados e precisam conter os arquivos
  verificados pelo preflight.

O instalador não foi executado como parte desta alteração; não há movimentação
de Bob, cópia ou deploy automático neste checkout.

## O que o instalador faz hoje
- baixa a release estavel do `oxmysql`
- baixa a release estavel do `ox_lib`
- sincroniza `mapmanager`, `spawnmanager` e `baseevents` do repositorio oficial `citizenfx/cfx-server-data`
- resolve todos os resources MZ declarados em `cfg/resources.cfg`, na mesma ordem do boot
- clona/atualiza os resources com repositorio Git confirmado
- copia da distribuicao indicada por `MZ_RESOURCE_SOURCE_ROOT` os resources que ainda nao possuem repositorio publico
- falha no preflight, antes de alterar dependencias, se algum resource distribution-only obrigatorio estiver ausente
- clona ou atualiza `screenshot-basic`
- instala `pma-voice` v7.0.0 no commit `3379e45dcf7aaa8b7651317190c108095a121553`
- aplica e verifica o patch `patches/pma-voice-server-authoritative-calls.patch`
- instala ou reutiliza `bob74_ipl` 2.7.1 no commit `037dd6c71f6894be771e9343008223d7a3305d79`
- instala `mz_cayo` da distribuição local em `resources/[maps]/mz_cayo`
- falha sem sobrescrever o Bob existente quando encontra versão/commit divergente ou mais de uma instalação

## O que o instalador nao faz hoje
- nao instala FXServer artifacts
- nao baixa a base stock completa do FiveM
- nao cria configs locais do servidor
- nao inventa URLs para resources MZ que ainda nao possuem repositorio publico confirmado

## Manifest de clean install

`cfg/resources.cfg` e a autoridade da ordem de boot MZ. Os instaladores Windows/Linux e `tools/check_structure.bat` devem cobrir exatamente essa lista.

No estado atual, estes resources sao `distribution-only`: `mz_status`, `mz_medical`, `mz_world_editor`, `mz_police`, `mz_prison`, `mz_dispatch`, `mz_furniture`, `mz_computer`, `mz_transport`, `mz_multichar`, `mz_warns` e `mz_feedback`. Eles precisam existir previamente em `resources/[mz]` ou estar na raiz indicada por `MZ_RESOURCE_SOURCE_ROOT`. `mz_cayo` é uma fonte MZ local separada e é provisionada em `resources/[maps]/mz_cayo`.

`mz_warns` e obrigatorio no perfil atual porque fornece a persistencia de advertencias consumida por `mz_admin`. `mz_computer` e obrigatorio no gate IC-0 porque sua sessao precisa ser revalidada contra `sessionId`/`characterId`. Nenhum dos dois e instalado por outro caminho conhecido.

## O que precisa ficar entendido sem ambiguidade
- O `mz_starter` nao substitui o `mz_core`.
- O `mz_starter` usa apenas uma parte do `cfx-server-data`, nao a base completa.
- O `mz_core` entra como recurso do framework dentro de `resources/[mz]/mz_core`.
- O `mz_notify` entra em `resources/[mz]/mz_notify`.
- O `mz_sync` entra em `resources/[mz]/mz_sync`.
- O `pma-voice` entra em `resources/[som]/pma-voice`.
- O `pma-voice` fica em detached HEAD no commit aprovado e com somente `server/module/phone.lua` modificado pelo patch MZ. O instalador falha se encontrar outro estado local, em vez de sobrescreve-lo.
- O `screenshot-basic` entra em `resources/[standalone]/screenshot-basic`.
- O `bob74_ipl` entra em `resources/[maps]/bob74_ipl`; uma instalação antiga única, limpa e validada em `[standalone]` é migrada de caminho.
- O `mz_cayo` entra em `resources/[maps]/mz_cayo`, usando a fonte separada `mz_cayo` na raiz MZ indicada.
- O Bob é obtido de `https://github.com/Bob74/bob74_ipl`, tag `2.7.1`, licença MIT. O preflight verifica a versão e os arquivos usados pelo starter.
- Os recursos oficiais sincronizados entram em `resources/[managers]` e `resources/[system]`; `chat`, `yarn` e `webpack` permanecem no `citizen/system_resources` do artifact.

## Resultado esperado
Ao final desta etapa, o ambiente deve ter:
- o `server.cfg` e os `cfg/*.cfg` do `mz_starter`
- `oxmysql` em `resources/[ox]/oxmysql`
- `ox_lib` em `resources/[ox]/ox_lib`
- `mapmanager` em `resources/[managers]/mapmanager`
- `spawnmanager` em `resources/[managers]/spawnmanager`
- `baseevents` em `resources/[system]/baseevents`
- `chat`, `yarn` e `webpack` no `citizen/system_resources` dos artifacts
- `pma-voice` em `resources/[som]/pma-voice`
- `screenshot-basic` em `resources/[standalone]/screenshot-basic`
- `bob74_ipl` 2.7.1 e `mz_cayo` em `resources/[maps]`, cada um em uma única pasta
- `mz_core` em `resources/[mz]/mz_core`
- `mz_notify` em `resources/[mz]/mz_notify`
- `mz_sync` em `resources/[mz]/mz_sync`
- todos os resources `mz_*` declarados em `cfg/resources.cfg`, incluindo `mz_warns` e `mz_computer`, em `resources/[mz]/`

Para a visao de fronteira entre os repositorios e a estrutura esperada do diretorio, veja [docs/structure.md](structure.md).

## Instalacao manual de screenshot-basic

Se o instalador nao for usado, instale manualmente:

```sh
mkdir -p "resources/[standalone]"
git clone https://github.com/citizenfx/screenshot-basic.git "resources/[standalone]/screenshot-basic"
```

No Windows PowerShell:

```powershell
New-Item -ItemType Directory -Force "resources/[standalone]"
git clone https://github.com/citizenfx/screenshot-basic.git "resources/[standalone]/screenshot-basic"
```

O resource deve iniciar antes de `mz_realestate` quando o `mz_realestate` estiver no boot do ambiente.

## Contrato de voz do telefone

O setup define OneSync e os convars de calls, radios, native audio, sending range e submix em `cfg/onesync.cfg`. `pma-voice` inicia antes de `mz_phone`; o telefone usa exclusivamente o export server `setPlayerCall(source, channel)`. O patch remove o handler upstream que aceitava `pma-voice:setPlayerCall` enviado pelo client.

Uma instalacao manual precisa reproduzir exatamente o pin e aplicar o patch:

```sh
git clone --branch v7.0.0 --depth 1 https://github.com/AvarianKnight/pma-voice.git "resources/[som]/pma-voice"
git -C "resources/[som]/pma-voice" checkout --detach 3379e45dcf7aaa8b7651317190c108095a121553
git -C "resources/[som]/pma-voice" apply "$(pwd)/patches/pma-voice-server-authoritative-calls.patch"
```

O hash da tag e a ausencia do handler client-authoritative devem ser verificados antes do boot. O runtime instalado na VPS nao foi acessado durante esta mudanca; sua versao deve ser comparada com esse commit no rollout.
