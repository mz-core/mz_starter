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
- mantem pastas locais que existam sem `.git`, para evitar sobrescrever alteracoes manuais
- mantem `pma-voice` fixado em v7.0.0/`3379e45dcf7aaa8b7651317190c108095a121553` e reaplica/verifica o patch server-authoritative de calls

`pma-voice` e a excecao ao `git pull --ff-only`: o instalador confere o commit exato e aceita somente a modificacao prevista em `server/module/phone.lua`. Qualquer outra mudanca local interrompe a atualizacao sem sobrescrever o resource. Uma nova versao exige atualizar o pin, revisar a API e revalidar o patch.

## Conferir estrutura
No Windows, use:

```bat
tools\check_structure.bat
```

Se algum resource aparecer como `[MISSING]`, rode o instalador ou confira se o link do repositorio daquele resource ainda precisa ser ajustado no script.

## Observacao
O `mz_starter` nao atualiza FXServer artifacts. Essa parte continua sendo uma decisao separada da infraestrutura do servidor.
