# Desarrollo de proveedores

[🇬🇧 English](provider-development.md) | [🇧🇷 Português](provider-development.pt-BR.md) | **🇪🇸 Español**

Un adaptador implementa detección, autenticación, ejecución, reanudación y lectura de cuota según las capacidades reales de la CLI. Registre el adaptador, mantenga los argumentos como una lista segura y no declare soporte cuando la herramienta no ofrece una operación.

Los datos contractuales usan IDs y enums estables en inglés. Los textos humanos creados por Nexus deben usar IDs de `internal/localization`; la salida libre del proveedor permanece intacta. Añada pruebas de detección, comando, aislamiento, clasificación de fallos y fallback.
