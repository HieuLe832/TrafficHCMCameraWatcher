package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- SESSION MANAGER ---

type SessionManager struct {
	client     *http.Client
	mu         sync.Mutex
	lastFetch  time.Time
	hasSession bool
}

func NewSessionManager() *SessionManager {
	jar, _ := cookiejar.New(nil)
	return &SessionManager{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

// EstablishSession visits map.aspx to get cookies
func (sm *SessionManager) EstablishSession() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// If session was established recently (within 5 minutes), reuse it
	if sm.hasSession && time.Since(sm.lastFetch) < 5*time.Minute {
		return nil
	}

	log.Println("Establishing session with map.aspx...")
	req, err := http.NewRequest("GET", "https://giaothong.hochiminhcity.gov.vn/map.aspx", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := sm.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to load map.aspx: status %d", resp.StatusCode)
	}

	sm.lastFetch = time.Now()
	sm.hasSession = true
	log.Println("Session established successfully.")
	return nil
}

func (sm *SessionManager) GetClient() *http.Client {
	return sm.client
}

var session = NewSessionManager()

// --- AJAXPRO DATATABLE PARSER ---

type Parser struct {
	src []rune
	pos int
	len int
}

func NewParser(s string) *Parser {
	runes := []rune(s)
	return &Parser{
		src: runes,
		pos: 0,
		len: len(runes),
	}
}

func (p *Parser) skipWhitespace() {
	for p.pos < p.len {
		r := p.src[p.pos]
		if r == ' ' || r == '\t' || r == '\r' || r == '\n' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *Parser) peek() rune {
	if p.pos >= p.len {
		return 0
	}
	return p.src[p.pos]
}

func (p *Parser) next() rune {
	if p.pos >= p.len {
		return 0
	}
	r := p.src[p.pos]
	p.pos++
	return r
}

func (p *Parser) parseValue() (interface{}, error) {
	p.skipWhitespace()
	r := p.peek()
	if r == 0 {
		return nil, errors.New("unexpected EOF")
	}

	// Object: { ... }
	if r == '{' {
		return p.parseObject()
	}
	// Array: [ ... ]
	if r == '[' {
		return p.parseArray()
	}
	// String: "..." or '...'
	if r == '"' || r == '\'' {
		return p.parseString()
	}
	// DataTable constructor: new Ajax.Web.DataTable(...)
	if r == 'n' {
		if p.pos+3 <= p.len && string(p.src[p.pos:p.pos+3]) == "new" {
			return p.parseNewExpression()
		}
		return p.parseLiteral()
	}
	// Number
	if (r >= '0' && r <= '9') || r == '-' || r == '.' {
		return p.parseNumber()
	}
	// Literals: true, false, null
	return p.parseLiteral()
}

func (p *Parser) parseObject() (interface{}, error) {
	p.next() // consume '{'
	obj := make(map[string]interface{})

	for {
		p.skipWhitespace()
		if p.peek() == '}' {
			p.next()
			return obj, nil
		}

		// Parse key (must be a string or identifier)
		p.skipWhitespace()
		keyR := p.peek()
		var key string
		var err error
		if keyR == '"' || keyR == '\'' {
			val, err := p.parseString()
			if err != nil {
				return nil, err
			}
			key = val.(string)
		} else {
			// Bare identifier key
			var sb strings.Builder
			for p.pos < p.len {
				curr := p.peek()
				if (curr >= 'a' && curr <= 'z') || (curr >= 'A' && curr <= 'Z') || (curr >= '0' && curr <= '9') || curr == '_' {
					sb.WriteRune(p.next())
				} else {
					break
				}
			}
			key = sb.String()
			if key == "" {
				return nil, fmt.Errorf("expected object key at position %d", p.pos)
			}
		}

		p.skipWhitespace()
		if p.next() != ':' {
			return nil, fmt.Errorf("expected ':' after object key at position %d", p.pos)
		}

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		obj[key] = val

		p.skipWhitespace()
		nextR := p.next()
		if nextR == '}' {
			return obj, nil
		}
		if nextR != ',' {
			return nil, fmt.Errorf("expected ',' or '}' in object at position %d, got %c", p.pos, nextR)
		}
	}
}

func (p *Parser) parseArray() (interface{}, error) {
	p.next() // consume '['
	arr := make([]interface{}, 0)

	for {
		p.skipWhitespace()
		if p.peek() == ']' {
			p.next()
			return arr, nil
		}

		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		arr = append(arr, val)

		p.skipWhitespace()
		nextR := p.next()
		if nextR == ']' {
			return arr, nil
		}
		if nextR != ',' {
			return nil, fmt.Errorf("expected ',' or ']' in array at position %d, got %c", p.pos, nextR)
		}
	}
}

func (p *Parser) parseString() (interface{}, error) {
	quote := p.next() // consume quote char
	var sb strings.Builder

	for p.pos < p.len {
		r := p.next()
		if r == quote {
			return sb.String(), nil
		}
		if r == '\\' {
			if p.pos >= p.len {
				return nil, errors.New("unexpected EOF in string escape sequence")
			}
			esc := p.next()
			switch esc {
			case '"':
				sb.WriteByte('"')
			case '\'':
				sb.WriteByte('\'')
			case '\\':
				sb.WriteByte('\\')
			case '/':
				sb.WriteByte('/')
			case 'b':
				sb.WriteByte('\b')
			case 'f':
				sb.WriteByte('\f')
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'u':
				// Read 4 hex chars
				if p.pos+4 > p.len {
					return nil, errors.New("unexpected EOF in hex string escape")
				}
				hexStr := string(p.src[p.pos : p.pos+4])
				p.pos += 4
				val, err := strconv.ParseUint(hexStr, 16, 32)
				if err != nil {
					return nil, err
				}
				sb.WriteRune(rune(val))
			default:
				sb.WriteRune(esc)
			}
		} else {
			sb.WriteRune(r)
		}
	}
	return nil, errors.New("unclosed string literal")
}

func (p *Parser) parseNumber() (interface{}, error) {
	var sb strings.Builder
	for p.pos < p.len {
		r := p.peek()
		if (r >= '0' && r <= '9') || r == '-' || r == '+' || r == '.' || r == 'e' || r == 'E' {
			sb.WriteRune(p.next())
		} else {
			break
		}
	}
	numStr := sb.String()
	if strings.Contains(numStr, ".") || strings.Contains(numStr, "e") || strings.Contains(numStr, "E") {
		return strconv.ParseFloat(numStr, 64)
	}
	val, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		// Fallback to float
		return strconv.ParseFloat(numStr, 64)
	}
	return val, nil
}

func (p *Parser) parseLiteral() (interface{}, error) {
	var sb strings.Builder
	for p.pos < p.len {
		r := p.peek()
		if r >= 'a' && r <= 'z' {
			sb.WriteRune(p.next())
		} else {
			break
		}
	}
	lit := sb.String()
	if lit == "null" {
		return nil, nil
	}
	if lit == "true" {
		return true, nil
	}
	if lit == "false" {
		return false, nil
	}
	return nil, fmt.Errorf("unknown literal %s at position %d", lit, p.pos)
}

func (p *Parser) parseNewExpression() (interface{}, error) {
	// Expect "new Ajax.Web.DataTable("
	const prefix = "new Ajax.Web.DataTable"
	var sb strings.Builder
	for i := 0; i < len(prefix); i++ {
		sb.WriteRune(p.next())
	}
	if sb.String() != prefix {
		return nil, fmt.Errorf("expected 'new Ajax.Web.DataTable' at position %d", p.pos-len(prefix))
	}

	p.skipWhitespace()
	if p.next() != '(' {
		return nil, fmt.Errorf("expected '(' in new expression at position %d", p.pos)
	}

	// Parse columns argument: [[colName, colType], ...]
	columnsVal, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()
	if p.next() != ',' {
		return nil, fmt.Errorf("expected ',' after columns in DataTable at position %d", p.pos)
	}

	// Parse rows argument: [[val1, val2, ...], ...]
	rowsVal, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	p.skipWhitespace()
	if p.next() != ')' {
		return nil, fmt.Errorf("expected ')' at end of DataTable constructor at position %d", p.pos)
	}

	// Transform columns and rows into a list of map records
	columnsArr, ok1 := columnsVal.([]interface{})
	rowsArr, ok2 := rowsVal.([]interface{})
	if !ok1 || !ok2 {
		return nil, errors.New("DataTable arguments must be arrays")
	}

	// Extract column names
	colNames := make([]string, len(columnsArr))
	for i, col := range columnsArr {
		colPair, ok := col.([]interface{})
		if ok && len(colPair) > 0 {
			if nameStr, ok := colPair[0].(string); ok {
				colNames[i] = nameStr
			}
		}
	}

	// Map each row to a dictionary
	records := make([]map[string]interface{}, 0, len(rowsArr))
	for _, row := range rowsArr {
		rowVals, ok := row.([]interface{})
		if !ok {
			continue
		}
		record := make(map[string]interface{})
		for i, colName := range colNames {
			if i < len(rowVals) && colName != "" {
				record[colName] = rowVals[i]
			}
		}
		records = append(records, record)
	}

	return records, nil
}

// --- CONTROLLER HANDLERS ---

type CameraResponse struct {
	CamId      string      `json:"id"`
	Code       string      `json:"code"`
	Name       string      `json:"name"`
	Latitude   float64     `json:"lat"`
	Longitude  float64     `json:"lng"`
	Status     string      `json:"status"`
	Angle      interface{} `json:"angle"`
	Streaming  bool        `json:"streaming"`
	VideoUrl   string      `json:"videoUrl,omitempty"`
}

func getCamerasHandler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		return
	}

	// Establish session
	err := session.EstablishSession()
	if err != nil {
		log.Println("Session establishment error:", err)
		http.Error(w, "Failed to establish backend session", http.StatusInternalServerError)
		return
	}

	// POST to retrieve camera list
	url := "https://giaothong.hochiminhcity.gov.vn/ajaxpro/VDMS.Web.Library.AJAX.FolderAjax,VDMS.Web.Library.ashx"
	postBody := []byte(`{
		"path": "/root/vdms/tangthu/data/layerdata/camera",
		"isInTree": false,
		"searchKey": "",
		"layer": ["CAMERA"],
		"detail": true,
		"page": 0,
		"limit": -1,
		"filterQuery": ["Publish:true"],
		"sortby": null,
		"returnFields": ["CamId", "Code", "Location", "SnapshotUrl", "CamType", "Disctrict", "Publish", "ManagementUnit", "CamStatus", "PTZ", "Angle", "DisplayName", "VideoUrl", "VideoStreaming"]
	}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("X-AjaxPro-Method", "SearchQuery")
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Referer", "https://giaothong.hochiminhcity.gov.vn/map.aspx")

	resp, err := session.GetClient().Do(req)
	if err != nil {
		log.Println("POST request error:", err)
		http.Error(w, "Remote server request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("POST response status:", resp.StatusCode)
		http.Error(w, fmt.Sprintf("Remote server returned status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	bodyStr := string(bodyBytes)

	// Check if session expired or unauthenticated
	if strings.Contains(bodyStr, "User is not authenticated") {
		log.Println("Session expired. Re-establishing and retrying...")
		session.mu.Lock()
		session.hasSession = false // Force session refresh
		session.mu.Unlock()

		err = session.EstablishSession()
		if err != nil {
			http.Error(w, "Failed to refresh session", http.StatusInternalServerError)
			return
		}

		// Retry POST request once
		req2, _ := http.NewRequest("POST", url, bytes.NewBuffer(postBody))
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req2.Header.Set("X-AjaxPro-Method", "SearchQuery")
		req2.Header.Set("Content-Type", "text/plain; charset=utf-8")
		req2.Header.Set("Referer", "https://giaothong.hochiminhcity.gov.vn/map.aspx")

		resp2, err := session.GetClient().Do(req2)
		if err != nil {
			http.Error(w, "Retry request failed", http.StatusBadGateway)
			return
		}
		defer resp2.Body.Close()
		bodyBytes, err = io.ReadAll(resp2.Body)
		if err != nil {
			http.Error(w, "Failed to read retry response", http.StatusInternalServerError)
			return
		}
		bodyStr = string(bodyBytes)
	}

	// Parse custom JavaScript/DataTable format
	parser := NewParser(bodyStr)
	parsedData, err := parser.parseValue()
	if err != nil {
		log.Println("Parsing error:", err)
		http.Error(w, "Failed to parse camera metadata payload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract cameras list
	dataMap, ok := parsedData.(map[string]interface{})
	if !ok {
		http.Error(w, "Invalid response payload format", http.StatusInternalServerError)
		return
	}

	valArr, ok := dataMap["value"].([]interface{})
	if !ok || len(valArr) < 2 {
		http.Error(w, "Missing camera list in response", http.StatusInternalServerError)
		return
	}

	camsAndMeta, ok := valArr[1].([]interface{})
	if !ok {
		http.Error(w, "Missing camera payload list", http.StatusInternalServerError)
		return
	}

	// Find the camera list DataTable (which translates to a slice of map[string]interface{})
	var camerasRaw []map[string]interface{}
	for _, item := range camsAndMeta {
		if rawList, ok := item.([]map[string]interface{}); ok {
			camerasRaw = rawList
			break
		}
	}

	if camerasRaw == nil {
		http.Error(w, "No DataTable found in camera payload", http.StatusInternalServerError)
		return
	}

	// Clean and format output cameras
	cleanCameras := make([]CameraResponse, 0, len(camerasRaw))
	for _, cam := range camerasRaw {
		id, _ := cam["CamId"].(string)
		code, _ := cam["Code"].(string)
		displayName, _ := cam["DisplayName"].(string)
		if displayName == "" {
			displayName, _ = cam["Title"].(string)
		}
		status, _ := cam["CamStatus"].(string)
		angle := cam["Angle"]
		streamingVal := cam["VideoStreaming"]
		streaming := false
		if fVal, ok := streamingVal.(float64); ok && fVal == 1 {
			streaming = true
		} else if iVal, ok := streamingVal.(int64); ok && iVal == 1 {
			streaming = true
		}
		videoUrl, _ := cam["VideoUrl"].(string)

		// Parse coordinates from Location DataTable
		var lat, lng float64
		if locList, ok := cam["Location"].([]map[string]interface{}); ok && len(locList) > 0 {
			shape, _ := locList[0]["Shape"].(string)
			// shape is like "POINT(106.691054105759 10.7918902432446)"
			shape = strings.Replace(shape, "POINT(", "", 1)
			shape = strings.Replace(shape, ")", "", 1)
			coords := strings.Split(shape, " ")
			if len(coords) >= 2 {
				lng, _ = strconv.ParseFloat(coords[0], 64)
				lat, _ = strconv.ParseFloat(coords[1], 64)
			}
		}

		// Only include cameras with coordinates
		if lat != 0 && lng != 0 {
			cleanCameras = append(cleanCameras, CameraResponse{
				CamId:     id,
				Code:      code,
				Name:      displayName,
				Latitude:  lat,
				Longitude: lng,
				Status:    status,
				Angle:     angle,
				Streaming: streaming,
				VideoUrl:  videoUrl,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(cleanCameras)
}

func getCameraImageHandler(w http.ResponseWriter, r *http.Request) {
	// Enable CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	if r.Method == "OPTIONS" {
		return
	}

	camId := r.URL.Query().Get("id")
	if camId == "" {
		http.Error(w, "Missing 'id' parameter", http.StatusBadRequest)
		return
	}

	width := r.URL.Query().Get("w")
	if width == "" {
		width = "300"
	}
	height := r.URL.Query().Get("h")
	if height == "" {
		height = "230"
	}

	// Establish session
	err := session.EstablishSession()
	if err != nil {
		http.Error(w, "Failed to establish session", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("https://giaothong.hochiminhcity.gov.vn:8007/Render/CameraHandler.ashx?id=%s&w=%s&h=%s&bg=black", camId, width, height)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://giaothong.hochiminhcity.gov.vn/map.aspx")

	resp, err := session.GetClient().Do(req)
	if err != nil {
		log.Println("Image proxy request error:", err)
		http.Error(w, "Remote server request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		log.Println("Image request returned 403 Forbidden. Refreshing session and retrying...")
		session.mu.Lock()
		session.hasSession = false // Force session refresh
		session.mu.Unlock()

		err = session.EstablishSession()
		if err != nil {
			http.Error(w, "Failed to refresh session", http.StatusInternalServerError)
			return
		}

		req2, _ := http.NewRequest("GET", url, nil)
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req2.Header.Set("Referer", "https://giaothong.hochiminhcity.gov.vn/map.aspx")

		resp2, err := session.GetClient().Do(req2)
		if err != nil {
			http.Error(w, "Retry request failed", http.StatusBadGateway)
			return
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusOK {
			http.Error(w, fmt.Sprintf("Remote server returned status %d on retry", resp2.StatusCode), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", resp2.Header.Get("Content-Type"))
		io.Copy(w, resp2.Body)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Remote server returned status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	io.Copy(w, resp.Body)
}

func main() {
	http.HandleFunc("/api/cameras", getCamerasHandler)
	http.HandleFunc("/api/camera/image", getCameraImageHandler)

	log.Println("Starting proxy server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
