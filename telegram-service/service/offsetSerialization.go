package service

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"

	"github.com/weoses/memelo/telegram-service/entity"
)

func writeStr(w io.Writer, s string) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(s))); err != nil {
		return err
	}
	_, err := io.WriteString(w, s)
	return err
}

func writeInt(buf io.Writer, i uint32) error {
	return binary.Write(buf, binary.LittleEndian, i)
}

func readStr(r io.Reader) (string, error) {
	var n uint32
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return "", err
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func readInt(r io.Reader) (uint32, error) {
	var i uint32
	err := binary.Read(r, binary.LittleEndian, &i)
	return i, err
}

func parseOffset(offset string) *entity.PaginationOffset {
	if offset == "" {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(offset)
	if err != nil {
		return nil
	}
	r := bytes.NewReader(raw)
	searcher, err := readStr(r)
	if err != nil {
		return nil
	}
	var listLen uint32
	if listLen, err = readInt(r); err != nil {
		return nil
	}
	sortingAfter := make([]string, listLen)
	for i := range listLen {
		v, err := readStr(r)
		if err != nil {
			return nil
		}
		sortingAfter[i] = v
	}
	return &entity.PaginationOffset{Searcher: searcher, SortingAfter: sortingAfter}
}

func serializeOffset(p *entity.PaginationOffset) string {
	if p == nil || (p.Searcher == "" && len(p.SortingAfter) == 0) {
		return ""
	}

	var buf bytes.Buffer

	if writeStr(&buf, p.Searcher) != nil {
		return ""
	}

	if writeInt(&buf, uint32(len(p.SortingAfter))) != nil {
		return ""
	}

	for _, v := range p.SortingAfter {
		if writeStr(&buf, v) != nil {
			return ""
		}
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
