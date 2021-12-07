package utils

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"

	"golang.org/x/text/encoding/charmap"
)

var charMaps = map[string]encoding.Encoding{
	"codepage037":       charmap.CodePage037,
	"codepage1047":      charmap.CodePage1047,
	"codepage1140":      charmap.CodePage1140,
	"codepage437":       charmap.CodePage437,
	"codepage850":       charmap.CodePage850,
	"codepage852":       charmap.CodePage852,
	"codepage855":       charmap.CodePage855,
	"codepage858":       charmap.CodePage858,
	"codepage860":       charmap.CodePage860,
	"codepage862":       charmap.CodePage862,
	"codepage863":       charmap.CodePage863,
	"codepage865":       charmap.CodePage865,
	"codepage866":       charmap.CodePage866,
	"iso8859_1":         charmap.ISO8859_1,
	"iso8859_10":        charmap.ISO8859_10,
	"iso8859_13":        charmap.ISO8859_13,
	"iso8859_14":        charmap.ISO8859_14,
	"iso8859_15":        charmap.ISO8859_15,
	"iso8859_16":        charmap.ISO8859_16,
	"iso8859_2":         charmap.ISO8859_2,
	"iso8859_3":         charmap.ISO8859_3,
	"iso8859_4":         charmap.ISO8859_4,
	"iso8859_5":         charmap.ISO8859_5,
	"iso8859_6":         charmap.ISO8859_6,
	"iso8859_7":         charmap.ISO8859_7,
	"iso8859_8":         charmap.ISO8859_8,
	"iso8859_9":         charmap.ISO8859_9,
	"koi8r":             charmap.KOI8R,
	"koi8u":             charmap.KOI8U,
	"macintosh":         charmap.Macintosh,
	"macintoshcyrillic": charmap.MacintoshCyrillic,
	"windows1250":       charmap.Windows1250,
	"windows1251":       charmap.Windows1251,
	"windows1252":       charmap.Windows1252,
	"windows1253":       charmap.Windows1253,
	"windows1254":       charmap.Windows1254,
	"windows1255":       charmap.Windows1255,
	"windows1256":       charmap.Windows1256,
	"windows1257":       charmap.Windows1257,
	"windows1258":       charmap.Windows1258,
	"windows874":        charmap.Windows874,
	"gb18030":           simplifiedchinese.GB18030,
	"gbk":               simplifiedchinese.GBK,
	"big5":              traditionalchinese.Big5,
	"eucjp":             japanese.EUCJP,
	"iso2022jp":         japanese.ISO2022JP,
	"shiftJIS":          japanese.ShiftJIS,
	"euckr":             korean.EUCKR,
	"utf16be":           unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
	"utf16le":           unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf8":              nil,
	"utf-8":             nil,
}

func Encode(encodingName, contentsUtf8 string) ([]byte, error) {
	cm, err := DetectCharMap(encodingName)
	if err != nil {
		return []byte{}, err
	}

	if cm == nil {
		return []byte(contentsUtf8), nil
	}

	enc := cm.NewEncoder()
	out, err := enc.String(contentsUtf8)

	return []byte(out), err
}

func Decode(encodingName string, data []byte) (string, error) {
	cm, err := DetectCharMap(encodingName)
	if err != nil {
		return "", err
	}

	if cm == nil {
		return string(data), nil
	}

	enc := cm.NewDecoder()
	out, err := enc.Bytes(data)

	return string(out), err
}

func DetectCharMap(encodingName string) (encoding.Encoding, error) {
	cm, ok := charMaps[strings.ToLower(encodingName)]
	if !ok {
		return nil, fmt.Errorf("unknown encoding: '%s'", encodingName)
	}

	return cm, nil
}

func WriteEncodedFile(encodingName, contentsUtf8, fileName string, perm os.FileMode) error {
	encodedData, err := Encode(encodingName, contentsUtf8)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, encodedData, perm)
}

func ReadEncodedFile(encodingName, fileName string) (contentsUtf8 string, err error) {
	contents, err := os.ReadFile(fileName)
	if err != nil {
		return
	}

	return Decode(encodingName, contents)
}
