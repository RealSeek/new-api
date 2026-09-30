package ratio_setting

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// ImagePriceConfig 按分辨率档位配置图片单张价格（USD）。
type ImagePriceConfig map[string]float64

var imagePriceMap = types.NewRWMap[string, ImagePriceConfig]()

// 图片分辨率档位与图片网关的路由口径保持一致。
var imagePriceResolutions = map[string]struct{}{
	"1k": {},
	"2k": {},
	"4k": {},
}

func ImagePrice2JSONString() string {
	return imagePriceMap.MarshalJSONString()
}

func ValidateImagePriceJSONString(jsonStr string) error {
	var configs map[string]ImagePriceConfig
	if err := common.UnmarshalJsonStr(jsonStr, &configs); err != nil {
		return err
	}
	for model, config := range configs {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("图片分辨率价格缺少模型名称")
		}
		for resolution, price := range config {
			if _, ok := imagePriceResolutions[strings.ToLower(strings.TrimSpace(resolution))]; !ok {
				return fmt.Errorf("模型 %s 的分辨率 %s 无效，仅支持 1k/2k/4k", model, resolution)
			}
			if price <= 0 {
				return fmt.Errorf("模型 %s 的分辨率 %s 价格必须大于 0", model, resolution)
			}
		}
	}
	return nil
}

func UpdateImagePriceByJSONString(jsonStr string) error {
	if err := ValidateImagePriceJSONString(jsonStr); err != nil {
		return err
	}
	return types.LoadFromJsonStringWithCallback(imagePriceMap, jsonStr, InvalidateExposedDataCache)
}

// GetImagePrice 返回模型指定分辨率档位的单张价格。
func GetImagePrice(model, resolution string) (float64, bool) {
	config, ok := imagePriceMap.Get(FormatMatchingModelName(model))
	if !ok {
		return 0, false
	}
	price, ok := config[strings.ToLower(strings.TrimSpace(resolution))]
	if !ok || price <= 0 {
		return 0, false
	}
	return price, true
}

func GetImagePriceConfig(model string) (ImagePriceConfig, bool) {
	return imagePriceMap.Get(FormatMatchingModelName(model))
}

func GetImagePriceCopy() map[string]ImagePriceConfig {
	return imagePriceMap.ReadAll()
}
