import http from "k6/http";
import { sleep } from "k6";

export default function () {
  const url = "http://localhost:8080/predict";
  const payload = JSON.stringify({
    sepal_length: 6.0,
    sepal_width: 3.0,
    petal_length: 5.5,
    petal_width: 2.0,
  });
  const params = { headers: { "Content-Type": "application/json" } };
  http.post(url, payload, params);
  sleep(0.1); // 模拟每秒 10 次请求/用户
}
