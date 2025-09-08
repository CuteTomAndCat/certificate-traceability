#!/bin/bash

API_BASE="http://localhost:8080/api/v1"

echo "=== 计量证书区块链API测试 ==="

# 1. 创建证书
echo "1. 创建证书..."
CERT_RESPONSE=$(curl -s -X POST ${API_BASE}/certificates \
-H "Content-Type: application/json" \
-d '{
  "certificateNo": "CERT-2025-001",
  "testUnit": "华为技术有限公司",
  "testDate": "2025-01-15",
  "testData": [
    {
      "parameter": "电压",
      "measuredValue": 220.5,
      "unit": "V",
      "uncertainty": 0.1,
      "method": "直接测量法",
      "equipment": "数字万用表"
    },
    {
      "parameter": "电流",
      "measuredValue": 5.2,
      "unit": "A",
      "uncertainty": 0.05,
      "method": "钳表测量法",
      "equipment": "数字钳表"
    }
  ],
  "inspectionOrg": "中国计量科学研究院",
  "inspector": "张三",
  "validUntil": "2026-01-15",
  "createdBy": "system"
}')

CERT_ID=$(echo $CERT_RESPONSE | jq -r '.certificateId')
echo "证书创建成功，ID: $CERT_ID"

# 2. 获取证书详情
echo -e "\n2. 获取证书详情..."
curl -s -X GET ${API_BASE}/certificates/${CERT_ID} | jq .

# 3. 签发证书
echo -e "\n3. 签发证书..."
curl -s -X POST ${API_BASE}/certificates/${CERT_ID}/issue \
-H "Content-Type: application/json" \
-d '{
  "operator": "张三"
}' | jq .

# 4. 获取更新后的证书
echo -e "\n4. 获取签发后的证书..."
curl -s -X GET ${API_BASE}/certificates/${CERT_ID} | jq .

# 5. 查询所有证书
echo -e "\n5. 查询所有证书..."
curl -s -X GET ${API_BASE}/certificates | jq .

# 6. 按状态查询证书
echo -e "\n6. 按状态查询证书..."
curl -s -X GET "${API_BASE}/certificates?status=issued" | jq .

# 7. 获取证书历史
echo -e "\n7. 获取证书历史..."
curl -s -X GET ${API_BASE}/certificates/${CERT_ID}/history | jq .

echo -e "\n=== API测试完成 ==="