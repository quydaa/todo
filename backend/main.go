package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// struct Note này sẽ ánh xạ trực tiếp tới bản notes
type Note struct {
	Id           int        `json:"id" gorm:"column:id;"`                       // struct tags bắt buộc phải được bao quanh bởi backtick "`"
	DetailedNote string     `json:"detailed_note" gorm:"column:detailed_note;"` // json: chuyển từ json -> truct, struct-> json
	UserId       int        `json:"user_id" gorm:"column:user_id;"`             // gorm: ánh xạ trường UserId tới cột user_id trong db
	CreatedAt    *time.Time `json:"created_at" gorm:"column:created_at;"`       // có thể dùng Auto Migration của GORM để tạo bảng từ struct này
	UpdatedAt    *time.Time `json:"updated_at" gorm:"column:updated_at;"`
} // phần này là phần xác định cấu trúc trường nào trong api ánh xạ hay truyền tới cột nào trong db; xác định trong json trả về có cái gì; xem dữ liệu gửi đi, nhận về có đúng định dạng không

func (Note) TableName() string { return "notes" } // xác định bẳng tương ứng mà struct ánh xạ tới trong db là gì. Câu hỏi: nếu liên đới nhiều bảng thì thế nào?
// (Note): tên struct; string: kiểu trả về tên bảng; return "notes": trả về tên bảng là notes
func main() {
	dsn := "root:@tcp(127.0.0.1:3306)/todo?charset=utf8mb4&parseTime=True&loc=Local" // thông tin để kết nối với db
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})                            // khỏi tạo liên kết với db bằng thư viện gorm
	// biến err này là biến chứa lỗi

	if err != nil {
		log.Fatalln("Cannot connect to MySQL:", err)
	} // nếu err không bằng nil -> có lỗi -> in thông báo kết nối thất bại

	log.Println("Connected to MySQL:", db) // nếu err ko chứa lỗi -> kết nối thành công

	router := gin.Default() // khoiẻ chạy router trong framework Gin. ( trong bài này dùng 2 cái thư viện là gin với gorm)

	/*  middleware -  là một hàm trung gian chạy trước hoặc sau handler chạy. Nó có tác dụng xử lý request và response trước
	    khi nó đến tay handler, như:
		 Logging	Ghi log lại mỗi lần có request.
	     Authentication	Kiểm tra token, quyền truy cập.
	     CORS	Cho phép frontend khác domain truy cập.
	     Gzip	Nén dữ liệu trả về để nhẹ hơn.
	     Rate limiting	Giới hạn số request một user có thể gửi trong 1 phút.
	     Recover	Bắt lỗi panic để server không crash.

		 Xử lý trước request:	Kiểm tra đăng nhập, ghi log, nén dữ liệu.
		 Xử lý sau response:	Thêm headers, ghi thời gian xử lý.
		 Ngăn chặn request không hợp lệ:	Chặn request thiếu token, sai IP.
		 Thêm tính năng chung:	CORS, rate limiting, caching.

		 Có nhiều loại middleware, nhưng bên dưới dùng là CORS - Cho phép frontend khác domain truy cập.
		 Bởi vì khi mở file index.html bằng live serve thì nó vhạy ở 5500, mà main.go thì nó ở 8080.
		 thành ra sẽ bị lỗi Access to fetch at 'http://localhost:8080/v1/notes' from origin 'http://127.0.0.1:5500' has been blocked
		 by CORS policy: No 'Access-Control-Allow-Origin' header is present on the requested resource và không gọi api được -> không hiện
		 được ở frontend. Nên bên dưới phải thêm CORS middleware */
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")            // Access-Control-Allow-Origin: Chỉ định rõ origin frontend được phép goi api
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS") //Access-Control-Allow-Methods: Các HTTP methods được phép
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")     //Access-Control-Allow-Headers: Các headers được phép gửi từ frontend.
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")                        //Access-Control-Allow-Credentials: Cho phép gửi cookie/credentials (khi dùng authentication).
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")                                 // Cache preflight request 24h: Thời gian cache (giây) cho preflight request (tránh gửi lặp).

		if c.Request.Method == "OPTIONS" { /*ví dụ mình đang ở 5500, gửi 1 request tới api đang ở 8080. Thì nó sẽ gửi 1 cái request kiểu OPTIONS
			để xin phép được gọi api hay là hỏi có đc gọi api không.*/
			c.AbortWithStatus(http.StatusNoContent) // nếu đúng là OPTIONS thì trả về NoContent rồi dừng xử lý, tương tự với cho phép truy cập api

			return // thoát khỏi middleware
		}

		c.Next() // chuyển request tới handler nếu ko phải OPTIONS (tức là đường đi bình thường, trường hợp bất thường là nằm khác cổng nên mới có OPTIONS)
	})

	v1 := router.Group("/v1") // tạo một nhóm các route có chung tiền tố  là /v1
	{
		v1.POST("/notes", createNote(db)) //Định nghĩa một API endpoint cho phương thức POST đến đường dẫn /v1/notes.
		//khi gửi request POST /v1/notes → hàm createNote(db) sẽ xử lý.
		//createNote(db) hàm createNote với db là tham số được truyền vào
		/*createNote(db) không phải gọi ngay hàm xử lý, mà là gọi một "factory function"(giống một cái vỏ bọc) – tức là
		   chạy nó rồi nó trả về một hàm handler, và chính handler đó mới được gán vào route.
		   Tại sao phải lòng vòng thế? vì Gin không hỗ trợ handler có 2 tham số, chỉ chấp nhận 1 tham số duy nhất: c *gin.Context,
		   nên dùng vòng ngoài xử lý tham số db, rồi hàm đc tạo ra xử lý request
		   createNote(db) được gọi trước → trả về một hàm handler.
		   Sau đó Gin dùng handler này để xử lý request /notes.
		   - db là biến chứa kết nối tới cơ sở dữ liệu (database).
		   Cụ thể hơn trong app Go dùng GORM, db là đối tượng kiểu *gorm.DB – nó cho phép bạn:
			Tạo dữ liệu mới: db.Create(&note)
			Truy vấn: db.First(&note, id)
			Cập nhật: db.Save(&note)
			Xoá: db.Delete(&note)*/
		v1.GET("/notes", getListOfNotes(db))
		v1.GET("/notes/:id", readNoteById(db))
		v1.PUT("/notes/:id", editNoteById(db))
		v1.DELETE("/notes/:id", deleteNoteById(db))
	}

	// Thêm route kiểm tra server hoạt động ổn không
	router.GET("/health", func(c *gin.Context) { /* khi có GET request gửi đến, nó chạy func(c *gin.Context) và trả về "status": "ok".
		vậy làm sao nó biết server die chưa? nếu nó die rồi thì sao mà nhận request, không nhân
		request thì ko in ra "status": "ok" => chỉ cần còn in ra "status": "ok" thì nó còn sống.*/
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	log.Println("Starting server on :8080") // in ra server đang chạy ở cổng 8080
	router.Run(":8080")                     //khởi động server ở cổng 8080
	//  phải đặt router.Run(":8080") ở cuối vì code sau dòng router.Run() sẽ ko đc thực thi
	// localhost:8080 → localhost là domain, 8080 là port
	// 127.0.0.1:5500 → 127.0.0.1 là IP/domain, 5500 là port

}

// [Các hàm createNote, readNoteById, getListOfNotes, editNoteById, deleteNoteById ]

func createNote(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dataNote Note

		if err := c.ShouldBind(&dataNote); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// preprocess detailed_note - trim all spaces
		dataNote.DetailedNote = strings.TrimSpace(dataNote.DetailedNote)

		if dataNote.DetailedNote == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "detailed_note cannot be blank"})
			return
		}

		if err := db.Create(&dataNote).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": dataNote.Id})
	}
}

func readNoteById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var dataNote Note

		id, err := strconv.Atoi(c.Param("id"))

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Where("id = ?", id).First(&dataNote).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": dataNote})
	}
}

func getListOfNotes(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type DataPaging struct {
			Page  int   `json:"page" form:"page"`
			Limit int   `json:"limit" form:"limit"`
			Total int64 `json:"total" form:"-"`
		}

		var paging DataPaging

		if err := c.ShouldBind(&paging); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if paging.Page <= 0 {
			paging.Page = 1
		}

		if paging.Limit <= 0 {
			paging.Limit = 10
		}

		offset := (paging.Page - 1) * paging.Limit

		var result []Note

		if err := db.Table(Note{}.TableName()).
			Count(&paging.Total).
			Offset(offset).
			Order("id desc").
			Find(&result).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func editNoteById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var dataNote Note

		if err := c.ShouldBind(&dataNote); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Where("id = ?", id).Updates(&dataNote).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": true})
	}
}

func deleteNoteById(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Table(Note{}.TableName()).
			Where("id = ?", id).
			Delete(nil).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": true})
	}
}
