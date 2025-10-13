package services

import (
	"fmt"
	"log"
	"tool-map/models"
	"tool-map/repositories"

	"gorm.io/gorm"
)

type OSMServiceInterface interface {
	FetchAndProcessRelation(relationID int64) (*models.OSMProcessingResult, error)
	GetBoundaryStringFromResult(result *models.OSMProcessingResult) string
	UpdateStringBoundaryToDatabase(id string, level int, boundaryString, wayAddress string, lonCenter, latCenter float64) error
}
type OSMService struct {
	client         *models.OSMApiClient
	dmTTRepo       repositories.DmTTRepositoryInterface
	dmPhuongXaRepo repositories.DmPhuongXaRepositoryInterface
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
	var centerPoints []models.CenterPoint
	for _, node := range osm.Nodes {
		nodes = append(nodes, node.ToAddress())

		// Only treat nodes that contain a 'capital' tag as center points
		isCenter := false
		for _, tag := range node.Tags {
			if tag.Key == "capital" {
				isCenter = true
				break
			}
		}
		if isCenter {
			centerPoints = append(centerPoints, node.ToCenterPoint())
		}
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

func (s *OSMService) UpdateStringBoundaryToDatabase(name string, level int, boundaryString, wayAddress string, lonCenter, latCenter float64) error {
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
		return s.dmTTRepo.UpdateDataAddressByMaTT(tt.MaTT, &boundaryString, &wayAddress, &lonCenter, &latCenter)
	case 6: // Xã/phường
		px, err := s.dmPhuongXaRepo.GetByName(name)
		if err != nil {
			return fmt.Errorf("không thể lấy dữ liệu xã/phường từ database: %w", err)
		}
		if px == nil {
			return fmt.Errorf("không tìm thấy xã/phường '%s' trong database", name)
		}
		return s.dmPhuongXaRepo.UpdateDataAddressByMaPhuongXa(px.MaPhuongXa, &boundaryString, &wayAddress, &lonCenter, &latCenter)
	default:
		return fmt.Errorf("level '%d' không được hỗ trợ", level)
	}
}
