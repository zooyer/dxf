package core

import (
	"unicode/utf8"

	"github.com/axgle/mahonia"
)

func IsUTF8(s string) bool {
	return utf8.ValidString(s)
}

func IsGBK(s string) bool {
	if IsUTF8(s) {
		return false
	}

	data := []byte(s)
	length := len(data)
	var i = 0
	for i < length {
		if data[i] <= 0xff {
			i++
			continue
		} else {
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}

	return true
}

func ConvertToString(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)

	srcResult := srcCoder.ConvertString(src)

	tagCoder := mahonia.NewDecoder(tagCode)

	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)

	result := string(cdata)

	return result
}

func GetString(str string) string {
	if IsGBK(str) {
		return ConvertToString(str, "gbk", "utf-8")
	}

	return str
}
