import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 300,
  duration: '180s',
};

export default function () {
  const payload = JSON.stringify({
    device_id: `test-${__VU}-${__ITER}`,
    value: Math.random() * 10 + 20
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const res = http.post('http://metrics.local/metric', payload, params);
  
  // Раздельные checks для разных статусов
  // Они появятся в итоговой статистике
  check(res, {
    'status is 202 (accepted)': (r) => r.status === 202,
  });
  
  check(res, {
    'status is 429 (rate limited)': (r) => r.status === 429,
  });
  
  // Общий check для отображения в результатах
  check(res, {
    'status is 202 or 429': (r) => r.status === 202 || r.status === 429,
  });
}