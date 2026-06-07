package imageregistry

const (
	SourceLabel    = "protos.io/image.source"
	SourceP2P      = "p2p"
	SourceLocalTar = "local-tar"
)

type Descriptor struct {
	MediaType   string
	Digest      string
	SizeBytes   uint64
	Platform    string
	Annotations map[string]string
}

type ImageContent struct {
	ImageRef    string
	Target      Descriptor
	Platform    string
	Labels      map[string]string
	Descriptors []Descriptor
}

type LoadedImage struct {
	ImageRef     string
	TargetDigest string
	Platform     string
}
