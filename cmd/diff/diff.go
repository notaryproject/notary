package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"fmt"
	"time"
	"path"

	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/docker/go-connections/tlsconfig"

	"github.com/theupdateframework/notary/tuf/data"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/utils"
	"github.com/theupdateframework/notary/client"
)

func (n *notaryCommander) Diff(cmd *cobra.Command, args []string) {
	cmd.Println("Running diff on", "gun")

	if len(args) < 3 {
		cmd.Usage()
		logrus.Fatalf("Must specify a GUN and two timestamp hashes")
	}
	config, err := n.parseConfig()
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	gun := data.GUN(args[0])
	hash1 := args[1]
	hash2 := args[2]

	baseURL := getRemoteTrustServer(config)

	rt, err := getTransport(config, gun, admin)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	diff, err := client.NewDiff(gun, baseURL, rt, hash1, hash2)
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	out, err := json.Marshal(diff)
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	cmd.Println(out)
}


type passwordStore struct {
	anonymous bool
}

func getUsername(input chan string) {
	in := bufio.NewReader(os.Stdin)
	result, err := in.ReadString('\n')
	if err != nil {
		logrus.Errorf("error processing username input: %s", err)
		input <- ""
	}
	input <- result
}

func (ps passwordStore) Basic(u *url.URL) (string, string) {
	// if it's not a terminal, don't wait on input
	if ps.anonymous {
		return "", ""
	}

	auth := os.Getenv("NOTARY_AUTH")
	if auth != "" {
		dec, err := base64.StdEncoding.DecodeString(auth)
		if err != nil {
			logrus.Error("Could not base64-decode authentication string")
			return "", ""
		}
		plain := string(dec)

		i := strings.Index(plain, ":")
		if i == 0 {
			logrus.Error("Authentication string with zero-legnth username")
			return "", ""
		} else if i > -1 {
			username := plain[:i]
			password := plain[i+1:]
			password = strings.TrimSpace(password)
			return username, password
		}

		logrus.Error("Malformatted authentication string; format must be <username>:<password>")
		return "", ""
	}

	stdin := bufio.NewReader(os.Stdin)
	input := make(chan string, 1)
	fmt.Fprintf(os.Stdout, "Enter username: ")
	go getUsername(input)
	var username string
	select {
	case i := <-input:
		username = strings.TrimSpace(i)
		if username == "" {
			return "", ""
		}
	case <-time.After(30 * time.Second):
		logrus.Error("timeout when retrieving username input")
		return "", ""
	}

	fmt.Fprintf(os.Stdout, "Enter password: ")
	passphrase, err := passphrase.GetPassphrase(stdin)
	fmt.Fprintln(os.Stdout)
	if err != nil {
		logrus.Errorf("error processing password input: %s", err)
		return "", ""
	}
	password := strings.TrimSpace(string(passphrase))

	return username, password
}

// to comply with the CredentialStore interface
func (ps passwordStore) RefreshToken(u *url.URL, service string) string {
	return ""
}

// to comply with the CredentialStore interface
func (ps passwordStore) SetRefreshToken(u *url.URL, service string, token string) {
}

type httpAccess int

const (
	readOnly httpAccess = iota
	readWrite
	admin
)

// It correctly handles the auth challenge/credentials required to interact
// with a notary server over both HTTP Basic Auth and the JWT auth implemented
// in the notary-server
// The readOnly flag indicates if the operation should be performed as an
// anonymous read only operation. If the command entered requires write
// permissions on the server, readOnly must be false
func getTransport(config *viper.Viper, gun data.GUN, permission httpAccess) (http.RoundTripper, error) {
	// Attempt to get a root CA from the config file. Nil is the host defaults.
	rootCAFile := utils.GetPathRelativeToConfig(config, "remote_server.root_ca")
	clientCert := utils.GetPathRelativeToConfig(config, "remote_server.tls_client_cert")
	clientKey := utils.GetPathRelativeToConfig(config, "remote_server.tls_client_key")

	insecureSkipVerify := false
	if config.IsSet("remote_server.skipTLSVerify") {
		insecureSkipVerify = config.GetBool("remote_server.skipTLSVerify")
	}

	if clientCert == "" && clientKey != "" || clientCert != "" && clientKey == "" {
		return nil, fmt.Errorf("either pass both client key and cert, or neither")
	}

	tlsConfig, err := tlsconfig.Client(tlsconfig.Options{
		CAFile:             rootCAFile,
		InsecureSkipVerify: insecureSkipVerify,
		CertFile:           clientCert,
		KeyFile:            clientKey,
		ExclusiveRootPools: true,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to configure TLS: %s", err.Error())
	}

	base := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
		DisableKeepAlives:   true,
	}
	trustServerURL := getRemoteTrustServer(config)
	return tokenAuth(trustServerURL, base, gun, permission)
}

func tokenAuth(trustServerURL string, baseTransport *http.Transport, gun data.GUN,
	permission httpAccess) (http.RoundTripper, error) {

	// TODO(dmcgowan): add notary specific headers
	authTransport := transport.NewTransport(baseTransport)
	pingClient := &http.Client{
		Transport: authTransport,
		Timeout:   5 * time.Second,
	}
	endpoint, err := url.Parse(trustServerURL)
	if err != nil {
		return nil, fmt.Errorf("Could not parse remote trust server url (%s): %s", trustServerURL, err.Error())
	}
	if endpoint.Scheme == "" {
		return nil, fmt.Errorf("Trust server url has to be in the form of http(s)://URL:PORT. Got: %s", trustServerURL)
	}
	subPath, err := url.Parse(path.Join(endpoint.Path, "/v2") + "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to parse v2 subpath. This error should not have been reached. Please report it as an issue at https://github.com/theupdateframework/notary/issues: %s", err.Error())
	}
	endpoint = endpoint.ResolveReference(subPath)
	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		logrus.Errorf("could not reach %s: %s", trustServerURL, err.Error())
		logrus.Info("continuing in offline mode")
		return nil, nil
	}
	// non-nil err means we must close body
	defer resp.Body.Close()
	if (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices) &&
		resp.StatusCode != http.StatusUnauthorized {
		// If we didn't get a 2XX range or 401 status code, we're not talking to a notary server.
		// The http client should be configured to handle redirects so at this point, 3XX is
		// not a valid status code.
		logrus.Errorf("could not reach %s: %d", trustServerURL, resp.StatusCode)
		logrus.Info("continuing in offline mode")
		return nil, nil
	}

	challengeManager := challenge.NewSimpleManager()
	if err := challengeManager.AddResponse(resp); err != nil {
		return nil, err
	}

	ps := passwordStore{anonymous: permission == readOnly}

	var actions []string
	switch permission {
	case admin:
		actions = []string{"*"}
	case readWrite:
		actions = []string{"push", "pull"}
	case readOnly:
		actions = []string{"pull"}
	default:
		return nil, fmt.Errorf("Invalid permission requested for token authentication of gun %s", gun)
	}

	tokenHandler := auth.NewTokenHandler(authTransport, ps, gun.String(), actions...)
	basicHandler := auth.NewBasicHandler(ps)

	modifier := auth.NewAuthorizer(challengeManager, tokenHandler, basicHandler)

	if permission != readOnly {
		return newAuthRoundTripper(transport.NewTransport(baseTransport, modifier)), nil
	}

	// Try to authenticate read only repositories using basic username/password authentication
	return newAuthRoundTripper(transport.NewTransport(baseTransport, modifier),
		transport.NewTransport(baseTransport, auth.NewAuthorizer(challengeManager, auth.NewTokenHandler(authTransport, passwordStore{anonymous: false}, gun.String(), actions...)))), nil
}

func getRemoteTrustServer(config *viper.Viper) string {
	if configRemote := config.GetString("remote_server.url"); configRemote != "" {
		return configRemote
	}
	return defaultServerURL
}

func getTrustPinning(config *viper.Viper) (trustpinning.TrustPinConfig, error) {
	var ok bool
	// Need to parse out Certs section from config
	certMap := config.GetStringMap("trust_pinning.certs")
	resultCertMap := make(map[string][]string)
	for gun, certSlice := range certMap {
		var castedCertSlice []interface{}
		if castedCertSlice, ok = certSlice.([]interface{}); !ok {
			return trustpinning.TrustPinConfig{}, fmt.Errorf("invalid format for trust_pinning.certs")
		}
		certsForGun := make([]string, len(castedCertSlice))
		for idx, certIDInterface := range castedCertSlice {
			if certID, ok := certIDInterface.(string); ok {
				certsForGun[idx] = certID
			} else {
				return trustpinning.TrustPinConfig{}, fmt.Errorf("invalid format for trust_pinning.certs")
			}
		}
		resultCertMap[gun] = certsForGun
	}
	return trustpinning.TrustPinConfig{
		DisableTOFU: config.GetBool("trust_pinning.disable_tofu"),
		CA:          config.GetStringMapString("trust_pinning.ca"),
		Certs:       resultCertMap,
	}, nil
}

// authRoundTripper tries to authenticate the requests via multiple HTTP transactions (until first succeed)
type authRoundTripper struct {
	trippers []http.RoundTripper
}

func newAuthRoundTripper(trippers ...http.RoundTripper) http.RoundTripper {
	return &authRoundTripper{trippers: trippers}
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	var resp *http.Response
	// Try all run all transactions
	for _, t := range a.trippers {
		var err error
		resp, err = t.RoundTrip(req)
		// Reject on error
		if err != nil {
			return resp, err
		}

		// Stop when request is authorized/unknown error
		if resp.StatusCode != http.StatusUnauthorized {
			return resp, nil
		}
	}

	// Return the last response
	return resp, nil
}
