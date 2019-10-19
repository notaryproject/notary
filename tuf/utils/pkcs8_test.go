package utils

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/tuf/data"
)

func getRSAKey() (data.PrivateKey, error) {
	raw := []byte(`INSERT PRIVATE KEY`)
	block, _ := pem.Decode(raw)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	privKey, err := RSAToPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return privKey, nil
}

func TestConvertTUFKeyToPKCS8(t *testing.T) {
	keys := []data.PrivateKey{}

	rsaKey, err := getRSAKey()
	require.NoError(t, err)
	keys = append(keys, rsaKey)

	ecKey, err := GenerateECDSAKey(rand.Reader)
	require.NoError(t, err)
	keys = append(keys, ecKey)

	edKey, err := GenerateED25519Key(rand.Reader)
	require.NoError(t, err)
	keys = append(keys, edKey)

	for _, k := range keys {
		testConvertKeyToPKCS8(t, k, nil)
		testConvertKeyToPKCS8(t, k, []byte("ponies"))
	}
}

func testConvertKeyToPKCS8(t *testing.T, privKey data.PrivateKey, password []byte) {
	der, err := ConvertTUFKeyToPKCS8(privKey, password)
	require.NoError(t, err, "could not convert private key to pkcs8")

	key, err := ParsePKCS8ToTufKey(der, password)
	require.NoError(t, err, "could not decrypt the newly created pkcs8 key")
	require.EqualValues(t, key.Private(), privKey.Private(), "private key did not match")
}

func TestParsePKCS8ToTufKey(t *testing.T) {
	keys := []struct {
		// algorithm (supports: rsa, ecdsa, ed25519)
		Algorithm string
		// unencrypted PKCS#8 key
		Unencrypted string
		// encrypted PKCS#8 key
		Encrypted string
		// password used to encrypt key
		Password string
	}{
		{
			Algorithm: "rsa",
			Unencrypted: `INSERT PRIVATE KEY
`,
			Encrypted: `INSERT PRIVATE KEY
`,
			Password: "ENTER PASSWORD",
		},
		{
			Algorithm: "ecdsa",
			Unencrypted: `-----BEGIN PRIVATE KEY-----
MHgCAQAwEAYHKoZIzj0CAQYFK4EEACEEYTBfAgEBBBxdqDSBsFWIAiQ99sRSQZrb
IFczI8UIRM7FD/SNoTwDOgAETbjLZYByEmU3oALoLIz4Xr814S8jMs3cAfJuywm/
kLGZ7y/1i56SXpTOByu6LHXrRokEi4hWQAc=
-----END PRIVATE KEY-----
`,
			Encrypted: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHOMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAjsfXVXuwOQnAICCAAw
HQYJYIZIAWUDBAEqBBB0O793rOzupOUavjLSiPmBBIGALJxsXCe8rLBfeviStIH0
A+1jCXUqXNm8D4npyNui/JRi/CjYPqgcO/2ulP8ppUAeTnLVQdhpv5ZOemK5ibMc
ECaNuzo40snnpve4duZEufkI9hXrO6MAMRT+G5ep1rKyIKboIPkzYUAdezj5ggUu
p1Gc8HB7j2SYjQX0Ybvlr6k=
-----END ENCRYPTED PRIVATE KEY-----
`,
			Password: "ponies",
		},
		{
			Algorithm: "ecdsa",
			Unencrypted: `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgiwt5YfD/xQdVwJZ0
2TpiJDQQ8DRHYVeWzIscya52BvChRANCAAT58IHVQJwbo3/MS/dFjh+xM85gVydX
xY+wxYDkaougZDPIgvu3+bQZ4xYSAnCGX7UJIiLloKuuuvbmXQlnSGqw
-----END PRIVATE KEY-----
`,
			Encrypted: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHeMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAhTsVpOdfLrSwICCAAw
HQYJYIZIAWUDBAEqBBDjOBeXbqjRH1FP6BkI6n3PBIGQgdYzQ7wOKaEp73WloPJl
966A0tiBCt2wy4LSueFjlh5NtF0o8odzg+zK9lHGD0MluWwM9LsDk5xtfcXCE6Ya
16PfcoAKE6l7VuGob4wDms/Y0G9DLhKXxErQOfzEolNZjN5RcZF9938ZPjQUDeIX
yYeijYPrkZWmdcrFPZUkKY5HQMIjXoULlUtlN9fFckLn
-----END ENCRYPTED PRIVATE KEY-----
`,
			Password: "ponies",
		},
		{
			Algorithm: "ecdsa",
			Unencrypted: `-----BEGIN PRIVATE KEY-----
MIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDCnjVESo9F+BLL4ZSt1
/ZU14MYlozCa7OyjdcdFjwMSajUZ4N0HVoBpJoeFh8DKaJ2hZANiAAQ4sTZRVUFU
p4IXBI9QEuwWh0Lsd/uUtZkpwXrjC4hpCQI3am7QC5Ct83VAtQ1WXBYg7EjIYNfi
CDbvJdq1y0IhdY138OQvsTaewiuYHUvRwjljxiSjpNEOB6AoD36FlqY=
-----END PRIVATE KEY-----
`,
			Encrypted: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIBDjBJBgkqhkiG9w0BBQ0wPDAbBgkqhkiG9w0BBQwwDgQIPvZXgw/gzOsCAggA
MB0GCWCGSAFlAwQBKgQQIEKz02wepEM1NmHfAHd76wSBwALvEtH7pz03/m6Z5Gkv
aafc7pfImJJzpLVIxGcNxrLkz1/WFoIpXHR8Bdde4dBEa8TYz91KvSNfnFjGE2xk
AiMSZyuObwGB0Vw1de8tDlpsVYftkZC4VrpRwUEksTUzYgHum/sqRlm7DmeXJq2t
540HZ9XhS2ZfT1bSqaCMX1s98KMpxDDHRDPX0SEBskyGqIWKLzLfYJnf07OlZ8r4
/oByTKigO+k9U40jNeuYW1aZeM8wqdApa89K48jWxftfNQ==
-----END ENCRYPTED PRIVATE KEY-----
`,
			Password: "ponies",
		},
		{
			Algorithm: "ecdsa",
			Unencrypted: `-----BEGIN PRIVATE KEY-----
MIHuAgEAMBAGByqGSM49AgEGBSuBBAAjBIHWMIHTAgEBBEIB0dZtwbNAy6K2iJF0
P9cTcwv2XnSCyeiIcOW/IG3I09pklXQNCw1igQdKSjZLZZRVS4OZMvuG774OPq9j
F7m/tkihgYkDgYYABADN4kHmO0/+mIHmIuC6id/lX04mZ9wZovU102l4VUdZA3e6
tZWDMdS2D3oqwhud2xCoHNw2ShxspzUISd/srH1pPAA3L2r2eZ6axrEqz1unbdBy
q1SyrsbtvDEJsP8STxiK3RSL9r00gqwlK44lp6dYQU3zd6IzS/69ACj/nmfX+YE4
AA==
-----END PRIVATE KEY-----
`,
			Encrypted: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIBTzBJBgkqhkiG9w0BBQ0wPDAbBgkqhkiG9w0BBQwwDgQI0ykvTci7+gkCAggA
MB0GCWCGSAFlAwQBKgQQdd7n4DwrOqViFfmLvN7ERASCAQCuGSdBA8+YCSPpVhcj
bYxIU0knqQPrlF5N3+BJMGVIz3468DVZFi9UtiiKRaHGfSxaUimqyQ0oUzXULEav
ZSp6abjxoBJZXPJWHu0f85s3DOjoks990a5o+J72gH/dH9yK/GgvR3MSXlkKoMut
Zqm3toQAF5RReU3E2wirnYEec8h4Zw5gy3FX63MuvX9mhlOtHTPhiZjoM2ogVQ3b
iH1BRu4nZF11wSZNxtWflLGMGZaQv3zBt4w7eu6AN25U5DiTy6ReWmsyB+kD/cZ5
eR9Noh2UlGMDPqTTrv2xqjjo/ieTusS7aGVQ0d8VESsxaAmCYx/kDY3FLtrKvsb5
kB1E
-----END ENCRYPTED PRIVATE KEY-----
`,
			Password: "ponies",
		},
		{
			Algorithm: "ed25519",
			Unencrypted: `-----BEGIN PRIVATE KEY-----
MHICAQAwCwYJKwYBBAHaRw8BBGDkASR4b08nd+A8txI3h+1hG+7EAIxE5cdbv3gt
rwib9ibygTpRt8XjscMv+vum4zFjI2pPZbhQn6lZlumHo7g35AEkeG9PJ3fgPLcS
N4ftYRvuxACMROXHW794La8Im/Y=
-----END PRIVATE KEY-----
`,
			Encrypted: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIHOMEkGCSqGSIb3DQEFDTA8MBsGCSqGSIb3DQEFDDAOBAhdzEpEYPy6ugICCAAw
HQYJYIZIAWUDBAEqBBCBSlYxcs3xzXsdF3JXvyvuBIGA9EaQcxFFk6d6jQiTzACC
66TFviduVfuqCr+VmulBAQZiCLj3PCpKugQ5z0aJ9CPfCRus5II3qS+qOXjI3OuH
hmevc2qO2C9bpsDteibfi9/tJJ8vVHdd+w44rSbLdLFro+p9CTT1R/VQillIFT4N
lXM12s3lyKsXT8bUcibd0gM=
-----END ENCRYPTED PRIVATE KEY-----
`,
			Password: "ponies",
		},
	}

	for _, k := range keys {
		testParsePKCS8ToTufKey(t, k.Algorithm, []byte(k.Unencrypted), []byte(k.Encrypted), []byte(k.Password))
	}
}

func testParsePKCS8ToTufKey(t *testing.T, alg string, unencypted, encrypted, password []byte) {
	block, _ := pem.Decode(unencypted)
	require.NotNil(t, block.Bytes, "could not decode PEM block properly")

	unencryptedKey, err := ParsePKCS8ToTufKey(block.Bytes, nil)
	require.NoError(t, err, "could not parse pkcs8 to tuf key")
	require.Equal(t, alg, unencryptedKey.Algorithm())

	block, _ = pem.Decode(encrypted)
	require.NotNil(t, block.Bytes, "could not decode PEM block properly")

	encryptedKey, err := ParsePKCS8ToTufKey(block.Bytes, password)
	require.NoError(t, err, "could not parse pkcs8 to tuf key")
	require.Equal(t, alg, encryptedKey.Algorithm())
	require.EqualValues(t, unencryptedKey.Private(), encryptedKey.Private())

	_, err = ParsePKCS8ToTufKey(block.Bytes, []byte("wrong password"))
	require.Error(t, err, "could parse key even with wrong password")
}

func TestPEMtoPEM(t *testing.T) {
	testInputPKCS1 := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpgIBAAKCAQEA3vRQI7s20MF0Zc3fywttsw72OkRXuTT0/JQrSuoilzOSaoLK
p7sYprIeIu9OeXqvBbwAxe3i1GViGwWM8cH9QqD05XhMz0Crr9vu2zHaZFEI9mgT
XcQxMQGntZ4xYV/rL/fddzj7+n1oKNvovS800NvPEMUkkgApdp5ES605V1q51tBp
LEYJ82xb5vT8cVseFYfA4G+gVqLNfQlasa0QsQT4YlVEDbbwT3/wuMG/m+wTx2p8
urhD+69oQbORkpqkNiEzMNidOrvtD7qyab+cUNamYU0CKOFn/KhWuoZV7EVYnc+o
evm7naYsenDq43Q5hGyacEuTjGtLnUG32d8RewIDAQABAoIBAQDeLOBfewSY6u8v
NAU7tVvP/6znS4uPiHJJ8O1jbgaiXkYd1dBVbWCXXRAjCA5PiC45rKuokfJkbdNh
0houIH5ck0D4GvWP4oY0bRqNXBShuw8PXY9O9V9/0oJpvga/XnJkDsCnOiX/7FCL
xvka7ZvYNfMWZx6WT4sCJZ0xPKHTpT09SzbhDOCLOsM4nWbs5xkXuEGPkD289z+N
OmENdcKDHz0mgYAr7hKJI3oAt2ogTjSyiQBLmxgudBUP5oJ1qY6/kYUCTYE3cjss
Y/mqfNcKtylQpTIUaUCw8BhOf3yJFA0GSI6C7hp96cjEk2dRQxAtYhSZJPA2uN+D
1/UIUeSBAoGBAO9VnVaTvkvc55upY9kUKnZcztJwG2Hf9FRrJrgC2RIaj3KNEImU
QyPgIVBXRzvdrvtvNJ6Tdb0cg6CBLJu7IeQjca2Lj4ACIzoSMF8ak6BP3cdB/fCc
2eHchqBKPWgZ23dq3CrpedtR6TbWLcswMrYdpZzpZ2eFPeStYxVhTLExAoGBAO56
tNX+Sp4N4cCeht6ttljLnGfAjeSBWv4z+xIqRyEoXbhchoppNxfnX34CrKmQM8MH
fEYHKo27u/SkhnMGyuDOtrd51+jhB0LXCOH3w6GI162HVTRJXND8nUtCPB9h/SpF
spr8Nk1Y2FtcfwkqhVphzExFzKm7eOPueevlsKJrAoGBALuvhh1Y60iOycpWggjA
ObRsf3yjkbWlbPOuu8Rd52C9F3UbjrZ1YFmH8FgSubgG1qwyvy8EMLbG36pE4niV
vbQs337bDQOzqXBmxywtqUt0llUmOUAxoOPwjlqxHYq/jE4PrOyx/2+wwpTQTUUk
XQBYK4Hrv718zdbA6gzgKsZhAoGBAMsnQufNMZmFL9xb737Assbf5QRJd1bCj1Zf
x7FIzMFFVtlYENDWIrW9R47cDmSAUGgC923cavbEh7A3e8V/ctKhpeuU40YidIIP
FyUQYNo57amI0R+yo1vw5roW2YrOedFKAIWg901asyzZFeskCufcyiHrkBbDeo+J
NtmrGJazAoGBAMOxKBm9HpFXB0h90+FA6aQgL6FfF578QTW+s5UsZUJjKunYuGmX
nYOb3iF6Kvvc/qmKDB1TInVtho1N5TcDpLTbO3/JPtJyolvYXjnBI7qXjpPxJeic
DbA919iDaEVtW1tQOQ9g7WBG0VorWUSroQSGi2o00tBJXEiEPmsJK2HL
-----END RSA PRIVATE KEY-----
`)

	testOutputPKCS8 := []byte(`-----BEGIN PRIVATE KEY-----
MIIEvgIBADALBgkqhkiG9w0BAQEEggSqMIIEpgIBAAKCAQEA3vRQI7s20MF0Zc3f
ywttsw72OkRXuTT0/JQrSuoilzOSaoLKp7sYprIeIu9OeXqvBbwAxe3i1GViGwWM
8cH9QqD05XhMz0Crr9vu2zHaZFEI9mgTXcQxMQGntZ4xYV/rL/fddzj7+n1oKNvo
vS800NvPEMUkkgApdp5ES605V1q51tBpLEYJ82xb5vT8cVseFYfA4G+gVqLNfQla
sa0QsQT4YlVEDbbwT3/wuMG/m+wTx2p8urhD+69oQbORkpqkNiEzMNidOrvtD7qy
ab+cUNamYU0CKOFn/KhWuoZV7EVYnc+oevm7naYsenDq43Q5hGyacEuTjGtLnUG3
2d8RewIDAQABAoIBAQDeLOBfewSY6u8vNAU7tVvP/6znS4uPiHJJ8O1jbgaiXkYd
1dBVbWCXXRAjCA5PiC45rKuokfJkbdNh0houIH5ck0D4GvWP4oY0bRqNXBShuw8P
XY9O9V9/0oJpvga/XnJkDsCnOiX/7FCLxvka7ZvYNfMWZx6WT4sCJZ0xPKHTpT09
SzbhDOCLOsM4nWbs5xkXuEGPkD289z+NOmENdcKDHz0mgYAr7hKJI3oAt2ogTjSy
iQBLmxgudBUP5oJ1qY6/kYUCTYE3cjssY/mqfNcKtylQpTIUaUCw8BhOf3yJFA0G
SI6C7hp96cjEk2dRQxAtYhSZJPA2uN+D1/UIUeSBAoGBAO9VnVaTvkvc55upY9kU
KnZcztJwG2Hf9FRrJrgC2RIaj3KNEImUQyPgIVBXRzvdrvtvNJ6Tdb0cg6CBLJu7
IeQjca2Lj4ACIzoSMF8ak6BP3cdB/fCc2eHchqBKPWgZ23dq3CrpedtR6TbWLcsw
MrYdpZzpZ2eFPeStYxVhTLExAoGBAO56tNX+Sp4N4cCeht6ttljLnGfAjeSBWv4z
+xIqRyEoXbhchoppNxfnX34CrKmQM8MHfEYHKo27u/SkhnMGyuDOtrd51+jhB0LX
COH3w6GI162HVTRJXND8nUtCPB9h/SpFspr8Nk1Y2FtcfwkqhVphzExFzKm7eOPu
eevlsKJrAoGBALuvhh1Y60iOycpWggjAObRsf3yjkbWlbPOuu8Rd52C9F3UbjrZ1
YFmH8FgSubgG1qwyvy8EMLbG36pE4niVvbQs337bDQOzqXBmxywtqUt0llUmOUAx
oOPwjlqxHYq/jE4PrOyx/2+wwpTQTUUkXQBYK4Hrv718zdbA6gzgKsZhAoGBAMsn
QufNMZmFL9xb737Assbf5QRJd1bCj1Zfx7FIzMFFVtlYENDWIrW9R47cDmSAUGgC
923cavbEh7A3e8V/ctKhpeuU40YidIIPFyUQYNo57amI0R+yo1vw5roW2YrOedFK
AIWg901asyzZFeskCufcyiHrkBbDeo+JNtmrGJazAoGBAMOxKBm9HpFXB0h90+FA
6aQgL6FfF578QTW+s5UsZUJjKunYuGmXnYOb3iF6Kvvc/qmKDB1TInVtho1N5TcD
pLTbO3/JPtJyolvYXjnBI7qXjpPxJeicDbA919iDaEVtW1tQOQ9g7WBG0VorWUSr
oQSGi2o00tBJXEiEPmsJK2HL
-----END PRIVATE KEY-----
`)

	block, _ := pem.Decode(testInputPKCS1)
	require.NotEmpty(t, block)

	rsaKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	testPrivKey, err := RSAToPrivateKey(rsaKey)
	require.NoError(t, err)

	der, err := ConvertTUFKeyToPKCS8(testPrivKey, nil)
	require.NoError(t, err, "could not convert pkcs1 to pkcs8")

	testOutput := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	require.EqualValues(t, testOutputPKCS8, testOutput)
}
