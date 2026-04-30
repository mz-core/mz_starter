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
- mantem pastas locais que existam sem `.git`, para evitar sobrescrever alteracoes manuais

## Conferir estrutura
No Windows, use:

```bat
tools\check_structure.bat
```

Se algum resource aparecer como `[MISSING]`, rode o instalador ou confira se o link do repositorio daquele resource ainda precisa ser ajustado no script.

## Observacao
O `mz_starter` nao atualiza FXServer artifacts. Essa parte continua sendo uma decisao separada da infraestrutura do servidor.
