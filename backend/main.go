package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"certificate-backend/handlers"
	"certificate-backend/fabric"
)

func main() {
	// 初始化Fabric连接
	fabricClient, err := fabric.NewFabricClient()
	if err != nil {
		log.Fatalf("Failed to initialize Fabric client: %v", err)
	}
	defer fabricClient.Close()

	// 创建Gin路由器
	r := gin.Default()

	// 添加CORS中间件
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	// 初始化处理器
	handler := handlers.NewCertificateHandler(fabricClient)

	// API路由
	api := r.Group("/api/v1")
	{
		// 证书相关路由
		api.POST("/certificates", handler.CreateCertificate)
		api.GET("/certificates/:id", handler.GetCertificate)
		api.PUT("/certificates/:id", handler.UpdateCertificate)
		api.POST("/certificates/:id/issue", handler.IssueCertificate)
		api.POST("/certificates/:id/revoke", handler.RevokeCertificate)
		api.GET("/certificates/:id/history", handler.GetCertificateHistory)
		api.GET("/certificates", handler.QueryCertificates)
	}

	// 启动服务器
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}