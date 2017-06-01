package passphrase

// NewDefaultPassphraseProtector instantiates a default passphrase protector.
func NewDefaultPassphraseProtector() Protector {
	return DefaultPassphraseProtector{}
}

// DefaultPassphraseProtector implements a place holder passphrase protector. This is because, by default the
// passphrase is maintained in clear.
type DefaultPassphraseProtector struct {
}

// Encrypt is a dummy function in the default implementation as the passphrase is maintained in clear.
func (dp DefaultPassphraseProtector) Encrypt(clearText string) (string, error) {
	return clearText, nil
}

// Decrypt is a dummy function in the default implementation as the passphrase is maintained in clear.
func (dp DefaultPassphraseProtector) Decrypt(cipherText string) (string, error) {
	return cipherText, nil
}
