package common

// Global variables for commission and attribution setting keys (frozen per contract 01-CONTRACT.md §2.4 + §3.4).
// Together with default registration in model/option.go InitOptionMap and the updateOptionMap cases, this forms the hot-reload trio (contract F2).
// Default values are given at the declaration site (following the constants.go FileUploadPermission pattern),
// ensuring OptionMap and the global variables stay consistent on first startup (when the DB has no rows).
var (
	CommissionEnabled   bool
	CommissionPayoutDay = 8
	AffRebindWindowMode = "days"
	AffRebindWindowDays = 7
)
