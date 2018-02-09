// +build pkcs11, linux

package opencryptoki

var possiblePkcs11Libs = []string{
	"/usr/local/lib/opencryptoki/libopencryptoki.so",
	"/usr/local/lib64/opencryptoki/libopencryptoki.so",
	"/usr/lib64/opencryptoki/libopencryptoki.so",
	"/usr/lib/opencryptoki/libopencryptoki.so",
}
