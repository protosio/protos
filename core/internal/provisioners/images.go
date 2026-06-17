package provisioners

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	ProtosCloudImagePrefix    = "protos-image-"
	ProtosCloudSnapshotPrefix = "protos-snapshot-"
	ProtosCloudNameDateLayout = "20060102150405"
)

var protosCloudNameDateSuffixRE = regexp.MustCompile(`-[0-9]{14}$`)

type ProtosCloudImageNames struct {
	LogicalName  string
	ImageName    string
	SnapshotName string
	DateSuffix   string
}

func NewProtosCloudImageNames(logicalName string, now time.Time) (ProtosCloudImageNames, error) {
	logicalName, err := NormalizeProtosCloudImageLogicalName(logicalName)
	if err != nil {
		return ProtosCloudImageNames{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}
	dateSuffix := now.UTC().Format(ProtosCloudNameDateLayout)
	return ProtosCloudImageNames{
		LogicalName:  logicalName,
		ImageName:    ProtosCloudImagePrefix + logicalName + "-" + dateSuffix,
		SnapshotName: ProtosCloudSnapshotPrefix + logicalName + "-" + dateSuffix,
		DateSuffix:   dateSuffix,
	}, nil
}

func NormalizeProtosCloudImageLogicalName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if logicalName, _, ok := ParseProtosCloudObjectName(name); ok {
		name = logicalName
	}
	name = strings.TrimPrefix(name, "protos-")
	var b strings.Builder
	previousDash := false
	for _, r := range name {
		allowed := r == '.' || r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r)
		if !allowed {
			r = '-'
		}
		if r == '-' {
			if previousDash {
				continue
			}
			previousDash = true
		} else {
			previousDash = false
		}
		r = unicode.ToLower(r)
		b.WriteRune(r)
	}
	normalized := strings.Trim(b.String(), "-._")
	if normalized == "" {
		return "", fmt.Errorf("image name is empty after normalization")
	}
	return normalized, nil
}

func ParseProtosCloudObjectName(name string) (logicalName string, dateSuffix string, ok bool) {
	name = strings.TrimSpace(name)
	var withoutPrefix string
	switch {
	case strings.HasPrefix(name, ProtosCloudImagePrefix):
		withoutPrefix = strings.TrimPrefix(name, ProtosCloudImagePrefix)
	case strings.HasPrefix(name, ProtosCloudSnapshotPrefix):
		withoutPrefix = strings.TrimPrefix(name, ProtosCloudSnapshotPrefix)
	default:
		return "", "", false
	}
	if !protosCloudNameDateSuffixRE.MatchString(withoutPrefix) {
		return "", "", false
	}
	dateSuffix = withoutPrefix[len(withoutPrefix)-len(ProtosCloudNameDateLayout):]
	logicalName = strings.TrimSuffix(withoutPrefix, "-"+dateSuffix)
	if logicalName == "" {
		return "", "", false
	}
	return logicalName, dateSuffix, true
}

func ProtosCloudImageInfo(id string, cloudName string, location string, logicalFallback string) ImageInfo {
	cloudName = strings.TrimSpace(cloudName)
	logicalFallback = strings.TrimSpace(logicalFallback)
	info := ImageInfo{
		ID:       strings.TrimSpace(id),
		Name:     cloudName,
		Location: strings.TrimSpace(location),
	}
	if info.Name == "" {
		info.Name = logicalFallback
	}
	if logicalName, dateSuffix, ok := ParseProtosCloudObjectName(cloudName); ok {
		info.LogicalName = logicalName
		info.DateSuffix = dateSuffix
		info.Canonical = true
		if parsed, err := time.ParseInLocation(ProtosCloudNameDateLayout, dateSuffix, time.UTC); err == nil {
			info.UpdatedAt = parsed
		}
		return info
	}
	if logicalFallback != "" {
		info.LogicalName = logicalFallback
	} else if strings.HasPrefix(cloudName, "protos-") {
		info.LogicalName = strings.TrimPrefix(cloudName, "protos-")
	}
	return info
}

func ProtosImageMatchesRef(image ImageInfo, ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	for _, candidate := range []string{image.ID, image.Name, image.LogicalName} {
		if strings.TrimSpace(candidate) == ref {
			return true
		}
	}
	normalizedRef, err := NormalizeProtosCloudImageLogicalName(ref)
	if err != nil {
		return false
	}
	for _, candidate := range []string{image.Name, image.LogicalName} {
		normalizedCandidate, err := NormalizeProtosCloudImageLogicalName(candidate)
		if err == nil && normalizedCandidate == normalizedRef {
			return true
		}
	}
	return false
}

func SelectProtosImageForRef(images map[string]ImageInfo, location string, ref string) (string, ImageInfo, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ImageInfo{}, false
	}
	location = strings.TrimSpace(location)

	exactIDMatches := make([]ImageInfo, 0, 1)
	exactNameMatches := make([]ImageInfo, 0, 1)
	logicalMatches := make([]ImageInfo, 0, len(images))
	for mapID, image := range images {
		if image.ID == "" {
			image.ID = strings.TrimSpace(mapID)
		}
		if location != "" && strings.TrimSpace(image.Location) != location {
			continue
		}
		if strings.TrimSpace(image.ID) == ref {
			exactIDMatches = append(exactIDMatches, image)
			continue
		}
		if strings.TrimSpace(image.Name) == ref {
			exactNameMatches = append(exactNameMatches, image)
			continue
		}
		if ProtosImageMatchesRef(image, ref) {
			logicalMatches = append(logicalMatches, image)
		}
	}
	for _, matches := range [][]ImageInfo{exactIDMatches, exactNameMatches, logicalMatches} {
		if len(matches) == 0 {
			continue
		}
		sort.Slice(matches, func(i, j int) bool {
			return protosImageSelectionLess(matches[j], matches[i])
		})
		selected := matches[0]
		return selected.ID, selected, true
	}
	return "", ImageInfo{}, false
}

func protosImageSelectionLess(a ImageInfo, b ImageInfo) bool {
	if !a.UpdatedAt.Equal(b.UpdatedAt) {
		if a.UpdatedAt.IsZero() {
			return true
		}
		if b.UpdatedAt.IsZero() {
			return false
		}
		return a.UpdatedAt.Before(b.UpdatedAt)
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.ID < b.ID
}
