# Testing

## Gates mínimos

```bash
gofmt -w <arquivos alterados>
go test ./internal/control/driver ./internal/control/launcher ./internal/core/quota ./internal/core/scheduler
go build ./cmd/nexus
git diff --check
```

## Release/build

`make build` deve falhar quando `go build` falhar e só então instalar o binário. A versão exibida por `nexus version` deve corresponder a `VERSION`.

## Web

```bash
yarn dev --force
```

O acesso padrão é local em `http://127.0.0.1:3000`, usando a URL Bootstrap emitida pelo Nexus.
