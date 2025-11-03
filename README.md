# Tool Map - OSM Data Integration

Dự án này tích hợp dữ liệu OpenStreetMap (OSM) để lấy thông tin biên giới hành chính và lưu vào database dưới dạng JSON.

## Tính năng chính

### 1. Gọi OSM API
- Lấy dữ liệu relation từ OpenStreetMap API
- Hỗ trợ lấy đầy đủ thông tin nodes, ways, và relations

### 2. Phân loại theo Capital Level
- **Capital level 4**: Tỉnh/Thành phố (DmTT)
- **Capital level 6**: Xã/Phường (DmPhuongXa)
- Tự động phân loại dựa trên tag `capital` trong OSM data

### 3. Service Layer Architecture
- **OSMService**: Xử lý logic chính cho OSM data
- **FetchAndProcessRelation()**: Lấy và xử lý dữ liệu relation
- **SaveToDatabase()**: Lưu dữ liệu đã xử lý vào database
- **Structured Results**: Trả về kết quả có cấu trúc rõ ràng

### 4. Xử lý tọa độ biên giới
- Trích xuất tọa độ từ administrative boundaries
- Chuyển đổi thành JSON array format
- Lưu vào database dưới dạng JSON string

### 3. Cấu trúc dữ liệu

#### DmPhuongXa (Phường/Xã)
```go
type DmPhuongXa struct {
    MaPhuongXa     string  `json:"maPhuongXa" gorm:"column:MA_PHUONG_XA;primarykey"`
    TenPhuongXa    string  `json:"tenPhuongXa" gorm:"column:TEN_PHUONG_XA"`
    TenPhuongXaEn  string  `json:"tenPhuongXaEn" gorm:"column:TEN_PHUONG_XA_EN"`
    ToaDoBienGioi  *string `json:"toaDoBienGioi" gorm:"column:TOA_DO_BIEN_GIOI;type:json"`
    // ... các trường khác
}
```

#### Coordinate (Tọa độ)
```go
type Coordinate struct {
    Lat float64 `json:"lat"`
    Lon float64 `json:"lon"`
}
```

## Cách sử dụng

### 1. Sử dụng Service Layer (Khuyến nghị)

```go
// Tạo OSM service
osmService := services.NewOSMService()

// Lấy relation 19283382 (Xã Tân Minh)
relationID := int64(19283382)
result, err := osmService.FetchAndProcessRelation(relationID)
if err != nil {
    log.Fatal(err)
}

// Lưu vào database
err = osmService.SaveToDatabase(result)
if err != nil {
    log.Fatal(err)
}
```

### 2. Sử dụng trực tiếp API Client

```go
// Tạo OSM API client
client := models.NewOSMApiClient()

// Lấy relation 19283382 (Xã Tân Minh)
relationID := int64(19283382)
osm, err := client.FetchRelationFull(relationID)
if err != nil {
    log.Fatal(err)
}
```

### 2. Trích xuất tọa độ biên giới

```go
// Lấy tọa độ biên giới
coordinates, err := osm.GetBoundaryCoordinates()
if err != nil {
    log.Fatal(err)
}

// Chuyển đổi thành JSON string
jsonString, err := models.EncodeCoordinatesToJSON(coordinates)
if err != nil {
    log.Fatal(err)
}
```

### 3. Lưu vào database

```go
phuongXa := entities.DmPhuongXa{
    MaPhuongXa:    "XA_TAN_MINH",
    TenPhuongXa:   "Xã Tân Minh",
    TenPhuongXaEn: "Tân Minh Commune",
    ToaDoBienGioi: &jsonString,
    // ... các trường khác
}

// Lưu vào database bằng GORM
db.Create(&phuongXa)
```

### 4. Đọc từ database

```go
var phuongXa entities.DmPhuongXa
db.Where("ma_phuong_xa = ?", "XA_TAN_MINH").First(&phuongXa)

// Decode JSON string về coordinates
coordinates, err := models.DecodeCoordinatesFromJSON(*phuongXa.ToaDoBienGioi)
if err != nil {
    log.Fatal(err)
}

// Sử dụng coordinates
for _, coord := range coordinates {
    fmt.Printf("Lat: %.6f, Lon: %.6f\n", coord.Lat, coord.Lon)
}
```

## Ví dụ JSON output

Dữ liệu tọa độ được lưu dưới dạng JSON array:

```json
[
  {"lat": 20.6909521, "lon": 106.5132050},
  {"lat": 20.7088896, "lon": 106.5148401},
  {"lat": 20.7068509, "lon": 106.5153615},
  {"lat": 20.7061564, "lon": 106.5154731},
  {"lat": 20.7054419, "lon": 106.5154645}
]
```

## API Endpoints sử dụng

- **OSM Relation API**: `https://www.openstreetmap.org/api/0.6/relation/{id}/full`
- **OSM Way API**: `https://www.openstreetmap.org/api/0.6/way/{id}/full`
- **OSM Node API**: `https://www.openstreetmap.org/api/0.6/node/{id}`

## Chạy chương trình

```bash
# Chạy chương trình chính
go run main.go

# Chạy ví dụ JSON usage (uncomment main function trong example_json_usage.go)
go run example_json_usage.go
```

## Lợi ích của JSON format

1. **Dễ đọc và debug**: JSON format dễ đọc hơn binary
2. **SQL JSON functions**: Có thể sử dụng các hàm JSON của SQL
3. **Frontend integration**: Dễ dàng sử dụng với JavaScript/frontend
4. **Flexibility**: Có thể thêm metadata vào JSON nếu cần

## Database Schema

```sql
CREATE TABLE DM_PHUONG_XA (
    MA_PHUONG_XA VARCHAR(50) PRIMARY KEY,
    TEN_PHUONG_XA VARCHAR(255),
    TEN_PHUONG_XA_EN VARCHAR(255),
    TOA_DO_BIEN_GIOI JSON,
    TRUC_THUOC_TINH VARCHAR(100),
    -- ... các trường khác
);
```

# MinIO Integration

## Cấu hình MinIO

Tạo file `.env` với thông tin MinIO:

```env
# MinIO Configuration
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY_ID=minioadmin
MINIO_SECRET_ACCESS_KEY=minioadmin
MINIO_BUCKET_NAME=osm-data
MINIO_USE_SSL=false
```

## Chạy MinIO với Docker

```bash
docker run -p 9000:9000 -p 9001:9001 \
  --name minio \
  -e "MINIO_ACCESS_KEY=minioadmin" \
  -e "MINIO_SECRET_KEY=minioadmin" \
  minio/minio server /data --console-address ":9001"
```

## Chức năng đã tích hợp

1. **Kết nối MinIO**: Tự động kết nối khi khởi động app
2. **Upload Polygon Data**: 
   - Provinces: `provinces/{name}_{id}_polygon.json`
   - Communes: `communes/{name}_{id}_polygon.json`
3. **Bucket Management**: Tự động tạo bucket `osm-data` nếu chưa có

## API MinIO Service

```go
// Upload file
minioService.UploadFile("bucket", "object", "/path/to/file")

// Upload bytes
minioService.UploadBytes("bucket", "object", []byte("data"), "application/json")

// Download file
minioService.DownloadFile("bucket", "object", "/path/to/save")

// Get object data
data, err := minioService.GetObject("bucket", "object")

// Generate presigned URL
url, err := minioService.GetPresignedURL("bucket", "object", 24*time.Hour)
```

## Truy cập MinIO Console

- URL: http://localhost:9001
- Username: minioadmin
- Password: minioadmin

