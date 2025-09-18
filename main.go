package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ==================== 数据模型定义 ====================

// Spot 模型（对应数据库中的景点表）
// gorm 标签 `primaryKey` 表示 ID 为主键，自增
type Spot struct {
	ID             uint   `gorm:"primaryKey"` // 景点ID，主键
	Name           string // 景点名称
	Description    string // 景点描述
	Ticket         string // 门票信息
	Transport      string // 交通信息
	RecommendCount int    // 推荐次数
	ImageURL       string // 图片URL
}

func main() {
	// ==================== 1. 连接数据库 ====================
	// 打开/创建 SQLite 数据库文件（spots.db）
	db, err := gorm.Open(sqlite.Open("spots.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("无法连接数据库:", err)
	}

	// 根据模型自动迁移数据库结构（不存在表就建表，添加缺失列）
	db.AutoMigrate(&Spot{})

	// 如果表为空，插入两条示例数据（初始化用）
	var count int64
	db.Model(&Spot{}).Count(&count)
	if count == 0 {
		db.Create(&Spot{
			Name:           "西湖",
			Description:    "杭州著名景点",
			Ticket:         "免费",
			Transport:      "公交可达",
			RecommendCount: 0,
		})
		db.Create(&Spot{
			Name:           "黄山",
			Description:    "中国名山",
			Ticket:         "门票230元",
			Transport:      "高铁+大巴",
			RecommendCount: 0,
		})
	}

	// ==================== 2. Gin 主程序（端口 8080） ====================
	// 创建 Gin 引擎，加载模板
	r1 := gin.Default()
	r1.LoadHTMLGlob("templates/*.html")

	// ---------- 首页：列出所有景点 ----------
	r1.GET("/", func(c *gin.Context) {
		var spots []Spot
		// 按推荐次数降序、ID升序排序
		db.Order("recommend_count desc, id asc").Find(&spots)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"spots": spots, // 模板可用 {{range .spots}} ... {{end}}
		})
	})

	// ---------- 添加新景点 ----------
	r1.POST("/add", func(c *gin.Context) {
		// 取表单字段
		name := c.PostForm("name")
		description := c.PostForm("description")
		ticket := c.PostForm("ticket")
		transport := c.PostForm("transport")
		imageURL := c.PostForm("imageurl")

		// 插入数据库
		db.Create(&Spot{
			Name:           name,
			Description:    description,
			Ticket:         ticket,
			Transport:      transport,
			ImageURL:       imageURL,
			RecommendCount: 0, // 新增景点推荐数初始为0
		})

		// 插入后重定向回首页
		c.Redirect(http.StatusFound, "/")
	})

	// ---------- 推荐景点（推荐次数 +1） ----------
	r1.POST("/recommend/:id", func(c *gin.Context) {
		id := c.Param("id") // URL路径参数，如 /recommend/3

		var spot Spot
		// 根据主键查询（注意：这里是字符串ID，GORM可自动转换）
		if err := db.First(&spot, id).Error; err == nil {
			// 找到则推荐次数+1，再保存回数据库
			spot.RecommendCount++
			db.Save(&spot)
		}
		// 不论是否成功，都重定向回首页
		c.Redirect(http.StatusFound, "/")
	})

	// ---------- 删除景点 ----------
	r1.POST("/delete/:id", func(c *gin.Context) {
		id := c.Param("id")
		// 根据ID删除记录
		db.Delete(&Spot{}, id)
		c.Redirect(http.StatusFound, "/")
	})

	// ---------- 更新景点信息 ----------
	r1.POST("/update/:id", func(c *gin.Context) {
		id := c.Param("id")

		// 取表单字段
		name := c.PostForm("name")
		description := c.PostForm("description")
		ticket := c.PostForm("ticket")
		transport := c.PostForm("transport")
		imageURL := c.PostForm("imageurl")

		// 找到对应的景点
		var spot Spot
		if err := db.First(&spot, id).Error; err != nil {
			// 没找到直接返回404
			c.String(http.StatusNotFound, "未找到ID为 %s 的景点", id)
			return
		}

		// 更新字段
		// 注意：Updates(Spot{}) 用struct会跳过零值（空字符串不会更新）
		db.Model(&spot).Updates(Spot{
			Name:        name,
			Description: description,
			Ticket:      ticket,
			Transport:   transport,
			ImageURL:    imageURL,
		})

		c.Redirect(http.StatusFound, "/")
	})

	// ---------- 搜索景点 ----------
	r1.GET("/search", func(c *gin.Context) {
		query := c.Query("q") // 获取搜索关键词（GET参数q=）

		var spots []Spot
		if query == "" {
			// 没关键词：返回全部
			db.Order("recommend_count desc, id asc").Find(&spots)
		} else {
			// 按名称或描述模糊搜索
			db.Where("name LIKE ? OR description LIKE ?", "%"+query+"%", "%"+query+"%").
				Order("recommend_count desc, id asc").Find(&spots)
		}

		c.HTML(http.StatusOK, "index.html", gin.H{
			"spots": spots,
		})
	})

	// ---------- 批量删除景点 ----------
	r1.POST("/batchdelete", func(c *gin.Context) {
		// 获取多个ID（表单checkbox name=ids）
		ids := c.PostFormArray("ids")
		if len(ids) > 0 {
			// WHERE id IN (...)
			db.Where("id IN ?", ids).Delete(&Spot{})
		}
		c.Redirect(http.StatusFound, "/")
	})

	// ---------- 启动主服务（8080端口） ----------
	// 因为后面还要再启动一个服务，所以这里放在goroutine里
	go func() {
		if err := r1.Run(":8080"); err != nil {
			log.Fatal("主程序启动失败:", err)
		}
	}()

	// ==================== 3. 第二个Gin实例（静态HTML，端口8081） ====================
	r2 := gin.Default()
	// 如果只有一个静态HTML，可以直接用StaticFile映射根路径
	r2.StaticFile("/", "./static/another.html")

	// 启动第二个服务（阻塞）
	if err := r2.Run(":8081"); err != nil {
		log.Fatal("静态HTML服务启动失败:", err)
	}
}
