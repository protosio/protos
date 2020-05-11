package cloud

type digitalocean struct {
}

func newDigitalOceanClient() (*digitalocean, error) {
	return &digitalocean{}, nil

}

func (do *digitalocean) NewInstance(image string) (string, error) {
	return "", nil
}

func (do *digitalocean) DeleteInstance(id string) error {
	return nil
}

func (do *digitalocean) StartInstance(id string) error {
	return nil
}

func (do *digitalocean) StopInstance(id string) error {
	return nil
}

func (do *digitalocean) AddImage(url string, hash string) error {
	return nil
}

func (do *digitalocean) RemoveImage(id string) error {
	return nil
}

func (do *digitalocean) AuthFields() []string {
	return []string{}
}

func (do *digitalocean) Init(credentials map[string]string) error {
	return nil
}
