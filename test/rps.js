import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 300,
  duration: '30s',
  // или точный RPS:
  // stages: [
  //   { duration: '30s', target: 1000 }, // 1000 RPS
  // ],
};

export default function () {
  const payload = JSON.stringify({
    device_id: `test-${__VU}-${__ITER}`,
    value: Math.random() < 0.1 ? 
      Math.random() * 50 + 100 :  // 10% аномалий
      Math.random() * 10 + 20     // 90% нормальных
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const res = http.post('http://metrics.local:30080/metric', payload, params);
  
  check(res, {
    'status is 202 or 429': (r) => r.status === 202 || r.status === 429,
  });
}