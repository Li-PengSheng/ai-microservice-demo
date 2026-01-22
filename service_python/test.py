import random
import time

import requests

# 配置地址
GATEWAY_URL = "http://localhost:8080/predict"
METRICS_URL = "http://localhost:8080/metrics"

def test_predict():
    """测试单次预测功能"""
    payload = {
        "sepal_length": 5.1,
        "sepal_width": 3.5,
        "petal_length": 1.4,
        "petal_width": 0.2
    }
    try:
        response = requests.post(GATEWAY_URL, json=payload)
        print(f"预测结果: {response.status_code}, 返回: {response.json()}")
    except Exception as e:
        print(f"预测接口异常: {e}")

def test_metrics():
    """检测 Prometheus 指标接口"""
    try:
        response = requests.get(METRICS_URL)
        if "http_requests_total" in response.text:
            print("Prometheus 指标接口正常，已检测到 http_requests_total")
        else:
            print("指标接口未包含预期数据")
    except Exception as e:
        print(f"指标接口异常: {e}")

def run_load_test(count=50):
    """生成测试数据以填充 Grafana 和 Jaeger"""
    print(f"开始生成 {count} 条测试数据...")
    for i in range(count):
        payload = {
            "sepal_length": round(random.uniform(4.0, 7.0), 1),
            "sepal_width": round(random.uniform(2.0, 4.5), 1),
            "petal_length": round(random.uniform(1.0, 6.0), 1),
            "petal_width": round(random.uniform(0.1, 2.5), 1)
        }
        requests.post(GATEWAY_URL, json=payload)
        if i % 10 == 0:
            print(f"已完成 {i} 次请求...")
        time.sleep(0.1)  # 模拟真实间隔

if __name__ == "__main__":
    test_predict()
    test_metrics()
    run_load_test(5000)
