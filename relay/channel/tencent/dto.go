package tencent

type TencentMessage struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

type TencentChatRequest struct {
	// Model name; supported values include hunyuan-lite, hunyuan-standard, hunyuan-standard-256K, hunyuan-pro.
	// See the [product overview](https://cloud.tencent.com/document/product/1729/104753) for details on each model.
	//
	// Note:
	// Billing differs by model; see the [purchase guide](https://cloud.tencent.com/document/product/1729/97731) and call as needed.
	Model *string `json:"Model"`
	// Chat context.
	// Notes:
	// 1. At most 40 entries, ordered from oldest to newest.
	// 2. Message.Role can be system, user, or assistant.
	// The system role is optional and, if present, must be first. user and assistant must alternate (question/answer), starting and ending with user, and Content must not be empty. Example Role order: [system (optional) user assistant user assistant user ...].
	// 3. The total Content length in Messages must not exceed the model's input limit (see the [product overview](https://cloud.tencent.com/document/product/1729/104753)); otherwise the leading content is truncated and only the tail is kept.
	Messages []*TencentMessage `json:"Messages"`
	// Streaming switch.
	// Notes:
	// 1. Defaults to non-streaming (false) when unset.
	// 2. In streaming mode results are returned incrementally over SSE (take values from Choices[n].Delta and concatenate them to get the full result).
	// 3. In non-streaming mode:
	// The call behaves like a normal HTTP request.
	// Responses take longer; **set this to true if you need lower latency**.
	// The final result is returned once (take values from Choices[n].Message).
	//
	// Note:
	// When calling via an SDK, streaming and non-streaming use **different ways** to read the result; see the SDK comments or examples (under examples/hunyuan/v20230901/ in each language SDK repository).
	Stream *bool `json:"Stream,omitempty"`
	// Notes:
	// 1. Affects the diversity of the output text; higher values produce more diverse text.
	// 2. Range [0.0, 1.0]; the per-model recommended value is used when unset.
	// 3. Not recommended unless necessary; unreasonable values degrade quality.
	TopP *float64 `json:"TopP,omitempty"`
	// Notes:
	// 1. Higher values make the output more random; lower values make it more focused and deterministic.
	// 2. Range [0.0, 2.0]; the per-model recommended value is used when unset.
	// 3. Not recommended unless necessary; unreasonable values degrade quality.
	Temperature *float64 `json:"Temperature,omitempty"`
}

type TencentError struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
}

type TencentUsage struct {
	PromptTokens     int `json:"PromptTokens"`
	CompletionTokens int `json:"CompletionTokens"`
	TotalTokens      int `json:"TotalTokens"`
}

type TencentResponseChoices struct {
	FinishReason string         `json:"FinishReason,omitempty"` // Streaming end marker; "stop" indicates the final packet
	Messages     TencentMessage `json:"Message,omitempty"`      // Content returned in synchronous mode; in streaming mode it is null; content is limited to 1024 tokens.
	Delta        TencentMessage `json:"Delta,omitempty"`        // Content returned in streaming mode; in synchronous mode it is null; content is limited to 1024 tokens.
}

type TencentChatResponse struct {
	Choices []TencentResponseChoices `json:"Choices,omitempty"` // Results
	Created int64                    `json:"Created,omitempty"` // Unix timestamp as a string
	Id      string                   `json:"Id,omitempty"`      // Session ID
	Usage   TencentUsage             `json:"Usage,omitempty"`   // Token counts
	Error   TencentError             `json:"Error,omitempty"`   // Error information; note this field may be null
	Note    string                   `json:"Note,omitempty"`    // Notes
	ReqID   string                   `json:"Req_id,omitempty"`  // Unique request ID, returned on every request; used for feedback on request parameters
}

type TencentChatResponseSB struct {
	Response TencentChatResponse `json:"Response,omitempty"`
}
