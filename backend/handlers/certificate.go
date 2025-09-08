package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"certificate-backend/fabric"
	"certificate-backend/models"
)

type CertificateHandler struct {
	fabricClient *fabric.FabricClient
}

func NewCertificateHandler(fabricClient *fabric.FabricClient) *CertificateHandler {
	return &CertificateHandler{
		fabricClient: fabricClient,
	}
}

// CreateCertificate 创建证书
func (h *CertificateHandler) CreateCertificate(c *gin.Context) {
	var req models.CreateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成证书ID
	certificateID := uuid.New().String()

	// 创建证书对象
	cert := models.Certificate{
		ID:            certificateID,
		CertificateNo: req.CertificateNo,
		TestUnit:      req.TestUnit,
		TestDate:      req.TestDate,
		TestData:      req.TestData,
		InspectionOrg: req.InspectionOrg,
		Inspector:     req.Inspector,
		ValidUntil:    req.ValidUntil,
		CreatedBy:     req.CreatedBy,
		Hash:          h.generateCertificateHash(req),
	}

	// 序列化证书数据
	certData, err := json.Marshal(cert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal certificate data"})
		return
	}

	// 调用智能合约创建证书
	_, err = h.fabricClient.SubmitTransaction("CreateCertificate", certificateID, string(certData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create certificate: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Certificate created successfully",
		"certificateId": certificateID,
	})
}

// GetCertificate 获取证书详情
func (h *CertificateHandler) GetCertificate(c *gin.Context) {
	id := c.Param("id")

	// 调用智能合约读取证书
	result, err := h.fabricClient.EvaluateTransaction("ReadCertificate", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}

	var cert models.Certificate
	if err := json.Unmarshal(result, &cert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal certificate data"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

// UpdateCertificate 更新证书
func (h *CertificateHandler) UpdateCertificate(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取现有证书
	existingResult, err := h.fabricClient.EvaluateTransaction("ReadCertificate", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}

	var existingCert models.Certificate
	if err := json.Unmarshal(existingResult, &existingCert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal existing certificate"})
		return
	}

	// 更新证书字段（只更新提供的字段）
	if req.CertificateNo != "" {
		existingCert.CertificateNo = req.CertificateNo
	}
	if req.TestUnit != "" {
		existingCert.TestUnit = req.TestUnit
	}
	if req.TestDate != "" {
		existingCert.TestDate = req.TestDate
	}
	if len(req.TestData) > 0 {
		existingCert.TestData = req.TestData
	}
	if req.InspectionOrg != "" {
		existingCert.InspectionOrg = req.InspectionOrg
	}
	if req.Inspector != "" {
		existingCert.Inspector = req.Inspector
	}
	if req.ValidUntil != "" {
		existingCert.ValidUntil = req.ValidUntil
	}

	// 重新生成哈希
	existingCert.Hash = h.generateCertificateHashFromCert(existingCert)

	// 序列化更新后的证书
	updatedCertData, err := json.Marshal(existingCert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal updated certificate data"})
		return
	}

	// 调用智能合约更新证书
	_, err = h.fabricClient.SubmitTransaction("UpdateCertificate", id, string(updatedCertData), req.Operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update certificate: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate updated successfully"})
}

// IssueCertificate 签发证书
func (h *CertificateHandler) IssueCertificate(c *gin.Context) {
	id := c.Param("id")
	var req models.IssueCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用智能合约签发证书
	_, err := h.fabricClient.SubmitTransaction("IssueCertificate", id, req.Operator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to issue certificate: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate issued successfully"})
}

// RevokeCertificate 撤销证书
func (h *CertificateHandler) RevokeCertificate(c *gin.Context) {
	id := c.Param("id")
	var req models.RevokeCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用智能合约撤销证书
	_, err := h.fabricClient.SubmitTransaction("RevokeCertificate", id, req.Operator, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to revoke certificate: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate revoked successfully"})
}

// GetCertificateHistory 获取证书历史记录
func (h *CertificateHandler) GetCertificateHistory(c *gin.Context) {
	id := c.Param("id")

	// 调用智能合约获取证书历史
	result, err := h.fabricClient.EvaluateTransaction("GetCertificateHistory", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get certificate history: %v", err)})
		return
	}

	var history []interface{}
	if err := json.Unmarshal(result, &history); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal history data"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// QueryCertificates 查询证书
func (h *CertificateHandler) QueryCertificates(c *gin.Context) {
	var req models.QueryCertificatesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var result []byte
	var err error

	// 根据查询参数调用不同的智能合约函数
	if req.TestUnit != "" {
		result, err = h.fabricClient.EvaluateTransaction("QueryCertificatesByTestUnit", req.TestUnit)
	} else if req.Status != "" {
		result, err = h.fabricClient.EvaluateTransaction("QueryCertificatesByStatus", req.Status)
	} else if req.InspectionOrg != "" {
		result, err = h.fabricClient.EvaluateTransaction("QueryCertificatesByInspectionOrg", req.InspectionOrg)
	} else {
		result, err = h.fabricClient.EvaluateTransaction("GetAllCertificates")
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query certificates: %v", err)})
		return
	}

	var certificates []models.Certificate
	if err := json.Unmarshal(result, &certificates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal certificates data"})
		return
	}

	c.JSON(http.StatusOK, certificates)
}

// generateCertificateHash 生成证书哈希
func (h *CertificateHandler) generateCertificateHash(req models.CreateCertificateRequest) string {
	data := fmt.Sprintf("%s%s%s%s%s%s%s",
		req.CertificateNo,
		req.TestUnit,
		req.TestDate,
		req.InspectionOrg,
		req.Inspector,
		req.ValidUntil,
		time.Now().Format(time.RFC3339))
	
	// 添加测试数据到哈希计算
	for _, testData := range req.TestData {
		data += fmt.Sprintf("%s%f%s%f%s%s",
			testData.Parameter,
			testData.MeasuredValue,
			testData.Unit,
			testData.Uncertainty,
			testData.Method,
			testData.Equipment)
	}

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// generateCertificateHashFromCert 从证书对象生成哈希
func (h *CertificateHandler) generateCertificateHashFromCert(cert models.Certificate) string {
	data := fmt.Sprintf("%s%s%s%s%s%s%s",
		cert.CertificateNo,
		cert.TestUnit,
		cert.TestDate,
		cert.InspectionOrg,
		cert.Inspector,
		cert.ValidUntil,
		cert.CreatedAt)
	
	// 添加测试数据到哈希计算
	for _, testData := range cert.TestData {
		data += fmt.Sprintf("%s%f%s%f%s%s",
			testData.Parameter,
			testData.MeasuredValue,
			testData.Unit,
			testData.Uncertainty,
			testData.Method,
			testData.Equipment)
	}

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}