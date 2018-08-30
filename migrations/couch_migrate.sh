#!/usr/bin/env sh

# When run in the docker containers, the working directory
# is the root of the repo.

iter=0

# have to poll for DB to come up
echo "trying to contact CouchDB for 30 seconds before failing"
case $SERVICE_NAME in
	notary_server)
		# have to poll for DB to come up
		until notary-server -config=fixtures/server-config.couch.json -bootstrap
		do
			iter=$(( iter+1 ))
			if [[ $iter -gt 30 ]]; then
				echo "CouchDB failed to come up within 30 seconds"
				exit 1;
			fi
			echo "waiting for CouchDB to come up."
			sleep 1
		done
		;;
	notary_signer)
		# have to poll for DB to come up
		until notary-signer -config=fixtures/signer-config.couch.json -bootstrap
		do
			iter=$(( iter+1 ))
			if [[ $iter -gt 30 ]]; then
				echo "CouchDB failed to come up within 30 seconds"
				exit 1;
			fi
			echo "waiting for CouchDB to come up."
			sleep 1
		done
		;;
esac
echo "successfully reached and updated CouchDB"
