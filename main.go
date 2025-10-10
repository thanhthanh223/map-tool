package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"tool-map/services"

	oracle "github.com/godoes/gorm-oracle"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== BẮT ĐẦU CHƯƠNG TRÌNH ===")

	// Kết nối Oracle database
	// db := connectDB()
	// fmt.Println("Đã kết nối Oracle database")

	// Tạo OSM service với database
	osmService := services.NewOSMServiceWithDB(nil)
	fmt.Println("Đã tạo OSM service")

	// Lấy relation 19282554 (Xã Vĩnh Hải)
	relationID := int64(19282554)
	fmt.Printf("Đang xử lý relation ID: %d\n", relationID)

	// Fetch và process dữ liệu OSM
	fmt.Println("Đang fetch và process dữ liệu OSM...")
	result, err := osmService.FetchAndProcessRelation(relationID)
	if err != nil {
		log.Fatalf("Lỗi khi xử lý dữ liệu OSM: %v", err)
	}
	fmt.Println("Đã fetch và process dữ liệu OSM thành công")

	// Hiển thị kết quả JSON
	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("KẾT QUẢ XỬ LÝ OSM DATA\n")
	fmt.Printf("%s\n", strings.Repeat("=", 60))

	// Lấy boundary string từ kết quả đã xử lý (không gọi API thêm)
	fmt.Println("Đang lấy boundary string từ kết quả...")
	boundaryString := osmService.GetBoundaryStringFromResult(result)

	// Hiển thị thông tin boundary string
	fmt.Printf("\n=== BOUNDARY STRING INFO ===\n")
	fmt.Printf("Boundary string length: %d characters\n", len(boundaryString))
	if len(boundaryString) > 0 {
		fmt.Printf("Boundary JSON string (first 200 chars):\n")
		if len(boundaryString) > 200 {
			fmt.Printf("%s...\n", boundaryString[:200])
		} else {
			fmt.Printf("%s\n", boundaryString)
		}
	} else {
		fmt.Println("Boundary string rỗng!")
	}

	// Lưu boundary string vào Oracle database
	fmt.Println("\n=== LƯU DATABASE ===")
	if len(boundaryString) > 0 {
		err = osmService.UpdateStringBoundaryToDatabase("Xã Vĩnh Hải", 6, boundaryString)
		if err != nil {
			fmt.Printf("Lỗi khi lưu vào database: %v\n", err)
		} else {
			fmt.Printf("Đã lưu boundary string cho 'Xã Vĩnh Hải' với level '6' (commune)\n")
		}
	} else {
		fmt.Println("Không có boundary string để lưu")
	}

	fmt.Printf("\n=== HOÀN THÀNH XỬ LÝ ===\n")
	fmt.Printf("Đã xử lý thành công relation %d\n", relationID)
	if result != nil && result.Boundaries != nil {
		fmt.Printf("- Tổng tọa độ: %d\n", result.Boundaries.TotalCoordinates)
	}
	if result != nil && result.Administrative != nil {
		fmt.Printf("- Tỉnh/thành phố: %d\n", len(result.Administrative["provinces"]))
		fmt.Printf("- Xã/phường: %d\n", len(result.Administrative["communes"]))

		// Hiển thị chi tiết các entities
		if len(result.Administrative["provinces"]) > 0 {
			fmt.Printf("\nCác tỉnh/thành phố:\n")
			for _, province := range result.Administrative["provinces"] {
				fmt.Printf("  - %s (ID: %d, AdminLevel: %d, CapitalLevel: %d)\n",
					province.Name, province.ID, province.AdminLevel, province.CapitalLevel)
			}
		}

		if len(result.Administrative["communes"]) > 0 {
			fmt.Printf("\nCác xã/phường:\n")
			for _, commune := range result.Administrative["communes"] {
				fmt.Printf("  - %s (ID: %d, AdminLevel: %d, CapitalLevel: %d)\n",
					commune.Name, commune.ID, commune.AdminLevel, commune.CapitalLevel)
			}
		}
	}

	fmt.Println("=== KẾT THÚC CHƯƠNG TRÌNH ===")
}

func connectDB() *gorm.DB {
	dsn := os.Getenv("dsn")

	db, err := gorm.Open(oracle.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect Oracle database: " + err.Error())
	}
	return db
}
