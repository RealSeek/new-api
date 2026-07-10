package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeHeaderNavCustomLinks(t *testing.T) {
	normalized, err := normalizeHeaderNavCustomLinks(`[
		{"title":" ","titles":{"zhCN":" 图片生成 ","en":" Image Generation ","other":"忽略"},"url":" https://image.realseek.wiki/ "},
		{"title":"Docs","url":"https://example.com/docs"}
	]`)

	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"title":"图片生成","titles":{"en":"Image Generation","zhCN":"图片生成"},"url":"https://image.realseek.wiki/"},
		{"title":"Docs","url":"https://example.com/docs"}
	]`, normalized)
}

func TestNormalizeHeaderNavCustomLinksAllowsEmptyList(t *testing.T) {
	normalized, err := normalizeHeaderNavCustomLinks(`[]`)

	require.NoError(t, err)
	assert.Equal(t, `[]`, normalized)
}

func TestNormalizeHeaderNavCustomLinksRejectsInvalidEntries(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		errorSubstr string
	}{
		{
			name:        "顶层不是数组",
			value:       `{"title":"Docs","url":"https://example.com"}`,
			errorSubstr: "JSON 数组",
		},
		{
			name:        "名称为空",
			value:       `[{"title":" ","titles":{},"url":"https://example.com"}]`,
			errorSubstr: "至少需要填写一个语言名称",
		},
		{
			name:        "禁止非 HTTP 协议",
			value:       `[{"title":"Unsafe","url":"javascript:alert(1)"}]`,
			errorSubstr: "HTTP 或 HTTPS URL",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeHeaderNavCustomLinks(test.value)

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorSubstr)
		})
	}
}
