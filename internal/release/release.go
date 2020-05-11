package release

import (
	"sort"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
)

type CloudImage struct {
	Provider    string
	URL         string
	Digest      string
	ReleaseDate time.Time `json:"release-date"`
}

type Release struct {
	CloudImages map[string]CloudImage `json:"cloud-images"`
	Version     string
	Description string
	ReleaseDate time.Time `json:"release-date"`
}

type Releases struct {
	Releases map[string]Release
}

//
// Releases methods
//

// GetLatest returns the latest version from the Releases collection
func (rls Releases) GetLatest() (Release, error) {
	var vs []*semver.Version
	for version := range rls.Releases {
		v, err := semver.NewVersion(version)
		if err != nil {
			return Release{}, errors.Wrap(err, "Error parsing version")
		}
		vs = append(vs, v)
	}

	vc := semver.Collection(vs)
	sort.Sort(vc)
	if len(vs) == 0 {
		return Release{}, errors.New("Could not get latest Protos release. 0 releases found")
	}
	latestVersion := vc[len(vc)-1].String()
	return rls.Releases[latestVersion], nil
}

// GetVersion takes a version as string and returns a Release struct
func (rls Releases) GetVersion(version string) (Release, error) {
	_, err := semver.NewVersion(version)
	if err != nil {
		return Release{}, errors.Wrapf(err, "Cant parse version '%s'", version)
	}
	versionConstraint, err := semver.NewConstraint("= " + version)
	if err != nil {
		return Release{}, errors.Wrap(err, "Error parsing version")
	}

	for lversion, release := range rls.Releases {
		v, err := semver.NewVersion(lversion)
		if err != nil {
			return Release{}, errors.Wrap(err, "Error parsing version from releases list")
		}
		if versionConstraint.Check(v) {
			return release, nil
		}
	}
	return Release{}, errors.Errorf("Failed to find a release with version '%s'", version)
}
