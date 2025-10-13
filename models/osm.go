package models

import (
	"encoding/xml"
	"strconv"
	"time"
)

// OSM represents the root element of an OpenStreetMap XML file
type OSM struct {
	XMLName     xml.Name   `xml:"osm"`
	Version     string     `xml:"version,attr"`
	Generator   string     `xml:"generator,attr"`
	Copyright   string     `xml:"copyright,attr"`
	Attribution string     `xml:"attribution,attr"`
	License     string     `xml:"license,attr"`
	Nodes       []Node     `xml:"node"`
	Ways        []Way      `xml:"way"`
	Relations   []Relation `xml:"relation"`
}

// Node represents an OSM node (point)
type Node struct {
	XMLName   xml.Name `xml:"node"`
	ID        int64    `xml:"id,attr"`
	Visible   bool     `xml:"visible,attr"`
	Version   int      `xml:"version,attr"`
	Changeset int64    `xml:"changeset,attr"`
	Timestamp string   `xml:"timestamp,attr"`
	User      string   `xml:"user,attr"`
	UID       int64    `xml:"uid,attr"`
	Lat       float64  `xml:"lat,attr"`
	Lon       float64  `xml:"lon,attr"`
	Tags      []Tag    `xml:"tag"`
}

// Way represents an OSM way (sequence of nodes forming a line or area)
type Way struct {
	XMLName   xml.Name  `xml:"way"`
	ID        int64     `xml:"id,attr"`
	Visible   bool      `xml:"visible,attr"`
	Version   int       `xml:"version,attr"`
	Changeset int64     `xml:"changeset,attr"`
	Timestamp string    `xml:"timestamp,attr"`
	User      string    `xml:"user,attr"`
	UID       int64     `xml:"uid,attr"`
	Nodes     []NodeRef `xml:"nd"`
	Tags      []Tag     `xml:"tag"`
}

// NodeRef represents a reference to a node in a way
type NodeRef struct {
	XMLName xml.Name `xml:"nd"`
	Ref     int64    `xml:"ref,attr"`
}

// Relation represents an OSM relation (grouping of nodes, ways, and other relations)
type Relation struct {
	XMLName   xml.Name `xml:"relation"`
	ID        int64    `xml:"id,attr"`
	Visible   bool     `xml:"visible,attr"`
	Version   int      `xml:"version,attr"`
	Changeset int64    `xml:"changeset,attr"`
	Timestamp string   `xml:"timestamp,attr"`
	User      string   `xml:"user,attr"`
	UID       int64    `xml:"uid,attr"`
	Members   []Member `xml:"member"`
	Tags      []Tag    `xml:"tag"`
}

// Member represents a member of a relation
type Member struct {
	XMLName xml.Name `xml:"member"`
	Type    string   `xml:"type,attr"`
	Ref     int64    `xml:"ref,attr"`
	Role    string   `xml:"role,attr"`
}

// Tag represents a key-value tag
type Tag struct {
	XMLName xml.Name `xml:"tag"`
	Key     string   `xml:"k,attr"`
	Value   string   `xml:"v,attr"`
}

// Coordinate represents a geographical coordinate
type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// AdminEntity represents an administrative entity with level information
type AdminEntity struct {
	Type         string `json:"type"`         // "province" or "commune"
	ID           int64  `json:"id"`           // OSM ID
	Name         string `json:"name"`         // Vietnamese name
	NameEn       string `json:"nameEn"`       // English name
	NameVi       string `json:"nameVi"`       // Vietnamese name with prefix
	AdminLevel   int    `json:"adminLevel"`   // admin_level from OSM
	CapitalLevel int    `json:"capitalLevel"` // capital level (4=province, 6=commune)
	Place        string `json:"place"`        // place type (town, city, etc.)
	Boundary     string `json:"boundary"`     // JSON string of boundary coordinates
}

// OSMProcessingResult contains processed OSM data
type OSMProcessingResult struct {
	BasicInfo       *BasicOSMInfo            `json:"basicInfo"`
	Boundaries      *BoundaryData            `json:"boundaries"`
	Administrative  map[string][]AdminEntity `json:"administrative"`
	CapitalStats    map[int]int              `json:"capitalStats"`
	JSONCoordinates string                   `json:"jsonCoordinates"`
}

// BasicOSMInfo contains basic OSM information
type BasicOSMInfo struct {
	Version    string  `json:"version"`
	Generator  string  `json:"generator"`
	TotalNodes int     `json:"totalNodes"`
	TotalWays  int     `json:"totalWays"`
	TotalRels  int     `json:"totalRelations"`
	Bounds     *Bounds `json:"bounds,omitempty"`
}

// Bounds represents geographical bounds
type Bounds struct {
	MinLat  float64 `json:"minLat"`
	MaxLat  float64 `json:"maxLat"`
	MinLon  float64 `json:"minLon"`
	MaxLon  float64 `json:"maxLon"`
	HasData bool    `json:"hasData"`
}

// BoundaryData contains boundary information
type BoundaryData struct {
	TotalCoordinates int                  `json:"totalCoordinates"`
	JSONString       string               `json:"jsonString"`
	FirstFiveCoords  []Coordinate         `json:"firstFiveCoords,omitempty"`
	Places           *PlacesData          `json:"places,omitempty"`
	AdminBoundaries  *AdminBoundariesData `json:"adminBoundaries,omitempty"`
}

// PlacesData contains place information
type PlacesData struct {
	PlaceNodes     int `json:"placeNodes"`
	PlaceRelations int `json:"placeRelations"`
}

// AdminBoundariesData contains administrative boundary information
type AdminBoundariesData struct {
	BoundaryWays      int `json:"boundaryWays"`
	BoundaryRelations int `json:"boundaryRelations"`
}

// GetTagValue returns the value of a tag by key, or empty string if not found
func (n *Node) GetTagValue(key string) string {
	for _, tag := range n.Tags {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

// GetTagValue returns the value of a tag by key, or empty string if not found
func (w *Way) GetTagValue(key string) string {
	for _, tag := range w.Tags {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

// GetTagValue returns the value of a tag by key, or empty string if not found
func (r *Relation) GetTagValue(key string) string {
	for _, tag := range r.Tags {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

// GetTimestamp returns the parsed timestamp as time.Time
func (n *Node) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, n.Timestamp)
}

// GetTimestamp returns the parsed timestamp as time.Time
func (w *Way) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, w.Timestamp)
}

// GetTimestamp returns the parsed timestamp as time.Time
func (r *Relation) GetTimestamp() (time.Time, error) {
	return time.Parse(time.RFC3339, r.Timestamp)
}

// IsAdministrativeBoundary checks if this way/relation is an administrative boundary
func (w *Way) IsAdministrativeBoundary() bool {
	boundary := w.GetTagValue("boundary")
	adminLevel := w.GetTagValue("admin_level")
	return boundary == "administrative" && adminLevel != ""
}

// IsAdministrativeBoundary checks if this relation is an administrative boundary
func (r *Relation) IsAdministrativeBoundary() bool {
	boundary := r.GetTagValue("boundary")
	adminLevel := r.GetTagValue("admin_level")
	return boundary == "administrative" && adminLevel != ""
}

// IsPlace checks if this node/relation is a place
func (n *Node) IsPlace() bool {
	return n.GetTagValue("place") != ""
}

// IsPlace checks if this relation is a place
func (r *Relation) IsPlace() bool {
	return r.GetTagValue("place") != ""
}

// GetName returns the name of the node/way/relation
func (n *Node) GetName() string {
	if name := n.GetTagValue("name"); name != "" {
		return name
	}
	if name := n.GetTagValue("name:vi"); name != "" {
		return name
	}
	if name := n.GetTagValue("name:en"); name != "" {
		return name
	}
	return ""
}

// GetName returns the name of the way
func (w *Way) GetName() string {
	if name := w.GetTagValue("name"); name != "" {
		return name
	}
	if name := w.GetTagValue("name:vi"); name != "" {
		return name
	}
	if name := w.GetTagValue("name:en"); name != "" {
		return name
	}
	return ""
}

// GetName returns the name of the relation
func (r *Relation) GetName() string {
	if name := r.GetTagValue("name"); name != "" {
		return name
	}
	if name := r.GetTagValue("name:vi"); name != "" {
		return name
	}
	if name := r.GetTagValue("name:en"); name != "" {
		return name
	}
	return ""
}

// GetAdminLevel returns the administrative level as integer
func (w *Way) GetAdminLevel() int {
	level := w.GetTagValue("admin_level")
	if level == "" {
		return -1
	}
	if adminLevel, err := strconv.Atoi(level); err == nil {
		return adminLevel
	}
	return -1
}

// GetAdminLevel returns the administrative level as integer
func (r *Relation) GetAdminLevel() int {
	level := r.GetTagValue("admin_level")
	if level == "" {
		return -1
	}
	if adminLevel, err := strconv.Atoi(level); err == nil {
		return adminLevel
	}
	return -1
}

// GetCapitalLevel returns the capital level as integer (4=tỉnh/tp, 6=xã)
func (n *Node) GetCapitalLevel() int {
	capital := n.GetTagValue("capital")
	if capital == "" {
		return -1
	}
	if capitalLevel, err := strconv.Atoi(capital); err == nil {
		return capitalLevel
	}
	return -1
}

// GetCapitalLevel returns the capital level as integer (4=tỉnh/tp, 6=xã)
func (r *Relation) GetCapitalLevel() int {
	capital := r.GetTagValue("capital")
	if capital == "" {
		return -1
	}
	if capitalLevel, err := strconv.Atoi(capital); err == nil {
		return capitalLevel
	}
	return -1
}

// IsProvinceOrCity checks if this is a province or city
func (n *Node) IsProvinceOrCity() bool {
	// Tỉnh/thành phố thường là Node với capital=4
	return n.GetCapitalLevel() == 4
}

// IsCommune checks if this is a commune/ward
func (n *Node) IsCommune() bool {
	// Commune/ward ít khi là Node, chủ yếu là Relation
	return n.GetCapitalLevel() == 6
}

// IsProvinceOrCity checks if this is a province or city
func (r *Relation) IsProvinceOrCity() bool {
	// Tỉnh/thành phố có thể là Relation với admin_level=4
	return r.GetAdminLevel() == 4
}

// IsCommune checks if this is a commune/ward
func (r *Relation) IsCommune() bool {
	// Commune/ward thường là Relation với admin_level=6
	return r.GetAdminLevel() == 6
}
