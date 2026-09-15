package pcsc

// NewOSBackend returns the production PC/SC backend for this platform.
// Monitoring and reading must use separate instances so each owns its
// native PC/SC context.
func NewOSBackend() (Backend, error) {
	return newPlatformBackend()
}

// NewSyntheticBackend returns the deterministic test backend.
func NewSyntheticBackend() Backend {
	return NewSynthetic()
}
