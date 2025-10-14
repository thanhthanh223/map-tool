package services

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"tool-map/entities"
	"tool-map/models"
	"tool-map/repositories"

	"gorm.io/gorm"
)

type OSMServiceInterface interface {
	FetchAndProcessRelation(relationID int64) (*models.OSMProcessingResult, error)
	GetBoundaryStringFromResult(result *models.OSMProcessingResult) string
	UpdateStringBoundaryToDatabase(id string, level int, boundaryString, wayAddress string, lonCenter, latCenter float64) error
	CreatePolygonFromWaysAndNodes(osm *models.OSM) ([][]float64, error)
	UpdatePolygonToDatabase(id string, level int, polygonData [][]float64) error
	FindCommuneByCoordinate(provinceCode string, lat, lon float64) (*entities.DmPhuongXa, error)
}
type OSMService struct {
	client         *models.OSMApiClient
	dmTTRepo       repositories.DmTTRepositoryInterface
	dmPhuongXaRepo repositories.DmPhuongXaRepositoryInterface
}

// WayCoordinates represents a way with its coordinates
type WayCoordinates struct {
	ID      int64
	Coords  [][]float64 // [lat, lon] pairs
	NodeIDs []int64
}

// Connection represents a connection between ways
type Connection struct {
	WayID   int64
	IsStart bool
	Coords  [][]float64
}

// NewOSMService creates a new OSM service without database
func NewOSMService() *OSMService {
	return &OSMService{
		client: models.NewOSMApiClient(),
	}
}

// NewOSMServiceWithDB creates a new OSM service with database repositories
func NewOSMServiceWithDB(db *gorm.DB) *OSMService {
	return &OSMService{
		client:         models.NewOSMApiClient(),
		dmTTRepo:       repositories.NewDmTTRepository(db),
		dmPhuongXaRepo: repositories.NewDmPhuongXaRepository(db),
	}
}

// FetchAndProcessRelation fetches OSM relation data and processes it
func (s *OSMService) FetchAndProcessRelation(relationID int64) (*models.OSMProcessingResult, error) {
	fmt.Printf("Gọi OSM API để lấy dữ liệu cho relation %d...\n", relationID)

	osm, err := s.client.FetchRelationFull(relationID)
	if err != nil {
		return nil, fmt.Errorf("không thể lấy dữ liệu từ OSM API: %w", err)
	}

	fmt.Printf("OSM Data Information (from API):\n")
	fmt.Printf("Version: %s\n", osm.Version)
	fmt.Printf("Generator: %s\n", osm.Generator)
	fmt.Printf("Total Nodes: %d\n", len(osm.Nodes))
	fmt.Printf("Total Ways: %d\n", len(osm.Ways))
	fmt.Printf("Total Relations: %d\n", len(osm.Relations))

	return s.processOSMData(osm)
}

// processOSMData processes OSM data and returns structured result
func (s *OSMService) processOSMData(osm *models.OSM) (*models.OSMProcessingResult, error) {
	// Basic info
	basicInfo := &models.BasicOSMInfo{
		Version:    osm.Version,
		Generator:  osm.Generator,
		TotalNodes: len(osm.Nodes),
		TotalWays:  len(osm.Ways),
		TotalRels:  len(osm.Relations),
	}

	// Get bounds
	minLat, maxLat, minLon, maxLon, hasData := osm.GetBounds()
	if hasData {
		basicInfo.Bounds = &models.Bounds{
			MinLat:  minLat,
			MaxLat:  maxLat,
			MinLon:  minLon,
			MaxLon:  maxLon,
			HasData: hasData,
		}
	}

	// Process boundary coordinates
	boundaryData, err := s.processBoundaryData(osm)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi xử lý boundary data: %w", err)
	}

	// Process administrative entities
	administrativeData, err := s.processAdministrativeEntities(osm)
	if err != nil {
		return nil, fmt.Errorf("lỗi khi xử lý administrative entities: %w", err)
	}

	// Get capital level statistics
	capitalStats := s.getCapitalLevelStats(osm)

	// Convert OSM Ways to WayAddress format
	var ways []models.WayAddress
	for _, way := range osm.Ways {
		ways = append(ways, way.ToWayAddress())
	}

	// Convert OSM Nodes to Address format
	var nodes []models.Address
	var centerPoints []models.AdministrativeCenter
	for _, node := range osm.Nodes {
		nodes = append(nodes, node.ToAddress())

		// Logic: node có capital tag, place tag, hoặc population tag = center point
		if node.IsCenterPoint() {
			centerPoints = append(centerPoints, node.ToCenterPoint())
		}
	}

	// Convert OSM Relations to RelationInfo format
	var relations []models.RelationInfo
	for _, relation := range osm.Relations {
		relations = append(relations, relation.ToRelationInfo())
	}

	return &models.OSMProcessingResult{
		BasicInfo:       basicInfo,
		Boundaries:      boundaryData,
		Administrative:  administrativeData,
		CapitalStats:    capitalStats,
		JSONCoordinates: boundaryData.JSONString,
		Ways:            ways,
		Nodes:           nodes,
		CenterPoints:    centerPoints,
		Relations:       relations,
	}, nil
}

// processBoundaryData processes boundary coordinate data
func (s *OSMService) processBoundaryData(osm *models.OSM) (*models.BoundaryData, error) {
	fmt.Printf("\nProcessing boundary coordinates...\n")

	coordinates, err := osm.GetBoundaryCoordinates()
	if err != nil {
		return nil, fmt.Errorf("failed to get boundary coordinates: %w", err)
	}

	fmt.Printf("Found %d boundary coordinates\n", len(coordinates))

	// Encode coordinates to JSON string
	jsonString, err := models.EncodeCoordinatesToJSON(coordinates)
	if err != nil {
		return nil, fmt.Errorf("failed to encode coordinates to JSON: %w", err)
	}

	fmt.Printf("JSON string length: %d characters\n", len(jsonString))

	// Decode back to verify
	decodedCoords, err := models.DecodeCoordinatesFromJSON(jsonString)
	if err != nil {
		log.Printf("Failed to decode coordinates from JSON: %v", err)
	}

	fmt.Printf("Decoded coordinates count: %d (matches original: %t)\n",
		len(decodedCoords), len(decodedCoords) == len(coordinates))

	// Get first 5 coordinates for display
	var firstFiveCoords []models.Coordinate
	if len(coordinates) > 0 {
		maxCoords := 5
		if len(coordinates) < maxCoords {
			maxCoords = len(coordinates)
		}
		firstFiveCoords = coordinates[:maxCoords]

		fmt.Printf("\nFirst %d coordinates:\n", maxCoords)
		for i, coord := range firstFiveCoords {
			fmt.Printf("  %d: [%.6f, %.6f]\n", i+1, coord.Lon, coord.Lat)
		}
	}

	// Get places data
	placeNodes, placeRelations := osm.GetPlaces()
	placesData := &models.PlacesData{
		PlaceNodes:     len(placeNodes),
		PlaceRelations: len(placeRelations),
	}

	fmt.Printf("\nPlaces found:\n")
	fmt.Printf("Place Nodes: %d\n", len(placeNodes))
	for _, node := range placeNodes {
		fmt.Printf("  - Node %d: %s (place=%s)\n", node.ID, node.GetName(), node.GetTagValue("place"))
	}
	fmt.Printf("Place Relations: %d\n", len(placeRelations))
	for _, relation := range placeRelations {
		fmt.Printf("  - Relation %d: %s (place=%s)\n", relation.ID, relation.GetName(), relation.GetTagValue("place"))
	}

	// Get administrative boundaries
	boundaryWays, boundaryRelations := osm.GetAdministrativeBoundaries()
	adminBoundariesData := &models.AdminBoundariesData{
		BoundaryWays:      len(boundaryWays),
		BoundaryRelations: len(boundaryRelations),
	}

	fmt.Printf("\nAdministrative Boundaries:\n")
	fmt.Printf("Boundary Ways: %d\n", len(boundaryWays))
	for _, way := range boundaryWays {
		fmt.Printf("  - Way %d: admin_level=%s\n", way.ID, way.GetTagValue("admin_level"))
	}
	fmt.Printf("Boundary Relations: %d\n", len(boundaryRelations))
	for _, relation := range boundaryRelations {
		fmt.Printf("  - Relation %d: %s (admin_level=%s)\n", relation.ID, relation.GetName(), relation.GetTagValue("admin_level"))
	}

	return &models.BoundaryData{
		TotalCoordinates: len(coordinates),
		JSONString:       jsonString,
		FirstFiveCoords:  firstFiveCoords,
		Places:           placesData,
		AdminBoundaries:  adminBoundariesData,
	}, nil
}

// getCapitalLevelStats gets capital level statistics
func (s *OSMService) getCapitalLevelStats(osm *models.OSM) map[int]int {
	fmt.Printf("\n=== CAPITAL LEVEL STATISTICS ===\n")

	capitalStats := make(map[int]int)
	for _, relation := range osm.Relations {
		if relation.IsAdministrativeBoundary() {
			capitalLevel := relation.GetCapitalLevel()
			if capitalLevel > 0 {
				capitalStats[capitalLevel]++
			}
		}
	}

	for level, count := range capitalStats {
		var levelName string
		switch level {
		case 4:
			levelName = "Tỉnh/Thành phố"
		case 6:
			levelName = "Xã/Phường"
		default:
			levelName = fmt.Sprintf("Cấp %d", level)
		}
		fmt.Printf("Capital level %d (%s): %d entities\n", level, levelName, count)
	}

	return capitalStats
}

// processAdministrativeEntities processes administrative entities and categorizes them by level
func (s *OSMService) processAdministrativeEntities(osm *models.OSM) (map[string][]models.AdminEntity, error) {
	fmt.Printf("\nProcessing administrative entities...\n")

	administrativeData := map[string][]models.AdminEntity{
		"provinces": []models.AdminEntity{},
		"communes":  []models.AdminEntity{},
	}

	// Process relations (boundaries)
	for _, relation := range osm.Relations {
		if !relation.IsAdministrativeBoundary() {
			continue
		}

		adminLevel := relation.GetAdminLevel()
		capitalLevel := relation.GetCapitalLevel()

		fmt.Printf("Processing relation %d: %s (admin_level=%d, capital=%d)\n",
			relation.ID, relation.GetName(), adminLevel, capitalLevel)

		// Get boundary coordinates for this relation
		coordinates, err := osm.GetBoundaryCoordinatesFromRelation(&relation)
		if err != nil {
			fmt.Printf("Warning: Could not get boundary coordinates for relation %d: %v\n", relation.ID, err)
			continue
		}

		// Encode coordinates to JSON
		boundaryJSON, err := models.EncodeCoordinatesToJSON(coordinates)
		if err != nil {
			fmt.Printf("Warning: Could not encode coordinates for relation %d: %v\n", relation.ID, err)
			continue
		}

		// Create AdminEntity
		entity := models.AdminEntity{
			ID:           relation.ID,
			Name:         relation.GetTagValue("name"),
			NameEn:       relation.GetTagValue("name:en"),
			NameVi:       relation.GetTagValue("name:vi"),
			AdminLevel:   adminLevel,
			CapitalLevel: capitalLevel,
			Place:        relation.GetTagValue("place"),
			Boundary:     boundaryJSON,
		}

		// Classify by level - Relation thường dùng admin_level
		if adminLevel == 4 {
			entity.Type = "province"
			administrativeData["provinces"] = append(administrativeData["provinces"], entity)
			fmt.Printf("  -> Added as PROVINCE (admin_level=4)\n")
		} else if adminLevel == 6 {
			entity.Type = "commune"
			administrativeData["communes"] = append(administrativeData["communes"], entity)
			fmt.Printf("  -> Added as COMMUNE (admin_level=6)\n")
		} else {
			fmt.Printf("  -> Skipped (admin_level=%d not recognized)\n", adminLevel)
		}
	}

	// Process nodes (places with capital level)
	for _, node := range osm.Nodes {
		capitalLevel := node.GetCapitalLevel()
		if capitalLevel <= 0 {
			continue
		}

		fmt.Printf("Processing node %d: %s (capital=%d)\n",
			node.ID, node.GetName(), capitalLevel)

		// Create AdminEntity for node
		entity := models.AdminEntity{
			ID:           node.ID,
			Name:         node.GetTagValue("name"),
			NameEn:       node.GetTagValue("name:en"),
			NameVi:       node.GetTagValue("name:vi"),
			AdminLevel:   -1, // Nodes don't have admin_level
			CapitalLevel: capitalLevel,
			Place:        node.GetTagValue("place"),
			Boundary:     "", // Nodes don't have boundary coordinates
		}

		// Classify by capital level - Node thường dùng capital level
		if capitalLevel == 4 {
			entity.Type = "province"
			administrativeData["provinces"] = append(administrativeData["provinces"], entity)
			fmt.Printf("  -> Added as PROVINCE (capital=4)\n")
		} else if capitalLevel == 6 {
			entity.Type = "commune"
			administrativeData["communes"] = append(administrativeData["communes"], entity)
			fmt.Printf("  -> Added as COMMUNE (capital=6)\n")
		} else {
			fmt.Printf("  -> Skipped (capital level %d not recognized)\n", capitalLevel)
		}
	}

	fmt.Printf("Administrative entities processed:\n")
	fmt.Printf("- Provinces: %d\n", len(administrativeData["provinces"]))
	fmt.Printf("- Communes: %d\n", len(administrativeData["communes"]))

	return administrativeData, nil
}

// GetBoundaryStringFromResult lấy string của viền từ kết quả đã xử lý
func (s *OSMService) GetBoundaryStringFromResult(result *models.OSMProcessingResult) string {
	if result == nil {
		return ""
	}

	// Lấy boundary string từ tổng quát (nếu có)
	if result.Boundaries != nil {
		boundaryString := result.Boundaries.JSONString
		fmt.Printf("Lấy boundary string từ kết quả đã xử lý:\n")
		fmt.Printf("- Số coordinates: %d\n", result.Boundaries.TotalCoordinates)
		fmt.Printf("- JSON string length: %d characters\n", len(boundaryString))
		return boundaryString
	}

	// Nếu không có tổng quát, lấy từ Administrative entities
	if result.Administrative != nil {
		fmt.Printf("Lấy boundary string từ Administrative entities:\n")

		// Lấy từ communes trước (thường có boundary chi tiết hơn)
		if communes, exists := result.Administrative["communes"]; exists && len(communes) > 0 {
			for _, commune := range communes {
				if commune.Boundary != "" {
					fmt.Printf("- Từ commune '%s' (ID: %d): %d characters\n",
						commune.Name, commune.ID, len(commune.Boundary))
					return commune.Boundary
				}
			}
		}

		// Nếu không có commune, lấy từ provinces
		if provinces, exists := result.Administrative["provinces"]; exists && len(provinces) > 0 {
			for _, province := range provinces {
				if province.Boundary != "" {
					fmt.Printf("- Từ province '%s' (ID: %d): %d characters\n",
						province.Name, province.ID, len(province.Boundary))
					return province.Boundary
				}
			}
		}
	}

	fmt.Printf("Không tìm thấy boundary string trong kết quả\n")
	return ""
}

func (s *OSMService) UpdateStringBoundaryToDatabase(name string, level int, maxLat, minLat, maxLon, minLon, lonCenter, latCenter float64) error {
	if s.dmTTRepo == nil || s.dmPhuongXaRepo == nil {
		return fmt.Errorf("database repositories not initialized, use NewOSMServiceWithDB()")
	}

	switch level {
	case 4: // Tỉnh/thành phố
		tt, err := s.dmTTRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("không thể lấy dữ liệu tỉnh/thành phố từ database: %w", err)
		}
		if tt == nil {
			return fmt.Errorf("không tìm thấy tỉnh/thành phố '%s' trong database", name)
		}
		return s.dmTTRepo.UpdateDataAddressByMaTT(tt.MaTT, &maxLat, &minLat, &maxLon, &minLon, &lonCenter, &latCenter)
	case 6: // Xã/phường
		px, err := s.dmPhuongXaRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("không thể lấy dữ liệu xã/phường từ database: %w", err)
		}
		if px == nil {
			return fmt.Errorf("không tìm thấy xã/phường '%s' trong database", name)
		}
		return s.dmPhuongXaRepo.UpdateDataAddressByMaPhuongXa(px.MaPhuongXa, &maxLat, &minLat, &maxLon, &minLon, &lonCenter, &latCenter)
	default:
		return fmt.Errorf("level '%d' không được hỗ trợ", level)
	}
}

// CreatePolygonFromWaysAndNodes tạo polygon từ ways và nodes
func (s *OSMService) CreatePolygonFromWaysAndNodes(ways []models.WayAddress, nodes []models.Address) ([][]float64, error) {
	fmt.Printf("\n=== TẠO POLYGON TỪ WAYS VÀ NOTES ===\n")

	// Xây dựng map cho nodes với ID là key, value là [lat, lon]
	nodeMap := make(map[int64][]float64)
	for _, node := range nodes {
		nodeMap[node.ID] = []float64{node.Lat, node.Lon}
	}

	fmt.Printf("Created node map with %d nodes\n", len(nodeMap))

	// Tạo danh sách WayCoordinates từ ways và note
	var wayCoords []WayCoordinates
	for _, way := range ways {
		var coords [][]float64
		var nodeIDs []int64

		// way.Nodes là slice []string, cần convert sang int64
		for _, nodeRefStr := range way.Nodes {
			if nodeRef, err := strconv.ParseInt(nodeRefStr, 10, 64); err == nil {
				if coord, exists := nodeMap[nodeRef]; exists {
					coords = append(coords, coord)
					nodeIDs = append(nodeIDs, nodeRef)
				}
			}
		}

		if len(coords) >= 2 {
			wayCoords = append(wayCoords, WayCoordinates{
				ID:      way.ID,
				Coords:  coords,
				NodeIDs: nodeIDs,
			})
		}
	}

	fmt.Printf("Created %d way coordinates\n", len(wayCoords))

	if len(wayCoords) == 0 {
		return nil, fmt.Errorf("no valid ways found")
	}

	// Tạo polygon liền mạch từ kết quả của ways và note
	polygon, err := s.buildConnectedPath(wayCoords)
	if err != nil {
		fmt.Printf("Warning: buildConnectedPath failed: %v\n", err)
		// Fallback: sử dụng convex hull từ ways & node
		polygon = s.fallbackPolygonConstruction(wayCoords)
		fmt.Printf("Using fallback convex hull with %d points\n", len(polygon))
	}

	fmt.Printf("Final polygon has %d coordinates\n", len(polygon))
	return polygon, nil
}

// buildConnectedPath
func (s *OSMService) buildConnectedPath(wayCoords []WayCoordinates) ([][]float64, error) {
	if len(wayCoords) == 0 {
		return nil, fmt.Errorf("no way coordinates provided")
	}

	// Tạo connection map
	connectionMap := make(map[string][]Connection)
	usedWays := make(map[int64]bool)
	var result [][]float64

	// Xây dựng connection map
	for _, way := range wayCoords {
		if len(way.Coords) >= 2 {
			start := fmt.Sprintf("%.6f,%.6f", way.Coords[0][0], way.Coords[0][1])
			end := fmt.Sprintf("%.6f,%.6f", way.Coords[len(way.Coords)-1][0], way.Coords[len(way.Coords)-1][1])

			if connectionMap[start] == nil {
				connectionMap[start] = []Connection{}
			}
			if connectionMap[end] == nil {
				connectionMap[end] = []Connection{}
			}

			connectionMap[start] = append(connectionMap[start], Connection{
				WayID:   way.ID,
				IsStart: true,
				Coords:  way.Coords,
			})
			connectionMap[end] = append(connectionMap[end], Connection{
				WayID:   way.ID,
				IsStart: false,
				Coords:  way.Coords,
			})
		}
	}

	// Tìm điểm bắt đầu (chỉ kết nối với 1 way)
	var startPoint string
	for point, connections := range connectionMap {
		if len(connections) == 1 {
			startPoint = point
			break
		}
	}

	// Nếu không tìm thấy điểm đặc biệt, dùng điểm đầu tiên
	if startPoint == "" && len(connectionMap) > 0 {
		for point := range connectionMap {
			startPoint = point
			break
		}
	}

	if startPoint == "" {
		return nil, fmt.Errorf("no starting point found")
	}

	// Xây dựng đường dẫn liên tục
	currentPoint := startPoint
	for len(connectionMap) > 0 {
		connections, exists := connectionMap[currentPoint]
		if !exists || len(connections) == 0 {
			break
		}

		// Tìm connection chưa được sử dụng
		var connection *Connection
		for i := range connections {
			if !usedWays[connections[i].WayID] {
				connection = &connections[i]
				break
			}
		}

		if connection == nil {
			break
		}

		usedWays[connection.WayID] = true

		coords := connection.Coords
		if !connection.IsStart {
			// Đảo ngược nếu cần
			coords = s.reverseCoords(coords)
		}

		// Thêm coordinates (bỏ điểm cuối để tránh trùng lặp)
		if len(coords) > 1 {
			result = append(result, coords[:len(coords)-1]...)
		}

		// Tìm điểm tiếp theo
		if len(coords) > 0 {
			lastCoord := coords[len(coords)-1]
			currentPoint = fmt.Sprintf("%.6f,%.6f", lastCoord[0], lastCoord[1])
		}
	}

	// Thêm điểm cuối cùng - không cần thiết vì đã được xử lý trong loop
	// Loại bỏ logic thừa này để tránh duplicate points

	return result, nil
}

// reverseCoords đảo ngược thứ tự coordinates
func (s *OSMService) reverseCoords(coords [][]float64) [][]float64 {
	reversed := make([][]float64, len(coords))
	for i, coord := range coords {
		reversed[len(coords)-1-i] = coord
	}
	return reversed
}

// fallbackPolygonConstruction tạo convex hull như fallback
func (s *OSMService) fallbackPolygonConstruction(wayCoords []WayCoordinates) [][]float64 {
	// Thu thập tất cả coordinates
	var allCoords [][]float64
	for _, way := range wayCoords {
		allCoords = append(allCoords, way.Coords...)
	}

	if len(allCoords) < 3 {
		return allCoords
	}

	// Tạo convex hull (thuật toán đơn giản)
	return s.convexHull(allCoords)
}

// convexHull tạo convex hull từ các điểm
func (s *OSMService) convexHull(points [][]float64) [][]float64 {
	if len(points) < 3 {
		return points
	}

	// Tìm điểm bottom-most (và left-most nếu bằng nhau)
	bottom := 0
	for i := 1; i < len(points); i++ {
		if points[i][0] < points[bottom][0] ||
			(points[i][0] == points[bottom][0] && points[i][1] < points[bottom][1]) {
			bottom = i
		}
	}

	// Sắp xếp theo góc từ điểm bottom-most
	bottomPoint := points[bottom]
	points = append(points[:bottom], points[bottom+1:]...)

	// Sắp xếp theo góc
	sort.Slice(points, func(i, j int) bool {
		angleI := s.angle(bottomPoint, points[i])
		angleJ := s.angle(bottomPoint, points[j])
		return angleI < angleJ
	})

	// Thêm điểm bottom-most vào đầu
	hull := [][]float64{bottomPoint}

	// Graham scan
	for _, point := range points {
		for len(hull) > 1 && s.cross(hull[len(hull)-2], hull[len(hull)-1], point) <= 0 {
			hull = hull[:len(hull)-1]
		}
		hull = append(hull, point)
	}

	return hull
}

// angle tính góc từ p1 đến p2
func (s *OSMService) angle(p1, p2 []float64) float64 {
	return math.Atan2(p2[1]-p1[1], p2[0]-p1[0])
}

// cross tính cross product của 3 điểm
func (s *OSMService) cross(p1, p2, p3 []float64) float64 {
	return (p2[0]-p1[0])*(p3[1]-p1[1]) - (p2[1]-p1[1])*(p3[0]-p1[0])
}

// UpdatePolygonToDatabase lưu polygon data vào database
func (s *OSMService) UpdatePolygonToDatabase(name string, level int, polygonData [][]float64) error {
	if s.dmTTRepo == nil || s.dmPhuongXaRepo == nil {
		return fmt.Errorf("database repositories not initialized, use NewOSMServiceWithDB()")
	}

	// Convert polygon data to JSON string
	polygonJSON, err := json.Marshal(polygonData)
	if err != nil {
		return fmt.Errorf("failed to marshal polygon data to JSON: %w", err)
	}
	polygonString := string(polygonJSON)

	switch level {
	case 4: // Tỉnh/thành phố
		tt, err := s.dmTTRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("không thể lấy dữ liệu tỉnh/thành phố từ database: %w", err)
		}
		if tt == nil {
			return fmt.Errorf("không tìm thấy tỉnh/thành phố '%s' trong database", name)
		}
		return s.dmTTRepo.UpdatePolygonDataByMaTT(tt.MaTT, &polygonString)
	case 6: // Xã/phường
		// TODO: Implement for communes if needed
		px, err := s.dmPhuongXaRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("không thể lấy dữ liệu xã/phường từ database: %w", err)
		}
		if px == nil {
			return fmt.Errorf("không tìm thấy xã/phường '%s' trong database", name)
		}
		return s.dmPhuongXaRepo.UpdatePolygonDataByMaPhuongXa(px.MaPhuongXa, &polygonString)
	default:
		return fmt.Errorf("level '%d' không được hỗ trợ", level)
	}
}

// FindCommuneByCoordinate tìm xã/phường từ tọa độ lat/lon và mã tỉnh thành
func (s *OSMService) FindCommuneByCoordinate(provinceCode string, lat, lon float64) (*entities.DmPhuongXa, error) {
	if s.dmTTRepo == nil {
		return nil, fmt.Errorf("database repositories not initialized, use NewOSMServiceWithDB()")
	}

	return s.dmTTRepo.FindCommuneByCoordinate(provinceCode, lat, lon)
}
