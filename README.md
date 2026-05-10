# Практическая работа №8 (семестр 2)

## Выполнила: Сорокина К.С., ЭФМО-01-25

## Тема: Настройка GitHub Actions CI/CD для деплоя приложения

### Цель:

Сделать автоматическую проверку качества и сборку проекта при каждом push, а также подготовить базовый конвейер доставки: тесты → сборка → упаковка Docker-образа → публикация в registry.

## Технологии

- **Go** — язык реализации
- **GitHub Actions** — CI/CD платформа
- **Docker / Dockerfile** — контейнеризация сервиса
- **GitHub Container Registry (ghcr.io)** — хранилище Docker-образов
- **GITHUB_TOKEN** — встроенный секрет для авторизации в registry

## Структура проекта

```
PR8_sem2/
├── .github/
│   └── workflows/
│       └── ci.yml              ← pipeline GitHub Actions
├── go.mod
├── go.sum
├── .dockerignore
├── shared/
├── proto/
├── services/
│   ├── auth/
│   │   └── cmd/auth/main.go
│   └── tasks/
│       ├── cmd/tasks/main.go
│       ├── internal/
│       └── Dockerfile
└── deploy/
    └── tls/
        └── docker-compose.yml
```

---

## Pipeline (.github/workflows/ci.yml)

```yaml
name: CI

on:
  push:
    branches: ["main"]
  pull_request:
    branches: ["main"]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Run tests
        run: go test ./...

      - name: Build tasks
        run: go build ./services/tasks/cmd/tasks

  docker:
    runs-on: ubuntu-latest
    needs: test
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: services/tasks/Dockerfile
          push: true
          tags: |
            ghcr.io/${{ github.repository_owner }}/techip-tasks:latest
            ghcr.io/${{ github.repository_owner }}/techip-tasks:${{ github.sha }}
```

---

## Описание шагов pipeline

Pipeline состоит из двух job, которые выполняются последовательно: `docker` запускается только после успешного прохождения `test`.

**Job `test`:**
1. `Checkout` — клонирует репозиторий в окружение CI
2. `Setup Go` — устанавливает версию Go из `go.mod`
3. `Run tests` — запускает `go test ./...` по всем пакетам проекта
4. `Build tasks` — компилирует бинарник сервиса tasks, проверяя что код собирается без ошибок

**Job `docker`:**
1. `Checkout` — повторно клонирует репозиторий
2. `Log in to GHCR` — авторизуется в GitHub Container Registry через встроенный секрет `GITHUB_TOKEN`
3. `Set up Docker Buildx` — настраивает расширенный движок сборки Docker
4. `Build and push` — собирает образ из `services/tasks/Dockerfile` и публикует его с двумя тегами

---

## Версионирование образов

Каждый образ публикуется с двумя тегами:

| Тег | Описание |
|---|---|
| `latest` | всегда указывает на последний успешный билд ветки main |
| `${{ github.sha }}` | полный hash коммита — позволяет точно знать из какого кода собран образ |

Образ доступен по адресу:
```
ghcr.io/krrristina/techip-tasks:latest
ghcr.io/krrristina/techip-tasks:<commit-sha>
```

Скачать образ:
```bash
docker pull ghcr.io/krrristina/techip-tasks:latest
```

---

## Секреты CI

| Секрет | Откуда берётся | Для чего используется |
|---|---|---|
| `GITHUB_TOKEN` | Встроенный секрет GitHub Actions | Авторизация в ghcr.io для push образа |

`GITHUB_TOKEN` создаётся GitHub автоматически для каждого запуска pipeline и не требует ручной настройки. Токены и пароли никогда не хранятся в репозитории и не указываются в `ci.yml` в открытом виде.

---

## Результат

Pipeline успешно прошёл за **1м 32с**:
- `test` — 34 секунды
- `docker` — 50 секунд

Образ опубликован в GitHub Container Registry с тегами `latest` и хешем коммита.

![](https://github.com/krrristina/PR8_sem2/blob/main/screenshots/тест%20выполнен.png)

![](https://github.com/krrristina/PR8_sem2/blob/main/screenshots/packages.png)

## Ответы на контрольные вопросы

**Чем CI отличается от CD?**
CI (Continuous Integration) — автоматическая проверка кода при каждом коммите: тесты, сборка, линтеры. Цель — быстро обнаружить ошибки. CD (Continuous Delivery/Deployment) — автоматическая доставка проверенного кода до окружения: staging или production. CI — это "мы убедились что код рабочий", CD — "мы его задеплоили".

**Почему go test должен запускаться в pipeline?**
Чтобы ошибки обнаруживались сразу при push, а не после деплоя на сервер. Если тест упал — job docker не запустится, сломанный образ не попадёт в registry. Это защита от регрессий.

**Что такое секреты CI и почему их нельзя хранить в репозитории?**
Секреты — это чувствительные данные (токены, пароли, SSH-ключи), которые нужны pipeline но не должны быть видны посторонним. Хранение в репозитории опасно: git-история публична, токен можно извлечь из любого коммита даже после удаления. GitHub Actions шифрует секреты и подставляет их только во время выполнения pipeline, не показывая в логах.

**Почему важно версионировать docker-образы?**
Тег `latest` всегда перезаписывается — непонятно какой код внутри. Тег с хешем коммита позволяет точно воспроизвести любой деплой: знаем коммит → знаем образ → знаем что задеплоено. Это критично при откате на предыдущую версию.

**Какие риски у автоматического деплоя без ручного контроля?**
Любой коммит в main автоматически уходит в production. Если тесты не покрывают какую-то ошибку — она сразу попадает к пользователям. Решение — добавить ручное подтверждение (environment protection rules в GitHub) или деплоить автоматически только на staging, а в production — вручную.
