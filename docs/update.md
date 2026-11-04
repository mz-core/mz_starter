# Atualizacao

## Objetivo
Este documento resume como atualizar um ambiente ja instalado com o `mz_starter`.

## Atualizar dependencias
Rode novamente o instalador da plataforma:

```bat
tools\install_deps.bat
```

```sh
bash tools/install_deps.sh
```

O instalador:
- atualiza repositorios Git ja existentes com `git pull --ff-only`
- baixa novamente as releases de `oxmysql` e `ox_lib`
- sincroniza os recursos CFX base usados no `cfg/resources.cfg`
- instala/atualiza `screenshot-basic` em `resources/[standalone]/screenshot-basic`
- instala/reutiliza `bob74_ipl` 2.7.1 em `resources/[maps]/bob74_ipl`
- copia `mz_cayo` para `resources/[maps]/mz_cayo` sem sobrescrever resource local
- mantem pastas locais que existam sem `.git`, para evitar sobrescrever alteracoes manuais
- exige `MZ_RESOURCE_SOURCE_ROOT` somente para instalar pela primeira vez resources distribution-only que ainda nao tenham sido colocados em `resources/[mz]`
- mantem `pma-voice` fixado em v7.0.0/`3379e45dcf7aaa8b7651317190c108095a121553` e reaplica/verifica o patch server-authoritative de calls
- mantem `bob74_ipl` fixado em 2.7.1/`037dd6c71f6894be771e9343008223d7a3305d79`; não atualiza nem substitui uma instalação divergente automaticamente

`pma-voice` e a excecao ao `git pull --ff-only`: o instalador confere o commit exato e aceita somente a modificacao prevista em `server/module/phone.lua`. Qualquer outra mudanca local interrompe a atualizacao sem sobrescrever o resource. Uma nova versao exige atualizar o pin, revisar a API e revalidar o patch.

Na primeira atualização do layout antigo, uma única instalação Bob em
`resources/[standalone]/bob74_ipl` só é movida automaticamente se o clone estiver
limpo e no commit pinado. Cópias alteradas, sem `.git`, divergentes, duplicadas
ou em outro grupo bloqueiam o instalador; faça backup e migre manualmente após
validar a instalação. O destino `[maps]` nunca é sobrescrito.

`mz_cayo` tem sua fonte separada em `MZ_RESOURCE_SOURCE_ROOT/mz_cayo`. Se já
existir em `resources/[maps]/mz_cayo`, o instalador a preserva; publique uma
atualização pelo processo normal do ambiente e valide o preflight. O boot usa
`ensure bob74_ipl`, `ensure mz_settings`, `ensure mz_cayo`, nesta ordem, não
`ensure [maps]`.

## Conferir estrutura
No Windows, use:

```bat
tools\check_structure.bat
```

No Linux, use:

```sh
bash tools/check_structure.sh
```

Se algum resource aparecer como `[MISSING]`, rode o instalador ou confira se o link do repositorio daquele resource ainda precisa ser ajustado no script.

Para `mz_status`, `mz_medical`, `mz_world_editor`, `mz_police`, `mz_prison`, `mz_dispatch`, `mz_furniture`, `mz_computer`, `mz_transport`, `mz_multichar`, `mz_warns` e `mz_feedback`, nao ha URL publica confirmada no starter. Nao invente uma URL: atualize a distribuicao local de forma controlada e mantenha `MZ_RESOURCE_SOURCE_ROOT` apontando para o pacote completo em um clean install.

## Observacao
O `mz_starter` nao atualiza FXServer artifacts. Essa parte continua sendo uma decisao separada da infraestrutura do servidor.
