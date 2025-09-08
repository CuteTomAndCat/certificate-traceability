package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Certificate struct {
	ID               string            `json:"id"`
	CertificateNo    string            `json:"certificateNo"`
	TestUnit         string            `json:"testUnit"`        // 送检单位
	TestDate         string            `json:"testDate"`        // 测试日期
	TestData         []TestDataItem    `json:"testData"`        // 测试数据
	InspectionOrg    string            `json:"inspectionOrg"`   // 检验机构
	Inspector        string            `json:"inspector"`       // 检验员
	Status           string            `json:"status"`          // 证书状态：draft, issued, revoked
	IssuedDate       string            `json:"issuedDate"`      // 签发日期
	ValidUntil       string            `json:"validUntil"`      // 有效期至
	Hash             string            `json:"hash"`            // 证书内容哈希
	CreatedBy        string            `json:"createdBy"`       // 创建者
	CreatedAt        string            `json:"createdAt"`       // 创建时间
	UpdatedAt        string            `json:"updatedAt"`       // 更新时间
	TraceHistory     []TraceRecord     `json:"traceHistory"`    // 溯源记录
}

type TestDataItem struct {
	Parameter    string  `json:"parameter"`    // 测试参数
	MeasuredValue float64 `json:"measuredValue"` // 测量值
	Unit         string  `json:"unit"`         // 单位
	Uncertainty  float64 `json:"uncertainty"`  // 不确定度
	Method       string  `json:"method"`       // 测试方法
	Equipment    string  `json:"equipment"`    // 测试设备
}

type TraceRecord struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Operator  string `json:"operator"`
	Details   string `json:"details"`
}

// InitLedger 初始化账本
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// 可以在这里初始化一些示例数据
	return nil
}

// CreateCertificate 创建证书
func (s *SmartContract) CreateCertificate(ctx contractapi.TransactionContextInterface, id string, certificateData string) error {
	exists, err := s.CertificateExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("certificate %s already exists", id)
	}

	var cert Certificate
	err = json.Unmarshal([]byte(certificateData), &cert)
	if err != nil {
		return fmt.Errorf("failed to unmarshal certificate data: %v", err)
	}

	cert.ID = id
	cert.Status = "draft"
	cert.CreatedAt = time.Now().Format(time.RFC3339)
	cert.UpdatedAt = time.Now().Format(time.RFC3339)

	// 添加创建记录到溯源历史
	traceRecord := TraceRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "CREATED",
		Operator:  cert.CreatedBy,
		Details:   "Certificate created",
	}
	cert.TraceHistory = append(cert.TraceHistory, traceRecord)

	certificateJSON, err := json.Marshal(cert)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, certificateJSON)
}

// IssueCertificate 签发证书
func (s *SmartContract) IssueCertificate(ctx contractapi.TransactionContextInterface, id string, operator string) error {
	cert, err := s.ReadCertificate(ctx, id)
	if err != nil {
		return err
	}

	if cert.Status != "draft" {
		return fmt.Errorf("certificate %s is not in draft status", id)
	}

	cert.Status = "issued"
	cert.IssuedDate = time.Now().Format(time.RFC3339)
	cert.UpdatedAt = time.Now().Format(time.RFC3339)

	// 添加签发记录到溯源历史
	traceRecord := TraceRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "ISSUED",
		Operator:  operator,
		Details:   "Certificate issued",
	}
	cert.TraceHistory = append(cert.TraceHistory, traceRecord)

	certificateJSON, err := json.Marshal(cert)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, certificateJSON)
}

// RevokeCertificate 撤销证书
func (s *SmartContract) RevokeCertificate(ctx contractapi.TransactionContextInterface, id string, operator string, reason string) error {
	cert, err := s.ReadCertificate(ctx, id)
	if err != nil {
		return err
	}

	if cert.Status == "revoked" {
		return fmt.Errorf("certificate %s is already revoked", id)
	}

	cert.Status = "revoked"
	cert.UpdatedAt = time.Now().Format(time.RFC3339)

	// 添加撤销记录到溯源历史
	traceRecord := TraceRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "REVOKED",
		Operator:  operator,
		Details:   fmt.Sprintf("Certificate revoked. Reason: %s", reason),
	}
	cert.TraceHistory = append(cert.TraceHistory, traceRecord)

	certificateJSON, err := json.Marshal(cert)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, certificateJSON)
}

// ReadCertificate 读取证书
func (s *SmartContract) ReadCertificate(ctx contractapi.TransactionContextInterface, id string) (*Certificate, error) {
	certificateJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate %s: %v", id, err)
	}
	if certificateJSON == nil {
		return nil, fmt.Errorf("certificate %s does not exist", id)
	}

	var cert Certificate
	err = json.Unmarshal(certificateJSON, &cert)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// UpdateCertificate 更新证书（仅草稿状态允许）
func (s *SmartContract) UpdateCertificate(ctx contractapi.TransactionContextInterface, id string, certificateData string, operator string) error {
	cert, err := s.ReadCertificate(ctx, id)
	if err != nil {
		return err
	}

	if cert.Status != "draft" {
		return fmt.Errorf("cannot update certificate %s: only draft certificates can be updated", id)
	}

	var updatedCert Certificate
	err = json.Unmarshal([]byte(certificateData), &updatedCert)
	if err != nil {
		return fmt.Errorf("failed to unmarshal certificate data: %v", err)
	}

	// 保留原有的元数据
	updatedCert.ID = cert.ID
	updatedCert.Status = cert.Status
	updatedCert.CreatedBy = cert.CreatedBy
	updatedCert.CreatedAt = cert.CreatedAt
	updatedCert.UpdatedAt = time.Now().Format(time.RFC3339)
	updatedCert.TraceHistory = cert.TraceHistory

	// 添加更新记录到溯源历史
	traceRecord := TraceRecord{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "UPDATED",
		Operator:  operator,
		Details:   "Certificate updated",
	}
	updatedCert.TraceHistory = append(updatedCert.TraceHistory, traceRecord)

	certificateJSON, err := json.Marshal(updatedCert)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, certificateJSON)
}

// GetCertificateHistory 获取证书历史记录
func (s *SmartContract) GetCertificateHistory(ctx contractapi.TransactionContextInterface, id string) ([]HistoryQueryResult, error) {
	resultsIterator, err := ctx.GetStub().GetHistoryForKey(id)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var records []HistoryQueryResult
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var cert Certificate
		if len(response.Value) > 0 {
			err = json.Unmarshal(response.Value, &cert)
			if err != nil {
				return nil, err
			}
		}

		record := HistoryQueryResult{
			TxId:      response.TxId,
			Timestamp: time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String(),
			IsDelete:  response.IsDelete,
			Value:     cert,
		}
		records = append(records, record)
	}

	return records, nil
}

type HistoryQueryResult struct {
	TxId      string      `json:"txId"`
	Timestamp string      `json:"timestamp"`
	IsDelete  bool        `json:"isDelete"`
	Value     Certificate `json:"value"`
}

// QueryCertificatesByTestUnit 按送检单位查询证书
func (s *SmartContract) QueryCertificatesByTestUnit(ctx contractapi.TransactionContextInterface, testUnit string) ([]*Certificate, error) {
	queryString := fmt.Sprintf(`{"selector":{"testUnit":"%s"}}`, testUnit)
	return s.getQueryResultForQueryString(ctx, queryString)
}

// QueryCertificatesByStatus 按状态查询证书
func (s *SmartContract) QueryCertificatesByStatus(ctx contractapi.TransactionContextInterface, status string) ([]*Certificate, error) {
	queryString := fmt.Sprintf(`{"selector":{"status":"%s"}}`, status)
	return s.getQueryResultForQueryString(ctx, queryString)
}

// QueryCertificatesByInspectionOrg 按检验机构查询证书
func (s *SmartContract) QueryCertificatesByInspectionOrg(ctx contractapi.TransactionContextInterface, inspectionOrg string) ([]*Certificate, error) {
	queryString := fmt.Sprintf(`{"selector":{"inspectionOrg":"%s"}}`, inspectionOrg)
	return s.getQueryResultForQueryString(ctx, queryString)
}

// getQueryResultForQueryString 执行富查询
func (s *SmartContract) getQueryResultForQueryString(ctx contractapi.TransactionContextInterface, queryString string) ([]*Certificate, error) {
	resultsIterator, err := ctx.GetStub().GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var certificates []*Certificate
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var cert Certificate
		err = json.Unmarshal(queryResponse.Value, &cert)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, &cert)
	}

	return certificates, nil
}

// CertificateExists 检查证书是否存在
func (s *SmartContract) CertificateExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	certificateJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read certificate %s: %v", id, err)
	}

	return certificateJSON != nil, nil
}

// GetAllCertificates 获取所有证书
func (s *SmartContract) GetAllCertificates(ctx contractapi.TransactionContextInterface) ([]*Certificate, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var certificates []*Certificate
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var cert Certificate
		err = json.Unmarshal(queryResponse.Value, &cert)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, &cert)
	}

	return certificates, nil
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating certificate chaincode: %v", err)
		return
	}

	if err := assetChaincode.Start(); err != nil {
		fmt.Printf("Error starting certificate chaincode: %v", err)
	}
}