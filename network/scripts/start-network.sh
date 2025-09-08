#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印函数
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker和Docker Compose
check_prerequisites() {
    print_info "检查环境依赖..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    # 检查Docker版本
    DOCKER_VERSION=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    print_info "Docker版本: $DOCKER_VERSION"
    
    # 检查Docker Compose版本
    COMPOSE_VERSION=$(docker-compose --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    print_info "Docker Compose版本: $COMPOSE_VERSION"
}

# 清理之前的网络
cleanup_network() {
    print_info "清理之前的网络..."
    
    # 停止并删除容器
    docker-compose -f docker-compose.yaml down --volumes --remove-orphans 2>/dev/null || true
    
    # 删除之前生成的文件
    sudo rm -rf crypto-config
    sudo rm -rf system-genesis-block
    sudo rm -rf channel-artifacts
    sudo rm -f mychannel.block
    sudo rm -f certificate.tar.gz
    
    # 清理Docker网络
    docker network prune -f
    
    # 清理未使用的卷
    docker volume prune -f
    
    print_info "网络清理完成"
}

# 生成加密材料
generate_crypto() {
    print_info "生成加密材料..."
    
    # 检查cryptogen工具
    if ! command -v cryptogen &> /dev/null; then
        print_warning "cryptogen工具未找到，使用Docker容器生成加密材料"
        docker run --rm -v $(pwd):/work -w /work \
            hyperledger/fabric-tools:2.5.12 \
            cryptogen generate --config=./crypto-config.yaml
    else
        cryptogen generate --config=./crypto-config.yaml
    fi
    
    if [ ! -d "crypto-config" ]; then
        print_error "加密材料生成失败"
        exit 1
    fi
    
    # 检查关键文件是否存在
    if [ ! -f "crypto-config/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt" ]; then
        print_error "Orderer TLS证书未生成"
        exit 1
    fi
    
    print_info "加密材料生成完成"
}

# 生成创世区块
generate_genesis() {
    print_info "生成创世区块..."
    
    mkdir -p system-genesis-block
    mkdir -p channel-artifacts
    
    # 检查configtxgen工具
    if ! command -v configtxgen &> /dev/null; then
        print_warning "configtxgen工具未找到，使用Docker容器生成创世区块"
        docker run --rm -v $(pwd):/work -w /work \
            hyperledger/fabric-tools:2.5.12 \
            configtxgen -profile CertOrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block
    else
        configtxgen -profile CertOrdererGenesis -channelID system-channel -outputBlock ./system-genesis-block/genesis.block
    fi
    
    if [ ! -f "system-genesis-block/genesis.block" ]; then
        print_error "创世区块生成失败"
        exit 1
    fi
    
    print_info "创世区块生成完成"
}

# 生成通道配置
generate_channel_artifacts() {
    print_info "生成通道配置..."
    
    # 生成通道配置交易
    if ! command -v configtxgen &> /dev/null; then
        docker run --rm -v $(pwd):/work -w /work \
            hyperledger/fabric-tools:2.5.12 \
            configtxgen -profile CertChannel -outputCreateChannelTx ./channel-artifacts/channel.tx -channelID mychannel
            
        # 生成锚节点配置
        docker run --rm -v $(pwd):/work -w /work \
            hyperledger/fabric-tools:2.5.12 \
            configtxgen -profile CertChannel -outputAnchorPeersUpdate ./channel-artifacts/CertOrgMSPanchors.tx -channelID mychannel -asOrg CertOrgMSP
            
        docker run --rm -v $(pwd):/work -w /work \
            hyperledger/fabric-tools:2.5.12 \
            configtxgen -profile CertChannel -outputAnchorPeersUpdate ./channel-artifacts/TestOrgMSPanchors.tx -channelID mychannel -asOrg TestOrgMSP
    else
        configtxgen -profile CertChannel -outputCreateChannelTx ./channel-artifacts/channel.tx -channelID mychannel
        configtxgen -profile CertChannel -outputAnchorPeersUpdate ./channel-artifacts/CertOrgMSPanchors.tx -channelID mychannel -asOrg CertOrgMSP
        configtxgen -profile CertChannel -outputAnchorPeersUpdate ./channel-artifacts/TestOrgMSPanchors.tx -channelID mychannel -asOrg TestOrgMSP
    fi
    
    print_info "通道配置生成完成"
}

# 启动网络
start_network() {
    print_info "启动Fabric网络..."
    
    # 启动网络
    docker-compose -f docker-compose.yaml up -d
    
    # 等待网络启动
    print_info "等待网络启动..."
    sleep 15
    
    # 检查容器状态
    print_info "检查容器状态..."
    docker-compose -f docker-compose.yaml ps
    
    print_info "Fabric网络启动完成"
}

# 创建通道
create_channel() {
    print_info "创建通道..."
    
    # 等待orderer完全启动
    sleep 10
    
    # 创建通道
    docker exec cli peer channel create -o orderer.example.com:7050 -c mychannel -f ./channel-artifacts/channel.tx --tls --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
    
    # 加入通道 - CertOrg peer0
    docker exec cli peer channel join -b mychannel.block
    
    # 加入通道 - CertOrg peer1
    docker exec -e CORE_PEER_ADDRESS=peer1.cert.example.com:8051 \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/cert.example.com/peers/peer1.cert.example.com/tls/ca.crt \
        cli peer channel join -b mychannel.block
    
    # 加入通道 - TestOrg peer0
    docker exec -e CORE_PEER_LOCALMSPID=TestOrgMSP \
        -e CORE_PEER_ADDRESS=peer0.test.example.com:9051 \
        -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/users/Admin@test.example.com/msp \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer0.test.example.com/tls/ca.crt \
        cli peer channel join -b mychannel.block
    
    # 加入通道 - TestOrg peer1
    docker exec -e CORE_PEER_LOCALMSPID=TestOrgMSP \
        -e CORE_PEER_ADDRESS=peer1.test.example.com:10051 \
        -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/users/Admin@test.example.com/msp \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer1.test.example.com/tls/ca.crt \
        cli peer channel join -b mychannel.block
    
    print_info "通道创建完成"
}

# 安装和实例化智能合约
deploy_chaincode() {
    print_info "部署智能合约..."
    
    # 打包智能合约
    docker exec cli peer lifecycle chaincode package certificate.tar.gz --path /opt/gopath/src/github.com/chaincode/certificate --lang golang --label certificate_1.0
    
    # 安装智能合约到所有peer
    print_info "安装智能合约到CertOrg peers..."
    docker exec cli peer lifecycle chaincode install certificate.tar.gz
    docker exec -e CORE_PEER_ADDRESS=peer1.cert.example.com:8051 cli peer lifecycle chaincode install certificate.tar.gz
    
    print_info "安装智能合约到TestOrg peers..."
    docker exec -e CORE_PEER_LOCALMSPID=TestOrgMSP \
        -e CORE_PEER_ADDRESS=peer0.test.example.com:9051 \
        -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/users/Admin@test.example.com/msp \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer0.test.example.com/tls/ca.crt \
        cli peer lifecycle chaincode install certificate.tar.gz
    
    docker exec -e CORE_PEER_LOCALMSPID=TestOrgMSP \
        -e CORE_PEER_ADDRESS=peer1.test.example.com:10051 \
        -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/users/Admin@test.example.com/msp \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer1.test.example.com/tls/ca.crt \
        cli peer lifecycle chaincode install certificate.tar.gz
    
    # 获取package ID
    PACKAGE_ID=$(docker exec cli peer lifecycle chaincode queryinstalled --output json | jq -r '.installed_chaincodes[0].package_id')
    print_info "Package ID: $PACKAGE_ID"
    
    if [ "$PACKAGE_ID" = "null" ] || [ -z "$PACKAGE_ID" ]; then
        print_error "无法获取Package ID"
        exit 1
    fi
    
    # 批准智能合约定义 - CertOrg
    print_info "CertOrg批准智能合约定义..."
    docker exec cli peer lifecycle chaincode approveformyorg \
        -o orderer.example.com:7050 \
        --channelID mychannel \
        --name certificate \
        --version 1.0 \
        --package-id $PACKAGE_ID \
        --sequence 1 \
        --tls \
        --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
    
    # 批准智能合约定义 - TestOrg
    print_info "TestOrg批准智能合约定义..."
    docker exec -e CORE_PEER_LOCALMSPID=TestOrgMSP \
        -e CORE_PEER_ADDRESS=peer0.test.example.com:9051 \
        -e CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/users/Admin@test.example.com/msp \
        -e CORE_PEER_TLS_ROOTCERT_FILE=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer0.test.example.com/tls/ca.crt \
        cli peer lifecycle chaincode approveformyorg \
        -o orderer.example.com:7050 \
        --channelID mychannel \
        --name certificate \
        --version 1.0 \
        --package-id $PACKAGE_ID \
        --sequence 1 \
        --tls \
        --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
    
    # 检查提交准备状态
    print_info "检查智能合约提交准备状态..."
    docker exec cli peer lifecycle chaincode checkcommitreadiness \
        --channelID mychannel \
        --name certificate \
        --version 1.0 \
        --sequence 1 \
        --tls \
        --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        --output json
    
    # 提交智能合约定义
    print_info "提交智能合约定义..."
    docker exec cli peer lifecycle chaincode commit \
        -o orderer.example.com:7050 \
        --channelID mychannel \
        --name certificate \
        --version 1.0 \
        --sequence 1 \
        --tls \
        --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        --peerAddresses peer0.cert.example.com:7051 \
        --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/cert.example.com/peers/peer0.cert.example.com/tls/ca.crt \
        --peerAddresses peer0.test.example.com:9051 \
        --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer0.test.example.com/tls/ca.crt
    
    # 初始化智能合约
    print_info "初始化智能合约..."
    docker exec cli peer chaincode invoke \
        -o orderer.example.com:7050 \
        --tls \
        --cafile /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
        -C mychannel \
        -n certificate \
        --peerAddresses peer0.cert.example.com:7051 \
        --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/cert.example.com/peers/peer0.cert.example.com/tls/ca.crt \
        --peerAddresses peer0.test.example.com:9051 \
        --tlsRootCertFiles /opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/test.example.com/peers/peer0.test.example.com/tls/ca.crt \
        -c '{"function":"InitLedger","Args":[]}'
    
    print_info "智能合约部署完成"
}

# 主函数
main() {
    print_info "开始启动Hyperledger Fabric网络..."
    
    check_prerequisites
    cleanup_network
    generate_crypto
    generate_genesis
    generate_channel_artifacts
    start_network
    create_channel
    deploy_chaincode
    
    print_info "==========================================="
    print_info "Hyperledger Fabric网络启动成功！"
    print_info "==========================================="
    print_info "网络信息："
    print_info "- Channel: mychannel"
    print_info "- Chaincode: certificate"
    print_info "- Orderer: orderer.example.com:7050"
    print_info "- CertOrg Peers: peer0.cert.example.com:7051, peer1.cert.example.com:8051"
    print_info "- TestOrg Peers: peer0.test.example.com:9051, peer1.test.example.com:10051"
    print_info "==========================================="
    print_info "可以开始启动后端服务了！"
}

# 执行主函数
main "$@"