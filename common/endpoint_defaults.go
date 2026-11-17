package common

import "github.com/QuantumNous/new-api/constant"

// EndpointInfo describes the default request information for a single endpoint
// path: upstream path
// method: HTTP request method, such as POST/GET
// All are currently POST; this can be extended later
//
// The json tag is used to serialize directly into the API output
// Example: {"path":"/v1/chat/completions","method":"POST"}

type EndpointInfo struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

// defaultEndpointInfoMap holds the default Path and Method for built-in endpoints
var defaultEndpointInfoMap = map[constant.EndpointType]EndpointInfo{
	constant.EndpointTypeOpenAI:                {Path: "/v1/chat/completions", Method: "POST"},
	constant.EndpointTypeOpenAIResponse:        {Path: "/v1/responses", Method: "POST"},
	constant.EndpointTypeOpenAIResponseCompact: {Path: "/v1/responses/compact", Method: "POST"},
	constant.EndpointTypeAnthropic:             {Path: "/v1/messages", Method: "POST"},
	constant.EndpointTypeGemini:                {Path: "/v1beta/models/{model}:generateContent", Method: "POST"},
	constant.EndpointTypeJinaRerank:            {Path: "/v1/rerank", Method: "POST"},
	constant.EndpointTypeImageGeneration:       {Path: "/v1/images/generations", Method: "POST"},
	constant.EndpointTypeEmbeddings:            {Path: "/v1/embeddings", Method: "POST"},
}

// GetDefaultEndpointInfo returns the default information for the given endpoint type and whether it exists
func GetDefaultEndpointInfo(et constant.EndpointType) (EndpointInfo, bool) {
	info, ok := defaultEndpointInfoMap[et]
	return info, ok
}
