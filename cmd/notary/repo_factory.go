package main

import (
	"github.com/spf13/viper"

	"github.com/docker/notary"
	"github.com/docker/notary/client"
	"github.com/docker/notary/client_api/api"
	"github.com/docker/notary/tuf/data"
	"net/http"
)

type repoFactory func(gun data.GUN) (client.Repository, error)

func ConfigureRepo(v *viper.Viper, retriever notary.PassRetriever, onlineOperation bool) repoFactory {
	localRepo := func(gun data.GUN) (client.Repository, error) {
		var rt http.RoundTripper
		trustPin, err := getTrustPinning(v)
		if err != nil {
			return nil, err
		}
		if onlineOperation {
			rt, err = getTransport(v, gun, admin)
			if err != nil {
				return nil, err
			}
		}
		return client.NewFileCachedNotaryRepository(
			v.GetString("trust_dir"),
			gun,
			getRemoteTrustServer(v),
			rt,
			retriever,
			trustPin,
		)
	}
	remoteRepo := func(gun data.GUN) (client.Repository, error) {
		return api.NewClient(nil), nil
	}
	if v.GetBool("remote") {
		return remoteRepo
	}
	return localRepo
}
