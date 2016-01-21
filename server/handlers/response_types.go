package handlers

// VersionResponse wraps a list of versions incase we want
// to add more metadata later without breaking old clients.
type VersionResponse struct {
	Versions [][]byte `json:"versions"`
}
