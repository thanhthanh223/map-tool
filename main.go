package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"tool-map/repositories"
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

	// Khởi tạo Redis (nếu được cấu hình qua env)
	services.InitRedis()

	// Kết nối Oracle database
	db := connectDB()
	fmt.Println("Đã kết nối Oracle database")

	// repo
	dmTTRepo := repositories.NewDmTTRepository(db)
	dmPXRepo := repositories.NewDmPhuongXaRepository(db)

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

		var provinceName string
		// Nếu có provinces, thao tác thêm cho từng commune trong m	ỗi province
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
					provinceName = name
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

					// Tạo polygon từ ways và nodes
					fmt.Println("\n=== TẠO POLYGON - PROVINCE ===")
					polygons, err := osmService.CreatePolygonFromWaysAndNodes(result.Ways, result.Nodes)
					if err != nil {
						fmt.Printf("Lỗi khi tạo polygon: %v\n", err)
					} else {
						fmt.Printf("Tạo thành công %d polygon(s)\n", len(polygons))
						// Tạo mảng để lưu các URL MinIO sau khi upload polygons
						var polygonUrls []string

						// Xử lý từng polygon
						for i, polygon := range polygons {
							fmt.Printf("Processing polygon %d with %d points\n", i+1, len(polygon))

							// Upload polygon data lên MinIO
							fmt.Println("Đang upload polygon lên MinIO...")
							polygonJSON, err := json.Marshal(polygon)
							if err != nil {
								fmt.Printf("Lỗi khi marshal polygon JSON: %v\n", err)
								continue
							}

							// Tạo tên file khác nhau cho mỗi polygon
							var objectName string
							if len(polygons) == 1 {
								objectName = fmt.Sprintf("provinces_%d_polygon.txt", relationID)
							} else {
								objectName = fmt.Sprintf("provinces_%d_polygon_%d.txt", relationID, i+1)
							}

							uploadPolygonURL, err := services.UploadPolygonData(polygonJSON, objectName)
							if err != nil {
								fmt.Printf("Lỗi khi upload polygon lên MinIO: %v\n", err)
								continue
							}

							// Thêm url vào mảng lưu trữ
							polygonUrls = append(polygonUrls, uploadPolygonURL)
						}

						// Convert mảng các url thành string dạng JSON
						polygonUrlsJSON, err := json.Marshal(polygonUrls)
						if err != nil {
							fmt.Printf("Lỗi khi convert polygon URLs array sang string: %v\n", err)
						} else {
							// Lưu string mảng các url vào database bằng hàm UpdatePolygonToDatabase
							fmt.Println("Đang lưu mảng polygon URLs vào database...")
							err = osmService.UpdatePolygonToDatabase(name, adminLevel, string(polygonUrlsJSON), "")
							if err != nil {
								fmt.Printf("Lỗi khi lưu mảng polygon URLs vào database: %v\n", err)
							} else {
								fmt.Printf("Đã lưu mảng polygon URLs cho '%s'\n", name)
							}
						}
					}

					// Lưu province vào database
					fmt.Println("\n=== LƯU DATABASE - PROVINCE ===")
					err = osmService.UpdateStringBoundaryToDatabase(name, adminLevel, maxLat, minLat, maxLon, minLon, LonCenter, LatCenter, "")
					if err != nil {
						fmt.Printf("Lỗi khi lưu province vào database: %v\n", err)
					} else {
						fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", name, adminLevel)
					}
				}
			}
		}

		TinhThanhInDb, err := dmTTRepo.GetByName(provinceName)
		if err != nil {
			fmt.Printf("Lỗi khi lấy dữ liệu tỉnh/thành phố từ database: %v\n", err)
			return
		}
		if TinhThanhInDb == nil {
			fmt.Printf("Không tìm thấy tỉnh/thành phố '%s' trong database\n", provinceName)
			return
		}

		for _, commune := range result.Relations {
			// Nếu là huyện thì skip
			if *commune.AdminLevel != 6 {
				continue
			}

			fmt.Printf("Tìm thấy commune: %s (admin_level: %d) trong database\n", commune.Name, commune.AdminLevel)
			fmt.Println("Đang lấy boundary string từ kết quả commune...")

			px, err := dmPXRepo.GetByName(commune.Name, TinhThanhInDb.MaTT)
			if err != nil || px == nil || px.MaPhuongXa == "" {
				fmt.Printf("Không tìm thấy phường xã '%s' trong database\n", commune.Name)
				continue
			}

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
			err = osmService.UpdateStringBoundaryToDatabase(commune.Name, *commune.AdminLevel, maxLat, minLat, maxLon, minLon, LonCenter, LatCenter, TinhThanhInDb.MaTT)
			if err != nil {
				fmt.Printf("Lỗi khi lưu commune vào database: %v\n", err)
			} else {
				fmt.Printf("Đã lưu boundary string cho '%s' với level '%d'\n", commune.Name, commune.AdminLevel)
			}

			// Tạo polygon từ ways và nodes
			fmt.Println("\n=== TẠO POLYGON - COMMUNE ===")
			polygons, err := osmService.CreatePolygonFromWaysAndNodes(communeDataResult.Ways, communeDataResult.Nodes)
			if err != nil {
				fmt.Printf("Lỗi khi tạo polygon: %v\n", err)
			} else {
				fmt.Printf("Tạo thành công %d polygon(s) cho commune\n", len(polygons))

				// Xử lý từng polygon
				for i, polygon := range polygons {
					fmt.Printf("Processing commune polygon %d with %d points\n", i+1, len(polygon))

					// Lưu polygon vào database (chỉ polygon đầu tiên)
					if i == 0 {
						fmt.Println("Đang lưu polygon chính vào database...")
						polygonJSON, err := json.Marshal(polygon)
						if err != nil {
							fmt.Printf("Lỗi khi marshal polygon JSON: %v\n", err)
						} else {
							// lưu polygon vào database là data cho phường xã
							err = osmService.UpdatePolygonToDatabase(commune.Name, 6, string(polygonJSON), TinhThanhInDb.MaTT)
							if err != nil {
								fmt.Printf("Lỗi khi lưu polygon vào database: %v\n", err)
							} else {
								fmt.Printf("Đã lưu polygon chính cho '%s'\n", commune.Name)
							}
						}
					} else {
						fmt.Printf("Commune polygon %d được tạo nhưng không lưu vào DB (chỉ lưu polygon chính)\n", i+1)
					}
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

	fmt.Println("Đang cập nhật tọa độ trung tâm của xã/phường...")
	err = osmService.UpdateLatLonCenterForPhuongXa()
	if err != nil {
		fmt.Printf("Lỗi khi cập nhật tọa độ trung tâm của xã/phường: %v\n", err)
	} else {
		fmt.Println("Đã cập nhật tọa độ trung tâm của xã/phường thành công")
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
