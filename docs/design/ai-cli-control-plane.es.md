# Arquitectura del plano de control

[🇬🇧 English](ai-cli-control-plane.md) | [🇧🇷 Português](ai-cli-control-plane.pt-BR.md) | **🇪🇸 Español**

IAPro Nexus separa presentación, planificación y ejecución. El CLI y las TUI llaman a la capa de aplicación; el servidor web expone API locales autenticadas; los registros mantienen proveedores y runtimes; scheduler, cuota, cooldown y fallback deciden qué perfil puede ejecutar; los adaptadores convierten la decisión para cada CLI.

Proyectos, Agentes y Misiones son identidades duraderas. Los runtimes y sesiones del proveedor son generaciones reemplazables bajo esas identidades. Workspace OS persiste el diseño sin mezclar estado visual con el proceso del terminal.

La internacionalización respeta la misma frontera: React usa catálogos `i18next`; CLI/TUI usan `go-i18n`; contratos JSON, IDs y enums no se localizan. Un idioma nuevo requiere registrar el locale, completar su catálogo y superar la prueba de paridad.

Límites de seguridad: loopback por defecto, autenticación y CSRF en la web, homes aislados, argumentos sin shell intermedio y redacción en handoffs. Los fallos degradan funciones concretas sin inventar estado ni capacidad.
