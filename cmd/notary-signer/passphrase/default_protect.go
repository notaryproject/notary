package passphrase

// NewDefaultPassphraseProtector instantiates a default passphrase protector.
func NewDefaultPassphraseProtector() PassphraseProtector {
	return DefaultPassphraseProtector{}
}

// DefaultPassphraseStore implements a place holder passphrase protector. This is because, by default the
// passphrase is maintained in clear.
type DefaultPassphraseProtector struct {
}

// In the default protect implementation Encrypt is a dummy function as the passphrase is maintained in clear.
func (dp DefaultPassphraseProtector) Encrypt(clearText string) (string, error) {
  return clearText, nil
}

// In the default protect implementation Decrypt is a dummy function as the passphrase is maintained in clear.
func (dp DefaultPassphraseProtector) Decrypt(cipherText string) (string, error) {
  return cipherText, nil
}
