# Seguridad

[🇬🇧 English](security.md) | [🇧🇷 Português](security.pt-BR.md) | **🇪🇸 Español**

Nexus es local-first. Los perfiles usan directorios aislados, el servidor web escucha en loopback por defecto y los handoffs aplican redacción de secretos. No exponga la interfaz directamente a Internet; use un túnel SSH o una VPN privada.

Ejecute `nexus security` para auditar los límites de archivos compartidos y `nexus doctor` para comprobar binarios, autenticación y permisos. Nunca incluya tokens, cookies ni credenciales en informes. Los mensajes y logs de proveedores conservan su idioma original como evidencia operativa.
