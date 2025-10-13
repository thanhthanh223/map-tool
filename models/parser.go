package models

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

// ParseOSMFromFile parses an OSM XML file from the given file path
func ParseOSMFromFile(filename string) (*OSM, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	return ParseOSMFromReader(file)
}

// ParseOSMFromReader parses OSM XML data from an io.Reader
func ParseOSMFromReader(reader io.Reader) (*OSM, error) {
	var osm OSM
	decoder := xml.NewDecoder(reader)

	err := decoder.Decode(&osm)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OSM XML: %w", err)
	}

	return &osm, nil
}

// ParseOSMFromBytes parses OSM XML data from a byte slice
func ParseOSMFromBytes(data []byte) (*OSM, error) {
	var osm OSM
	err := xml.Unmarshal(data, &osm)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal OSM XML: %w", err)
	}

	return &osm, nil
}

// FilterNodesByTag filters nodes by a specific tag key-value pair
func (osm *OSM) FilterNodesByTag(key, value string) []Node {
	var filtered []Node
	for _, node := range osm.Nodes {
		if node.GetTagValue(key) == value {
			filtered = append(filtered, node)
		}
	}
	return filtered
}

// FilterWaysByTag filters ways by a specific tag key-value pair
func (osm *OSM) FilterWaysByTag(key, value string) []Way {
	var filtered []Way
	for _, way := range osm.Ways {
		if way.GetTagValue(key) == value {
			filtered = append(filtered, way)
		}
	}
	return filtered
}

// FilterRelationsByTag filters relations by a specific tag key-value pair
func (osm *OSM) FilterRelationsByTag(key, value string) []Relation {
	var filtered []Relation
	for _, relation := range osm.Relations {
		if relation.GetTagValue(key) == value {
			filtered = append(filtered, relation)
		}
	}
	return filtered
}

// GetAdministrativeBoundaries returns all administrative boundary ways and relations
func (osm *OSM) GetAdministrativeBoundaries() ([]Way, []Relation) {
	var boundaryWays []Way
	var boundaryRelations []Relation

	for _, way := range osm.Ways {
		if way.IsAdministrativeBoundary() {
			boundaryWays = append(boundaryWays, way)
		}
	}

	for _, relation := range osm.Relations {
		if relation.IsAdministrativeBoundary() {
			boundaryRelations = append(boundaryRelations, relation)
		}
	}

	return boundaryWays, boundaryRelations
}

// GetPlaces returns all place nodes and relations
func (osm *OSM) GetPlaces() ([]Node, []Relation) {
	var placeNodes []Node
	var placeRelations []Relation

	for _, node := range osm.Nodes {
		if node.IsPlace() {
			placeNodes = append(placeNodes, node)
		}
	}

	for _, relation := range osm.Relations {
		if relation.IsPlace() {
			placeRelations = append(placeRelations, relation)
		}
	}

	return placeNodes, placeRelations
}

// FindNodeByID finds a node by its ID
func (osm *OSM) FindNodeByID(id int64) (*Node, bool) {
	for _, node := range osm.Nodes {
		if node.ID == id {
			return &node, true
		}
	}
	return nil, false
}

// FindWayByID finds a way by its ID
func (osm *OSM) FindWayByID(id int64) (*Way, bool) {
	for _, way := range osm.Ways {
		if way.ID == id {
			return &way, true
		}
	}
	return nil, false
}

// FindRelationByID finds a relation by its ID
func (osm *OSM) FindRelationByID(id int64) (*Relation, bool) {
	for _, relation := range osm.Relations {
		if relation.ID == id {
			return &relation, true
		}
	}
	return nil, false
}

// GetNodesInWay returns all nodes that are referenced in a way
func (osm *OSM) GetNodesInWay(way *Way) []Node {
	var nodes []Node
	for _, nodeRef := range way.Nodes {
		if node, found := osm.FindNodeByID(nodeRef.Ref); found {
			nodes = append(nodes, *node)
		}
	}
	return nodes
}

// GetWayCoordinates returns coordinates of all nodes in a way
func (osm *OSM) GetWayCoordinates(way *Way) []Coordinate {
	var coordinates []Coordinate
	for _, nodeRef := range way.Nodes {
		if node, found := osm.FindNodeByID(nodeRef.Ref); found {
			coordinates = append(coordinates, Coordinate{
				ID:  node.ID, // Include OSM Node ID
				Lat: node.Lat,
				Lon: node.Lon,
			})
		}
	}
	return coordinates
}

// GetBounds returns the bounding box of all nodes in the OSM data
func (osm *OSM) GetBounds() (minLat, maxLat, minLon, maxLon float64, hasData bool) {
	if len(osm.Nodes) == 0 {
		return 0, 0, 0, 0, false
	}

	minLat = osm.Nodes[0].Lat
	maxLat = osm.Nodes[0].Lat
	minLon = osm.Nodes[0].Lon
	maxLon = osm.Nodes[0].Lon

	for _, node := range osm.Nodes {
		if node.Lat < minLat {
			minLat = node.Lat
		}
		if node.Lat > maxLat {
			maxLat = node.Lat
		}
		if node.Lon < minLon {
			minLon = node.Lon
		}
		if node.Lon > maxLon {
			maxLon = node.Lon
		}
	}

	return minLat, maxLat, minLon, maxLon, true
}
