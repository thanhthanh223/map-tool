package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"tool-map/services"

	oracle "github.com/godoes/gorm-oracle"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic recovered: %v\n", r)
		}
	}()

	fmt.Println("=== BẮT ĐẦU CHƯƠNG TRÌNH ===")
	fmt.Println("Debug: Chương trình đã bắt đầu chạy...")

	_ = godotenv.Load(".env")

	// Kết nối Oracle database
	db := connectDB()
	fmt.Println("Đã kết nối Oracle database")

	// Tạo OSM service với database
	osmService := services.NewOSMServiceWithDB(db)
	fmt.Println("Đã tạo OSM service")

	// Đọc danh sách relation IDs từ file id.txt
	idFile, err := os.ReadFile("id.txt")
	if err != nil {
		log.Fatalf("Lỗi khi đọc file id.txt: %v", err)
	}
	idLines := strings.Split(string(idFile), "\n")

	var relationIDs []int64
	for _, line := range idLines {
		line = strings.TrimSpace(line)
		if line == "" || line == "\r" {
			continue
		}
		var id int64
		_, err := fmt.Sscanf(line, "%d", &id)
		if err != nil {
			fmt.Printf("Bỏ qua dòng không hợp lệ: %s\n", line)
			continue
		}
		relationIDs = append(relationIDs, id)
	}

	for _, relationID := range relationIDs {
		fmt.Printf("\n------------------------------\n")
		fmt.Printf("Đang xử lý relation ID: %d\n", relationID)

		fmt.Println("Đang fetch và process dữ liệu OSM...")
		result, err := osmService.FetchAndProcessRelation(relationID)
		if err != nil {
			fmt.Printf("Lỗi khi xử lý dữ liệu OSM (ID %d): %v\n", relationID, err)
			continue
		}
		fmt.Println("Đã fetch và process dữ liệu OSM thành công")

		// Hiển thị kết quả JSON
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("KẾT QUẢ XỬ LÝ OSM DATA\n")
		fmt.Printf("%s\n", strings.Repeat("=", 60))

		// Nếu có provinces, thao tác thêm cho từng commune trong mỗi province
		if result.Administrative != nil {
			if provinces, exists := result.Administrative["provinces"]; exists && len(provinces) > 0 {
				for _, province := range provinces {
					if province.Boundary == "" {
						continue
					}
					name := province.Name
					adminLevel := province.AdminLevel
					fmt.Printf("Tìm thấy province: %s (admin_level: %d)\n", name, adminLevel)
					fmt.Println("Đang lấy boundary string từ kết quả province...")
					provinceBoundaryString := province.Boundary
					// Lưu province
					fmt.Println("\n=== LƯU DATABASE - PROVINCE ===")
					err = osmService.UpdateStringBoundaryToDatabase(name, adminLevel, provinceBoundaryString)
					if err != nil {
						fmt.Printf("Lỗi khi lưu province vào database: %v\n", err)
					} else {
						fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", name, adminLevel)
					}

					// Nếu có communes bên trong, gọi tiếp cho từng commune
					if communes, existsC := result.Administrative["communes"]; existsC && len(communes) > 0 {
						fmt.Printf("\nTìm thấy %d commune(s) trong province '%s', tiến hành xử lý từng commune\n", len(communes), name)
						for _, commune := range communes {
							if commune.Boundary == "" {
								continue
							}

							resultPX, errPX := osmService.FetchAndProcessRelation(commune.ID)
							if errPX != nil {
								fmt.Printf("Lỗi khi xử lý dữ liệu OSM (ID %d): %v\n", commune.ID, errPX)
								continue
							}
							communeBoundaryString := resultPX.Boundaries.JSONString
							err = osmService.UpdateStringBoundaryToDatabase(commune.Name, commune.AdminLevel, communeBoundaryString)
							if err != nil {
								fmt.Printf("Lỗi khi lưu commune vào database: %v\n", err)
							} else {
								fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", commune.Name, commune.AdminLevel)
							}
						}
					}
				}
			}
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
	}
	fmt.Println("=== KẾT THÚC CHƯƠNG TRÌNH ===")
}

func connectDB() *gorm.DB {
	host := os.Getenv("ORACLE_HOST")
	portStr := os.Getenv("ORACLE_PORT")
	service := os.Getenv("ORACLE_SERVICE")
	user := os.Getenv("ORACLE_USER")
	password := os.Getenv("ORACLE_PASSWORD")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("PORT không hợp lệ: %v", err)
	}

	// Dùng oracle.BuildUrl (giả sử bạn đang dùng thư viện hỗ trợ)
	url := oracle.BuildUrl(
		host,
		port,
		service,
		user,
		password,
		nil, // options nếu có
	)

	fmt.Printf("Connecting to Oracle with URL: %s\n", url)

	db, err := gorm.Open(oracle.Open(url), &gorm.Config{})
	if err != nil {
		log.Fatalf("Lỗi khi kết nối Oracle database: %v", err)
	}
	return db
}
