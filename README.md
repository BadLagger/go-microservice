# Простая микросервисная реализация бэкэнда на GO #

## Описание ##

Отказоустойчивый, высокопроизводительный микросервис для приема и обработки метрик от IoT-устройств, способный обрабатывать не менее 1000 запросов в секунду (RPS) с гарантированным временем отклика и автоматическим масштабированием под нагрузкой.

## Технологический стэк ##

- __Go 1.24+__ - основной язык разработки
- __Gorilla Mux__ - роутинг HTTP запросов
- __Redis__ - хранилище
- __Prometheus__ - сбор и хранение метрик
- __Grafana__ - визуализация и дашборды
- __minikube__ - оркестрация контейнеров

## Архитектура ##

```text
┌─────────────────┐     HTTP/HTTPS     ┌──────────────────┐
│   IoT Devices   │───────────────────▶│   Nginx Ingress  │
│  (1000+ units)  │                    │    Controller    │
└─────────────────┘                    └─────────┬────────┘
                                                 │
                                                 ▼
                                        ┌──────────────────┐
                                        │   Go Service     │
                                        │  (metrics-app)   │
                                        └─────────┬────────┘
                                              │   │   │
                ┌────────────────┬────────────┘   │   └──────────────┐
                ▼                ▼                ▼                  ▼
        ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌────────────────┐
        │   Rate       │ │   Metrics    │ │   Anomaly    │ │   Health       │
        │   Limiter    │ │   Storage    │ │   Detection  │ │   Checks       │
        │  (1000 RPS)  │ │  (Redis)     │ │  (Analytics) │ │  (Readiness)   │
        └──────────────┘ └──────────────┘ └──────────────┘ └────────────────┘
```

## Доступные сервисы ##
|Сервис| Порт | Описание |
|------|------|----------|
| Go Application | 30080 | Основной API сервер |
| Prometheus | 30090 | Система мониторинга и сбора метрик |
| Grafana | 	30300 | Визуализация метрик и дашборды |

## API Endpoints ##

### Метрики IoT ###
- POST /metric - Добавить метрику
- GET //metrics/latest - Получить последнюю метрику

### Тестирование и мониторинг ###

- GET /metrics - Prometheus метрики
- GET /health - Health check

##  Развертывание Minikube ##
```bash
# удаляем старый кластер если есть
minikube delete

# создаём новый кластер
#minikube start --cpus=4 --memory=8192 --disk-size=20gb
minikube start --driver=hyperv --cpus=4 --memory=8192 --disk-size=20gb

# Это для Linux
# eval $(minikube docker-env)
# Это для Windows
minikube docker-env | Invoke-Expression

# Включаем метрики
minikube addons enable metrics-server

# Включаем Ingress
minikube addons enable ingress

# Собираем образ приложения
docker build -f k8s/Dockerfile -t metrics-app:latest .

kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/app.yaml
kubectl apply -f k8s/hpa.yaml
kubectl apply -f k8s/prometheus-config.yaml
kubectl apply -f k8s/prometheus.yaml
kubectl apply -f .\k8s\grafana-configs.yaml
kubectl apply -f .\k8s\grafana.yaml
kubectl apply -f .\k8s\ingress.yaml


minikube ip
# Далее всё для Windows 11 (Power Shell)
# Проверяем хосты
Get-Content C:\Windows\System32\drivers\etc\hosts

# Если в хостах уже есть metrics.local чистим
$hostsPath = "C:\Windows\System32\drivers\etc\hosts"
$cleanLines = Get-Content $hostsPath | Where-Object {
     $_ -notmatch "metrics\.local" -and
     $_ -notmatch "^\s*$"  # удалить пустые строки
 }

 Set-Content -Path $hostsPath -Value $cleanLines -Force

# Сохраняем в переменную
$ip = minikube ip

# Прописываем в хосты
Add-Content -Path "C:\Windows\System32\drivers\etc\hosts" -Value "`n${ip} metrics.local" -Force
```

## Мониторинг и метрики ##

### Дашборды Grafana ###

Включены готовые дашборды:
1. RPS график - запросы в секунду
2. Latency график - время ответа
3. IoT метрики

## Нагрузочное тестирование ##
```bash
choco install k6
# Для тестирования:
k6.exe run .\rps.js
```

### Интерпретация результатов ###
- Успешные запросы: ~1000 RPS (ограничено rate limiter)
- 429 ошибки: при превышении лимита
- Latency: стабильная при нагрузке


