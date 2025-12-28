#!/usr/bin/env python3
"""
Сверхпростой генератор без внешних зависимостей.
Использует только встроенные библиотеки.
"""

import urllib.request
import json
import random
import sys

def send_simple_metric(device_id, value):
    """Отправляет метрику используя только urllib (нет зависимостей)."""
    url = "http://172.27.185.72:30080/metric"
    data = json.dumps({"device_id": f"sensor_{device_id}", "value": value})
    
    try:
        req = urllib.request.Request(
            url,
            data=data.encode('utf-8'),
            headers={'Content-Type': 'application/json'},
            method='POST'
        )
        
        with urllib.request.urlopen(req, timeout=5) as response:
            if response.status == 202:
                return True
        return False
    except Exception as e:
        print(f"Ошибка для sensor_{device_id}: {e}")
        return False

def main():
    print("Генерация тестовых данных...")
    print("Нажмите Ctrl+C для остановки")
    print()
    
    i = 1
    try:
        while True:
            # Генерация значения
            if i % 10 == 0:
                value = random.uniform(100, 150)  # аномалия
                marker = "[АНОМАЛИЯ]"
            else:
                value = random.uniform(20, 30)    # нормальное
                marker = ""
            
            # Отправка
            success = send_simple_metric(i, value)
            symbol = "✓" if success else "✗"
            
            # Выводим только аномалии для наглядности
            if i % 10 == 0 or not success:
                print(f"{symbol} sensor_{i}: {value:.1f} {marker}")
            
            i += 1
            
            # Небольшая задержка для наглядности
            import time
            time.sleep(0.1)
            
    except KeyboardInterrupt:
        print(f"\n\nСгенерировано {i-1} метрик")
        print("Завершение...")

if __name__ == "__main__":
    main()