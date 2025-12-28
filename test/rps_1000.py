#!/usr/bin/env python3
"""
Нагрузочный тест 1000 RPS для метрик сервиса
"""
import urllib.request
import json
import random
import threading
import time
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed

TARGET_URL = "http://metrics.local/metric"
#TARGET_URL = "http://localhost:9898/metric"
TOTAL_REQUESTS = 1000  # сколько всего запросов
CONCURRENT_WORKERS = 300  # параллельных потоков
ANOMALY_RATE = 0.1  # 10% аномалий

success_count = 0
failure_count = 0
lock = threading.Lock()

def send_request(request_id):
    """Отправляет один запрос"""
    global success_count, failure_count
    
    # Генерация значения (10% аномалий)
    if random.random() < ANOMALY_RATE:
        value = random.uniform(100, 150)  # аномалия
    else:
        value = random.uniform(20, 30)    # нормальное
    
    data = json.dumps({
        "device_id": f"load-test-{request_id}",
        "value": value
    })
    
    try:
        req = urllib.request.Request(
            TARGET_URL,
            data=data.encode('utf-8'),
            headers={'Content-Type': 'application/json'},
            method='POST'
        )
        
        with urllib.request.urlopen(req, timeout=2) as response:
            if response.status == 202:
                with lock:
                    success_count += 1
                return True, request_id, value
        
        with lock:
            failure_count += 1
        return False, request_id, value
        
    except Exception as e:
        with lock:
            failure_count += 1
        return False, request_id, value

def main():
    print(f"=== Нагрузочный тест 1000 RPS ===")
    print(f"Цель: {TARGET_URL}")
    print(f"Всего запросов: {TOTAL_REQUESTS}")
    print(f"Потоков: {CONCURRENT_WORKERS}")
    print(f"Аномалий: {ANOMALY_RATE*100}%")
    print("-" * 40)
    
    start_time = time.time()
    
    # Используем ThreadPoolExecutor для параллельных запросов
    with ThreadPoolExecutor(max_workers=CONCURRENT_WORKERS) as executor:
        futures = []
        
        for i in range(1, TOTAL_REQUESTS + 1):
            future = executor.submit(send_request, i)
            futures.append(future)
        
        # Прогресс-бар
        completed = 0
        for future in as_completed(futures):
            completed += 1
            if completed % 100 == 0:
                print(f"Прогресс: {completed}/{TOTAL_REQUESTS}")
    
    end_time = time.time()
    duration = end_time - start_time
    
    print("-" * 40)
    print(f"РЕЗУЛЬТАТЫ:")
    print(f"Успешно: {success_count}/{TOTAL_REQUESTS}")
    print(f"Ошибок: {failure_count}/{TOTAL_REQUESTS}")
    print(f"Время: {duration:.2f} секунд")
    print(f"RPS: {TOTAL_REQUESTS/duration:.1f}")
    print(f"RPM: {(TOTAL_REQUESTS/duration)*60:.0f}")
    
    if duration > 0:
        actual_rps = TOTAL_REQUESTS / duration
        if actual_rps >= 1000:
            print("✅ ТЕСТ ПРОЙДЕН: 1000+ RPS достигнуто!")
        else:
            print(f"⚠️  Цель не достигнута: {actual_rps:.1f} RPS (требуется 1000)")
    
    print("\nПроверь HPA:")
    print("kubectl get hpa -w")

if __name__ == "__main__":
    main()