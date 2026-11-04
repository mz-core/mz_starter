# mz_starter

Camada de boot, configuracao e instalacao para subir um servidor FiveM usando `mz_core`.

## O que este repositorio e
- `mz_starter` organiza o `server.cfg`, os arquivos em `cfg/` e o fluxo basico de dependencias do ambiente.
- Ele existe para preparar e iniciar o servidor, nao para concentrar a logica do framework.

## O que este repositorio nao e
- `mz_starter` nao e o framework.
- A logica de dominio do servidor fica no `mz_core`: player, sessao, orgs, contas, inventario, veiculos e demais regras do core.

## Relacao com recursos oficiais
- O instalador sincroniza os recursos stock necessarios para o boot atual a partir do repositorio oficial `citizenfx/cfx-server-data`.
- Hoje ele traz `mapmanager`, `spawnmanager` e `baseevents`, mantendo o restante da base stock fora do escopo deste repositorio. `chat`, `yarn` e `webpack` sao fornecidos por `citizen/system_resources` nos artifacts atuais. O perfil suportado usa OneSync; portanto, nao instala nem inicia o `sessionmanager` legado de servidores sem OneSync.

## O que o instalador faz hoje
- baixa `oxmysql`
- baixa `ox_lib`
- sincroniza os recursos CFX base usados no `cfg/resources.cfg`
- clona ou atualiza `mz_core`
- clona ou atualiza `mz_notify`
- clona ou atualiza `mz_sync`
- clona ou atualiza os resources MZ listados no boot atual
- clona ou atualiza `screenshot-basic` em `resources/[standalone]`
- instala/reutiliza Bob74 e provisiona `mz_cayo` no grupo `resources/[maps]`
- instala `pma-voice` v7.0.0 no commit `3379e45dcf7aaa8b7651317190c108095a121553`
- aplica e verifica o patch MZ que remove o ingresso em call channel solicitado pelo client, preservando o export server `setPlayerCall`

## O que o instalador nao faz hoje
- nao instala FXServer artifacts
- nao provisiona a base stock completa do FiveM
- nao substitui configuracoes locais do servidor

## Instalacao rapida
1. Prepare os FXServer artifacts do ambiente.
2. Configure o banco em `cfg/database.cfg`.
3. Configure a license key em `cfg/base.cfg`.
4. Se a distribuicao ainda contiver resources sem repositorio publico, aponte `MZ_RESOURCE_SOURCE_ROOT` para a raiz do pacote MZ completo.
5. Rode `tools/install_deps.bat` ou `tools/install_deps.sh`.
6. Rode `tools/check_structure.bat` (Windows) ou `bash tools/check_structure.sh` (Linux) e somente inicie o servidor com a estrutura completa.

No workspace oficial Windows, por exemplo:

```bat
set "MZ_RESOURCE_SOURCE_ROOT=D:\git-hub\repo_oficial"
tools\install_deps.bat
tools\check_structure.bat
```

`mz_warns` e `mz_computer` fazem parte do runtime atual. O instalador interrompe o clean install antes de alterar dependencias se eles - ou qualquer outro resource distribution-only - nao estiverem na raiz-fonte nem previamente instalados.

## Catalogo de opcoes do FXServer

Veja [cfg/fivem_options.example.cfg](cfg/fivem_options.example.cfg) para exemplos comentados de configuracao geral, OneSync, pure mode, rede, listagem, seguranca e integracoes opcionais.

O catalogo nao e carregado pelo `server.cfg`; todas as opcoes permanecem desativadas. Copie somente as linhas que decidir usar para os arquivos existentes em `cfg/`, sem duplicar nem substituir configuracoes ja definidas. Convars especificas de resources podem nao estar no catalogo e variam conforme o artifact/resource instalado.

## Documentacao
- [Instalacao](docs/install.md)
- [Estrutura do ambiente](docs/structure.md)

O boot declara `ensure bob74_ipl`, `ensure mz_settings` e `ensure mz_cayo`, nessa
ordem; nao substitua por `ensure [maps]`. O instalador migra um Bob antigo
somente quando o clone está limpo e validado, e falha sem sobrescrever cópias
alteradas ou duplicadas. Consulte [Instalacao](docs/install.md) para o
procedimento do layout antigo.

## screenshot-basic

O `screenshot-basic` e dependencia do `mz_feedback`, fica em `resources/[standalone]/screenshot-basic` e inicia antes dos resources MZ.

Ele sera usado em uma fase futura do `mz_realestate` para fotos de imoveis. No estado atual, anuncios, placas e detalhes publicos continuam funcionando sem implementar captura de imagens.

Instalacao manual, se necessario:

```sh
mkdir -p "resources/[standalone]"
git clone https://github.com/citizenfx/screenshot-basic.git "resources/[standalone]/screenshot-basic"
```

## pma-voice

O `pma-voice` e instalado em `resources/[som]/pma-voice` antes de `mz_phone` e `mz_radio`. O pin e o patch fazem parte do contrato de seguranca do telefone: apenas resources server-side podem atribuir membership de chamada. O instalador aborta sem sobrescrever quando encontra uma copia sem `.git` ou alteracoes locais diferentes do patch esperado.

Nao atualize esse resource com `git pull` isolado. Para atualizar, mude tag, commit e patch juntos nos dois instaladores, valide o spoof de `pma-voice:setPlayerCall` e execute as regressões de `mz_phone` e `mz_radio`.
