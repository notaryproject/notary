package passwordwrap

// NewDefaultPasswordProtector instantiates a default password protector.
func NewDefaultPasswordProtector() Protector {
	return DefaultPasswordProtector{}
}

// DefaultPasswordProtector implements a place holder password protector. This is because, by default the
// password is maintained in clear.
type DefaultPasswordProtector struct {
}

// Encrypt is a dummy function in the default implementation as the password is maintained in clear.
func (dp DefaultPasswordProtector) Encrypt(clearText string) (string, error) {
	return clearText, nil
}

// Decrypt is a dummy function in the default implementation as the password is maintained in clear.
func (dp DefaultPasswordProtector) Decrypt(cipherText string) (string, error) {
	return cipherText, nil
}
