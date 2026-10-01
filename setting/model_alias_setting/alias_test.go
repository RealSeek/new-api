package model_alias_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAlias() ResolutionAlias {
	return ResolutionAlias{
		Resolutions: map[string]string{
			"480p":   "MiniMaxH3-480p",
			"720p":   "MiniMaxH3-720p",
			"2k":     "MiniMaxH3-2k",
			"2k-pro": "MiniMaxH3-2k-pro",
		},
	}
}

func TestResolveAliasVariant(t *testing.T) {
	alias := testAlias()

	variant, label, err := ResolveAliasVariant(alias, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-720p", variant)
	assert.Equal(t, "720p", label)

	variant, label, err = ResolveAliasVariant(alias, []string{" 2K "}, nil)
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-2k", variant)
	assert.Equal(t, "2k", label)

	// 客户端显式给了分辨率但不支持时必须报错，不能静默按默认档位计价。
	_, _, err = ResolveAliasVariant(alias, []string{"4k"}, nil)
	require.Error(t, err)

	// 第一个候选未命中时继续尝试后面的候选。
	variant, label, err = ResolveAliasVariant(alias, []string{"1080p", "2k-pro"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-2k-pro", variant)
	assert.Equal(t, "2k-pro", label)

	// size 推断属于弱线索：命中则采用，不命中回退默认档位而不是报错。
	variant, label, err = ResolveAliasVariant(alias, nil, []string{"720p"})
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-720p", variant)
	assert.Equal(t, "720p", label)

	variant, label, err = ResolveAliasVariant(alias, nil, []string{"1080p"})
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-720p", variant)
	assert.Equal(t, "720p", label)
}

func TestResolveAliasVariantUsesConfiguredDefault(t *testing.T) {
	alias := testAlias()
	alias.DefaultResolution = "2k"

	variant, label, err := ResolveAliasVariant(alias, []string{""}, nil)
	require.NoError(t, err)
	assert.Equal(t, "MiniMaxH3-2k", variant)
	assert.Equal(t, "2k", label)
}

func TestResolutionLabelsOrder(t *testing.T) {
	alias := testAlias()
	assert.Equal(t, []string{"480p", "720p", "2k", "2k-pro"}, alias.ResolutionLabels())
}

func TestValidateModelAliasesJSONString(t *testing.T) {
	require.NoError(t, ValidateModelAliasesJSONString(`{"MiniMaxH3":{"default_resolution":"720p","resolutions":{"720p":"MiniMaxH3-720p"}}}`))
	require.Error(t, ValidateModelAliasesJSONString(`{"MiniMaxH3":{"resolutions":{}}}`))
	require.Error(t, ValidateModelAliasesJSONString(`{"MiniMaxH3":{"resolutions":{"720p":""}}}`))
	require.Error(t, ValidateModelAliasesJSONString(`{"MiniMaxH3":{"default_resolution":"2k","resolutions":{"720p":"MiniMaxH3-720p"}}}`))
}
