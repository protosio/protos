package apic

import (
	"context"
	"fmt"
	"sort"
	"strings"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/provisioners"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	instanceDeployFieldText   = "text"
	instanceDeployFieldSelect = "select"
	instanceDeployFieldHidden = "hidden"
)

type instanceDeployFieldMeta struct {
	label    string
	kind     string
	required bool
	helper   string
}

var instanceDeployFieldMetadata = map[string]instanceDeployFieldMeta{
	"name": {
		label:    "Name",
		kind:     instanceDeployFieldText,
		required: true,
	},
	"cloud_name": {
		label:    "Provisioner",
		kind:     instanceDeployFieldSelect,
		required: true,
	},
	"cloud_location": {
		label:    "Location",
		kind:     instanceDeployFieldSelect,
		required: true,
	},
	"machine_type": {
		label:    "Size",
		kind:     instanceDeployFieldSelect,
		required: true,
	},
	"protos_version": {
		label:    "Protos version",
		kind:     instanceDeployFieldSelect,
		required: true,
	},
	"dev_img": {
		label:  "Existing image",
		kind:   instanceDeployFieldHidden,
		helper: "Advanced image override.",
	},
}

func (b *Backend) GetInstanceDeployOptions(ctx context.Context, in *pbApic.GetInstanceDeployOptionsRequest) (*pbApic.GetInstanceDeployOptionsResponse, error) {
	if err := b.requireProvisionerCapability("inspect instance deployment options"); err != nil {
		return nil, err
	}
	if b.protosClient == nil || b.protosClient.ProvisionerManager == nil {
		return nil, fmt.Errorf("provisioner manager is not configured")
	}

	provisionerOptions, err := b.instanceDeployProvisionerOptions()
	if err != nil {
		return nil, err
	}

	selectedProvisioner := strings.TrimSpace(in.GetProvisioner())
	selectedLocation := strings.TrimSpace(in.GetLocation())
	var locationOptions []*pbApic.InstanceDeployFieldOption
	var machineOptions []*pbApic.InstanceDeployFieldOption
	var releaseOptions []*pbApic.InstanceDeployFieldOption
	var releaseHelper string
	var imageOptions []*pbApic.InstanceDeployFieldOption
	var imageHelper string
	var provisionerType string

	if selectedProvisioner != "" {
		provisioner, err := b.protosClient.ProvisionerManager.GetProvisionerOrDefault(selectedProvisioner)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve provisioner '%s': %w", selectedProvisioner, err)
		}
		computeProvisioner, ok := provisioner.(provisioners.ComputeProvisioner)
		if !ok {
			return nil, fmt.Errorf("provisioner '%s'(%s) does not support compute operations", provisioner.NameStr(), provisioner.TypeStr())
		}
		if _, ok := provisioner.(provisioners.ImageProvisioner); !ok {
			return nil, fmt.Errorf("provisioner '%s'(%s) does not support image operations", provisioner.NameStr(), provisioner.TypeStr())
		}
		if _, ok := provisioner.(provisioners.VolumeProvisioner); !ok {
			return nil, fmt.Errorf("provisioner '%s'(%s) does not support volume operations", provisioner.NameStr(), provisioner.TypeStr())
		}
		if err := provisioner.Init(); err != nil {
			return nil, fmt.Errorf("error reaching provisioner '%s'(%s) API: %w", provisioner.NameStr(), provisioner.TypeStr(), err)
		}

		provisionerType = provisioner.TypeStr()
		locations := computeProvisioner.SupportedLocations()
		sort.Strings(locations)
		for _, location := range locations {
			locationOptions = append(locationOptions, &pbApic.InstanceDeployFieldOption{
				Value: location,
				Label: location,
			})
		}
		if selectedLocation == "" && len(locations) > 0 {
			selectedLocation = locations[0]
		}
		if selectedLocation != "" && !stringInSlice(selectedLocation, locations) {
			return nil, fmt.Errorf("location '%s' is not available for provisioner '%s'", selectedLocation, selectedProvisioner)
		}
		if selectedLocation != "" {
			machines, err := computeProvisioner.SupportedMachines(selectedLocation)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve supported machines for '%s': %w", selectedProvisioner, err)
			}
			machineOptions = instanceDeployMachineOptions(machines)
		}
		releaseOptions, releaseHelper = b.instanceDeployReleaseOptions(provisionerType)
		imageOptions, imageHelper = instanceDeployImageOptions(provisioner, selectedLocation)
	}

	fields := baseInstanceDeployFields()
	for _, field := range fields {
		switch field.GetName() {
		case "name":
			field.Visible = selectedProvisioner != ""
		case "cloud_name":
			field.Visible = true
			field.Value = selectedProvisioner
			field.Options = provisionerOptions
			if len(provisionerOptions) == 0 {
				field.Helper = "No deployable provisioners are configured."
			}
		case "cloud_location":
			field.Visible = selectedProvisioner != ""
			field.Value = selectedLocation
			field.Options = locationOptions
		case "machine_type":
			field.Visible = selectedProvisioner != "" && selectedLocation != ""
			field.Options = machineOptions
			if len(machineOptions) > 0 {
				field.Value = preferredInstanceDeployMachine(provisionerType, machineOptions)
			}
		case "protos_version":
			field.Visible = selectedProvisioner != "" && len(releaseOptions) > 0
			field.Required = field.Visible
			field.Options = releaseOptions
			if len(releaseOptions) > 0 {
				field.Value = releaseOptions[0].GetValue()
			}
			field.Helper = releaseHelper
		case "dev_img":
			field.Kind = instanceDeployFieldSelect
			field.Visible = selectedProvisioner != "" && len(releaseOptions) == 0
			field.Required = field.Visible
			field.Options = imageOptions
			if len(imageOptions) > 0 {
				field.Value = imageOptions[0].GetValue()
			}
			if field.Visible {
				field.Helper = imageHelper
				if field.Helper == "" {
					field.Helper = "Using an existing image because release options are unavailable."
				}
			}
		}
	}

	return &pbApic.GetInstanceDeployOptionsResponse{Fields: fields}, nil
}

func baseInstanceDeployFields() []*pbApic.InstanceDeployField {
	descriptor := (&pbApic.DeployInstanceRequest{}).ProtoReflect().Descriptor()
	fields := make([]*pbApic.InstanceDeployField, 0, descriptor.Fields().Len())
	for i := 0; i < descriptor.Fields().Len(); i++ {
		field := descriptor.Fields().Get(i)
		name := string(field.Name())
		meta, found := instanceDeployFieldMetadata[name]
		if !found {
			meta = instanceDeployFieldMeta{
				label:    humanizeProtoFieldName(field.Name()),
				kind:     instanceDeployFieldText,
				required: false,
			}
		}
		fields = append(fields, &pbApic.InstanceDeployField{
			Name:     name,
			Label:    meta.label,
			Kind:     meta.kind,
			Required: meta.required,
			Helper:   meta.helper,
			Visible:  meta.kind != instanceDeployFieldHidden,
		})
	}
	return fields
}

func (b *Backend) instanceDeployProvisionerOptions() ([]*pbApic.InstanceDeployFieldOption, error) {
	provisionersList, err := b.protosClient.ProvisionerManager.GetProvisioners()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioners: %w", err)
	}

	options := make([]*pbApic.InstanceDeployFieldOption, 0, len(provisionersList))
	seen := map[string]struct{}{}
	for _, provisioner := range provisionersList {
		if !supportsInstanceDeploy(provisioner) {
			continue
		}
		seen[provisioner.NameStr()] = struct{}{}
		options = append(options, &pbApic.InstanceDeployFieldOption{
			Value:       provisioner.NameStr(),
			Label:       provisionerDisplayName(provisioner.NameStr()),
			Description: provisioner.TypeStr(),
		})
	}
	for _, provisionerType := range b.protosClient.ProvisionerManager.SupportedProvisioners() {
		if _, found := seen[provisionerType]; found {
			continue
		}
		authFields, err := b.protosClient.ProvisionerManager.ProvisionerAuthFields(provisionerType)
		if err != nil {
			return nil, err
		}
		if len(authFields) > 0 {
			continue
		}
		provisioner, err := b.protosClient.ProvisionerManager.GetProvisionerOrDefault(provisionerType)
		if err != nil || !supportsInstanceDeploy(provisioner) {
			continue
		}
		options = append(options, &pbApic.InstanceDeployFieldOption{
			Value:       provisionerType,
			Label:       provisionerDisplayName(provisionerType),
			Description: provisioner.TypeStr(),
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return strings.ToLower(options[i].GetLabel()) < strings.ToLower(options[j].GetLabel())
	})
	return options, nil
}

func (b *Backend) instanceDeployReleaseOptions(provisionerType string) ([]*pbApic.InstanceDeployFieldOption, string) {
	releases, err := b.protosClient.GetProtosAvailableReleases()
	if err != nil {
		return nil, fmt.Sprintf("Could not load release options: %s", err.Error())
	}

	versions := make([]string, 0, len(releases.Releases))
	for version, release := range releases.Releases {
		if provisionerType != "" {
			if _, found := release.CloudImages[provisionerType]; !found {
				continue
			}
		}
		versions = append(versions, version)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	options := make([]*pbApic.InstanceDeployFieldOption, 0, len(versions))
	for _, version := range versions {
		release := releases.Releases[version]
		options = append(options, &pbApic.InstanceDeployFieldOption{
			Value:       version,
			Label:       version,
			Description: release.Description,
		})
	}
	if len(options) == 0 {
		return nil, "No compatible Protos releases are available for this provisioner."
	}
	return options, ""
}

func instanceDeployMachineOptions(machines map[string]provisioners.MachineSpec) []*pbApic.InstanceDeployFieldOption {
	names := make([]string, 0, len(machines))
	for name := range machines {
		names = append(names, name)
	}
	sort.Strings(names)

	options := make([]*pbApic.InstanceDeployFieldOption, 0, len(names))
	for _, name := range names {
		options = append(options, &pbApic.InstanceDeployFieldOption{
			Value:       name,
			Label:       name,
			Description: machineSpecDescription(machines[name]),
		})
	}
	return options
}

func preferredInstanceDeployMachine(provisionerType string, options []*pbApic.InstanceDeployFieldOption) string {
	if len(options) == 0 {
		return ""
	}
	if provisionerType == "local_macos" {
		for _, option := range options {
			if option.GetValue() == "vz-2c-2g" {
				return option.GetValue()
			}
		}
	}
	return options[0].GetValue()
}

func instanceDeployImageOptions(provisioner provisioners.Provisioner, location string) ([]*pbApic.InstanceDeployFieldOption, string) {
	imageProvisioner, ok := provisioner.(provisioners.ImageProvisioner)
	if !ok {
		return nil, "This provisioner does not support image lookup."
	}
	images, err := imageProvisioner.GetProtosImages()
	if err != nil {
		return nil, fmt.Sprintf("Could not load existing image options: %s", err.Error())
	}

	imageList := make([]provisioners.ImageInfo, 0, len(images))
	for _, image := range images {
		if location != "" && image.Location != "" && image.Location != location {
			continue
		}
		imageList = append(imageList, image)
	}
	sort.Slice(imageList, func(i, j int) bool {
		return instanceDeployImageLess(imageList[i], imageList[j])
	})

	options := make([]*pbApic.InstanceDeployFieldOption, 0, len(imageList))
	for _, image := range imageList {
		options = append(options, &pbApic.InstanceDeployFieldOption{
			Value:       image.Name,
			Label:       image.Name,
			Description: instanceDeployImageDescription(image),
		})
	}
	if len(options) == 0 {
		return nil, "No existing Protos images are available for this provisioner."
	}
	return options, ""
}

func instanceDeployImageLess(left provisioners.ImageInfo, right provisioners.ImageInfo) bool {
	if !left.UpdatedAt.IsZero() || !right.UpdatedAt.IsZero() {
		if left.UpdatedAt.IsZero() {
			return false
		}
		if right.UpdatedAt.IsZero() {
			return true
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
	}
	return strings.ToLower(left.Name) < strings.ToLower(right.Name)
}

func instanceDeployImageDescription(image provisioners.ImageInfo) string {
	parts := make([]string, 0, 2)
	if image.Location != "" {
		parts = append(parts, image.Location)
	}
	if !image.UpdatedAt.IsZero() {
		parts = append(parts, "updated "+image.UpdatedAt.Format("2006-01-02 15:04"))
	}
	return strings.Join(parts, " / ")
}

func machineSpecDescription(spec provisioners.MachineSpec) string {
	parts := []string{
		fmt.Sprintf("%d CPU", spec.Cores),
		fmt.Sprintf("%.1f GiB RAM", float64(spec.Memory)/1024),
		fmt.Sprintf("%d GB disk", spec.DefaultStorage),
	}
	if spec.PriceMonthly > 0 {
		parts = append(parts, fmt.Sprintf("%.2f/mo", spec.PriceMonthly))
	}
	if spec.Baremetal {
		parts = append(parts, "bare metal")
	}
	return strings.Join(parts, " / ")
}

func supportsInstanceDeploy(provisioner provisioners.Provisioner) bool {
	if provisioner == nil {
		return false
	}
	if _, ok := provisioner.(provisioners.ComputeProvisioner); !ok {
		return false
	}
	if _, ok := provisioner.(provisioners.ImageProvisioner); !ok {
		return false
	}
	if _, ok := provisioner.(provisioners.VolumeProvisioner); !ok {
		return false
	}
	return true
}

func provisionerDisplayName(value string) string {
	switch value {
	case "local_macos":
		return "Local macOS"
	default:
		return value
	}
}

func humanizeProtoFieldName(name protoreflect.Name) string {
	parts := strings.Split(string(name), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
