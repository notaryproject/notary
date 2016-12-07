package trustmanager

import "github.com/docker/docker-credential-helpers/secretservice"

const defaultCredentialsStore = "secretservice"

var helper = secretservice.Secretservice{}
