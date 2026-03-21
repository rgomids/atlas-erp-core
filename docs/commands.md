# Command Reference

Este arquivo centraliza os principais comandos operacionais da foundation e deve ser atualizado sempre que novos fluxos recorrentes forem introduzidos.

## Setup e runtime

```bash
make setup
make up
make down
make run
make build
```

## Qualidade

```bash
make fmt
make lint
make test
make test-unit
make test-integration
make test-functional
```

## Banco e migrations

```bash
make migrate-up
make migrate-down
```

## Healthcheck manual

```bash
curl http://localhost:8080/health
```
