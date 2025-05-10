# Pubsub-Service
Сервис реализует паттерн Publisher-Subscriber (Pub/Sub) через gRPC, позволяя:

 - Клиентам подписываться на события по ключам (stream-соединение)

 - Клиентам публиковать события для всех подписчиков конкретного ключа

 - Обрабатывать сотни одновременных подключений

### API
``` 
rpc Subscribe(SubscribeRequest) returns (stream Event);
rpc Publish(PublishRequest) returns (google.protobuf.Empty); 
```
Использует пакет subpub: 

- Управляет подписками через Subscribe()/Unsubscribe()

- Рассылает события через Publish()

Ошибки:
- OK - успешное выполнение

- Internal - внутренние ошибки сервера

- InvalidArgument - невалидные аргументы

- DeadlineExceeded - превышено время ожидания

### Сборка 
***Go 1.23.3***

```
git clone https://github.com/nutockk/vk-test
cd vk-test
go mod tidy   
go build ./cmd
```
./config/config.yaml
```
GRPCPort: 50051
ShutdownTimeout: 10s
```
### Паттерны
**Dependency Injection:**

- Логирование инжектируется во все слои

**Graceful Shutdown:**

- Oбработка сигналов завершения

- Ожидание завершения активных подключений

**Publisher-Subscriber:**

- Асинхронная доставка сообщений

**Слоистая архитектура:**

- Отдельный слой для работы с gRPC

- Отдельный слой для работы с subpub