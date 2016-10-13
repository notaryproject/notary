package notary

import (
	"time"
)

// application wide constants
const (
	// MaxDownloadSize is the maximum size we'll download for metadata if no limit is given
	MaxDownloadSize int64 = 100 << 20
	// MaxTimestampSize is the maximum size of timestamp metadata - 1MiB.
	MaxTimestampSize int64 = 1 << 20
	// MinRSABitSize is the minimum bit size for RSA keys allowed in notary
	MinRSABitSize = 2048
	// MinThreshold requires a minimum of one threshold for roles; currently we do not support a higher threshold
	MinThreshold = 1
	// PrivKeyPerms are the file permissions to use when writing private keys to disk
	PrivKeyPerms = 0700
	// PubCertPerms are the file permissions to use when writing public certificates to disk
	PubCertPerms = 0755
	// Sha256HexSize is how big a Sha256 hex is in number of characters
	Sha256HexSize = 64
	// Sha512HexSize is how big a Sha512 hex is in number of characters
	Sha512HexSize = 128
	// SHA256 is the name of SHA256 hash algorithm
	SHA256 = "sha256"
	// SHA512 is the name of SHA512 hash algorithm
	SHA512 = "sha512"
	// TrustedCertsDir is the directory, under the notary repo base directory, where trusted certs are stored
	TrustedCertsDir = "trusted_certificates"
	// PrivDir is the directory, under the notary repo base directory, where private keys are stored
	PrivDir = "private"
	// RootKeysSubdir is the subdirectory under PrivDir where root private keys are stored
	// DEPRECATED: The only reason we need this constant is compatibility with older versions
	RootKeysSubdir = "root_keys"
	// NonRootKeysSubdir is the subdirectory under PrivDir where non-root private keys are stored
	// DEPRECATED: The only reason we need this constant is compatibility with older versions
	NonRootKeysSubdir = "tuf_keys"
	// KeyExtension is the file extension to use for private key files
	KeyExtension = "key"

	// Day is a duration of one day
	Day  = 24 * time.Hour
	Year = 365 * Day

	// NotaryRootExpiry is the duration representing the expiry time of the Root role
	NotaryRootExpiry      = 10 * Year
	NotaryTargetsExpiry   = 3 * Year
	NotarySnapshotExpiry  = 3 * Year
	NotaryTimestampExpiry = 14 * Day

	ConsistentMetadataCacheMaxAge = 30 * Day
	CurrentMetadataCacheMaxAge    = 5 * time.Minute
	// CacheMaxAgeLimit is the generally recommended maximum age for Cache-Control headers
	// (one year, in seconds, since one year is forever in terms of internet
	// content)
	CacheMaxAgeLimit = 1 * Year

	MySQLBackend     = "mysql"
	MemoryBackend    = "memory"
	PostgresBackend  = "postgres"
	SQLiteBackend    = "sqlite3"
	RethinkDBBackend = "rethinkdb"

	DefaultImportRole = "delegation"

	// HealthCheckKeyManagement and HealthCheckSigner are the grpc service name
	// for "KeyManagement" and "Signer" respectively which used for health check.
	// The "Overall" indicates the querying for overall status of the server.
	HealthCheckKeyManagement = "grpc.health.v1.Health.KeyManagement"
	HealthCheckSigner        = "grpc.health.v1.Health.Signer"
	HealthCheckOverall       = "grpc.health.v1.Health.Overall"

	// PrivExecPerms indicates the file permissions for directory
	// and PrivNoExecPerms for file.
	PrivExecPerms   = 0700
	PrivNoExecPerms = 0600
)

// enum to use for setting and retrieving values from contexts
const (
	CtxKeyMetaStore CtxKey = iota
	CtxKeyKeyAlgo
	CtxKeyCryptoSvc
	CtxKeyRepo
)

// NotaryDefaultExpiries is the construct used to configure the default expiry times of
// the various role files.
var NotaryDefaultExpiries = map[string]time.Duration{
	"root":      NotaryRootExpiry,
	"targets":   NotaryTargetsExpiry,
	"snapshot":  NotarySnapshotExpiry,
	"timestamp": NotaryTimestampExpiry,
}

// NotarySupportedBackends contains the backends we would like to support at present
var NotarySupportedBackends = []string{
	MemoryBackend,
	MySQLBackend,
	SQLiteBackend,
	RethinkDBBackend,
	PostgresBackend,
}
