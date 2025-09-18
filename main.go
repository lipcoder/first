package main

import (
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "github.com/gin-gonic/gin"
    "net/http"
    "log"
)

// Spot 模型
type Spot struct {
    ID             uint   `gorm:"primaryKey"`
    Name           string
    Description    string
    Ticket         string
    Transport      string
    RecommendCount int
    ImageURL       string
}

func main() {
    // 连接数据库
    db, err := gorm.Open(sqlite.Open("spots.db"), &gorm.Config{})
    if err != nil {
        log.Fatal("无法连接数据库:", err)
    }
    // 自动迁移
    db.AutoMigrate(&Spot{})

    // 如果数据库为空，插入两条示例数据
    var count int64
    db.Model(&Spot{}).Count(&count)
    if count == 0 {
        db.Create(&Spot{Name: "西湖", Description: "杭州著名景点", Ticket: "免费", Transport: "公交可达", RecommendCount: 0})
        db.Create(&Spot{Name: "黄山", Description: "中国名山", Ticket: "门票230元", Transport: "高铁+大巴", RecommendCount: 0})
    }

    // ---------------- 主程序（8080） ----------------
    r1 := gin.Default()
    r1.LoadHTMLGlob("templates/*.html")

    // 首页：列出景
    r1.GET("/", func(c *gin.Context) {
        var spots []Spot
        db.Order("recommend_count desc, id asc").Find(&spots)
        c.HTML(http.StatusOK, "index.html", gin.H{
            "spots": spots,
        })
    })

    // 添加新景点
    r1.POST("/add", func(c *gin.Context) {
        name := c.PostForm("name")
        description := c.PostForm("description")
        ticket := c.PostForm("ticket")
        transport := c.PostForm("transport")
	imageURL := c.PostForm("imageurl")

        db.Create(&Spot{
		Name: name, 
		Description: description, 
		Ticket: ticket, 
		Transport: transport, 
		ImageURL: imageURL,
		RecommendCount: 0})
        c.Redirect(http.StatusFound, "/")
    })

    // 推荐功能
    r1.POST("/recommend/:id", func(c *gin.Context) {
        id := c.Param("id")
        var spot Spot
        if err := db.First(&spot, id).Error; err == nil {
            spot.RecommendCount++
            db.Save(&spot)
        }
        c.Redirect(http.StatusFound, "/")
    })

    // 删除功能
    r1.POST("/delete/:id", func(c *gin.Context) {
        id := c.Param("id")
        db.Delete(&Spot{}, id)
        c.Redirect(http.StatusFound, "/")
    })

    // 更新景点
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
        c.String(http.StatusNotFound, "未找到ID为 %s 的景点", id)
        return
    }

    // 更新字段
    db.Model(&spot).Updates(Spot{
        Name:        name,
        Description: description,
        Ticket:      ticket,
        Transport:   transport,
        ImageURL:    imageURL,
    })

    c.Redirect(http.StatusFound, "/")
})

	// 搜索景点
r1.GET("/search", func(c *gin.Context) {
    query := c.Query("q") // 获取搜索关键词
    var spots []Spot

    // 如果没有关键词就返回全部
    if query == "" {
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

	r1.POST("/batchdelete", func(c *gin.Context) {
    ids := c.PostFormArray("ids")
    if len(ids) > 0 {
        db.Where("id IN ?", ids).Delete(&Spot{})
    }
    c.Redirect(http.StatusFound, "/")
})

	
    // 可以加搜索、查看等功能
    go func() {
        if err := r1.Run(":8080"); err != nil {
            log.Fatal("主程序启动失败:", err)
        }
    }()

    // ---------------- 静态HTML（8081） ----------------
    r2 := gin.Default()
    // 如果你只有一个HTML，用StaticFile
    r2.StaticFile("/", "./static/another.html")

    if err := r2.Run(":8081"); err != nil {
        log.Fatal("静态HTML服务启动失败:", err)
    }
}

