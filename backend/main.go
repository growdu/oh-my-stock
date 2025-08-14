package main

import (
	"log"

	"oh-my-stock/config"
	"oh-my-stock/controllers"
	_ "oh-my-stock/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title 股票信息 API
// @version 1.0
// @description 股票基础信息管理 RESTful API
// @BasePath  /api/v1
// @host 192.168.3.99:3003
func main() {
	// 加载 JSON 配置
	config.LoadConfig("config.json")

	// 初始化数据库
	config.InitDB()

	r := gin.Default()

	// CORS 配置
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type"},
	}))

	// 统一 API 版本
	// 路由分组
	v1 := r.Group("/api/v1")
	//stockRoutes := r.Group("/api/v1/stocks")
	{
		stockRoutes := v1.Group("/stocks")
		stockRoutes.GET("", controllers.GetStocks)
		stockRoutes.GET("/:id", controllers.GetStockByID)
		stockRoutes.POST("", controllers.CreateStock)
		stockRoutes.PUT("/:id", controllers.UpdateStock)
		stockRoutes.DELETE("/:id", controllers.DeleteStock)

		stockRoutes.GET("/symbol/:symbol", controllers.GetStockBySymbol)
		stockRoutes.DELETE("/symbol/:symbol", controllers.DeleteStockBySymbol)

		stockDaily := v1.Group("/stock-daily-data")
		{
			stockDaily.GET("", controllers.GetAllStockDailyData)
			stockDaily.GET("/:symbol", controllers.GetStockDailyData)
			stockDaily.POST("", controllers.CreateStockDailyData)
			stockDaily.DELETE("/:symbol", controllers.DeleteStockDailyData)
		}

	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Println("🚀 服务启动，监听端口 3003")

	r.Run(":3003")
}
