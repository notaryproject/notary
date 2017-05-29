package passphrase

// PasswordStore is the interface to store/retrieve passphrase from any specific implementation type.
type PasswordStore interface {
  // Get passphrase from storage.
	GetPassword(alias string) (string, error)

  // Set passphrase in storage.
	SetPassword(alias string, newPassword string) error
}

// PasswordProtector is the interface to wrap/unwrap passphrase using any specific implementation type.
type PasswordProtector interface {
  // Wrap the passphrase passed in as clear text.
	Encrypt(clearText string) (string, error)

  // Unwrap the passphrase passed in as cipher text.
	Decrypt(cipherText string) (string, error)
}
