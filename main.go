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
	fmt.Println("Starting OSM processing...")
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
					if strings.Contains(province.Name, "Thành phố") || strings.Contains(province.Name, "Tỉnh") {
						province.Name = strings.ReplaceAll(province.Name, "Thành phố ", "")
						province.Name = strings.ReplaceAll(province.Name, "Tỉnh ", "")
						province.Name = strings.TrimSpace(province.Name)
					}
					name := province.Name
					adminLevel := province.AdminLevel
					fmt.Printf("Tìm thấy province: %s (admin_level: %d)\n", name, adminLevel)
					fmt.Println("Đang lấy boundary string từ kết quả province...")

					// Lấy boundary từ province
					LonCenter := result.CenterPoints[0].Lon
					LatCenter := result.CenterPoints[0].Lat
					maxLat := result.BasicInfo.Bounds.MaxLat
					minLat := result.BasicInfo.Bounds.MinLat
					maxLon := result.BasicInfo.Bounds.MaxLon
					minLon := result.BasicInfo.Bounds.MinLon

					// Xuất JSON cho province - 2 file riêng biệt
					fmt.Println("\n=== XUẤT JSON - PROVINCE ===")

					// Tạo polygon từ ways và nodes
					fmt.Println("\n=== TẠO POLYGON - PROVINCE ===")
					polygon, err := osmService.CreatePolygonFromWaysAndNodes(result.Ways, result.Nodes)
					if err != nil {
						fmt.Printf("Lỗi khi tạo polygon: %v\n", err)
					} else {
						fmt.Printf("Polygon tạo thành công với %d điểm\n", len(polygon))

						// Lưu polygon vào database
						fmt.Println("Đang lưu polygon vào database...")
						err = osmService.UpdatePolygonToDatabase(name, adminLevel, polygon)
						if err != nil {
							fmt.Printf("Lỗi khi lưu polygon vào database: %v\n", err)
						} else {
							fmt.Printf("Đã lưu polygon cho '%s'\n", name)
						}
					}

					// Lưu province vào database
					fmt.Println("\n=== LƯU DATABASE - PROVINCE ===")
					err = osmService.UpdateStringBoundaryToDatabase(name, adminLevel, maxLat, minLat, maxLon, minLon, LonCenter, LatCenter)
					if err != nil {
						fmt.Printf("Lỗi khi lưu province vào database: %v\n", err)
					} else {
						fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", name, adminLevel)
					}
				}
			}
		}

		for _, commune := range result.Relations {
			// Nếu là huyện thì skip
			if *commune.AdminLevel != 6 {
				continue
			}

			fmt.Printf("Tìm thấy commune: %s (admin_level: %d)\n", commune.Name, commune.AdminLevel)
			fmt.Println("Đang lấy boundary string từ kết quả commune...")

			communeDataResult, err := osmService.FetchAndProcessRelation(commune.ID)
			if err != nil {
				fmt.Printf("Lỗi khi lấy dữ liệu OSM (ID %d): %v\n", commune.ID, err)
				continue
			}

			// Lấy boundary từ commune
			maxLat := communeDataResult.BasicInfo.Bounds.MaxLat
			minLat := communeDataResult.BasicInfo.Bounds.MinLat
			maxLon := communeDataResult.BasicInfo.Bounds.MaxLon
			minLon := communeDataResult.BasicInfo.Bounds.MinLon

			var LonCenter float64
			var LatCenter float64
			if len(communeDataResult.CenterPoints) > 0 {
				LonCenter = communeDataResult.CenterPoints[0].Lon
				LatCenter = communeDataResult.CenterPoints[0].Lat
			} else {
				LonCenter = 0
				LatCenter = 0
			}
			// Lưu commune
			fmt.Println("\n=== LƯU DATABASE - COMMUNE ===")
			err = osmService.UpdateStringBoundaryToDatabase(commune.Name, *commune.AdminLevel, maxLat, minLat, maxLon, minLon, LonCenter, LatCenter)
			if err != nil {
				fmt.Printf("Lỗi khi lưu commune vào database: %v\n", err)
			} else {
				fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", commune.Name, commune.AdminLevel)
			}

			// Tạo polygon từ ways và nodes
			fmt.Println("\n=== TẠO POLYGON - COMMUNE ===")
			polygon, err := osmService.CreatePolygonFromWaysAndNodes(communeDataResult.Ways, communeDataResult.Nodes)
			if err != nil {
				fmt.Printf("Lỗi khi tạo polygon: %v\n", err)
			} else {
				fmt.Printf("Polygon tạo thành công với %d điểm\n", len(polygon))

				// Lưu polygon vào database
				fmt.Println("Đang lưu polygon vào database...")
				err = osmService.UpdatePolygonToDatabase(commune.Name, 6, polygon)
				if err != nil {
					fmt.Printf("Lỗi khi lưu polygon vào database: %v\n", err)
				} else {
					fmt.Printf("Đã lưu polygon cho '%s'\n", commune.Name)
				}
			}
		}

		fmt.Printf("\n=== HOÀN THÀNH XỬ LÝ ===\n")
		fmt.Printf("Đã xử lý thành công relation %d\n", relationID)
		if result != nil {
			if boundaries := result.Boundaries; boundaries != nil {
				fmt.Printf("- Tổng tọa độ: %d\n", boundaries.TotalCoordinates)
			}
		}
		if result != nil && result.Administrative != nil {
			fmt.Printf("- Tỉnh/thành phố: %d\n", len(result.Administrative["provinces"]))
			fmt.Printf("- Xã/phường: %d\n", len(result.Administrative["communes"]))
			fmt.Printf("- Nodes: %d\n", len(result.Nodes))
			fmt.Printf("- Ways: %d\n", len(result.Ways))
			fmt.Printf("- Relations: %d\n", len(result.Relations))
			fmt.Printf("- Center Points: %d\n", len(result.CenterPoints))

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
