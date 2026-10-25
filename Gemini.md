Да, лучше без ТГ-бота: логичнее сделать **Go-сервис / CLI-парсер Etsy**, который мониторит новые листинги, фильтрует их и сохраняет результат в JSON/CSV/SQLite.[1][2]

Ниже готовое ТЗ именно под такой вариант — без Telegram, только ядро парсинга и фильтрации.[3][4]

## Цель

Разработать приложение на **Go**, которое регулярно получает **новые активные товары Etsy**, обогащает их данными магазина продавца, применяет заданные фильтры и сохраняет подходящие результаты в локальное хранилище и/или экспортные файлы.[2][1]

## Формат проекта

Нужен не бот, а один из двух режимов работы:

- **CLI-режим**: запуск командой вручную, один проход парсинга.
- **Daemon-режим**: фоновый процесс с циклическим опросом Etsy каждые N секунд.[1]

Приложение должно поддерживать оба режима.

## Стек

- Go 1.22+.
- Etsy Open API v3.[2]
- Хранилище: SQLite.
- Экспорт: JSON Lines и CSV.
- Конфиг: `.env` + optional `config.yaml`.

## Источник данных

Использовать Etsy Open API v3 для получения новых листингов.[2]

Основной endpoint:
- `GET /v3/application/listings?state=active&limit=...&offset=...`.[3][1]

Правила сортировки:
- Основной порядок — newest first по creation date, так как `findAllListingsActive` возвращает активные листинги с пагинацией по дате создания.[1]
- Учитывать, что `sort_order`/`sort_on` в Etsy работают не во всех сценариях одинаково, поэтому нужно опираться на реальный `creation_timestamp` и дополнительно сортировать локально при необходимости.[5][1]

Для каждого найденного листинга догружать данные магазина через shop endpoint, чтобы получить shop-level метрики вроде `listing_active_count`, `review_average`, `review_count`, `transaction_sold_count`, `create_date`.[4][6]

## Авторизация

Поддержать Etsy OAuth 2.0 Authorization Code Flow и работу с Bearer token.[2]

Обязательные заголовки:
- `x-api-key`
- `Authorization: Bearer <access_token>`.[7][2]

Минимальные scope’ы:
- `shops_r`
- `listings_r`.[2]

## Основной сценарий работы

### Режим single run
По команде запуска приложение:
1. Получает первую страницу новых активных листингов Etsy.[1]
2. Для каждого листинга получает shop-данные.[6][4]
3. Применяет фильтры.
4. Сохраняет прошедшие результаты.
5. Печатает краткую сводку в stdout.

### Режим daemon
В цикле:
1. Раз в `SCAN_INTERVAL_SEC` получает свежие листинги.[1]
2. Отбрасывает уже обработанные `listing_id`.
3. Обогащает новыми данными магазина.
4. Применяет фильтры.
5. Сохраняет совпадения.
6. Обновляет state последнего прохода.

## Что собирать по каждому объекту

### Данные листинга
Обязательные поля:
- `listing_id`
- `shop_id`
- `title`
- `url`
- `price`
- `currency_code`
- `creation_timestamp`
- `state` если доступно
- `image_url` если доступна главная картинка.[7][1]

Опциональные поля:
- `views`
- `description_short`
- `tags`
- `materials`
- `taxonomy_id`
- `shipping_origin` если Etsy реально возвращает.[6][1]

### Данные магазина
Обязательные поля:
- `shop_id`
- `shop_name`
- `create_date`
- `listing_active_count`
- `review_count`
- `review_average`
- `transaction_sold_count`
- `shop_location_country_iso` или страна, если доступно.[4][6]

Опциональные:
- `is_vacation`
- `currency_code`
- `user_id`
- `seller_name/display_name`, если доступно отдельным запросом.[6]

## Фильтры

### Обязательные фильтры
Система фильтров должна быть конфигурируемой и позволять включать/выключать каждый фильтр отдельно.

Поддержать:

- **Цена товара**
  - disabled
  - min
  - max
  - range

- **Дата регистрации продавца**
  - фильтр по `shop.create_date`
  - примеры: `registered_before`, `registered_after`

- **Возраст объявления**
  - например: “не старше 30 минут”, “не старше 6 часов”

- **Количество объявлений продавца**
  - фильтр по `listing_active_count`
  - пример: `>= 5`.[4]

- **Количество продаж продавца**
  - фильтр по `transaction_sold_count`.[6]

- **Рейтинг продавца**
  - фильтр по `review_average`
  - отдельно по `review_count`, чтобы рейтинг 5.0 с 1 отзывом не считался сильным сигналом.[6]

- **Страна / регион**
  - whitelist / blacklist стран.

### Поля, которые не делать обязательными
Следующие поля не должны быть основой логики в MVP:

- Гендер продавца — официального публичного поля нет.[2]
- Количество купленных товаров продавцом — это не публичная shop-метрика через обычный публичный сценарий Etsy API.[8][2]
- Просмотры “за период” — Etsy не дает такой надежной метрики в официальном API; максимум использовать lifetime views, если поле действительно приходит.[9][1]

Если поле недоступно, приложение должно сохранять `null`, а не выдумывать значение.

## Выходные данные

### SQLite
Сохранять данные минимум в таблицы:

- `listings_raw`
- `shops_raw`
- `matches`
- `scan_runs`
- `processed_listing_ids`

### JSONL
Каждый прошедший фильтр листинг сохранять отдельной строкой JSON в файл:
- `output/matches.jsonl`

### CSV
Сохранять плоский экспорт:
- `output/matches.csv`

Поля CSV:
- listing_id
- title
- price
- currency_code
- listing_url
- image_url
- created_at
- listing_age_minutes
- shop_id
- shop_name
- shop_created_at
- shop_age_days
- listing_active_count
- transaction_sold_count
- review_average
- review_count
- country
- matched_at

## Dedup и state

Нужно реализовать защиту от дублей:

- Один и тот же `listing_id` не должен повторно записываться в `matches`, если уже был сохранен ранее.
- После рестарта процесса приложение должно помнить уже обработанные товары.
- Дополнительно хранить:
  - `last_seen_created_timestamp`
  - `last_scan_time`
  - `last_successful_scan_time`

## Архитектура

Рекомендуемая структура:

- `cmd/etsy-monitor` — main entrypoint.
- `internal/config`
- `internal/etsy`
- `internal/model`
- `internal/filters`
- `internal/storage`
- `internal/export`
- `internal/worker`
- `internal/logging`

## Конфигурация

Через env:

- `ETSY_API_KEY`
- `ETSY_ACCESS_TOKEN`
- `SCAN_INTERVAL_SEC`
- `SQLITE_PATH`
- `OUTPUT_DIR`
- `WORKER_CONCURRENCY`
- `HTTP_TIMEOUT_SEC`
- `LOG_LEVEL`

Через config-файл:
- фильтры
- лимиты
- режим запуска
- экспортные опции

Пример фильтров в `config.yaml`:

```yaml
mode: daemon
scan_interval_sec: 60

filters:
  price:
    enabled: false
    min: 0
    max: 0

  seller_registered:
    enabled: false
    before: ""
    after: ""

  listing_age:
    enabled: true
    max_minutes: 30

  shop_listing_count:
    enabled: true
    min: 5

  shop_sales:
    enabled: false
    min: 0

  rating:
    enabled: false
    min_average: 0
    min_reviews: 0

  country:
    enabled: false
    allow: []
    deny: []
```

## Обработка ошибок

Приложение должно:

- корректно обрабатывать 429 и 5xx от Etsy;
- использовать exponential backoff;
- иметь таймауты на HTTP-запросы;
- ограничивать параллелизм запросов к shop endpoint;
- логировать ошибки без падения всего процесса.[2]

Если Etsy временно возвращает `null` в некоторых shop-полях, приложение не должно падать на десериализации; поля магазина должны быть nullable, потому что такие кейсы уже наблюдались у разработчиков.[6]

## Логирование

Нужно structured logging с уровнями:
- debug
- info
- warn
- error

Логировать:
- старт/стоп процесса
- начало и конец scan run
- количество найденных листингов
- количество прошедших фильтры
- количество дублей
- ошибки API
- ошибки записи в БД / экспорт

## Производительность

Приложение должно стабильно работать на недорогом VPS и не утекать по памяти при длительном daemon-режиме.[1]

Ожидаемое поведение:
- первая страница листингов: до 100 элементов за проход.[1]
- shop enrichment — ограниченный concurrency pool.
- без повторного запроса к shop, если он уже был получен недавно и есть кэш в SQLite, например TTL 10–30 минут.

## Тесты

Минимально покрыть тестами:

- price filter
- listing age filter
- shop listing count filter
- rating filter
- dedup logic
- CSV export formatter
- human time conversion

## Deliverables

Кодер должен сдать:

- исходный код на Go;
- `README.md`;
- `.env.example`;
- `config.example.yaml`;
- SQL migration или auto-init схемы;
- пример запуска в CLI-режиме;
- пример запуска в daemon-режиме;
- unit tests для ключевой логики;
- пример экспортируемых `matches.jsonl` и `matches.csv`.

## Критерии приемки

Работа считается выполненной, если:

1. Приложение стартует локально с валидными Etsy credentials.[2]
2. Успешно получает новые активные листинги Etsy.[1]
3. Обогащает их данными магазина.[4][6]
4. Корректно применяет фильтр `listing_active_count >= 5`.[4]
5. Сохраняет только подходящие записи в SQLite и CSV/JSONL.
6. Не создает дубликаты после рестарта.
7. Не падает, если часть shop-полей пришла `null`.[6]
8. Не использует неофициальный scraping как основную механику MVP.[2]

Если хочешь, я сейчас сделаю еще **сильно сокращенную версию ТЗ**, буквально как сообщение на 15–20 строк, чтобы ты просто переслал кодеру.

Sources
[1] Reference | Etsy Open API v3 https://developers.etsy.com/documentation/reference/
[2] Authentication | Etsy Open API v3 https://developer.etsy.com/documentation/essentials/authentication/
[3] Summary of Changes | Etsy Open API v3 https://developers.etsy.com/documentation/migration/summary-changes/
[4] 596 Service Not Found Error https://groups.google.com/g/etsy-api-v2/c/Reg15CtHIZA
[5] etsy-open-api-client/docs/ShopListingApi.md at main · gordonturner/etsy-open-api-client https://github.com/gordonturner/etsy-open-api-client/blob/main/docs/ShopListingApi.md
[6] Suddenly getting null values on shop data · etsy open-api · Discussion #1279 https://github.com/etsy/open-api/discussions/1279
[7] Listings Tutorial | Etsy Open API v3 https://developer.etsy.com/documentation/tutorials/listings
[8] Authentication | Etsy Open API v3 https://developer.etsy.com/documentation/essentials/authentication
[9] Add shop statistics to the Etsy API · etsy open-api · Discussion #1386 https://github.com/etsy/open-api/discussions/1386
[10] Etsy .../listings/active API call returns the wrong amount of active listings https://stackoverflow.com/questions/79031128/etsy-listings-active-api-call-returns-the-wrong-amount-of-active-listings
[11] ShopListingApi - etsy https://etsy.guimauve.digital/ShopListingApi.html
[12] etsy/open-api: We are working on a new version of Etsy's ... - GitHub https://github.com/etsy/open-api
[13] Etsy Scraping API - Extract Handmade Product & Shop Data https://www.browsable.app/apis/etsy/
[14] gordonturner/etsy-open-api-client: Java library providing ... - GitHub https://github.com/gordonturner/etsy-open-api-client
[15] Does Etsy Limit The Number of Search Result Records You Can Access? · etsy open-api · Discussion #1188 https://github.com/etsy/open-api/discussions/1188
[16] Testing: Trust, But Verify https://rollout.com/integration-guides/etsy/reading-and-writing-data-using-the-etsy-api
