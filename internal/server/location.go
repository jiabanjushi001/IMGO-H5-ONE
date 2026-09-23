package server

import (
	"encoding/binary"
	"github.com/mozillazg/go-pinyin"
	"net"
	"os"
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
	return &ipDatabase{b, off}, nil
}
func (d *ipDatabase) find(ip string) string {
	if d == nil {
		return ""
	}
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
func (a *App) location(ip any) string { return a.ipdb.find(str(ip)) }
