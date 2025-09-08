### 1. 克隆项目
```bash
git clone <repository-url>
cd certificate-traceability
```

### 2. 启动区块链网络
```bash
cd network
chmod +x scripts/*.sh
./scripts/start-network.sh
```

### 3. 启动后端服务
```bash
cd backend
go mod tidy
go run main.go
```
