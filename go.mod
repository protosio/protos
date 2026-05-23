module github.com/protosio/protos

go 1.26.3

replace (
	github.com/dolthub/dolt/go => github.com/nustiueudinastea/dolt/go v0.0.0-20260519092250-dc384c58ba80
	github.com/dolthub/driver => github.com/nustiueudinastea/doltsqldriver v0.0.0-20260513100158-ab0b5e5fa8c2
	github.com/protosio/protos/network/modules/ipc => ./network/modules/ipc
	github.com/protosio/protos/network/modules/networkextension => ./network/modules/networkextension
	github.com/protosio/protos/network/modules/wireguard => ./network/modules/wireguard
)

require (
	filippo.io/edwards25519 v1.2.0
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Masterminds/semver v1.5.0
	github.com/birros/go-libp2p-grpc v0.0.0
	github.com/bokwoon95/sq v0.5.1
	github.com/bramvdbogaerde/go-scp v1.6.0
	github.com/cheggaaa/pb/v3 v3.1.7
	github.com/containerd/containerd/v2 v2.3.0
	github.com/containerd/errdefs v1.0.0
	github.com/containerd/platforms v1.0.0-rc.4
	github.com/dennwc/btrfs v0.0.0-20260222081608-edfb8b9e4f55
	github.com/go-playground/validator/v10 v10.30.2
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/hetznercloud/hcloud-go/v2 v2.40.0
	github.com/jsimonetti/rtnetlink v1.4.2
	github.com/libp2p/go-libp2p v0.48.0
	github.com/martinlindhe/base36 v1.1.1
	github.com/miekg/dns v1.1.72
	github.com/mikesmitty/edkey v0.0.0-20170222072505-3356ea4e686a
	github.com/multiformats/go-multiaddr v0.16.1
	github.com/opencontainers/runtime-spec v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/protosio/protos/network/modules/ipc v0.0.0
	github.com/protosio/protos/network/modules/networkextension v0.0.0
	github.com/protosio/protos/network/modules/wireguard v0.0.0
	github.com/rakyll/statik v0.1.8
	github.com/rs/xid v1.6.0
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.36
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/sirupsen/logrus v1.9.4
	github.com/tmc/apple v0.6.3
	github.com/urfave/cli/v2 v2.27.7
	golang.org/x/crypto v0.51.0
	golang.org/x/sys v0.44.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
	google.golang.org/grpc v1.81.1
	google.golang.org/protobuf v1.36.12-0.20260120151049-f2248ac996af
	gopkg.in/yaml.v2 v2.4.0
)

require (
	cel.dev/expr v0.25.2 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/monitoring v1.29.0 // indirect
	cuelang.org/go v0.16.1 // indirect
	filippo.io/bigmod v0.1.1-0.20260103110540-f8a47775ebe5 // indirect
	filippo.io/keygen v0.0.0-20260114151900-8e2790ea4c5b // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.7.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.32.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.56.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.56.0 // indirect
	github.com/abiosoft/readline v0.0.0-20180607040430-155bce2042db // indirect
	github.com/andreyvit/diff v0.0.0-20170406064948-c7f18ee00883 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cncf/xds/go v0.0.0-20260202195803-dba9d589def2 // indirect
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/cockroachdb/errors v1.13.0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.8 // indirect
	github.com/containerd/cgroups/v3 v3.1.3 // indirect
	github.com/containerd/containerd/api v1.11.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/plugin v1.1.0 // indirect
	github.com/containernetworking/cni v1.3.0 // indirect
	github.com/containernetworking/plugins v1.9.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dolthub/eventsapi_schema v0.0.0-20260310172945-37a9265ade69 // indirect
	github.com/dolthub/ishell v0.0.0-20260414231531-5f031e3e9037 // indirect
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/ebitengine/purego v0.10.0 // indirect
	github.com/edsrzf/mmap-go v1.2.0 // indirect
	github.com/emicklei/proto v1.14.3 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.3 // indirect
	github.com/felixge/fgprof v0.9.5 // indirect
	github.com/filecoin-project/go-clock v0.1.0 // indirect
	github.com/flynn-archive/go-shlex v0.0.0-20150515145356-3f9db97f8568 // indirect
	github.com/getsentry/sentry-go v0.46.2 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/gocraft/dbr/v2 v2.7.7 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/google/go-github/v57 v57.0.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20260507013755-92041b743c96 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/grafana/thema v0.0.0-20240605110052-2016107581da // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/libp2p/go-libp2p-pubsub v0.16.0 // indirect
	github.com/libp2p/go-yamux/v5 v5.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/mpvl/unique v0.0.0-20150818121801-cbe035fff7de // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pion/datachannel v1.6.0 // indirect
	github.com/pion/dtls/v3 v3.1.2 // indirect
	github.com/pion/ice/v4 v4.2.5 // indirect
	github.com/pion/interceptor v0.1.45 // indirect
	github.com/pion/logging v0.2.4 // indirect
	github.com/pion/mdns/v2 v2.1.0 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.16 // indirect
	github.com/pion/rtp v1.10.2 // indirect
	github.com/pion/sctp v1.9.5 // indirect
	github.com/pion/sdp/v3 v3.0.18 // indirect
	github.com/pion/srtp/v3 v3.0.10 // indirect
	github.com/pion/stun/v3 v3.1.2 // indirect
	github.com/pion/transport/v4 v4.0.1 // indirect
	github.com/pion/turn/v5 v5.0.4 // indirect
	github.com/pion/webrtc/v4 v4.2.12 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/profile v1.7.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/protocolbuffers/txtpbfmt v0.0.0-20260217160748-a481f6a22f94 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/sony/gobreaker/v2 v2.4.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/tealeg/xlsx v1.0.5 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/vishvananda/netlink v1.3.1 // indirect
	github.com/wlynxg/anet v0.0.5 // indirect
	github.com/xtaci/smux v1.5.57 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk v1.43.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/telemetry v0.0.0-20260508192327-42602be52be6 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/knftables v0.0.21 // indirect
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.11.0 // indirect
	cloud.google.com/go/storage v1.62.1 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.2.0 // indirect
	github.com/Microsoft/go-winio v0.6.3-0.20251027160822-ad3df93bed29 // indirect
	github.com/Microsoft/hcsshim v0.15.0-rc.1 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible // indirect
	github.com/apache/arrow/go/arrow v0.0.0-20211112161151-bc219186db40 // indirect
	github.com/apache/thrift v0.23.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.7 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.17 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.22.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.101.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/bcicen/jstream v1.0.1 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/continuity v0.5.0 // indirect
	github.com/containerd/fifo v1.1.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/ttrpc v1.2.8 // indirect
	github.com/containerd/typeurl/v2 v2.2.3 // indirect
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/denisbrodbeck/machineid v1.0.1 // indirect
	github.com/dennwc/ioctl v1.0.1 // indirect
	github.com/dolthub/aws-sdk-go-ini-parser v0.0.0-20250305001723-2821c37f6c12 // indirect
	github.com/dolthub/dolt/go v0.40.5-0.20260507231811-577c666cc34c // indirect
	github.com/dolthub/driver v0.0.0-00010101000000-000000000000 // indirect
	github.com/dolthub/flatbuffers/v23 v23.3.3-dh.2 // indirect
	github.com/dolthub/fslock v0.0.3 // indirect
	github.com/dolthub/go-icu-regex v0.0.0-20260412212219-49724d547866 // indirect
	github.com/dolthub/go-mysql-server v0.20.1-0.20260507202550-43d6daf5958b // indirect
	github.com/dolthub/gozstd v0.0.0-20240423170813-23a2903bca63 // indirect
	github.com/dolthub/jsonpath v0.0.2-0.20240227200619-19675ab05c71 // indirect
	github.com/dolthub/vitess v0.0.0-20260505163811-77e5224be390
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/esote/minmaxheap v1.0.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/flynn/noise v1.1.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.13 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-sql-driver/mysql v1.10.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.15 // indirect
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/ipfs/go-cid v0.6.1 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/juju/gnuflag v1.0.0 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kch42/buzhash v0.0.0-20160816060738-9bdec3dec7c6 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/koron/go-ssdp v0.9.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/strftime v1.1.1 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.3.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.4.1 // indirect
	github.com/libp2p/go-msgio v0.3.0 // indirect
	github.com/libp2p/go-netroute v0.4.0 // indirect
	github.com/libp2p/go-reuseport v0.4.0 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/mdlayher/genetlink v1.4.0 // indirect
	github.com/mdlayher/netlink v1.11.2 // indirect
	github.com/mdlayher/socket v0.6.0 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/sha256-simd v1.0.1 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/signal v0.7.1 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/mohae/uvarint v0.0.0-20160208145430-c3f9e62bf2b0 // indirect
	github.com/mr-tron/base58 v1.3.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.2.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.5.0 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multicodec v0.10.0 // indirect
	github.com/multiformats/go-multihash v0.2.3 // indirect
	github.com/multiformats/go-multistream v0.6.1 // indirect
	github.com/multiformats/go-varint v0.1.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/oracle/oci-go-sdk/v65 v65.114.2 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.1 // indirect
	github.com/quic-go/webtransport-go v0.10.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/safchain/ethtool v0.7.0 // indirect
	github.com/sergi/go-diff v1.4.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/tidwall/gjson v1.19.0 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/vbauerster/mpb/v8 v8.12.0 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/xitongsys/parquet-go v1.6.2 // indirect
	github.com/xitongsys/parquet-go-source v0.0.0-20241021075129-b732d2ac9c9b // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.68.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/fx v1.24.0 // indirect
	go.uber.org/mock v0.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/exp v0.0.0-20260508232706-74f9aab9d74a // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/term v0.43.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb // indirect
	google.golang.org/api v0.279.0 // indirect
	google.golang.org/genproto v0.0.0-20260511170946-3700d4141b60 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260511170946-3700d4141b60 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/src-d/go-errors.v1 v1.0.0 // indirect
	lukechampine.com/blake3 v1.4.1 // indirect
	swarmion.dev/protocol v0.0.0
	swarmion.dev/runtime v0.0.0
	swarmion.dev/schema-engines/cue v0.0.0
	swarmion.dev/schema-engines/declarative v0.0.0
	swarmion.dev/transports v0.0.0
)

replace github.com/grafana/thema => github.com/nustiueudinastea/thema v0.0.0-20240605110052-2016107581da

replace cuelang.org/go => github.com/grafana/cue v0.0.0-20230926092038-971951014e3f

replace github.com/protocolbuffers/txtpbfmt => github.com/protocolbuffers/txtpbfmt v0.0.0-20220428173112-74888fd59c2b

replace swarmion.dev/runtime => ../swarmion/runtime

replace swarmion.dev/protocol => ../swarmion/protocol

replace swarmion.dev/transports => ../swarmion/transports

replace swarmion.dev/schema-engines/cue => ../swarmion/schema-engines/cue

replace swarmion.dev/schema-engines/declarative => ../swarmion/schema-engines/declarative
