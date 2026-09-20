package read

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func detectTextDecoder(sample []byte) (string, transform.Transformer) {
	if validUTF8AllowingTruncation(sample) {
		return "utf-8", nil
	}
	if looksLikeGB18030(sample) {
		return "gb18030", simplifiedchinese.GB18030.NewDecoder()
	}
	return "iso-8859-1", charmap.ISO8859_1.NewDecoder()
}

func utf8BOMLen(sample []byte) int {
	if len(sample) >= 3 && sample[0] == 0xEF && sample[1] == 0xBB && sample[2] == 0xBF {
		return 3
	}
	return 0
}

// validUTF8AllowingTruncation 允许样本末尾被固定窗口切在多字节中间。
// 中间出现非法 UTF-8 仍判失败，避免真 GB18030 被当成截断 UTF-8。
func validUTF8AllowingTruncation(b []byte) bool {
	for len(b) > 0 {
		if !utf8.FullRune(b) {
			return true
		}
		r, size := utf8.DecodeRune(b)
		if r == utf8.RuneError && size == 1 {
			return false
		}
		b = b[size:]
	}
	return true
}

func looksLikeGB18030(sample []byte) bool {
	// 探测窗口也可能切在 GB 多字节中间；最多丢 3 个尾字节再判。
	// 解码出现替换符就不是干净的 GB18030——x/text 几乎不会返回 error。
	for drop := 0; drop <= 3 && drop <= len(sample); drop++ {
		chunk := sample[:len(sample)-drop]
		if len(chunk) == 0 || validUTF8AllowingTruncation(chunk) {
			continue
		}
		decoded, _, err := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), chunk)
		if err != nil || bytes.ContainsRune(decoded, unicode.ReplacementChar) {
			continue
		}
		return true
	}
	return false
}
