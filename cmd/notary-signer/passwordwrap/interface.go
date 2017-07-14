package passwordwrap

// Storage is the interface to store/retrieve password from any specific implementation type.
type Storage interface {
	// Get password from storage.
	Get(alias string) (string, error)

	// Set password in storage.
	Set(alias string, newPassword string) error
}
