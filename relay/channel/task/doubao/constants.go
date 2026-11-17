package doubao

import "strings"

var ModelList = []string{
	"doubao-seedance-1-0-pro-250528",
	"doubao-seedance-1-0-lite-t2v",
	"doubao-seedance-1-0-lite-i2v",
	"doubao-seedance-1-5-pro-251215",
	"doubao-seedance-2-0-260128",
	"doubao-seedance-2-0-fast-260128",
}

var ChannelName = "doubao-video"

// videoPriceKey is the price table key: the output resolution tier (both is1080p and is4k false
// means the 480p/720p base tier) and whether the input contains video.
type videoPriceKey struct {
	is1080p  bool
	is4k     bool
	hasVideo bool
}

// videoPriceTable holds the unit price of each model for each (output resolution tier, whether
// the input contains video) combination, in CNY per million tokens.
// The zero-value key {480p/720p, no video} is the base price, equal to the ModelRatio the
// administrator should configure; billing uses actual price / base price as the OtherRatio.
var videoPriceTable = map[string]map[videoPriceKey]float64{
	"doubao-seedance-2-0-260128": {
		{hasVideo: false}:                46.0,
		{hasVideo: true}:                 28.0,
		{is1080p: true, hasVideo: false}: 51.0,
		{is1080p: true, hasVideo: true}:  31.0,
		{is4k: true, hasVideo: false}:    26.0,
		{is4k: true, hasVideo: true}:     16.0,
	},
	"doubao-seedance-2-0-fast-260128": {
		{hasVideo: false}: 37.0,
		{hasVideo: true}:  22.0,
	},
}

// GetVideoInputRatio returns the billing multiplier relative to the base price for the given
// model, output resolution, and whether the input contains video.
// The second return value reports whether the model has a configured price table; when the
// multiplier is 1.0 the caller can ignore this OtherRatio.
func GetVideoInputRatio(modelName, resolution string, hasVideo bool) (float64, bool) {
	prices, ok := videoPriceTable[modelName]
	base := prices[videoPriceKey{}] // Zero-value key = {480p/720p, no video} base price
	if !ok || base <= 0 {
		return 0, false
	}
	res := strings.ToLower(strings.TrimSpace(resolution))
	price, ok := prices[videoPriceKey{is1080p: res == "1080p", is4k: res == "4k", hasVideo: hasVideo}]
	if !ok {
		// Unconfigured combinations (e.g. fast has no 1080p/4k; upstream reports an error itself) are billed at the base price.
		return 1.0, true
	}
	return price / base, true
}
