package read

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func TestTool_InfoName(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, ToolName, info.Name)
}

func TestTool_InvokableRun_ReadsPagedContent(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "x.txt"), []byte("l1\nl2\nl3\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"x.txt","offset":2,"limit":2}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "encoding=")
	assert.Contains(t, out, "2|l2")
	assert.Contains(t, out, "3|l3")
}

func TestTool_InvokableRun_ReadsFileOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()
	targetPath := filepath.Join(outsideDir, "x.txt")
	require.NoError(t, os.WriteFile(targetPath, []byte("external"), 0o644))

	tl, err := New(Config{DefaultBaseDir: baseDir})
	require.NoError(t, err)

	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)

	out, runErr := invokable.InvokableRun(context.Background(), `{"file_path":"`+targetPath+`"}`)
	require.NoError(t, runErr)
	assert.Contains(t, out, "external")
}

func utf8CutAtRuneSample() []byte {
	return append(bytes.Repeat([]byte("x"), 4095), []byte("结构相似度\n")...)
}

func dropIncompleteUTF8Suffix(b []byte) []byte {
	i := 0
	for i < len(b) {
		if !utf8.FullRune(b[i:]) {
			return b[:i]
		}
		_, size := utf8.DecodeRune(b[i:])
		i += size
	}
	return b
}

func invokeRead(t *testing.T, tl *Tool, args string) string {
	t.Helper()
	invokable, ok := any(tl).(interface {
		InvokableRun(context.Context, string, ...tool.Option) (string, error)
	})
	require.True(t, ok)
	out, err := invokable.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	return out
}

func TestTool_InfoStaysGeneric(t *testing.T) {
	tl, err := New(Config{})
	require.NoError(t, err)
	info, err := tl.Info(context.Background())
	require.NoError(t, err)
	blob := info.Desc
	if info.ParamsOneOf != nil {
		params, paramsErr := info.ParamsOneOf.ToJSONSchema()
		require.NoError(t, paramsErr)
		encoded, marshalErr := json.Marshal(params)
		require.NoError(t, marshalErr)
		blob += string(encoded)
	}
	for _, leak := range []string{"结构相似度", "notes.md", "简体中文页面"} {
		assert.NotContains(t, blob, leak)
	}
}

func TestDetectTextDecoder_UTF8SampleCutAtRuneStaysUTF8(t *testing.T) {
	buf := utf8CutAtRuneSample()
	require.True(t, utf8.Valid(buf))
	require.False(t, utf8.Valid(buf[:4096]))

	name, dec := detectTextDecoder(buf[:4096])
	assert.Equal(t, "utf-8", name)
	assert.Nil(t, dec)
}

func TestTool_InvokableRun_UTF8CutAtRuneIsNotGB18030Mojibake(t *testing.T) {
	buf := utf8CutAtRuneSample()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "notes.md"), buf, 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	out := invokeRead(t, tl, `{"file_path":"notes.md"}`)
	assert.Contains(t, out, "encoding=utf-8")
	assert.NotContains(t, out, "encoding=gb18030")
	assert.Contains(t, out, "结构相似度")

	garbled, _, transformErr := transform.Bytes(simplifiedchinese.GB18030.NewDecoder(), []byte("结构相似度"))
	require.NoError(t, transformErr)
	require.NotEqual(t, "结构相似度", string(garbled))
	assert.NotContains(t, out, string(garbled))
}

func TestTool_InvokableRun_ShortUTF8CJKWithoutBOM(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "short.md"), []byte("结构相似度\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	out := invokeRead(t, tl, `{"file_path":"short.md"}`)
	assert.Contains(t, out, "encoding=utf-8")
	assert.Contains(t, out, "1|结构相似度")
}

func TestTool_InvokableRun_UTF8BOMIsNotContent(t *testing.T) {
	tmpDir := t.TempDir()
	body := append([]byte{0xEF, 0xBB, 0xBF}, []byte("结构相似度\n")...)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "bom.md"), body, 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	out := invokeRead(t, tl, `{"file_path":"bom.md"}`)
	assert.Contains(t, out, "encoding=utf-8")
	assert.Contains(t, out, "1|结构相似度")
	assert.NotContains(t, out, "\ufeff")
	assert.False(t, strings.Contains(out, "1|\ufeff"))
}

func TestDetectTextDecoder_RealGB18030(t *testing.T) {
	raw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte("简体中文页面"))
	require.NoError(t, err)
	require.False(t, utf8.Valid(raw))
	require.False(t, utf8.Valid(dropIncompleteUTF8Suffix(raw)), "裁掉不完整尾字节后仍应是非法 UTF-8")

	name, dec := detectTextDecoder(raw)
	assert.Equal(t, "gb18030", name)
	require.NotNil(t, dec)

	longRaw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte(strings.Repeat("简", 3000)))
	require.NoError(t, err)
	require.Greater(t, len(longRaw), 4096)
	cut := longRaw[:4096]
	require.False(t, utf8.Valid(dropIncompleteUTF8Suffix(cut)))
	name, dec = detectTextDecoder(cut)
	assert.Equal(t, "gb18030", name)
	require.NotNil(t, dec)
}

func TestTool_InvokableRun_RealGB18030DecodesHan(t *testing.T) {
	raw, err := simplifiedchinese.GB18030.NewEncoder().Bytes([]byte("简体中文页面\n"))
	require.NoError(t, err)
	require.False(t, utf8.Valid(raw))

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "legacy.txt"), raw, 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	out := invokeRead(t, tl, `{"file_path":"legacy.txt"}`)
	assert.Contains(t, out, "encoding=gb18030")
	assert.Contains(t, out, "1|简体中文页面")
	assert.NotContains(t, out, string(raw))
}

func TestDetectTextDecoder_NonUTF8NonGB18030FallsBackToLatin1(t *testing.T) {
	// 0xE9 / 0xFF 非法 UTF-8；GB18030 只能靠替换符吞掉，不能当中文页。
	sample := []byte{'c', 'a', 'f', 0xE9, 0xFF}
	require.False(t, utf8.Valid(sample))

	name, dec := detectTextDecoder(sample)
	assert.Equal(t, "iso-8859-1", name)
	require.NotNil(t, dec)
}

func TestTool_InvokableRun_Latin1Fallback(t *testing.T) {
	raw := []byte{'c', 'a', 'f', 0xE9, 0xFF}
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "latin.txt"), raw, 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	out := invokeRead(t, tl, `{"file_path":"latin.txt"}`)
	assert.Contains(t, out, "encoding=iso-8859-1")
	assert.Contains(t, out, "1|caféÿ")
	assert.NotContains(t, out, "encoding=gb18030")
}

func TestDetectTextDecoder_ASCIIAndEmptyAreUTF8(t *testing.T) {
	name, dec := detectTextDecoder(nil)
	assert.Equal(t, "utf-8", name)
	assert.Nil(t, dec)

	name, dec = detectTextDecoder([]byte{})
	assert.Equal(t, "utf-8", name)
	assert.Nil(t, dec)

	name, dec = detectTextDecoder([]byte("hello"))
	assert.Equal(t, "utf-8", name)
	assert.Nil(t, dec)
}

func TestTool_InvokableRun_ASCIIAndEmptyAreUTF8(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "empty.txt"), []byte{}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "ascii.txt"), []byte("hello\n"), 0o644))

	tl, err := New(Config{DefaultBaseDir: tmpDir})
	require.NoError(t, err)

	emptyOut := invokeRead(t, tl, `{"file_path":"empty.txt"}`)
	assert.Contains(t, emptyOut, "encoding=utf-8")

	asciiOut := invokeRead(t, tl, `{"file_path":"ascii.txt"}`)
	assert.Contains(t, asciiOut, "encoding=utf-8")
	assert.Contains(t, asciiOut, "1|hello")
}
