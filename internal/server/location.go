package server

import (
	"encoding/binary"
	"github.com/mozillazg/go-pinyin"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func namePinyin(name string) string {
	args := pinyin.NewArgs()
	args.Fallback = func(r rune, _ pinyin.Args) []string { return []string{string(r)} }
	parts := pinyin.Pinyin(name, args)
	var b strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			b.WriteString(p[0])
		}
	}
	s := []rune(b.String())
	if len(s) > 64 {
		s = s[:64]
	}
	return string(s)
}

type ipDatabase struct {
	data   []byte
	offset uint32
	qqzeng *qqZengDatabase
}

type qqZengPrefix struct {
	start uint32
	end   uint32
}

type qqZengDatabase struct {
	data       []byte
	firstIndex uint32
	prefixes   map[byte]qqZengPrefix
}

func loadIPDatabase(path string) (*ipDatabase, error) {
	if path == "" {
		return nil, nil
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	if len(b) < 1028 {
		return nil, clientError{"IP 数据库无效", 500}
	}
	off := binary.BigEndian.Uint32(b[:4])
	if off < 1028 || int(off) > len(b) {
		return nil, clientError{"IP 数据库索引无效", 500}
	}
	database := &ipDatabase{data: b, offset: off}
	qqzengPath := filepath.Join(filepath.Dir(path), "qqzeng-ip-utf8.dat")
	if qqzeng, qqzengErr := loadQQZengDatabase(qqzengPath); qqzengErr == nil {
		database.qqzeng = qqzeng
	}
	return database, nil
}

func loadQQZengDatabase(path string) (*qqZengDatabase, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 16 {
		return nil, clientError{"QQZeng IP 数据库无效", 500}
	}
	firstIndex := binary.LittleEndian.Uint32(b[0:4])
	prefixStart := binary.LittleEndian.Uint32(b[8:12])
	prefixEnd := binary.LittleEndian.Uint32(b[12:16])
	if firstIndex >= uint32(len(b)) || prefixStart >= uint32(len(b)) || prefixEnd < prefixStart || uint64(prefixEnd)+9 > uint64(len(b)) {
		return nil, clientError{"QQZeng IP 数据库索引无效", 500}
	}
	prefixes := make(map[byte]qqZengPrefix, 256)
	for pos := prefixStart; pos <= prefixEnd; pos += 9 {
		prefixes[b[pos]] = qqZengPrefix{
			start: binary.LittleEndian.Uint32(b[pos+1 : pos+5]),
			end:   binary.LittleEndian.Uint32(b[pos+5 : pos+9]),
		}
		if prefixEnd-pos < 9 {
			break
		}
	}
	return &qqZengDatabase{data: b, firstIndex: firstIndex, prefixes: prefixes}, nil
}

func (d *qqZengDatabase) find(ip string) string {
	if d == nil {
		return ""
	}
	v := net.ParseIP(ip).To4()
	if v == nil || v[0] == 127 || v[0] == 10 || (v[0] == 192 && v[1] == 168) || (v[0] == 172 && v[1] >= 16 && v[1] <= 31) {
		return ""
	}
	prefix, ok := d.prefixes[v[0]]
	if !ok || prefix.start > prefix.end {
		return ""
	}
	target := binary.BigEndian.Uint32(v)
	low, high := prefix.start, prefix.end
	match := prefix.start
	for low <= high {
		mid := low + (high-low)/2
		pos := uint64(d.firstIndex) + uint64(mid)*12
		if pos+12 > uint64(len(d.data)) {
			return ""
		}
		endIP := binary.LittleEndian.Uint32(d.data[pos+4 : pos+8])
		if endIP >= target {
			match = mid
			if mid == 0 {
				break
			}
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	pos := uint64(d.firstIndex) + uint64(match)*12
	if pos+12 > uint64(len(d.data)) {
		return ""
	}
	startIP := binary.LittleEndian.Uint32(d.data[pos : pos+4])
	endIP := binary.LittleEndian.Uint32(d.data[pos+4 : pos+8])
	if target < startIP || target > endIP {
		return ""
	}
	offset := uint32(d.data[pos+8]) | uint32(d.data[pos+9])<<8 | uint32(d.data[pos+10])<<16
	length := uint32(d.data[pos+11])
	if uint64(offset)+uint64(length) > uint64(len(d.data)) {
		return ""
	}
	return string(d.data[offset : offset+length])
}

func (d *ipDatabase) find(ip string) string {
	if d == nil {
		return ""
	}
	if location := formatQQZengLocation(d.qqzeng.find(ip)); location != "" {
		return location
	}
	return formatLegacyLocation(d.findLegacy(ip))
}

func (d *ipDatabase) findLegacy(ip string) string {
	v := net.ParseIP(ip).To4()
	if v == nil {
		return ""
	}
	target := binary.BigEndian.Uint32(v)
	start := int(binary.LittleEndian.Uint32(d.data[4+int(v[0])*4:]))*8 + 1024
	index := d.data[4:d.offset]
	for pos := start; pos+8 <= len(index)-1024; pos += 8 {
		if binary.BigEndian.Uint32(index[pos:]) >= target {
			offset := int(index[pos+4]) | int(index[pos+5])<<8 | int(index[pos+6])<<16
			length := int(index[pos+7])
			begin := int(d.offset) + offset - 1024
			if begin < 0 || begin+length > len(d.data) {
				return ""
			}
			return strings.Join(strings.Fields(string(d.data[begin:begin+length])), " ")
		}
	}
	return ""
}

func formatQQZengLocation(raw string) string {
	parts := strings.Split(raw, "|")
	if len(parts) < 4 {
		return ""
	}
	country := strings.TrimSpace(parts[1])
	province := strings.TrimSpace(parts[2])
	city := strings.TrimSpace(parts[3])
	if country == "保留" {
		return ""
	}
	if country == "中国" {
		return joinLocationParts(province, city)
	}
	return joinLocationParts(country, province, city)
}

func formatLegacyLocation(raw string) string {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	if parts[0] == "中国" && len(parts) > 1 {
		parts = parts[1:]
	}
	return joinLocationParts(parts...)
}

func joinLocationParts(parts ...string) string {
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || (len(result) > 0 && result[len(result)-1] == part) {
			continue
		}
		result = append(result, part)
	}
	return strings.Join(result, "-")
}

func (a *App) location(ip any) string { return a.ipdb.find(str(ip)) }
