// oh-my-stock 后端入口
// @title oh-my-stock API
// @version 1.0
// @description 个人股票分析系统 RESTful API
// @BasePath /api/v1
package main

import (
	"log"
	"os"
	"time"

	"oh-my-stock/config"
	"oh-my-stock/controllers"
	_ "oh-my-stock/docs" //nolint:unused
	"oh-my-stock/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files" //nolint
	ginSwagger "github.com/swaggo/gin-swagger" //nolint
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}
	config.LoadConfig(configPath)
	config.InitDB()

	r := gin.Default()

	// CORS
	corsOrigin := config.GetFrontOrigin()
	if corsOrigin == "" {
		corsOrigin = "*"
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{corsOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-User-Id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: corsOrigin != "*",
		MaxAge:           12 * time.Hour,
	}))

	v1 := r.Group("/api/v1")

	// ============ 公开路由（不需要 JWT）============
	v1.POST("/user/register", controllers.Register)
	v1.POST("/user/login", controllers.Login)

	// ============ 用户域（需要 JWT）============
	user := v1.Group("/user", middleware.JWTAuth())
	{
		user.POST("/favorites", controllers.AddFavorite)
		user.GET("/favorites", controllers.GetFavorites)
		user.DELETE("/favorites/:id", controllers.DeleteFavorite)
		user.DELETE("/favorites/symbol/:symbol", controllers.DeleteFavoriteBySymbol)

		user.POST("/rules", controllers.AddRule)
		user.GET("/rules", controllers.GetRules)
		user.PUT("/rules/:id", controllers.UpdateRule)
		user.DELETE("/rules/:id", controllers.DeleteRule)
		user.POST("/rules/:id/run", controllers.RunRule)
		user.POST("/rules/preview", controllers.PreviewRule)
	}

	// ============ 股票域（公开）============
	stock := v1.Group("/stocks")
	{
		stock.GET("/:id", controllers.GetStockByID)
		stock.POST("", controllers.CreateStock)
		stock.PUT("/:id", controllers.UpdateStock)
		stock.GET("/symbol/:symbol", controllers.GetStockBySymbol)
		stock.GET("/history", controllers.GetStockHistory)
		stock.GET("/info", controllers.GetStockHistoryInfo)
		stock.GET("/list", controllers.GetStockList)
		stock.GET("/search", controllers.SearchStocks)
		stock.GET("/hot", controllers.GetHotStocks)
	}

	stockDaily := v1.Group("/stock-daily-data")
	{
		stockDaily.GET("", controllers.GetAllStockDailyData)
		stockDaily.GET("/:symbol", controllers.GetStockDailyData)
		stockDaily.POST("", controllers.CreateStockDailyData)
	}

	indicator := v1.Group("/stock-indicators")
	{
		indicator.POST("", controllers.CreateStockIndicator)
		indicator.GET("", controllers.GetStockIndicators)
		indicator.GET("/:id", controllers.GetStockIndicatorByID)
		indicator.GET("/symbol/:symbol", controllers.GetStockIndicatorBySymbolAndDate)
		indicator.PUT("/:id", controllers.UpdateStockIndicator)
	}

	flowAll := v1.Group("/stock-money-flow-all")
	{
		flowAll.POST("", controllers.CreateStockMoneyFlowAll)
		flowAll.GET("", controllers.GetStockMoneyFlowAlls)
		flowAll.GET("/:id", controllers.GetStockMoneyFlowAllByID)
		flowAll.GET("/symbol/:symbol", controllers.GetStockMoneyFlowAllBySymbolAndDate)
		flowAll.PUT("/:id", controllers.UpdateStockMoneyFlowAll)
		flowAll.DELETE("/:id", controllers.DeleteStockMoneyFlowAll)
	}

	flow := v1.Group("/stock-money-flow")
	{
		flow.POST("", controllers.CreateStockMoneyFlow)
		flow.GET("", controllers.GetStockMoneyFlows)
		flow.GET("/:id", controllers.GetStockMoneyFlowByID)
		flow.GET("/symbol/:symbol", controllers.GetStockMoneyFlowBySymbolAndDate)
		flow.PUT("/:id", controllers.UpdateStockMoneyFlowV1)
		flow.DELETE("/:id", controllers.DeleteStockMoneyFlowV1)
	}

	target := v1.Group("/target-stocks")
	{
		target.GET("", controllers.ListTargetStocks)
	}

	// Swagger / health
	// offline: disabled
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	host := config.GetServerHost()
	port := config.GetServerPort()
	addr := host + ":" + port
	log.Printf("🚀 服务启动，监听 %s (所有接口)", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
