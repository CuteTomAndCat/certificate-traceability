#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 停止网络
stop_network() {
    print_info "停止Fabric网络..."
    
    # 停止所有容器
    docker-compose -f docker-compose.yaml down --volumes --remove-orphans
    
    # 删除生成的文件（可选）
    read -p "是否删除生成的加密材料和配置文件？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "删除生成的文件..."
        sudo rm -rf crypto-config
        sudo rm -rf system-genesis-block
        sudo rm -rf channel-artifacts
        sudo rm -f mychannel.block
        sudo rm -f certificate.tar.gz
    fi
    
    # 清理Docker资源
    print_info "清理Docker资源..."
    docker network prune -f
    docker volume prune -f
    
    print_info "Fabric网络已停止"
}

stop_network