package util

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"

	"github.com/google/uuid"
	"github.com/weoses/memelo/telegram-service/entity"
)

const (
	typUint32 = iota
	typString
	typUuid
)

func writeStr(w io.Writer, s string) error {
	err := writeShort(w, uint16(len(s)))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

func writeByte(w io.Writer, b byte) error {
	return binary.Write(w, binary.LittleEndian, b)
}

func writeShort(w io.Writer, b uint16) error {
	return binary.Write(w, binary.LittleEndian, b)
}

func writeInt(buf io.Writer, i uint32) error {
	return binary.Write(buf, binary.LittleEndian, i)
}

func readStr(r io.Reader) (string, error) {
	n, err := readShort(r)
	if err != nil {
		return "", err
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func readShort(r io.Reader) (uint16, error) {
	var n uint16
	if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
		return 0, err
	}
	return n, nil
}

func readUUID(r io.Reader) (uuid.UUID, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return uuid.Nil, err
	}

	return uuid.FromBytes(buf)
}

func readInt(r io.Reader) (uint32, error) {
	var i uint32
	err := binary.Read(r, binary.LittleEndian, &i)
	return i, err
}
func readByte(r io.Reader) (uint8, error) {
	var b uint8
	if err := binary.Read(r, binary.LittleEndian, &b); err != nil {
		return b, err
	}
	return b, nil
}

func readTypObj(r io.Reader) (string, error) {
	typ, err := readByte(r)
	if err != nil {
		return "", err
	}
	switch typ {
	case typString:
		return readStr(r)
	case typUuid:
		u, err := readUUID(r)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	case typUint32:
		i, err := readInt(r)
		if err != nil {
			return "", err
		}
		return strconv.FormatUint(uint64(i), 10), nil
	default:
		return "", fmt.Errorf("invalid type %d", typ)
	}
}

func writeTypObj(w io.Writer, data string) error {
	asUuid, err := uuid.Parse(data)
	if err == nil {
		var binUuid []byte
		binUuid, err = asUuid.MarshalBinary()
		if err != nil {
			return err
		}
		err = writeByte(w, typUuid)
		if err != nil {
			return err
		}
		return binary.Write(w, binary.LittleEndian, binUuid)

	}
	asInt, err := strconv.ParseUint(data, 10, 32)
	if err == nil {
		asInt32 := uint32(asInt)
		err = writeByte(w, typUint32)
		if err != nil {
			return err
		}
		return writeInt(w, asInt32)
	}

	err = writeByte(w, typString)
	if err != nil {
		return err
	}
	return writeStr(w, data)
}
func ParseOffset(offset string) *entity.PaginationOffset {
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
	var listLen uint8
	if listLen, err = readByte(r); err != nil {
		return nil
	}
	sortingAfter := make([]string, listLen)
	for i := range listLen {
		v, err := readTypObj(r)
		if err != nil {
			return nil
		}
		sortingAfter[i] = v
	}
	return &entity.PaginationOffset{Searcher: searcher, SortingAfter: sortingAfter}
}

func SerializeOffset(p *entity.PaginationOffset) string {
	if p == nil || (p.Searcher == "" && len(p.SortingAfter) == 0) {
		return ""
	}

	var buf bytes.Buffer

	if writeStr(&buf, p.Searcher) != nil {
		return ""
	}

	if writeByte(&buf, uint8(len(p.SortingAfter))) != nil {
		return ""
	}

	for _, v := range p.SortingAfter {
		if writeTypObj(&buf, v) != nil {
			return ""
		}
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
