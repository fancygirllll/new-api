package service

import (
	"math"
	"strings"
	"sync"
	"unicode"
)

// Provider defines the model vendor category
type Provider string

const (
	OpenAI  Provider = "openai"  // e.g. GPT-3.5, GPT-4, GPT-4o
	Gemini  Provider = "gemini"  // e.g. Gemini 1.0, 1.5 Pro/Flash
	Claude  Provider = "claude"  // e.g. Claude 3, 3.5 Sonnet
	Unknown Provider = "unknown" // Fallback default
)

// multipliers defines the billing weights for each vendor
type multipliers struct {
	Word       float64 // English word (per word)
	Number     float64 // Number (per contiguous digit run)
	CJK        float64 // CJK character (per character)
	Symbol     float64 // Regular punctuation (per symbol)
	MathSymbol float64 // Math symbol (sum, integral, partial, sqrt, etc.; per symbol)
	URLDelim   float64 // URL delimiter (/,:,?,&,=,#,%) - handled well by tokenizers
	AtSign     float64 // @ sign - splits words, relatively expensive
	Emoji      float64 // Emoji (per emoji)
	Newline    float64 // Newline/tab (per character)
	Space      float64 // Space (per space)
	BasePad    int     // Base overhead (Start/End tokens)
}

var (
	multipliersMap = map[Provider]multipliers{
		Gemini: {
			Word: 1.15, Number: 2.8, CJK: 0.68, Symbol: 0.38, MathSymbol: 1.05, URLDelim: 1.2, AtSign: 2.5, Emoji: 1.08, Newline: 1.15, Space: 0.2, BasePad: 0,
		},
		Claude: {
			Word: 1.13, Number: 1.63, CJK: 1.21, Symbol: 0.4, MathSymbol: 4.52, URLDelim: 1.26, AtSign: 2.82, Emoji: 2.6, Newline: 0.89, Space: 0.39, BasePad: 0,
		},
		OpenAI: {
			Word: 1.02, Number: 1.55, CJK: 0.85, Symbol: 0.4, MathSymbol: 2.68, URLDelim: 1.0, AtSign: 2.0, Emoji: 2.12, Newline: 0.5, Space: 0.42, BasePad: 0,
		},
	}
	multipliersLock sync.RWMutex
)

// getMultipliers returns the weight config for a vendor
func getMultipliers(p Provider) multipliers {
	multipliersLock.RLock()
	defer multipliersLock.RUnlock()

	switch p {
	case Gemini:
		return multipliersMap[Gemini]
	case Claude:
		return multipliersMap[Claude]
	case OpenAI:
		return multipliersMap[OpenAI]
	default:
		// Default fallback (use OpenAI weights)
		return multipliersMap[OpenAI]
	}
}

// EstimateToken estimates the token count
func EstimateToken(provider Provider, text string) int {
	m := getMultipliers(provider)
	var count float64

	// State machine variables
	type WordType int
	const (
		None WordType = iota
		Latin
		Number
	)
	currentWordType := None

	for _, r := range text {
		// 1. Handle spaces and newlines
		if unicode.IsSpace(r) {
			currentWordType = None
			// Newlines and tabs use the Newline weight
			if r == '\n' || r == '\t' {
				count += m.Newline
			} else {
				// Regular spaces use the Space weight
				count += m.Space
			}
			continue
		}

		// 2. Handle CJK - billed per character
		if isCJK(r) {
			currentWordType = None
			count += m.CJK
			continue
		}

		// 3. Handle Emoji - use the dedicated Emoji weight
		if isEmoji(r) {
			currentWordType = None
			count += m.Emoji
			continue
		}

		// 4. Handle Latin letters/digits (English words)
		if isLatinOrNumber(r) {
			isNum := unicode.IsNumber(r)
			newType := Latin
			if isNum {
				newType = Number
			}

			// If not previously inside a word, or the type changed (letter <-> digit), treat it as a new token
			// Note: for OpenAI, "version 3.5" is usually split, and "abc123xyz" is sometimes split too
			// For simplicity, add weight when switching between letters and digits
			if currentWordType == None || currentWordType != newType {
				if newType == Number {
					count += m.Number
				} else {
					count += m.Word
				}
				currentWordType = newType
			}
			// Characters in the middle of a word are not charged extra
			continue
		}

		// 5. Handle punctuation/special characters - different weights per type
		currentWordType = None
		if isMathSymbol(r) {
			count += m.MathSymbol
		} else if r == '@' {
			count += m.AtSign
		} else if isURLDelim(r) {
			count += m.URLDelim
		} else {
			count += m.Symbol
		}
	}

	// Round up and add the base padding
	return int(math.Ceil(count)) + m.BasePad
}

// Helper: check whether a character is CJK
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		(r >= 0x3040 && r <= 0x30FF) || // Japanese
		(r >= 0xAC00 && r <= 0xD7A3) // Korean
}

// Helper: check whether a character is a word body (letter or digit)
func isLatinOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// Helper: check whether a character is Emoji
func isEmoji(r rune) bool {
	// Unicode ranges for Emoji
	// Basic range: 0x1F300-0x1F9FF (Emoticons, Symbols, Pictographs)
	// Supplementary range: 0x2600-0x26FF (Misc Symbols), 0x2700-0x27BF (Dingbats)
	// Emoticons: 0x1F600-0x1F64F
	// Other: 0x1F900-0x1F9FF (Supplemental Symbols and Pictographs)
	return (r >= 0x1F300 && r <= 0x1F9FF) ||
		(r >= 0x2600 && r <= 0x26FF) ||
		(r >= 0x2700 && r <= 0x27BF) ||
		(r >= 0x1F600 && r <= 0x1F64F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x1FA00 && r <= 0x1FAFF) // Symbols and Pictographs Extended-A
}

// Helper: check whether a character is a math symbol
func isMathSymbol(r rune) bool {
	// Math operators and symbols
	// Basic math symbols: sum, integral, partial, sqrt, infinity, <=, >=, !=, approx, +/-, x, /
	// Superscript/subscript digits
	// Greek letters are also commonly used in math
	mathSymbols := "∑∫∂√∞≤≥≠≈±×÷∈∉∋∌⊂⊃⊆⊇∪∩∧∨¬∀∃∄∅∆∇∝∟∠∡∢°′″‴⁺⁻⁼⁽⁾ⁿ₀₁₂₃₄₅₆₇₈₉₊₋₌₍₎²³¹⁴⁵⁶⁷⁸⁹⁰"
	for _, m := range mathSymbols {
		if r == m {
			return true
		}
	}
	// Mathematical Operators (U+2200–U+22FF)
	if r >= 0x2200 && r <= 0x22FF {
		return true
	}
	// Supplemental Mathematical Operators (U+2A00–U+2AFF)
	if r >= 0x2A00 && r <= 0x2AFF {
		return true
	}
	// Mathematical Alphanumeric Symbols (U+1D400–U+1D7FF)
	if r >= 0x1D400 && r <= 0x1D7FF {
		return true
	}
	return false
}

// Helper: check whether a character is a URL delimiter (tokenizers handle these well)
func isURLDelim(r rune) bool {
	// Common URL delimiters, usually optimized by tokenizers
	urlDelims := "/:?&=;#%"
	for _, d := range urlDelims {
		if r == d {
			return true
		}
	}
	return false
}

func EstimateTokenByModel(model, text string) int {
	// strings.Contains(model, "gpt-4o")
	if text == "" {
		return 0
	}

	model = strings.ToLower(model)
	if strings.Contains(model, "gemini") {
		return EstimateToken(Gemini, text)
	} else if strings.Contains(model, "claude") {
		return EstimateToken(Claude, text)
	} else {
		return EstimateToken(OpenAI, text)
	}
}
