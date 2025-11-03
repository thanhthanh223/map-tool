package util

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Hàm trả về một điểm (lat, lon) trung tâm nằm bên trong polygon
// polygon: slice các [2]float64, mỗi phần tử là [lat, lon]
func PolygonInteriorCentroid(polygon [][2]float64) (float64, float64) {
	if len(polygon) < 3 {
		return 0, 0 // không hợp lệ
	}

	// Tìm centroid hình học (coi như diện tích đều)
	var area float64
	var cx, cy float64
	for i := 0; i < len(polygon); i++ {
		j := (i + 1) % len(polygon)
		x0 := polygon[i][1]
		y0 := polygon[i][0]
		x1 := polygon[j][1]
		y1 := polygon[j][0]

		a := x0*y1 - x1*y0
		area += a
		cx += (x0 + x1) * a
		cy += (y0 + y1) * a
	}
	area *= 0.5
	if area == 0 {
		// không phải polygon thực sự, trả giá trị đầu vào
		return polygon[0][0], polygon[0][1]
	}
	cx /= (6 * area)
	cy /= (6 * area)

	centroidLat := cy
	centroidLon := cx

	// Kiểm tra centroid có nằm trong polygon không
	if pointInPolygon(centroidLat, centroidLon, polygon) {
		return centroidLat, centroidLon
	}

	// Nếu không nằm trong polygon, lùi dần về trọng tâm có thể điều chỉnh (Lerp từ centroid tới điểm đầu đến khi vào polygon)
	const stepCount = 20
	for t := 0.95; t >= 0; t -= 1.0 / stepCount {
		testLat := t*centroidLat + (1-t)*polygon[0][0]
		testLon := t*centroidLon + (1-t)*polygon[0][1]
		if pointInPolygon(testLat, testLon, polygon) {
			return testLat, testLon
		}
	}

	// Nếu vẫn không ổn, trả về một điểm đầu
	return polygon[0][0], polygon[0][1]
}

// Hàm kiểm tra một điểm có nằm trong polygon hay không
func pointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	n := len(polygon)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		yi, xi := polygon[i][0], polygon[i][1]
		yj, xj := polygon[j][0], polygon[j][1]
		intersect := ((xi > lon) != (xj > lon)) &&
			(lat < (yj-yi)*(lon-xi)/(xj-xi+1e-14)+yi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

// RemoveVietnameseAccent loại bỏ dấu tiếng Việt, tuy nhiên chữ "phường bảy" (với "bảy" bị viết là "bảy" sử dụng Unicode tổ hợp, ký tự 'a' + dấu '̉') sẽ không xử lý được với map rune2rune
// Giải pháp: chuẩn hóa Unicode về dạng NFC->NFD, sau đó loại bỏ các ký tự dấu (Mn: nonspacing mark)

func RemoveVietnameseAccent(s string) string {
	// Chuẩn hóa các ký tự gạch nối về dạng "-"
	normalizeDash := func(ss string) string {
		replacer := strings.NewReplacer("–", "-", "—", "-", "―", "-")
		return replacer.Replace(ss)
	}
	s = normalizeDash(s)

	// Normalize to decomposed form (NFD)
	normStr := norm.NFD.String(s)
	out := make([]rune, 0, len(normStr))
	for _, r := range normStr {
		if unicode.Is(unicode.Mn, r) {
			// Bỏ qua các non-spacing mark (dấu sắc, huyền,...)
			continue
		}
		// chuyển đ sang d, Đ sang D
		if r == 'đ' {
			out = append(out, 'd')
		} else if r == 'Đ' {
			out = append(out, 'D')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
