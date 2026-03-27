# codex-switch

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | Español | [한국어](./README.ko.md)

Usa directorios `CODEX_HOME` separados para mantener aisladas varias cuentas de Codex.

La idea es simple:

- cada cuenta tiene su propio `CODEX_HOME`
- `codex-switch` inicia `codex` con el perfil que elijas
- los accesos directos se generan como aliases del shell, así que no hace falta reescribir un `~/.codex/auth.json` compartido

## Por Qué Hacerlo Así

Separar los `CODEX_HOME` evita que se mezclen `sessions`, metadatos de uso y snapshots de autenticación. Además, el comportamiento queda claro: la cuenta activa es la que corresponde al `CODEX_HOME` con el que se lanzó `codex`.

## Estructura Del Proyecto

La CLI sigue una estructura de proyecto Go bastante estándar:

```text
cmd/codex-switch/      # punto de entrada del binario
internal/cli/          # parseo de comandos y comportamiento visible al usuario
internal/config/       # modelo de configuración y persistencia
internal/profile/      # operaciones sobre CODEX_HOME y auth.json
internal/runner/       # abstracción para ejecutar comandos externos
internal/shellinit/    # instalación/desinstalación de snippets de inicialización del shell
```

## Instalación

La forma más simple de compilar es usar el `Makefile` incluido:

```sh
make build
```

Eso inyecta automáticamente la versión `dev`. Si quieres compilar una versión concreta:

```sh
make build VERSION=v1.0.0
```

También puedes compilarlo directamente con `go build`:

```sh
go build -o codex-switch ./cmd/codex-switch
```

Si compilas desde el código fuente y luego quieres ejecutar `codex-switch` directamente, mueve el binario a un directorio que ya esté en tu `PATH` o ejecútalo como `./codex-switch`.

## Pruebas Y Desarrollo

Ejecutar las pruebas:

```sh
make test
```

Ejecutar las comprobaciones estáticas:

```sh
make vet
```

## Homebrew

Si distribuyes esta CLI con Homebrew, el flujo normal sería algo así:

```sh
brew install <tap>/codex-switch
codex-switch doctor
```

Comandos habituales del ciclo de vida:

```sh
brew upgrade codex-switch
brew uninstall codex-switch
```

Notas:

- Sustituye `<tap>` por el tap real que publiques, por ejemplo `your-org/tap`.
- `brew uninstall codex-switch` elimina el binario gestionado por Homebrew.
- No borra snippets de inicialización del shell ni datos del usuario bajo `~/.codex-switch` o `CODEX_SWITCH_HOME`.
- Usa `codex-switch cleanup` para quitar la integración con el shell, y `codex-switch cleanup --purge-data` si también quieres borrar los datos gestionados.

## Inicio Rápido

Para una primera prueba, esto suele bastar:

```sh
go build -o codex-switch ./cmd/codex-switch

codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

Si quieres que `cwork .` funcione de inmediato en el shell actual:

```sh
eval "$(codex-switch aliases --shell zsh)"
cwork .
```

Si quieres que los accesos directos se carguen automáticamente en terminales futuras:

```sh
codex-switch init-shell --shell zsh
source ~/.zshrc
cwork .
```

Conviene tener esto presente:

- `add --shortcut` solo guarda el nombre del acceso directo en la configuración.
- No crea por sí solo un comando nuevo en la terminal que ya está abierta.
- `init-shell` afecta a los shells futuros. El shell actual todavía necesita `source ~/.zshrc` o una terminal nueva.
- Un alias de acceso directo antepone `codex` automáticamente, así que `cwork .` equivale a `codex-switch run work -- codex .`.

## Comandos

```sh
codex-switch add <name> [--home path] [--label text] [--shortcut command] [--default]
codex-switch list
codex-switch show <name>
codex-switch status [--json]
codex-switch doctor [--json]
codex-switch remove <name>
codex-switch set-default <name>
codex-switch run [<profile>] [-- command args...]
codex-switch login [<profile>] [--copy-from-current] [-- command args...]
codex-switch import-auth <profile> [--from path]
codex-switch env [<profile>] [--shell sh|bash|zsh|fish]
codex-switch aliases [--shell bash|zsh|fish]
codex-switch init-shell [--shell bash|zsh|fish] [--rc-file path]
codex-switch uninit-shell [--shell bash|zsh|fish] [--rc-file path]
codex-switch cleanup [--shell bash|zsh|fish] [--rc-file path] [--purge-data]
codex-switch version
```

## Ejemplos

Añadir dos perfiles aislados:

```sh
codex-switch add work --shortcut cwork --default
codex-switch add personal --shortcut cpersonal
```

Iniciar sesión directamente en un perfil:

```sh
codex-switch login work
codex-switch login personal -- codex login
codex-switch login personal --copy-from-current
```

Importar un archivo de autenticación existente a un perfil:

```sh
codex-switch import-auth work --from ~/.codex/auth.json
codex-switch import-auth personal --from /path/to/old-codex-home
```

Ejecutar Codex dentro de un perfil:

```sh
codex-switch run work -- codex .
codex-switch run personal -- codex chat
codex-switch run -- codex .   # usa el perfil predeterminado
```

Revisar los perfiles configurados:

```sh
codex-switch list
codex-switch show work
codex-switch status
```

Si hay datos de autenticación, `list` y `show` incluyen directamente el resumen de la cuenta enlazada, con campos como `auth_email`, `auth_account_id` y `auth_name`. Eso ayuda bastante a comprobar que dos perfiles no están apuntando a la misma cuenta por accidente.

Exportar un perfil al shell actual:

```sh
eval "$(codex-switch env work)"
codex .
```

Generar aliases de accesos directos:

```sh
eval "$(codex-switch aliases --shell zsh)"

cwork .
cpersonal chat
```

Estos accesos directos pasan el resto de argumentos tal cual a `codex`, así que formas como `cwork .` y `cpersonal chat` funcionan sin más.

Instalar el cargador de aliases en el archivo rc del shell:

```sh
codex-switch init-shell --shell zsh
codex-switch init-shell --shell bash
codex-switch init-shell --shell fish
```

Eliminar el snippet gestionado del shell:

```sh
codex-switch uninit-shell --shell zsh
codex-switch uninit-shell --shell bash
codex-switch uninit-shell --shell fish
```

Limpiar la integración del shell y, si quieres, borrar también los datos almacenados:

```sh
codex-switch cleanup --shell zsh
codex-switch cleanup --shell zsh --purge-data
```

Hacer una comprobación rápida del entorno:

```sh
codex-switch doctor
codex-switch doctor --json
```

`doctor` también muestra el correo, el ID de cuenta y el nombre visible de cada perfil. Viene bien para detectar rápido cuentas cruzadas o un perfil predeterminado mal configurado.

## Flujos De Trabajo Típicos

Crear un perfil aislado nuevo e iniciar sesión desde cero:

```sh
codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

Reutilizar la cuenta actual sin abrir un nuevo flujo de inicio de sesión:

```sh
codex-switch add personal --shortcut cpersonal
codex-switch login personal --copy-from-current
codex-switch run personal -- codex .
```
