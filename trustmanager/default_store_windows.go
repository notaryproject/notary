package trustmanager

import "github.com/docker/docker-credential-helpers/wincred"

const defaultCredentialsStore = "wincred"

var helper = wincred.Wincred{}
