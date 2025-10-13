package models

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	OSMBaseURL = "https://www.openstreetmap.org/api/0.6"
)

type ApiInterface interface {
	FetchRelationFull(relationID int64) (*OSM, error)
	FetchWayFull(wayID int64) (*OSM, error)
	FetchNode(nodeID int64) (*OSM, error)

	EncodeCoordinatesToJSON(coordinates []Coordinate) (string, error)
	DecodeCoordinatesFromJSON(jsonString string) ([]Coordinate, error)
	EncodeCoordinatesToBinary(coordinates []Coordinate) (string, error)
	DecodeCoordinatesFromBinary(binaryString string) ([]Coordinate, error)
	GetBoundaryCoordinates() ([]Coordinate, error)
	GetBoundaryCoordinatesFromRelation(relation *Relation) ([]Coordinate, error)
	GetWayCoordinates(way *Way) []Coordinate
	GetNodesInWay(way *Way) []Node
	GetPlaces() ([]Node, []Relation)
	GetAdministrativeBoundaries() ([]Way, []Relation)
	FindNodeByID(id int64) (*Node, bool)
	FindWayByID(id int64) (*Way, bool)
	FindRelationByID(id int64) (*Relation, bool)
	IsAdministrativeBoundary() bool
	IsPlace() bool
	GetName() string
	GetTagValue(key string) string
	GetAdminLevel() int
	GetCapitalLevel() int
	IsProvinceOrCity() bool
	IsCommune() bool
	GetTimestamp() (time.Time, error)
}

// OSMApiClient represents an OSM API client
type OSMApiClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewOSMApiClient creates a new OSM API client
func NewOSMApiClient() *OSMApiClient {
	return &OSMApiClient{
		BaseURL: OSMBaseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		UserAgent: "tool-map/1.0",
	}
}

// FetchRelationFull fetches a relation with all its members (nodes, ways, and sub-relations)
func (client *OSMApiClient) FetchRelationFull(relationID int64) (*OSM, error) {
	url := fmt.Sprintf("%s/relation/%d/full", client.BaseURL, relationID)
	return client.fetchOSMData(url)
}

// FetchWayFull fetches a way with all its node members
func (client *OSMApiClient) FetchWayFull(wayID int64) (*OSM, error) {
	url := fmt.Sprintf("%s/way/%d/full", client.BaseURL, wayID)
	return client.fetchOSMData(url)
}

// FetchNode fetches a single node
func (client *OSMApiClient) FetchNode(nodeID int64) (*OSM, error) {
	url := fmt.Sprintf("%s/node/%d", client.BaseURL, nodeID)
	return client.fetchOSMData(url)
}

// fetchOSMData fetches OSM data from the given URL
func (client *OSMApiClient) fetchOSMData(url string) (*OSM, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent header (required by OSM API)
	req.Header.Set("User-Agent", client.UserAgent)

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse OSM XML data
	osm, err := ParseOSMFromBytes(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OSM data: %w", err)
	}

	return osm, nil
}

// EncodeCoordinatesToJSON encodes a list of coordinates to JSON string
func EncodeCoordinatesToJSON(coordinates []Coordinate) (string, error) {
	if len(coordinates) == 0 {
		return "[]", nil
	}

	// Convert coordinates to JSON array format
	jsonData, err := json.Marshal(coordinates)
	if err != nil {
		return "", fmt.Errorf("failed to marshal coordinates to JSON: %w", err)
	}

	return string(jsonData), nil
}

// EncodeCoordinatesToBinary encodes a list of coordinates to binary string (legacy function)
func EncodeCoordinatesToBinary(coordinates []Coordinate) (string, error) {
	if len(coordinates) == 0 {
		return "", nil
	}

	// Create a byte buffer to store binary data
	var binaryData []byte

	for _, coord := range coordinates {
		// Encode latitude as 8 bytes (float64)
		latBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(latBytes, math.Float64bits(coord.Lat))
		binaryData = append(binaryData, latBytes...)

		// Encode longitude as 8 bytes (float64)
		lonBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(lonBytes, math.Float64bits(coord.Lon))
		binaryData = append(binaryData, lonBytes...)
	}

	// Encode binary data to base64 string
	return base64.StdEncoding.EncodeToString(binaryData), nil
}

// DecodeCoordinatesFromJSON decodes a JSON string back to coordinates
func DecodeCoordinatesFromJSON(jsonString string) ([]Coordinate, error) {
	if jsonString == "" || jsonString == "[]" {
		return nil, nil
	}

	var coordinates []Coordinate
	err := json.Unmarshal([]byte(jsonString), &coordinates)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON string: %w", err)
	}

	return coordinates, nil
}

// DecodeCoordinatesFromBinary decodes a binary string back to coordinates (legacy function)
func DecodeCoordinatesFromBinary(binaryString string) ([]Coordinate, error) {
	if binaryString == "" {
		return nil, nil
	}

	// Decode base64 string to binary data
	binaryData, err := base64.StdEncoding.DecodeString(binaryString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}

	// Each coordinate pair consists of 16 bytes (8 for lat + 8 for lon)
	if len(binaryData)%16 != 0 {
		return nil, fmt.Errorf("invalid binary data length: expected multiple of 16 bytes")
	}

	var coordinates []Coordinate
	for i := 0; i < len(binaryData); i += 16 {
		// Decode latitude (first 8 bytes)
		latBytes := binaryData[i : i+8]
		lat := binary.LittleEndian.Uint64(latBytes)
		latitude := math.Float64frombits(lat)

		// Decode longitude (next 8 bytes)
		lonBytes := binaryData[i+8 : i+16]
		lon := binary.LittleEndian.Uint64(lonBytes)
		longitude := math.Float64frombits(lon)

		coordinates = append(coordinates, Coordinate{
			ID:  0, // Binary data doesn't contain ID, set to 0
			Lat: latitude,
			Lon: longitude,
		})
	}

	return coordinates, nil
}

// GetBoundaryCoordinates extracts coordinates from administrative boundary ways and relations
func (osm *OSM) GetBoundaryCoordinates() ([]Coordinate, error) {
	var allCoordinates []Coordinate

	// Process administrative boundary ways
	for _, way := range osm.Ways {
		if way.IsAdministrativeBoundary() {
			coordinates := osm.GetWayCoordinates(&way)
			allCoordinates = append(allCoordinates, coordinates...)
		}
	}

	// Process administrative boundary relations
	for _, relation := range osm.Relations {
		if relation.IsAdministrativeBoundary() {
			// Get coordinates from all ways in the relation
			for _, member := range relation.Members {
				if member.Type == "way" {
					if way, found := osm.FindWayByID(member.Ref); found {
						coordinates := osm.GetWayCoordinates(way)
						allCoordinates = append(allCoordinates, coordinates...)
					}
				}
			}
		}
	}

	return allCoordinates, nil
}

// ConvertToDmPhuongXaFormat converts OSM data to DmPhuongXa format (communes only)
func (osm *OSM) ConvertToDmPhuongXaFormat() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Process administrative boundary relations (communes)
	for _, relation := range osm.Relations {
		if relation.IsAdministrativeBoundary() && relation.GetAdminLevel() == 6 { // Commune level
			// Get boundary coordinates
			coordinates, err := osm.GetBoundaryCoordinatesFromRelation(&relation)
			if err != nil {
				continue // Skip this relation if we can't get coordinates
			}

			// Encode coordinates to JSON string
			jsonString, err := EncodeCoordinatesToJSON(coordinates)
			if err != nil {
				continue // Skip this relation if we can't encode coordinates
			}

			result := map[string]interface{}{
				"tenPhuongXa":   relation.GetName(),
				"tenPhuongXaEn": relation.GetTagValue("name:en"),
				"toaDoBienGioi": &jsonString,
				"admin_level":   relation.GetAdminLevel(),
				"capital_level": relation.GetCapitalLevel(),
				"place":         relation.GetTagValue("place"),
				"osm_id":        relation.ID,
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// ConvertToDmTTFormat converts OSM data to DmTT format (provinces/cities only)
func (osm *OSM) ConvertToDmTTFormat() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Process administrative boundary relations (provinces/cities)
	for _, relation := range osm.Relations {
		if relation.IsAdministrativeBoundary() && relation.GetAdminLevel() == 4 { // Province/City level
			// Get boundary coordinates
			coordinates, err := osm.GetBoundaryCoordinatesFromRelation(&relation)
			if err != nil {
				continue // Skip this relation if we can't get coordinates
			}

			// Encode coordinates to JSON string
			jsonString, err := EncodeCoordinatesToJSON(coordinates)
			if err != nil {
				continue // Skip this relation if we can't encode coordinates
			}

			result := map[string]interface{}{
				"tenTT":         relation.GetName(),
				"tenTTEn":       relation.GetTagValue("name:en"),
				"toaDoBienGioi": &jsonString,
				"admin_level":   relation.GetAdminLevel(),
				"capital_level": relation.GetCapitalLevel(),
				"place":         relation.GetTagValue("place"),
				"osm_id":        relation.ID,
			}

			results = append(results, result)
		}
	}

	return results, nil
}

// ConvertToAllAdministrativeLevels converts OSM data to all administrative levels
func (osm *OSM) ConvertToAllAdministrativeLevels() (map[string][]map[string]interface{}, error) {
	result := map[string][]map[string]interface{}{
		"provinces": {},
		"communes":  {},
	}

	// Process all administrative boundary relations
	for _, relation := range osm.Relations {
		if !relation.IsAdministrativeBoundary() {
			continue
		}

		// Get boundary coordinates
		coordinates, err := osm.GetBoundaryCoordinatesFromRelation(&relation)
		if err != nil {
			continue // Skip this relation if we can't get coordinates
		}

		// Encode coordinates to JSON string
		jsonString, err := EncodeCoordinatesToJSON(coordinates)
		if err != nil {
			continue // Skip this relation if we can't encode coordinates
		}

		adminLevel := relation.GetAdminLevel()
		capitalLevel := relation.GetCapitalLevel()

		baseData := map[string]interface{}{
			"name":          relation.GetName(),
			"nameEn":        relation.GetTagValue("name:en"),
			"toaDoBienGioi": &jsonString,
			"admin_level":   adminLevel,
			"capital_level": capitalLevel,
			"place":         relation.GetTagValue("place"),
			"osm_id":        relation.ID,
		}

		// Classify by capital level
		if capitalLevel == 4 { // Province/City
			result["provinces"] = append(result["provinces"], baseData)
		} else if capitalLevel == 6 { // Commune
			result["communes"] = append(result["communes"], baseData)
		}
	}

	return result, nil
}

// GetBoundaryCoordinatesFromRelation extracts coordinates from a specific relation
func (osm *OSM) GetBoundaryCoordinatesFromRelation(relation *Relation) ([]Coordinate, error) {
	// Lấy tất cả ways của relation
	var ways []Way
	for _, member := range relation.Members {
		if member.Type == "way" && member.Role == "outer" {
			if way, found := osm.FindWayByID(member.Ref); found {
				ways = append(ways, *way)
			}
		}
	}

	if len(ways) == 0 {
		return []Coordinate{}, fmt.Errorf("no outer ways found in relation")
	}

	// Sắp xếp các ways thành một chuỗi liền mạch
	sortedCoordinates, err := osm.connectWaysIntoPolygon(ways)
	if err != nil {
		return []Coordinate{}, fmt.Errorf("failed to connect ways: %w", err)
	}

	return sortedCoordinates, nil
}

// connectWaysIntoPolygon connects multiple ways into a single polygon (OSM-style)
func (osm *OSM) connectWaysIntoPolygon(ways []Way) ([]Coordinate, error) {
	if len(ways) == 0 {
		return []Coordinate{}, fmt.Errorf("no ways to connect")
	}

	// Bước 1: Tạo map để lưu trữ coordinates của từng way
	wayCoordsMap := make(map[int64][]Coordinate)
	for _, way := range ways {
		coords := osm.GetWayCoordinates(&way)
		if len(coords) > 0 {
			wayCoordsMap[way.ID] = coords
		}
	}

	if len(wayCoordsMap) == 0 {
		return []Coordinate{}, fmt.Errorf("no valid ways with coordinates")
	}

	// Bước 2: Tìm đường đi liền mạch giữa các ways (giống OSM website)
	connectedPath, err := osm.buildConnectedPath(wayCoordsMap)
	if err != nil {
		// Nếu không thể nối được, fallback về cách cũ
		fmt.Printf("Warning: Could not build connected path, using fallback: %v\n", err)
		return osm.fallbackPolygonConstruction(wayCoordsMap), nil
	}

	// Bước 3: Đảm bảo polygon được đóng kín
	if len(connectedPath) > 0 {
		first := connectedPath[0]
		last := connectedPath[len(connectedPath)-1]
		if !osm.isCloseEnough(first, last) {
			connectedPath = append(connectedPath, first)
		}
	}

	return connectedPath, nil
}

// buildConnectedPath tìm đường đi liền mạch giữa các ways (OSM-style algorithm)
func (osm *OSM) buildConnectedPath(wayCoordsMap map[int64][]Coordinate) ([]Coordinate, error) {
	if len(wayCoordsMap) == 1 {
		// Chỉ có 1 way, trả về luôn
		for _, coords := range wayCoordsMap {
			return coords, nil
		}
	}

	// Tạo map để track các điểm kết nối
	pointToWays := make(map[string][]int64)
	for wayID, coords := range wayCoordsMap {
		if len(coords) > 0 {
			// Điểm đầu
			startKey := osm.coordinateKey(coords[0])
			pointToWays[startKey] = append(pointToWays[startKey], wayID)

			// Điểm cuối
			endKey := osm.coordinateKey(coords[len(coords)-1])
			pointToWays[endKey] = append(pointToWays[endKey], wayID)
		}
	}

	// Tìm điểm bắt đầu (điểm chỉ có 1 way kết nối)
	var startWayID int64
	var isReversed bool

	for pointKey, wayIDs := range pointToWays {
		if len(wayIDs) == 1 {
			startWayID = wayIDs[0]
			coords := wayCoordsMap[startWayID]
			if len(coords) > 0 {
				// Kiểm tra xem điểm này là đầu hay cuối của way
				if osm.coordinateKey(coords[0]) == pointKey {
					isReversed = false
				} else {
					isReversed = true
				}
				break
			}
		}
	}

	if startWayID == 0 {
		// Không tìm thấy điểm bắt đầu, dùng cách khác
		return []Coordinate{}, fmt.Errorf("could not find starting point")
	}

	// Xây dựng đường đi
	var result []Coordinate
	processedWays := make(map[int64]bool)

	// Thêm way đầu tiên
	coords := wayCoordsMap[startWayID]
	if isReversed {
		// Đảo ngược thứ tự
		for i := len(coords) - 1; i >= 0; i-- {
			result = append(result, coords[i])
		}
	} else {
		result = append(result, coords...)
	}
	processedWays[startWayID] = true

	// Tìm các ways tiếp theo
	for len(processedWays) < len(wayCoordsMap) {
		lastPoint := result[len(result)-1]
		lastPointKey := osm.coordinateKey(lastPoint)

		// Tìm way tiếp theo
		var nextWayID int64
		var nextIsReversed bool

		for wayID, wayCoords := range wayCoordsMap {
			if processedWays[wayID] {
				continue
			}

			if len(wayCoords) > 0 {
				firstKey := osm.coordinateKey(wayCoords[0])
				lastKey := osm.coordinateKey(wayCoords[len(wayCoords)-1])

				if firstKey == lastPointKey {
					nextWayID = wayID
					nextIsReversed = false
					break
				} else if lastKey == lastPointKey {
					nextWayID = wayID
					nextIsReversed = true
					break
				}
			}
		}

		if nextWayID == 0 {
			// Không tìm thấy way tiếp theo
			break
		}

		// Thêm way tiếp theo
		coords := wayCoordsMap[nextWayID]
		if nextIsReversed {
			// Đảo ngược và bỏ điểm đầu (vì đã có rồi)
			for i := len(coords) - 2; i >= 0; i-- {
				result = append(result, coords[i])
			}
		} else {
			// Bỏ điểm đầu (vì đã có rồi)
			result = append(result, coords[1:]...)
		}
		processedWays[nextWayID] = true
	}

	// Thêm các ways còn lại nếu có
	for wayID, coords := range wayCoordsMap {
		if !processedWays[wayID] {
			result = append(result, coords...)
		}
	}

	return result, nil
}

// coordinateKey tạo key duy nhất cho coordinate (để so sánh)
func (osm *OSM) coordinateKey(coord Coordinate) string {
	// Làm tròn để tránh lỗi floating point
	lat := math.Round(coord.Lat*1000000) / 1000000
	lon := math.Round(coord.Lon*1000000) / 1000000
	return fmt.Sprintf("%.6f,%.6f", lat, lon)
}

// fallbackPolygonConstruction fallback method khi không thể nối ways
func (osm *OSM) fallbackPolygonConstruction(wayCoordsMap map[int64][]Coordinate) []Coordinate {
	var allCoords []Coordinate
	for _, coords := range wayCoordsMap {
		allCoords = append(allCoords, coords...)
	}

	// Sử dụng convex hull
	sortedCoords := osm.sortCoordinatesByConvexHull(allCoords)

	// Đóng polygon
	if len(sortedCoords) > 0 {
		first := sortedCoords[0]
		last := sortedCoords[len(sortedCoords)-1]
		if !osm.isCloseEnough(first, last) {
			sortedCoords = append(sortedCoords, first)
		}
	}

	return sortedCoords
}

// sortCoordinatesByConvexHull sắp xếp coordinates theo thứ tự convex hull
func (osm *OSM) sortCoordinatesByConvexHull(coords []Coordinate) []Coordinate {
	if len(coords) <= 3 {
		return coords
	}

	// Tìm điểm có latitude thấp nhất (nam nhất)
	minLat := coords[0].Lat
	minIndex := 0
	for i, coord := range coords {
		if coord.Lat < minLat || (coord.Lat == minLat && coord.Lon < coords[minIndex].Lon) {
			minLat = coord.Lat
			minIndex = i
		}
	}

	// Đặt điểm thấp nhất làm điểm đầu
	startPoint := coords[minIndex]

	// Sắp xếp các điểm còn lại theo góc từ điểm đầu (theo chiều kim đồng hồ)
	type PointWithAngle struct {
		coord Coordinate
		angle float64
		index int
	}

	var pointsWithAngle []PointWithAngle
	for i, coord := range coords {
		if i != minIndex {
			angle := osm.calculateAngle(startPoint, coord)
			pointsWithAngle = append(pointsWithAngle, PointWithAngle{
				coord: coord,
				angle: angle,
				index: i,
			})
		}
	}

	// Sắp xếp theo góc
	for i := 0; i < len(pointsWithAngle)-1; i++ {
		for j := i + 1; j < len(pointsWithAngle); j++ {
			if pointsWithAngle[i].angle > pointsWithAngle[j].angle {
				pointsWithAngle[i], pointsWithAngle[j] = pointsWithAngle[j], pointsWithAngle[i]
			}
		}
	}

	// Tạo kết quả cuối cùng
	result := []Coordinate{startPoint}
	for _, pwa := range pointsWithAngle {
		result = append(result, pwa.coord)
	}

	return result
}

// calculateAngle tính góc từ điểm start đến điểm end
func (osm *OSM) calculateAngle(start, end Coordinate) float64 {
	dx := end.Lon - start.Lon
	dy := end.Lat - start.Lat
	return math.Atan2(dy, dx)
}

// isCloseEnough checks if two coordinates are close enough to be considered the same point
func (osm *OSM) isCloseEnough(coord1, coord2 Coordinate) bool {
	const tolerance = 0.000001 // ~0.1 meters
	latDiff := coord1.Lat - coord2.Lat
	lonDiff := coord1.Lon - coord2.Lon
	return (latDiff*latDiff + lonDiff*lonDiff) < (tolerance * tolerance)
}

// PrettyPrintCoordinates prints coordinates in a readable format
func PrettyPrintCoordinates(coordinates []Coordinate) string {
	if len(coordinates) == 0 {
		return "No coordinates"
	}

	var parts []string
	for _, coord := range coordinates {
		parts = append(parts, fmt.Sprintf("[%.6f, %.6f]", coord.Lon, coord.Lat))
	}

	return strings.Join(parts, ", ")
}
