# Log Processor

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)

Инструмент для анализа логов распределённых систем с конкурентной обработкой, обнаружением ошибок и построением временной хронологии событий.

## Возможности

- **Сопоставление записей по request_id** — группировка событий из разных сервисов по идентификатору запроса
- **Обнаружение ошибок** — автоматическое выявление failed requests с уровнем WARN/ERROR
- **Построение хронологии** — восстановление правильной последовательности событий для каждого запроса
- **Конкурентная обработка** — параллельное чтение файлов через worker pool с graceful shutdown
- **JSON-отчёты** — структурированный вывод с метриками и детальным описанием ошибок
- **Гибкий CLI** — настройка входного каталога и выходного файла через флаги

## Установка

### Требования

- Go 1.25 или выше
- Файлы логов в формате:
```json
"2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated, request_id=req_abc123, user_id=12345"
```

### Сборка

```bash
go build -o log-processor main.go
```

Или для всех платформ:

```bash
GOOS=linux GOARCH=amd64 go build -o log-processor main.go
GOOS=darwin GOARCH=arm64 go build -o log-processor main.go
```

## Использование

### Базовый запуск

```bash
./log-processor --input-dir ./logs --output-file results.json
```

### Доступные флаги

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--input-dir` | `.` | Каталог с `.log` файлами для анализа |
| `--output-file` | `results.json` | Путь к выходному JSON-файлу с результатами |

### Примеры

#### Анализ логов в текущей директории

```bash
./log-processor
```

#### Анализ конкретного каталога

```bash
./log-processor --input-dir /var/log/microservices/
```

#### Полный пример с указанием всех параметров

```bash
./log-processor \
--input-dir /var/log/microservices/ \
--output-file incident-report.json
```

#### Получение справки

```bash
./log-processor --help
```

Пример вывода:

```text
Usage of log-processor:
-input-dir string
Directory containing .log files (default ".")
-output-file string
JSON output file path (default "results.json")
```
## Формат логов
Инструмент ожидает логи в следующем формате:

`TIMESTAMP [LEVEL] SERVICE: MESSAGE, request_id=REQUEST_ID, user_id=USER_ID`


Пример:
```text
2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated, request_id=req_abc123, user_id=12345
2023-12-25T14:30:15.456Z [INFO] payment-service: Processing payment, request_id=req_abc123
2023-12-25T14:30:15.567Z [ERROR] payment-service: Card declined, insufficient_funds, request_id=req_abc123
```
### Поля

- **TIMESTAMP** — время в формате ISO 8601 (RFC3339Nano)
- **LEVEL** — уровень лога: `INFO`, `WARN`, `ERROR`
- **SERVICE** — имя сервиса (например, `user-service`, `payment-service`)
- **MESSAGE** — текст сообщения
- **request_id** — обязательное поле для сопоставления событий
- **user_id** — опциональное поле

## Пример отчёта

```json
{
  "total_entries_processed": 396,
  "failed_requests_found": 12,
  "processing_time_seconds": 1.23,
  "failed_requests": [
    {
      "request_id": "req_abc123",
      "failing_service": "payment-service",
      "error_message": "Card declined, insufficient_funds",
      "timeline": [
        "14:30:15.123Z [INFO] user-service: User authenticated",
        "14:30:15.234Z [INFO] account-service: Balance checked, amount=500",
        "14:30:15.456Z [INFO] payment-service: Processing payment...",
        "14:30:15.567Z [ERROR] payment-service: Card declined, insufficient_funds",
        "14:30:15.678Z [INFO] order-service: Order creation aborted",
        "14:30:15.789Z [INFO] notification-service: Failure email sent"
      ]
    }
  ]
}
```
