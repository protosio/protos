module github.com/protosio/protos

go 1.13

replace github.com/docker/docker => github.com/docker/engine v1.4.2-0.20180816081446-320063a2ad06

require (
	github.com/Masterminds/semver v1.4.2
	github.com/asdine/storm v2.1.2+incompatible
	github.com/cnf/structhash v0.0.0-20180104161610-62a607eb0224
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v1.13.1
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-units v0.4.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/emirpasic/gods v1.12.0
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/websocket v1.4.1
	github.com/heroku/docker-registry-client v0.0.0-20181004091502-47ecf50fd8d4
	github.com/icholy/killable v0.0.0-20170925194751-168925335d1e
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/pkg/errors v0.8.1
	github.com/rakyll/statik v0.1.6
	github.com/rs/xid v1.2.1
	github.com/shirou/gopsutil v2.18.12+incompatible
	github.com/sirupsen/logrus v1.4.2
	github.com/tidwall/gjson v1.3.2
	github.com/unrolled/render v1.0.1
	github.com/urfave/cli v1.21.0
	github.com/urfave/negroni v1.0.0
	github.com/vjeantet/jodaTime v0.0.0-20170816150230-be924ce213fb
	go.etcd.io/bbolt v1.3.3 // indirect
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	gopkg.in/yaml.v2 v2.2.2
)
