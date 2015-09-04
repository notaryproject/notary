#!/usr/bin/env bash
set -e

cd "$(dirname "$BASH_SOURCE")/.."
rm -rf vendor/
source 'hack/.vendor-helpers.sh'

# global (used in multiple deps)
clone git github.com/Sirupsen/logrus v0.8.2
clone git github.com/agl/ed25519 d2b94fd789ea21d12fac1a4443dd3a3f79cda72c
clone git github.com/tent/canonical-json-go 96e4ba3a7613a1216cbd1badca4efe382adea337
clone git github.com/dvsekhvalnov/jose2go 5307afb3bb6169b0f68cdf519a5964c843344441
clone git github.com/go-sql-driver/mysql 0cc29e9fe8e25c2c58cf47bcab566e029bbaa88b
clone git github.com/miekg/pkcs11 88c9f842544e629ec046105d7fb50d5daafae737
clone git github.com/golang/protobuf 655cdfa588ea190e901bc5590e65d5621688847c
clone git golang.org/x/net 1dfe7915deaf3f80b962c163b918868d8a6d8974 https://github.com/golang/net.git
clone git github.com/mattn/go-sqlite3 b4142c444a8941d0d92b0b7103a24df9cd815e42

clone git github.com/jinzhu/gorm 82d726bbfd8cefbe2dcdc7f7f0484551c0d40433
clone git github.com/lib/pq 0dad96c0b94f8dee039aa40467f767467392a0af

clone git github.com/gorilla/mux e444e69cbd2e2e3e0749a2f3c717cec491552bbf
clone git github.com/gorilla/context 14f550f51af52180c2eefed15e5fd18d63c0a64a

# testing deps
clone git github.com/stretchr/testify 089c7181b8c728499929ff09b62d3fdd8df8adff
clone git github.com/DATA-DOG/go-sqlmock ed4836e31d3e9e77420e442ed9b864df55370ee0

# gotuf pkg and deps
clone git github.com/endophage/gotuf 4c04df9067a595ead06309f38021ea445acc1d1c
clone git github.com/google/gofuzz bbcb9da2d746f8bdbd6a936686a0a6067ada0ec5
clone hg code.google.com/p/gosqlite 74691fb6f83716190870cde1b658538dd4b18eb0
clone git github.com/google/gofuzz bbcb9da2d746f8bdbd6a936686a0a6067ada0ec5

# grpc deps
clone git google.golang.org/grpc 97f42dd262e97f4632986eddbc74c19fa022ea08 https://github.com/grpc/grpc-go.git
clone git github.com/bradfitz/http2 97124afb234048ae0c91b8883c59fcd890bf8145
clone git golang.org/x/oauth2 ce5ea7da934b76b1066c527632359e2b8f65db97 https://github.com/golang/oauth2.git
clone git google.golang.org/cloud f20d6dcccb44ed49de45ae3703312cb46e627db1 https://github.com/GoogleCloudPlatform/gcloud-golang.git

# bugsnag deps
clone git github.com/bugsnag/bugsnag-go 13fd6b8acda029830ef9904df6b63be0a83369d0
clone git github.com/bugsnag/osext 0dd3f918b21bec95ace9dc86c7e70266cfc5c702
clone git github.com/bugsnag/panicwrap e5f9854865b9778a45169fc249e99e338d4d6f27

# distribution dep
clone git github.com/docker/distribution fed58bd2d3c096055c0e69c2fb86c9a4965d1b8b
clone git github.com/docker/libtrust fa567046d9b14f6aa788882a950d69651d230b21
clone git golang.org/x/crypto bfc286917c5fcb7420d7e3092b50bbfd31b38a98 https://github.com/golang/crypto.git

# docker dep
clone git github.com/docker/docker 786b29d4db80a6175e72b47a794ee044918ba734

# cobra deps
clone git github.com/spf13/cobra 312092086bed4968099259622145a0c9ae280064
clone git github.com/spf13/pflag 5644820622454e71517561946e3d94b9f9db6842
clone git github.com/inconshreveable/mousetrap 76626ae9c91c4f2a10f34cad8ce83ea42c93bb75

# viper deps
clone git github.com/spf13/viper be5ff3e4840cf692388bde7a057595a474ef379e
clone git github.com/spf13/cast 4d07383ffe94b5e5a6fa3af9211374a4507a0184
clone git github.com/spf13/jwalterweatherman 3d60171a64319ef63c78bd45bd60e6eab1e75f8b
clone git github.com/kr/pretty bc9499caa0f45ee5edb2f0209fbd61fbf3d9018f
clone git github.com/kr/text 6807e777504f54ad073ecef66747de158294b639
clone git github.com/magiconair/properties 624009598839a9432bd97bb75552389422357723
clone git github.com/mitchellh/mapstructure 2caf8efc93669b6c43e0441cdc6aed17546c96f3
clone git gopkg.in/yaml.v2 bef53efd0c76e49e6de55ead051f886bea7e9420
clone git github.com/BurntSushi/toml bd2bdf7f18f849530ef7a1c29a4290217cab32a1

clean
