package model_alias_setting

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// DefaultAliasResolution 是客户端未指定分辨率时使用的默认档位。
const DefaultAliasResolution = "720p"

// ResolutionAlias 描述一个合并模型：客户端只请求别名，
// 平台按请求中的分辨率解析到具体的变体模型名（用于转发给上游与计价）。
type ResolutionAlias struct {
	DefaultResolution string            `json:"default_resolution,omitempty"`
	Resolutions       map[string]string `json:"resolutions"`
}

var resolutionAliasMap = types.NewRWMap[string, ResolutionAlias]()

func ModelAliasesJSONString() string {
	return resolutionAliasMap.MarshalJSONString()
}

func ValidateModelAliasesJSONString(jsonStr string) error {
	var configs map[string]ResolutionAlias
	if err := common.UnmarshalJsonStr(jsonStr, &configs); err != nil {
		return err
	}
	for alias, config := range configs {
		if strings.TrimSpace(alias) == "" {
			return fmt.Errorf("合并模型缺少别名")
		}
		if len(config.Resolutions) == 0 {
			return fmt.Errorf("合并模型 %s 至少需要配置一个分辨率变体", alias)
		}
		for resolution, model := range config.Resolutions {
			if strings.TrimSpace(resolution) == "" {
				return fmt.Errorf("合并模型 %s 存在空的分辨率档位", alias)
			}
			if strings.TrimSpace(model) == "" {
				return fmt.Errorf("合并模型 %s 的分辨率 %s 缺少对应模型名", alias, resolution)
			}
		}
		if config.DefaultResolution != "" {
			if _, ok := config.Resolutions[normalizeResolution(config.DefaultResolution)]; !ok {
				return fmt.Errorf("合并模型 %s 的默认分辨率 %s 未配置变体", alias, config.DefaultResolution)
			}
		}
	}
	return nil
}

func UpdateModelAliasesByJSONString(jsonStr string) error {
	if err := ValidateModelAliasesJSONString(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonString(resolutionAliasMap, jsonStr)
}

func GetResolutionAlias(model string) (ResolutionAlias, bool) {
	return resolutionAliasMap.Get(model)
}

func GetModelAliasesCopy() map[string]ResolutionAlias {
	return resolutionAliasMap.ReadAll()
}

// ResolveAliasVariant 解析别名模型在本次请求下的变体模型名。
// explicit 是客户端显式给出的分辨率（如 resolution / metadata.resolution），
// 显式指定但不支持时返回错误，避免静默按默认档位计价；
// fallback 是弱推断（如 size 归一化的档位），只在能命中时才采用。
// 两者都未命中时回退默认档位（配置缺省 720p）。
func ResolveAliasVariant(alias ResolutionAlias, explicit, fallback []string) (string, string, error) {
	explicitProvided := false
	for _, candidate := range explicit {
		label := normalizeResolution(candidate)
		if label == "" {
			continue
		}
		explicitProvided = true
		if model, ok := alias.Resolutions[label]; ok {
			return model, label, nil
		}
	}
	for _, candidate := range fallback {
		label := normalizeResolution(candidate)
		if label == "" {
			continue
		}
		if model, ok := alias.Resolutions[label]; ok {
			return model, label, nil
		}
	}
	defaultLabel := normalizeResolution(alias.DefaultResolution)
	if defaultLabel == "" {
		defaultLabel = DefaultAliasResolution
	}
	if model, ok := alias.Resolutions[defaultLabel]; ok {
		if explicitProvided {
			return "", "", fmt.Errorf("不支持所选分辨率，可选：%s", strings.Join(alias.ResolutionLabels(), " / "))
		}
		return model, defaultLabel, nil
	}
	return "", "", fmt.Errorf("缺少默认分辨率 %s 的变体配置", defaultLabel)
}

// ResolutionLabels 返回按 480p/720p/2k/2k-pro 自然顺序排列的分辨率档位。
func (a ResolutionAlias) ResolutionLabels() []string {
	labels := make([]string, 0, len(a.Resolutions))
	for label := range a.Resolutions {
		labels = append(labels, label)
	}
	sort.Slice(labels, func(i, j int) bool {
		rankI, numberI, variantI := resolutionSortKey(labels[i])
		rankJ, numberJ, variantJ := resolutionSortKey(labels[j])
		if rankI != rankJ {
			return rankI < rankJ
		}
		if numberI != numberJ {
			return numberI < numberJ
		}
		if variantI != variantJ {
			return variantI < variantJ
		}
		return labels[i] < labels[j]
	})
	return labels
}

func normalizeResolution(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// resolutionSortKey 让 480p/720p/1080p 排在 2k/4k 之前，同档按数值与 -pro 变体排序。
// 返回 (档位分组, 数值, 变体序号)，第三位保证 2k 排在 2k-pro 前且排序稳定。
func resolutionSortKey(label string) (int, int, int) {
	lower := normalizeResolution(label)
	if strings.HasSuffix(lower, "p") {
		if number, err := strconv.Atoi(strings.TrimSuffix(lower, "p")); err == nil {
			return 0, number, 0
		}
	}
	if strings.HasSuffix(lower, "-pro") {
		if number, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSuffix(lower, "-pro"), "k")); err == nil {
			return 1, number, 1
		}
	}
	if strings.HasSuffix(lower, "k") {
		if number, err := strconv.Atoi(strings.TrimSuffix(lower, "k")); err == nil {
			return 1, number, 0
		}
	}
	return 2, 0, 0
}
