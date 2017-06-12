package passwordwrap

// Storage is the interface to store/retrieve password from any specific implementation type.
type Storage interface {
	// Get password from storage.
	GetPassword(alias string) (string, error)

	// Set password in storage.
	SetPassword(alias string, newPassword string) error
}

// Protector is the interface to wrap/unwrap password using any specific implementation type.
type Protector interface {
	// Wrap the password passed in as clear text.
	Encrypt(clearText string) (string, error)

	// Unwrap the password passed in as cipher text.
	Decrypt(cipherText string) (string, error)
}
