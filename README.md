# wb-L0
This is a training task from WB

### Установка
```bash
git clone git@github.com:renegatemaster/wb-l0.git
cd wb-l0/
docker compose up -d
```

В docker compose два контейнера: Nats Streaming и PostgreSQL
Создадим таблицу для наших данных в PostgreSQL
```bash
docker exec -it db psql -U test_user -d test
```
```SQL
CREATE TABLE orders (
uid varchar(50) UNIQUE NOT NULL PRIMARY KEY,
data jsonb NOT NULL
);
```

### Использование
(не забудьте настроить файл .env) <br>
Для публикации данных в канал исполните команду:
```bash
go run pub/pub.go
```
Для активирования подписчика исполните команду:
```bash
go run main.go
```
Данные из базы данных будут загружены в кэш сервиса <br>
Входящие сообщения будут записываться в базу данных и кэш

Получить данные по API можно по GET-запросам вида:
```
http://localhost:8080/orders/b563feb7b2b84b6test
```
Тестовые запросы уже заготовлены в папке requests
