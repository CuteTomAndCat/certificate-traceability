package fabric

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type FabricClient struct {
	gateway    *client.Gateway
	network    *client.Network
	contract   *client.Contract
	connection *grpc.ClientConn
}

func NewFabricClient() (*FabricClient, error) {
	// 证书和密钥路径（需要根据实际路径调整）
	certPath := "/root/go/src/certificate-traceability/network/crypto-config/peerOrganizations/cert.example.com/users/User1@cert.example.com/msp/signcerts/User1@cert.example.com-cert.pem"
	keyPath := "/root/go/src/certificate-traceability/network/crypto-config/peerOrganizations/cert.example.com/users/User1@cert.example.com/msp/keystore/"
	tlsCertPath := "/root/go/src/certificate-traceability/network/crypto-config/peerOrganizations/cert.example.com/peers/peer0.cert.example.com/tls/ca.crt"

	// 连接到peer
	conn, err := newGrpcConnection(tlsCertPath, "peer0.cert.example.com:7051")
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	// 创建身份
	id, err := newIdentity(certPath, "CertOrgMSP")
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create identity: %v", err)
	}

	// 创建签名
	sign, err := newSign(keyPath)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create sign: %v", err)
	}

	// 创建网关
	gateway, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to connect to gateway: %v", err)
	}

	// 获取网络
	network := gateway.GetNetwork("mychannel")

	// 获取合约
	contract := network.GetContract("certificate")

	return &FabricClient{
		gateway:    gateway,
		network:    network,
		contract:   contract,
		connection: conn,
	}, nil
}

func (fc *FabricClient) Close() {
	if fc.gateway != nil {
		fc.gateway.Close()
	}
	if fc.connection != nil {
		fc.connection.Close()
	}
}

func (fc *FabricClient) SubmitTransaction(name string, args ...string) ([]byte, error) {
	result, err := fc.contract.SubmitTransaction(name, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction %s: %v", name, err)
	}
	return result, nil
}

func (fc *FabricClient) EvaluateTransaction(name string, args ...string) ([]byte, error) {
	result, err := fc.contract.EvaluateTransaction(name, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate transaction %s: %v", name, err)
	}
	return result, nil
}

func newGrpcConnection(tlsCertPath, peerEndpoint string) (*grpc.ClientConn, error) {
	certificate, err := loadCertificate(tlsCertPath)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, "peer0.cert.example.com")

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	return connection, nil
}

func newIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certificate, err := loadCertificate(certPath)
	if err != nil {
		return nil, err
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, err
	}

	return id, nil
}

func newSign(keyPath string) (identity.Sign, error) {
	files, err := ioutil.ReadDir(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key directory: %v", err)
	}

	var privateKeyPEM []byte
	for _, file := range files {
		if !file.IsDir() {
			privateKeyPEM, err = ioutil.ReadFile(filepath.Join(keyPath, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file: %v", err)
			}
			break
		}
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, err
	}

	return sign, nil
}

func loadCertificate(filename string) (*x509.Certificate, error) {
	certificatePEM, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	return identity.CertificateFromPEM(certificatePEM)
}