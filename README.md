# Open questions

Как выбор структуры базы данных (SQL или NoSQL) влияет на дизайн CRUD API? `в SQL строгая схема базы данных также API должен учитывать связи между таблицами нужны валидации данных. NoSQL удобен на ранних стадиях когда еще не сформировалась точная структура `

Какие проблемы могут возникнуть при массовых обновлениях данных через API? `нагрузка на бд, потяря данных, проблемы с сетью`

Почему важно использовать правильные HTTP-методы (GET, POST, PUT, DELETE), а не только POST? `для удобства что бы не создавать отдельные ендпоинты для каждой операции и было логически понятно`

Какие уязвимости могут возникнуть при хранении JWT на клиентской стороне? `XSS CSRF атаки`

В каких случаях стоит ограничивать время жизни JWT, и какие проблемы это создаёт для UX? `когда высокие требования безопастноси, нужно часто логиниться `

Как логирование помогает в расследовании инцидентов безопасности? `есть запись всех действий что присходили на сервере можно просто найти источник и причину инцидента`

В чём разница между горизонтальным и вертикальным масштабированием, и как это связано с кэшированием? `вертикально увеличиваем ресурсы мощности рабочей машины, горизонтально добавляем новые рабочие машины или инстансы приложения. Горизонтальное масштабирование требует распределённых кэшей`

Какой риск несут фоновые задачи при сбое очереди сообщений?` Потеря или дублирование данных или событий, нарушение порядка`

Почему важно учитывать идемпотентность задач при их повторном выполнении?` Позволяет безопасно повторять задачи без побочных эффектов.`

Что сложнее поддерживать в большой системе: код или документацию? Почему?` Документацию так как поддерживать код это необходимость а на доку могут просто забивать `

Какие плюсы и минусы у ручного написания README по сравнению с автогенерацией документации? `Более понятный, дружелюбный для людей текст, но требует ручного постоянного обновления. Автогенерация всегда синхронизирована с кодом/API но сухая, формальная`

Как документация помогает при онбординге новых разработчиков в команду? `Быстрое понимание архитектуры, процессов. Меньше вопросов к коллегам`
 
 
 # Overview

The client process tracking service allows tracking the stages of client interaction with a product or service,
supporting full CRUD operations (create, read, update, delete) for clients, as well as providing metrics for business analysis. The
service provides validation of transitions between stages, email validation, duplicate prevention, structured logging, optimized interaction with MongoDB, and
visual indicators for key data (for example, highlighting in red for unset parameters). The service is universal and can
be adapted to any product or service registration process, defined by the configuration of stages.

## What's inside:

- Migrations
- Swagger docs
- Environment configuration
- Docker development environment
- Redis caching
- MongoDB database
- Exporting metrics to Prometheus
- Unit and integration tests of handlers, service and repositories layers with coverage > 80%
- Github Actions CI/CD pipeline

## Usage

1. Copy .env.dist to .env and set the environment variables. In the .env file set these variables:
2. Change for local development in [docker-compose.yml](docker-compose.yml#L52) the following line:
```yaml
  - /trackme/prometheus.yml:/etc/prometheus/prometheus.yml # Change this line to your local path
  - ./prometheus.yml:/etc/prometheus/prometheus.yml # To this or your local path
```
3. Build the Docker image:

```sh
docker compose build
```

4. Run the Docker container:

```sh
docker compose up
```

4. Browse to {HTTP_HOST}:{HTTP_PORT}/swagger/index.html. You will see Swagger 2.0 API documents.


## OpenAPI Documentation
The OpenAPI documentation is generated using the swagger and available [here](docs/swagger.json).

## Client Management API

The service provides a complete REST API for managing clients throughout their lifecycle.

### Create Client
#### `POST /{base-path}/clients`

Creates a new client with validation:
- **Email validation**: Ensures valid email format using regex pattern
- **Duplicate prevention**: Checks if a client with the same email already exists
- **Stage validation**: Validates that the initial stage is valid according to the configured stages

#### Request body:
```json
{
   "name": "John Doe",
   "email": "john.doe@example.com",
   "stage": "registration",
   "is_active": true,
   "source": "website",
   "channel": "organic",
   "app": "not_installed",
   "last_login": "2024-01-15T10:00:00Z",
   "contracts": [
      {
         "id": "contract123",
         "autopayment": "enabled"
      }
   ]
}
```

#### Response (201 Created):
Returns the created client with all fields populated.

#### Errors:
- `400 Bad Request`: Invalid email format or invalid initial stage
- `409 Conflict`: Client with this email already exists
- `500 Internal Server Error`: Server error

---

### List Clients
#### `GET /{base-path}/clients`

Retrieves a paginated list of clients with optional filtering.

#### Query parameters:
- `id` - Filter by client ID
- `stage` - Filter by current stage
- `source` - Filter by source
- `channel` - Filter by channel
- `app` - Filter by app status (e.g., "installed", "not_installed")
- `is_active` - Filter by active status (default: true)
- `updated` - Filter by last updated after date (format: YYYY-MM-DD)
- `last_login` - Filter by last login date after (format: YYYY-MM-DD)
- `limit` - Pagination limit (default: 50)
- `offset` - Pagination offset (default: 0)

#### Response (200 OK):
```json
{
   "data": [
      {
         "id": "client123",
         "name": "John Doe",
         "email": "john.doe@example.com",
         "stage": "active",
         "is_active": true,
         "registration_date": "2024-01-15T10:00:00Z",
         "last_updated": "2024-01-20T15:30:00Z",
         "source": "website",
         "channel": "organic",
         "app": {
            "status": "installed",
            "highlight": false
         },
         "last_login": {
            "date": "2024-01-20",
            "highlight": false
         },
         "contracts": [
            {
               "id": "contract123",
               "autopayment": {
                  "status": "enabled",
                  "highlight": false
               }
            }
         ]
      }
   ],
   "meta": {
      "total": 100,
      "limit": 50,
      "offset": 0
   }
}
```

---

### Update Client Stage
#### `PUT /{base-path}/clients/{id}/stage`

Updates an existing client's information and stage. Does NOT create a new client if not found (returns 404).

#### Path parameters:
- `id` - Client ID (required)

#### Request body:
Same structure as Create Client request.

#### Validation:
- **Stage transition validation**: Ensures the stage transition is valid according to configured rules
- **Email validation**: Validates email format if provided
- **Not found check**: Returns 404 if client doesn't exist (no automatic creation)

#### Response:
- `200 OK`: Client updated successfully (existing client)
- `400 Bad Request`: Invalid stage transition or invalid data
- `404 Not Found`: Client with specified ID not found
- `500 Internal Server Error`: Server error

---

### Delete Client
#### `DELETE /{base-path}/clients/{id}`

Deletes a client from the system.

#### Path parameters:
- `id` - Client ID (required)

#### Response:
- `204 No Content`: Client deleted successfully
- `404 Not Found`: Client with specified ID not found
- `500 Internal Server Error`: Server error

---

## Metrics Overview
Project calculates several metrics like mau, dau, conversions, application install rate, etc. Metrics calculation triggers 
by cron job every midnight for daily every week and every first day of the month for week and month metrics respectively.

### Metrics calculation can be triggered manually by this endpoint:
#### `GET /{base-path}/metrics/calculate`
#### Query parameters:
- `interval` - the interval for which the metrics should be calculated. Possible values: `day`, `week`, `month`

#### Response:
```json
{
   "data": {
      "message": "triggerred success"
   }
}
```

### Visual Indicators (Highlights)
The service provides visual indicators to highlight important information that requires attention:


Mobile Application Status: Highlighted in red when the client's mobile app is not installed (not_installed).


Last Login Date: Highlighted in red when the client hasn't logged in for more than 30 days, indicating potential disengagement.


Autopayment Status: Highlighted in red for contracts where autopayment is disabled (disabled), which might require manual payment attention.

Example response with highlights:
```json
{
   "app": {
      "status": "not_installed",
      "highlight": true
   },
   "last_login": {
      "date": "2024-05-01",
      "highlight": true
   },
   "contracts": [
      {
         "autopayment": {
            "status": "disabled",
            "highlight": true
         }
      }
   ]
}
```


## Directories

1. **main.go**: contains the application's main entry point(s) or command-line interfaces (CLIs). Each subdirectory
   represents a different executable within the project
2. **/internal**: houses the internal components of your application that are not intended to be imported by external
   projects. This directory typically contains packages/modules related to business logic, domain models, repositories,
   services, and configuration.
3. **/internal/app**: this section may include any initialization code that needs to be executed before the application
   starts. For example, setting up configuration, connecting to databases, or initializing logging.
4. **/internal/cache**: directory allows for the separation of caching concerns from other parts of the application,
   promoting modularity and maintainability. By isolating caching-related code, it becomes easier to manage and test
   caching functionality independently. However, the specific directory structure and organization may vary based on the
   project's needs and preferences.
5. **/internal/config**: holds the configuration-related code and files. It includes the logic to read and parse
   configuration files, environment variables, or other sources of configuration data. It provides a centralized way to
   manage and access application configuration throughout the codebase.
6. **/internal/domain**: directory, you separate the core business logic from infrastructure-specific or
   framework-specific code. This separation helps keep your code clean, maintainable, and easier to test. It also allows
   for better reusability and modularity, as the domain layer can be used independently of the specific infrastructure
   or framework being used.
7. **/internal/handler**: contains the HTTP or RPC handlers for the application. These handlers are responsible for
   receiving incoming requests, parsing them, invoking the necessary business logic, and returning the appropriate
   responses. Each handler typically corresponds to a specific endpoint or operation in the application's API.
8. **/internal/repository**: contains the implementation of data access and persistence logic. It provides an
   abstraction over the data storage layer, allowing the application to interact with databases, or other external
   systems. Repositories handle the CRUD operations and data querying required by the application.
9. **/internal/service**: contains the implementation of the application's business logic. It encapsulates the core
   functionality of the application and provides high-level operations that the handlers can use to accomplish specific
   tasks. Services interact with data repositories, external APIs, or other dependencies to fulfill the application's
   requirements.
10. **/migrations/{store}**: contains database migration scripts, which are used to manage database schema changes over
    time.
11. **/pkg**: contains packages that can be imported and used by external projects. These packages are typically
    utilities, libraries, or modules that have potential for reuse across different projects.

## Libraries

1. Router: https://github.com/go-chi/chi
2. Migrations: https://github.com/golang-migrate/migrate
3. Swagger: https://github.com/swaggo/swag


# Swagger: HTTP tutorial for beginners

1. Add comments to your API source code, See [Declarative Comments Format](#declarative-comments-format).

2. Download swag by using:

```sh
go install github.com/swaggo/swag/cmd/swag@latest
```

To build from source you need [Go](https://golang.org/dl/) (1.17 or newer).

Or download a pre-compiled binary from the [release page](https://github.com/swaggo/swag/releases).

3. Run `swag init` in the project's root folder which contains the `main.go` file. This will parse your comments and
   generate the required files (`docs` folder and `docs/docs.go`).

```sh
swag init
```

Make sure to import the generated `docs/docs.go` so that your specific configuration gets `init`'ed. If your General API
annotations do not live in `main.go`, you can let swag know with `-g` flag.

  ```sh
  swag init -g internal/handler/handler.go
  ```

4. (optional) Use `swag fmt` format the SWAG comment. (Please upgrade to the latest version)

  ```sh
  swag fmt

  ```




