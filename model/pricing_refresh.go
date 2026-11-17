package model

// RefreshPricing forces an immediate recomputation of pricing-related caches.
// Used by internal admin APIs that need the freshest data,
// so it bypasses the default 1-minute delayed refresh.
func RefreshPricing() {
	updatePricingLock.Lock()
	defer updatePricingLock.Unlock()

	modelSupportEndpointsLock.Lock()
	defer modelSupportEndpointsLock.Unlock()

	updatePricing()
}
