package signer_test

// This module tests the Signer RPC interface using the Signer client

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/docker/notary/cryptoservice"
	pb "github.com/docker/notary/proto"
	"github.com/docker/notary/signer"
	"github.com/docker/notary/signer/api"
	"github.com/docker/notary/signer/client"
	"github.com/docker/notary/trustmanager"
	"github.com/docker/notary/tuf/data"
	"github.com/docker/notary/tuf/signed"
	"github.com/docker/notary/tuf/testutils/interfaces"
	"github.com/docker/notary/tuf/utils"
	"github.com/stretchr/testify/require"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func socketDialer(socketAddr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", socketAddr, timeout)
}

func setUpSignerClient(t *testing.T, grpcServer *grpc.Server) (*client.NotarySigner, *grpc.ClientConn, func()) {
	socketFile, err := ioutil.TempFile("", "notary-grpc-test")
	require.NoError(t, err)
	socketFile.Close()
	os.Remove(socketFile.Name())

	lis, err := net.Listen("unix", socketFile.Name())
	require.NoError(t, err, "unable to open socket to listen")

	go grpcServer.Serve(lis)

	// client setup
	clientConn, err := grpc.Dial(socketFile.Name(), grpc.WithInsecure(), grpc.WithDialer(socketDialer))
	require.NoError(t, err, "unable to connect to socket as a GRPC client")

	signerClient := client.NewNotarySigner(clientConn)

	cleanup := func() {
		clientConn.Close()
		grpcServer.Stop()
		os.Remove(socketFile.Name())
	}

	return signerClient, clientConn, cleanup
}

type stubServer struct {
	healthServer *health.Server
}

func (s stubServer) CreateKey(ctx context.Context, req *pb.CreateKeyRequest) (*pb.PublicKey, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s stubServer) DeleteKey(ctx context.Context, keyID *pb.KeyID) (*pb.Void, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s stubServer) GetKeyInfo(ctx context.Context, keyID *pb.KeyID) (*pb.GetKeyInfoResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s stubServer) Sign(ctx context.Context, sr *pb.SignatureRequest) (*pb.Signature, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s stubServer) CheckHealth(ctx context.Context, v *pb.Void) (*pb.HealthStatus, error) {
	return &pb.HealthStatus{}, nil
}

func getStubbedHealthServer(hs *health.Server) *grpc.Server {
	s := stubServer{healthServer: hs}
	gServer := grpc.NewServer()
	pb.RegisterKeyManagementServer(gServer, s)
	pb.RegisterSignerServer(gServer, s)

	if s.healthServer != nil {
		healthpb.RegisterHealthServer(gServer, s.healthServer)
	}

	return gServer
}

// CheckHealth does not succeed if the KM server is unhealthy
func TestHealthCheckKMUnhealthy(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_NOT_SERVING)

	s := getStubbedHealthServer(hs)
	signerClient, _, cleanup := setUpSignerClient(t, s)
	defer cleanup()
	require.Error(t, signerClient.CheckHealth(1*time.Second, "grpc.health.v1.Health.KeyManagement"))
}

// CheckHealth does not succeed if the health check to the KM server errors
func TestHealthCheckKMError(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_NOT_SERVING)

	s := getStubbedHealthServer(hs)
	signerClient, _, cleanup := setUpSignerClient(t, s)
	defer cleanup()
	require.Error(t, signerClient.CheckHealth(1*time.Second, "grpc.health.v1.Health.KeyManagement"))
}

// CheckHealth does not succeed if the health check to the KM server times out
func TestHealthCheckKMTimeout(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_NOT_SERVING)

	s := getStubbedHealthServer(hs)
	signerClient, _, cleanup := setUpSignerClient(t, s)
	defer cleanup()

	err := signerClient.CheckHealth(0*time.Second, "grpc.health.v1.Health.KeyManagement")
	require.Error(t, err)
	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
}

// CheckHealth succeeds if KM is healthy and reachable.
func TestHealthCheckKMHealthy(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_SERVING)

	s := getStubbedHealthServer(hs)
	signerClient, _, cleanup := setUpSignerClient(t, s)
	defer cleanup()
	require.NoError(t, signerClient.CheckHealth(1*time.Second, "grpc.health.v1.Health.KeyManagement"))
}

// CheckHealth fails immediately if not connected to the server.
func TestHealthCheckConnectionDied(t *testing.T) {
	hs := health.NewServer()
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_NOT_SERVING)

	s := getStubbedHealthServer(hs)
	signerClient, conn, cleanup := setUpSignerClient(t, s)
	defer cleanup()
	conn.Close()
	require.Error(t, signerClient.CheckHealth(1*time.Second, "grpc.health.v1.Health.KeyManagement"))
}

// TestHealthCheckForOverallStatus query for signer's overall health status
// with the empty provided service name.
func TestHealthCheckForOverallStatus(t *testing.T) {
	hs := health.NewServer()

	s := getStubbedHealthServer(hs)
	signerClient, _, cleanup := setUpSignerClient(t, s)
	defer cleanup()

	// both of the service are NOT SERVING, expect the health check for overall status to be failed.
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_NOT_SERVING)
	hs.SetServingStatus("grpc.health.v1.Health.Signer", healthpb.HealthCheckResponse_NOT_SERVING)
	err := signerClient.CheckHealth(1*time.Second, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOT_SERVING, want SERVING")

	// change the status of KeyManagement to SERVING and keep the status of Siger
	// still be NOT SERVING, expect the health check for overall status to be failed.
	hs.SetServingStatus("grpc.health.v1.Health.KeyManagement", healthpb.HealthCheckResponse_SERVING)
	err = signerClient.CheckHealth(1*time.Second, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "NOT_SERVING, want SERVING")

	// change the status of Signer to SERVING, expect the health check for overall status to success.
	hs.SetServingStatus("grpc.health.v1.Health.Signer", healthpb.HealthCheckResponse_SERVING)
	err = signerClient.CheckHealth(1*time.Second, "")
	require.NoError(t, err)
}

var constPass = func(string, string, bool, int) (string, bool, error) {
	return "constant", false, nil
}

func setUpSignerServer(t *testing.T, store trustmanager.KeyStore) *grpc.Server {
	cryptoService := cryptoservice.NewCryptoService(store)
	cryptoServices := signer.CryptoServiceIndex{
		data.ED25519Key: cryptoService,
		data.RSAKey:     cryptoService,
		data.ECDSAKey:   cryptoService,
	}

	fakeHealth := func() map[string]string { return nil }

	//server setup
	grpcServer := grpc.NewServer()
	pb.RegisterKeyManagementServer(grpcServer, &api.KeyManagementServer{
		CryptoServices: cryptoServices,
		HealthChecker:  fakeHealth,
	})
	pb.RegisterSignerServer(grpcServer, &api.SignerServer{
		CryptoServices: cryptoServices,
		HealthChecker:  fakeHealth,
	})

	return grpcServer
}

func TestGetPrivateKeyAndSignWithExistingKey(t *testing.T) {
	key, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err, "could not generate key")

	memStore := trustmanager.NewKeyMemoryStore(constPass)
	err = memStore.AddKey(trustmanager.KeyInfo{Role: data.CanonicalTimestampRole, Gun: "gun"}, key)
	require.NoError(t, err, "could not add key to store")

	signerClient, _, cleanup := setUpSignerClient(t, setUpSignerServer(t, memStore))
	defer cleanup()

	privKey, role, err := signerClient.GetPrivateKey(key.ID())
	require.NoError(t, err)
	require.Equal(t, data.CanonicalTimestampRole, role)
	require.NotNil(t, privKey)

	msg := []byte("message!")
	sig, err := privKey.Sign(rand.Reader, msg, nil)
	require.NoError(t, err)

	err = signed.Verifiers[data.ECDSASignature].Verify(
		data.PublicKeyFromPrivate(key), sig, msg)
	require.NoError(t, err)
}

func TestCannotSignWithKeyThatDoesntExist(t *testing.T) {
	memStore := trustmanager.NewKeyMemoryStore(constPass)

	_, conn, cleanup := setUpSignerClient(t, setUpSignerServer(t, memStore))
	defer cleanup()

	key, err := utils.GenerateECDSAKey(rand.Reader)
	require.NoError(t, err, "could not generate key")

	remotePrivKey := client.NewRemotePrivateKey(data.PublicKeyFromPrivate(key), pb.NewSignerClient(conn))

	msg := []byte("message!")
	_, err = remotePrivKey.Sign(rand.Reader, msg, nil)
	require.Error(t, err)
	// error translated into grpc error, so compare the text
	require.Equal(t, trustmanager.ErrKeyNotFound{KeyID: key.ID()}.Error(), grpc.ErrorDesc(err))
}

// Signer conforms to the signed.CryptoService interface behavior
func TestCryptoSignerInterfaceBehavior(t *testing.T) {
	memStore := trustmanager.NewKeyMemoryStore(constPass)
	signerClient, _, cleanup := setUpSignerClient(t, setUpSignerServer(t, memStore))
	defer cleanup()

	interfaces.EmptyCryptoServiceInterfaceBehaviorTests(t, signerClient)
	interfaces.CreateGetKeyCryptoServiceInterfaceBehaviorTests(t, signerClient, data.ECDSAKey)
	// can't test AddKey, because the signer does not support adding keys, and can't test listing
	// keys because the signer doesn't support listing keys.
}
