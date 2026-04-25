Để thiết kế một entity `File` chung nhất trong Golang, có thể lưu trữ đa dạng metadata (Image, Video, Document...) dưới dạng `JSONB` trong PostgreSQL và dễ dàng marshal/unmarshal khi trả về API, bạn cần sử dụng một kiểu dữ liệu có thể linh hoạt giữ nguyên cấu trúc JSON chưa parse (như `json.RawMessage`) hoặc dạng `map[string]any`. 

Tuy nhiên, cách tối ưu và Strongly-typed nhất trong Go là sử dụng `json.RawMessage` kết hợp với việc implement 2 interface của `database/sql` là **`driver.Valuer`** (để lưu vào DB) và **`sql.Scanner`** (để đọc từ DB ra).

Dưới đây là blueprint chuẩn xác nhất cho bài toán này:

### 1. Định nghĩa Entity & Type JSONB Custom

Ta sẽ tạo một custom type `FileMetadata` dựa trên `json.RawMessage`. Nhờ vậy, khi trả về API bằng `gin` hoặc `fiber`, nó sẽ tự động được parse thành JSON object hợp lệ chứ không bị biến thành string.

```go
package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// FileMetadata là custom type xử lý JSONB cho cả Database và HTTP Response
type FileMetadata json.RawMessage

// Scan implements the sql.Scanner interface (Database -> Golang)
func (m *FileMetadata) Scan(value any) error {
	if value == nil {
		*m = FileMetadata("{}") // Default rỗng nếu null
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	*m = append((*m)[0:0], bytes...)
	return nil
}

// Value implements the driver.Valuer interface (Golang -> Database)
func (m FileMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil // Lưu null vào db nếu rỗng
	}
	return []byte(m), nil
}

// File Entity chung nhất
type File struct {
	ID        uuid.UUID    `json:"id" gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Name      string       `json:"name" gorm:"type:varchar(255);not null"`
	URL       string       `json:"url" gorm:"type:text;not null"`
	Size      int64        `json:"size" gorm:"not null"` // Kích thước (bytes)
	MimeType  string       `json:"mime_type" gorm:"type:varchar(100);not null"`
	Provider  string       `json:"provider" gorm:"type:varchar(50)"` // S3, GCS, Local...
	Metadata  FileMetadata `json:"metadata" gorm:"type:jsonb"`       // Cột ăn tiền
	CreatedAt time.Time    `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time    `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### 2. Thiết kế các Sub-struct cho Metadata

Bây giờ bạn định nghĩa các struct con cụ thể cho từng loại file. Vì ta dùng `json.RawMessage`, ta có thể trì hoãn (defer) việc parse JSON cho đến khi biết chính xác file đó thuộc `MimeType` nào.

```go
// Các struct con mô tả metadata chi tiết
type ImageMetadata struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	ColorSpace  string `json:"color_space,omitempty"`
	Compression string `json:"compression,omitempty"`
}

type VideoMetadata struct {
	Duration int    `json:"duration"` // Thời lượng (giây)
	Bitrate  int    `json:"bitrate"`
	FPS      int    `json:"fps"`
	Codec    string `json:"codec"`
}

type DocumentMetadata struct {
	Pages  int    `json:"pages"`
	Author string `json:"author,omitempty"`
}
```

### 3. Utility Methods (Cách sử dụng)

Để code gọn gàng, bạn nên viết thêm các helper methods gắn vào `File` entity để ép kiểu (unmarshal) từ `FileMetadata` ra struct con tương ứng một cách an toàn.

```go
// Lấy metadata của Image
func (f *File) ParseImageMeta() (*ImageMetadata, error) {
	if f.Metadata == nil {
		return nil, nil
	}
	var meta ImageMetadata
	err := json.Unmarshal(f.Metadata, &meta)
	return &meta, err
}

// Lấy metadata của Video
func (f *File) ParseVideoMeta() (*VideoMetadata, error) {
	if f.Metadata == nil {
		return nil, nil
	}
	var meta VideoMetadata
	err := json.Unmarshal(f.Metadata, &meta)
	return &meta, err
}

// Hàm helper để gán metadata từ struct bất kỳ vào File
func (f *File) SetMetadata(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	f.Metadata = FileMetadata(b)
	return nil
}
```

### 4. Ví dụ thực tế khi thao tác

**Khi insert vào Database:**
```go
imgMeta := ImageMetadata{Width: 1920, Height: 1080, ColorSpace: "sRGB"}
file := entity.File{
	Name:     "background.jpg",
	MimeType: "image/jpeg",
	Size:     1024500,
}
_ = file.SetMetadata(imgMeta)

// Lưu xuống Postgres (GORM / sqlx / pgx đều nhận diện được interface Valuer)
db.Create(&file) 
```

**Khi return API (Gin/Fiber):**
```go
// Lấy từ DB lên
var f entity.File
db.First(&f, "id = ?", someID)

// Trả thẳng ra API. json.RawMessage tự động render ra object, KHÔNG bị stringify.
return c.JSON(200, f)
```
*Kết quả JSON client nhận được sẽ rất clean:*
```json
{
  "id": "...",
  "name": "background.jpg",
  "mime_type": "image/jpeg",
  "metadata": {
    "width": 1920,
    "height": 1080,
    "color_space": "sRGB"
  }
}
```

### 5. Lưu ý khi Query trực tiếp JSONB trong Postgres

Vì bạn dùng Postgres, việc lưu vào `JSONB` cho phép bạn query trên các trường metadata rất dễ dàng. Ví dụ bạn muốn tìm tất cả các file ảnh có `width` > 1000:

```go
// GORM query với toán tử của JSONB (->>)
db.Where("mime_type LIKE ? AND (metadata->>'width')::int > ?", "image/%", 1000).Find(&files)
```

**Tóm lại:** Việc dùng `json.RawMessage` + implement `Scanner`/`Valuer` là pattern chuẩn mực nhất cho Go. Nó đảm bảo hiệu năng cao (zero-allocation khi không cần unmarshal các field sâu bên trong), decouple rõ ràng giữa DB format và Golang Struct, và giữ được tính đa hình hoàn hảo cho mọi loại file.