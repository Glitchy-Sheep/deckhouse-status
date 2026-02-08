# Релиз

## Как выпустить новую версию

1. Убедись что `main` в актуальном состоянии:
   ```bash
   git checkout main && git pull
   ```

2. Создай тег и запушь:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

3. GitHub Actions автоматически:
   - Соберёт бинарники для linux/darwin x amd64/arm64
   - Создаст GitHub Release с changelog
   - Загрузит бинарники как release assets

4. Проверь результат: `https://github.com/glitchy-sheep/deckhouse-status/releases`

## Что происходит под капотом

- Воркфлоу `.github/workflows/release.yaml` запускается на пуш тега `v*`
- Использует [GoReleaser](https://goreleaser.com/) с конфигом `.goreleaser.yaml`
- Собирает статические бинарники (`CGO_ENABLED=0`, stripped)
- Именует файлы как `deckhouse-status-{os}-{arch}` (без архивов)

## Версионирование

Используется [semver](https://semver.org/): `vMAJOR.MINOR.PATCH`

- **patch** (`v1.0.1`) — баг-фиксы
- **minor** (`v1.1.0`) — новые фичи, обратно совместимые
- **major** (`v2.0.0`) — ломающие изменения

## Локальная проверка (опционально)

```bash
goreleaser release --snapshot --clean
```

Соберёт всё локально без публикации — полезно для проверки конфига.
