package common

import (
	"encoding/json"
	"strings"
)

// OutboundErrorFallbackMessage is the generic English fallback returned to clients:
// When an outbound error message contains Chinese characters, it is replaced entirely with this text so overseas customers never see Chinese errors.
const OutboundErrorFallbackMessage = "Service error. Please try again later or contact support."

// ContainsCJK reports whether a string contains at least one CJK unified ideograph.
// Covered ranges: U+4E00-U+9FFF (basic), U+3400-U+4DBF (extension A), U+F900-U+FAFF (compatibility ideographs).
func ContainsCJK(s string) bool {
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0xF900 && r <= 0xFAFF) {
			return true
		}
	}
	return false
}

// SanitizeOutboundErrorMessage sanitizes the language of an outbound error message:
// Replace it with OutboundErrorFallbackMessage when it contains Chinese; otherwise return it unchanged.
func SanitizeOutboundErrorMessage(msg string) string {
	if ContainsCJK(msg) {
		return OutboundErrorFallbackMessage
	}
	return msg
}

// SanitizeErrorChunkData sanitizes an OpenAI-format streaming error chunk:
// Only when the message of the top-level error object in the chunk is a string containing Chinese
// is it rewritten to OutboundErrorFallbackMessage and re-serialized;
// all other cases (normal content chunks, English errors, non-string messages, nested errors, etc.) are returned unchanged, byte for byte.
func SanitizeErrorChunkData(data string) string {
	// Pre-check: the word "error" inside a JSON string value is always escaped as \"error\", which does not match the pre-check string,
	// so only chunks with a top-level error key proceed to parsing.
	if !strings.Contains(data, `"error"`) {
		return data
	}

	var topLevel map[string]json.RawMessage
	if err := UnmarshalJsonStr(data, &topLevel); err != nil {
		return data
	}

	rawError, ok := topLevel["error"]
	if !ok || GetJsonType(rawError) != "object" {
		return data
	}

	var errorObject map[string]json.RawMessage
	if err := Unmarshal(rawError, &errorObject); err != nil {
		return data
	}

	rawMessage, ok := errorObject["message"]
	if !ok {
		return data
	}

	// Only string messages are handled; numbers, objects, arrays, and null are treated as non-matches and passed through.
	var message string
	if err := Unmarshal(rawMessage, &message); err != nil {
		return data
	}
	if !ContainsCJK(message) {
		return data
	}

	fallback, err := Marshal(OutboundErrorFallbackMessage)
	if err != nil {
		return data
	}
	errorObject["message"] = json.RawMessage(fallback)

	newError, err := Marshal(errorObject)
	if err != nil {
		return data
	}
	topLevel["error"] = json.RawMessage(newError)

	newData, err := Marshal(topLevel)
	if err != nil {
		return data
	}
	return string(newData)
}
