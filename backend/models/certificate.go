package models

type Certificate struct {
	ID               string            `json:"id"`
	CertificateNo    string            `json:"certificateNo"`
	TestUnit         string            `json:"testUnit"`
	TestDate         string            `json:"testDate"`
	TestData         []TestDataItem    `json:"testData"`
	InspectionOrg    string            `json:"inspectionOrg"`
	Inspector        string            `json:"inspector"`
	Status           string            `json:"status"`
	IssuedDate       string            `json:"issuedDate"`
	ValidUntil       string            `json:"validUntil"`
	Hash             string            `json:"hash"`
	CreatedBy        string            `json:"createdBy"`
	CreatedAt        string            `json:"createdAt"`
	UpdatedAt        string            `json:"updatedAt"`
	TraceHistory     []TraceRecord     `json:"traceHistory"`
}

type TestDataItem struct {
	Parameter     string  `json:"parameter"`
	MeasuredValue float64 `json:"measuredValue"`
	Unit          string  `json:"unit"`
	Uncertainty   float64 `json:"uncertainty"`
	Method        string  `json:"method"`
	Equipment     string  `json:"equipment"`
}

type TraceRecord struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Operator  string `json:"operator"`
	Details   string `json:"details"`
}

type CreateCertificateRequest struct {
	CertificateNo string         `json:"certificateNo" binding:"required"`
	TestUnit      string         `json:"testUnit" binding:"required"`
	TestDate      string         `json:"testDate" binding:"required"`
	TestData      []TestDataItem `json:"testData" binding:"required"`
	InspectionOrg string         `json:"inspectionOrg" binding:"required"`
	Inspector     string         `json:"inspector" binding:"required"`
	ValidUntil    string         `json:"validUntil" binding:"required"`
	CreatedBy     string         `json:"createdBy" binding:"required"`
}

type UpdateCertificateRequest struct {
	CertificateNo string         `json:"certificateNo"`
	TestUnit      string         `json:"testUnit"`
	TestDate      string         `json:"testDate"`
	TestData      []TestDataItem `json:"testData"`
	InspectionOrg string         `json:"inspectionOrg"`
	Inspector     string         `json:"inspector"`
	ValidUntil    string         `json:"validUntil"`
	Operator      string         `json:"operator" binding:"required"`
}

type IssueCertificateRequest struct {
	Operator string `json:"operator" binding:"required"`
}

type RevokeCertificateRequest struct {
	Operator string `json:"operator" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

type QueryCertificatesRequest struct {
	TestUnit      string `form:"testUnit"`
	Status        string `form:"status"`
	InspectionOrg string `form:"inspectionOrg"`
}