package passphrase

// NewDefaultPasswordProtector instantiates a default password protector.
func NewDefaultPasswordProtector() PasswordProtector {
	return DefaultPasswordProtector{}
}

// DefaultPasswordStore implements a place holder password protector. This is because, by default the
// password is maintained in clear.
type DefaultPasswordProtector struct {
}

// In the default protect implementation Encrypt is a dummy function as the passphrase is maintained in clear.
func (dp DefaultPasswordProtector) Encrypt(clearText string) (string, error) {
  return clearText, nil
}

// In the default protect implementation Decrypt is a dummy function as the passphrase is maintained in clear.
func (dp DefaultPasswordProtector) Decrypt(cipherText string) (string, error) {
  return cipherText, nil
}
