// This is a generated file - do not edit.
//
// Generated from apic/proto/apic.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use initRequestDescriptor instead')
const InitRequest$json = {
  '1': 'InitRequest',
  '2': [
    {'1': 'username', '3': 1, '4': 1, '5': 9, '10': 'username'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'organisation', '3': 3, '4': 1, '5': 9, '10': 'organisation'},
  ],
};

/// Descriptor for `InitRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List initRequestDescriptor = $convert.base64Decode(
    'CgtJbml0UmVxdWVzdBIaCgh1c2VybmFtZRgBIAEoCVIIdXNlcm5hbWUSEgoEbmFtZRgCIAEoCV'
    'IEbmFtZRIiCgxvcmdhbmlzYXRpb24YAyABKAlSDG9yZ2FuaXNhdGlvbg==');

@$core.Deprecated('Use initResponseDescriptor instead')
const InitResponse$json = {
  '1': 'InitResponse',
};

/// Descriptor for `InitResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List initResponseDescriptor =
    $convert.base64Decode('CgxJbml0UmVzcG9uc2U=');

@$core.Deprecated('Use userDeviceDescriptor instead')
const UserDevice$json = {
  '1': 'UserDevice',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'public_key', '3': 3, '4': 1, '5': 9, '10': 'publicKey'},
    {
      '1': 'public_key_wireguard',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'publicKeyWireguard'
    },
  ],
};

/// Descriptor for `UserDevice`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List userDeviceDescriptor = $convert.base64Decode(
    'CgpVc2VyRGV2aWNlEg4KAmlkGAEgASgJUgJpZBISCgRuYW1lGAIgASgJUgRuYW1lEh0KCnB1Ym'
    'xpY19rZXkYAyABKAlSCXB1YmxpY0tleRIwChRwdWJsaWNfa2V5X3dpcmVndWFyZBgEIAEoCVIS'
    'cHVibGljS2V5V2lyZWd1YXJk');

@$core.Deprecated('Use getUserDevicesRequestDescriptor instead')
const GetUserDevicesRequest$json = {
  '1': 'GetUserDevicesRequest',
};

/// Descriptor for `GetUserDevicesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserDevicesRequestDescriptor =
    $convert.base64Decode('ChVHZXRVc2VyRGV2aWNlc1JlcXVlc3Q=');

@$core.Deprecated('Use getUserDevicesResponseDescriptor instead')
const GetUserDevicesResponse$json = {
  '1': 'GetUserDevicesResponse',
  '2': [
    {
      '1': 'devices',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.UserDevice',
      '10': 'devices'
    },
  ],
};

/// Descriptor for `GetUserDevicesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserDevicesResponseDescriptor =
    $convert.base64Decode(
        'ChZHZXRVc2VyRGV2aWNlc1Jlc3BvbnNlEioKB2RldmljZXMYASADKAsyEC5hcGljLlVzZXJEZX'
        'ZpY2VSB2RldmljZXM=');

@$core.Deprecated('Use getUserInfoRequestDescriptor instead')
const GetUserInfoRequest$json = {
  '1': 'GetUserInfoRequest',
};

/// Descriptor for `GetUserInfoRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserInfoRequestDescriptor =
    $convert.base64Decode('ChJHZXRVc2VySW5mb1JlcXVlc3Q=');

@$core.Deprecated('Use getUserInfoResponseDescriptor instead')
const GetUserInfoResponse$json = {
  '1': 'GetUserInfoResponse',
  '2': [
    {'1': 'username', '3': 1, '4': 1, '5': 9, '10': 'username'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'is_admin', '3': 3, '4': 1, '5': 8, '10': 'isAdmin'},
    {'1': 'organisation_id', '3': 4, '4': 1, '5': 9, '10': 'organisationId'},
    {
      '1': 'organisation_name',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'organisationName'
    },
  ],
};

/// Descriptor for `GetUserInfoResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getUserInfoResponseDescriptor = $convert.base64Decode(
    'ChNHZXRVc2VySW5mb1Jlc3BvbnNlEhoKCHVzZXJuYW1lGAEgASgJUgh1c2VybmFtZRISCgRuYW'
    '1lGAIgASgJUgRuYW1lEhkKCGlzX2FkbWluGAMgASgIUgdpc0FkbWluEicKD29yZ2FuaXNhdGlv'
    'bl9pZBgEIAEoCVIOb3JnYW5pc2F0aW9uSWQSKwoRb3JnYW5pc2F0aW9uX25hbWUYBSABKAlSEG'
    '9yZ2FuaXNhdGlvbk5hbWU=');

@$core.Deprecated('Use organisationDescriptor instead')
const Organisation$json = {
  '1': 'Organisation',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'created_at', '3': 3, '4': 1, '5': 9, '10': 'createdAt'},
  ],
};

/// Descriptor for `Organisation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List organisationDescriptor = $convert.base64Decode(
    'CgxPcmdhbmlzYXRpb24SDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbWUSHQoKY3'
    'JlYXRlZF9hdBgDIAEoCVIJY3JlYXRlZEF0');

@$core.Deprecated('Use listOrganisationsRequestDescriptor instead')
const ListOrganisationsRequest$json = {
  '1': 'ListOrganisationsRequest',
};

/// Descriptor for `ListOrganisationsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOrganisationsRequestDescriptor =
    $convert.base64Decode('ChhMaXN0T3JnYW5pc2F0aW9uc1JlcXVlc3Q=');

@$core.Deprecated('Use listOrganisationsResponseDescriptor instead')
const ListOrganisationsResponse$json = {
  '1': 'ListOrganisationsResponse',
  '2': [
    {
      '1': 'organisations',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Organisation',
      '10': 'organisations'
    },
  ],
};

/// Descriptor for `ListOrganisationsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listOrganisationsResponseDescriptor =
    $convert.base64Decode(
        'ChlMaXN0T3JnYW5pc2F0aW9uc1Jlc3BvbnNlEjgKDW9yZ2FuaXNhdGlvbnMYASADKAsyEi5hcG'
        'ljLk9yZ2FuaXNhdGlvblINb3JnYW5pc2F0aW9ucw==');

@$core.Deprecated('Use startDeviceInviteRequestDescriptor instead')
const StartDeviceInviteRequest$json = {
  '1': 'StartDeviceInviteRequest',
  '2': [
    {'1': 'organisation_id', '3': 1, '4': 1, '5': 9, '10': 'organisationId'},
    {'1': 'channel', '3': 2, '4': 1, '5': 9, '10': 'channel'},
    {'1': 'join_mode', '3': 3, '4': 1, '5': 9, '10': 'joinMode'},
    {'1': 'username', '3': 4, '4': 1, '5': 9, '10': 'username'},
  ],
};

/// Descriptor for `StartDeviceInviteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startDeviceInviteRequestDescriptor = $convert.base64Decode(
    'ChhTdGFydERldmljZUludml0ZVJlcXVlc3QSJwoPb3JnYW5pc2F0aW9uX2lkGAEgASgJUg5vcm'
    'dhbmlzYXRpb25JZBIYCgdjaGFubmVsGAIgASgJUgdjaGFubmVsEhsKCWpvaW5fbW9kZRgDIAEo'
    'CVIIam9pbk1vZGUSGgoIdXNlcm5hbWUYBCABKAlSCHVzZXJuYW1l');

@$core.Deprecated('Use startDeviceInviteResponseDescriptor instead')
const StartDeviceInviteResponse$json = {
  '1': 'StartDeviceInviteResponse',
  '2': [
    {'1': 'invite_id', '3': 1, '4': 1, '5': 9, '10': 'inviteId'},
    {'1': 'expires_at_unix', '3': 2, '4': 1, '5': 3, '10': 'expiresAtUnix'},
    {'1': 'advertise_name', '3': 3, '4': 1, '5': 9, '10': 'advertiseName'},
    {
      '1': 'advertise_service',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'advertiseService'
    },
    {'1': 'channel', '3': 5, '4': 1, '5': 9, '10': 'channel'},
    {
      '1': 'verification_code',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'verificationCode'
    },
    {'1': 'join_mode', '3': 7, '4': 1, '5': 9, '10': 'joinMode'},
  ],
};

/// Descriptor for `StartDeviceInviteResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startDeviceInviteResponseDescriptor = $convert.base64Decode(
    'ChlTdGFydERldmljZUludml0ZVJlc3BvbnNlEhsKCWludml0ZV9pZBgBIAEoCVIIaW52aXRlSW'
    'QSJgoPZXhwaXJlc19hdF91bml4GAIgASgDUg1leHBpcmVzQXRVbml4EiUKDmFkdmVydGlzZV9u'
    'YW1lGAMgASgJUg1hZHZlcnRpc2VOYW1lEisKEWFkdmVydGlzZV9zZXJ2aWNlGAQgASgJUhBhZH'
    'ZlcnRpc2VTZXJ2aWNlEhgKB2NoYW5uZWwYBSABKAlSB2NoYW5uZWwSKwoRdmVyaWZpY2F0aW9u'
    'X2NvZGUYBiABKAlSEHZlcmlmaWNhdGlvbkNvZGUSGwoJam9pbl9tb2RlGAcgASgJUghqb2luTW'
    '9kZQ==');

@$core.Deprecated('Use nearbyOrganisationDescriptor instead')
const NearbyOrganisation$json = {
  '1': 'NearbyOrganisation',
  '2': [
    {'1': 'organisation_id', '3': 1, '4': 1, '5': 9, '10': 'organisationId'},
    {
      '1': 'organisation_name',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'organisationName'
    },
    {'1': 'device_name', '3': 3, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'peer_id', '3': 4, '4': 1, '5': 9, '10': 'peerId'},
    {'1': 'invite_id', '3': 5, '4': 1, '5': 9, '10': 'inviteId'},
    {'1': 'channel', '3': 6, '4': 1, '5': 9, '10': 'channel'},
    {'1': 'join_mode', '3': 7, '4': 1, '5': 9, '10': 'joinMode'},
  ],
};

/// Descriptor for `NearbyOrganisation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nearbyOrganisationDescriptor = $convert.base64Decode(
    'ChJOZWFyYnlPcmdhbmlzYXRpb24SJwoPb3JnYW5pc2F0aW9uX2lkGAEgASgJUg5vcmdhbmlzYX'
    'Rpb25JZBIrChFvcmdhbmlzYXRpb25fbmFtZRgCIAEoCVIQb3JnYW5pc2F0aW9uTmFtZRIfCgtk'
    'ZXZpY2VfbmFtZRgDIAEoCVIKZGV2aWNlTmFtZRIXCgdwZWVyX2lkGAQgASgJUgZwZWVySWQSGw'
    'oJaW52aXRlX2lkGAUgASgJUghpbnZpdGVJZBIYCgdjaGFubmVsGAYgASgJUgdjaGFubmVsEhsK'
    'CWpvaW5fbW9kZRgHIAEoCVIIam9pbk1vZGU=');

@$core.Deprecated('Use listNearbyOrganisationsRequestDescriptor instead')
const ListNearbyOrganisationsRequest$json = {
  '1': 'ListNearbyOrganisationsRequest',
  '2': [
    {'1': 'channel', '3': 1, '4': 1, '5': 9, '10': 'channel'},
  ],
};

/// Descriptor for `ListNearbyOrganisationsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listNearbyOrganisationsRequestDescriptor =
    $convert.base64Decode(
        'Ch5MaXN0TmVhcmJ5T3JnYW5pc2F0aW9uc1JlcXVlc3QSGAoHY2hhbm5lbBgBIAEoCVIHY2hhbm'
        '5lbA==');

@$core.Deprecated('Use listNearbyOrganisationsResponseDescriptor instead')
const ListNearbyOrganisationsResponse$json = {
  '1': 'ListNearbyOrganisationsResponse',
  '2': [
    {
      '1': 'organisations',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.NearbyOrganisation',
      '10': 'organisations'
    },
  ],
};

/// Descriptor for `ListNearbyOrganisationsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listNearbyOrganisationsResponseDescriptor =
    $convert.base64Decode(
        'Ch9MaXN0TmVhcmJ5T3JnYW5pc2F0aW9uc1Jlc3BvbnNlEj4KDW9yZ2FuaXNhdGlvbnMYASADKA'
        'syGC5hcGljLk5lYXJieU9yZ2FuaXNhdGlvblINb3JnYW5pc2F0aW9ucw==');

@$core.Deprecated('Use joinOrganisationRequestDescriptor instead')
const JoinOrganisationRequest$json = {
  '1': 'JoinOrganisationRequest',
  '2': [
    {'1': 'organisation_id', '3': 1, '4': 1, '5': 9, '10': 'organisationId'},
    {'1': 'peer_id', '3': 2, '4': 1, '5': 9, '10': 'peerId'},
    {'1': 'invite_id', '3': 3, '4': 1, '5': 9, '10': 'inviteId'},
    {'1': 'username', '3': 4, '4': 1, '5': 9, '10': 'username'},
    {'1': 'name', '3': 5, '4': 1, '5': 9, '10': 'name'},
    {'1': 'channel', '3': 6, '4': 1, '5': 9, '10': 'channel'},
    {
      '1': 'verification_code',
      '3': 7,
      '4': 1,
      '5': 9,
      '10': 'verificationCode'
    },
    {'1': 'join_mode', '3': 8, '4': 1, '5': 9, '10': 'joinMode'},
  ],
};

/// Descriptor for `JoinOrganisationRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List joinOrganisationRequestDescriptor = $convert.base64Decode(
    'ChdKb2luT3JnYW5pc2F0aW9uUmVxdWVzdBInCg9vcmdhbmlzYXRpb25faWQYASABKAlSDm9yZ2'
    'FuaXNhdGlvbklkEhcKB3BlZXJfaWQYAiABKAlSBnBlZXJJZBIbCglpbnZpdGVfaWQYAyABKAlS'
    'CGludml0ZUlkEhoKCHVzZXJuYW1lGAQgASgJUgh1c2VybmFtZRISCgRuYW1lGAUgASgJUgRuYW'
    '1lEhgKB2NoYW5uZWwYBiABKAlSB2NoYW5uZWwSKwoRdmVyaWZpY2F0aW9uX2NvZGUYByABKAlS'
    'EHZlcmlmaWNhdGlvbkNvZGUSGwoJam9pbl9tb2RlGAggASgJUghqb2luTW9kZQ==');

@$core.Deprecated('Use joinOrganisationResponseDescriptor instead')
const JoinOrganisationResponse$json = {
  '1': 'JoinOrganisationResponse',
};

/// Descriptor for `JoinOrganisationResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List joinOrganisationResponseDescriptor =
    $convert.base64Decode('ChhKb2luT3JnYW5pc2F0aW9uUmVzcG9uc2U=');

@$core.Deprecated('Use getLocalSSHKeyRequestDescriptor instead')
const GetLocalSSHKeyRequest$json = {
  '1': 'GetLocalSSHKeyRequest',
};

/// Descriptor for `GetLocalSSHKeyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLocalSSHKeyRequestDescriptor =
    $convert.base64Decode('ChVHZXRMb2NhbFNTSEtleVJlcXVlc3Q=');

@$core.Deprecated('Use getLocalSSHKeyResponseDescriptor instead')
const GetLocalSSHKeyResponse$json = {
  '1': 'GetLocalSSHKeyResponse',
  '2': [
    {'1': 'public', '3': 1, '4': 1, '5': 9, '10': 'public'},
    {'1': 'private', '3': 2, '4': 1, '5': 9, '10': 'private'},
  ],
};

/// Descriptor for `GetLocalSSHKeyResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLocalSSHKeyResponseDescriptor =
    $convert.base64Decode(
        'ChZHZXRMb2NhbFNTSEtleVJlc3BvbnNlEhYKBnB1YmxpYxgBIAEoCVIGcHVibGljEhgKB3ByaX'
        'ZhdGUYAiABKAlSB3ByaXZhdGU=');

@$core.Deprecated('Use appDescriptor instead')
const App$json = {
  '1': 'App',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'version', '3': 3, '4': 1, '5': 9, '10': 'version'},
    {'1': 'status', '3': 4, '4': 1, '5': 9, '10': 'status'},
    {'1': 'instance_name', '3': 5, '4': 1, '5': 9, '10': 'instanceName'},
    {'1': 'ip', '3': 6, '4': 1, '5': 9, '10': 'ip'},
    {'1': 'installer', '3': 7, '4': 1, '5': 9, '10': 'installer'},
    {'1': 'persistence', '3': 8, '4': 1, '5': 8, '10': 'persistence'},
  ],
};

/// Descriptor for `App`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List appDescriptor = $convert.base64Decode(
    'CgNBcHASDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbWUSGAoHdmVyc2lvbhgDIA'
    'EoCVIHdmVyc2lvbhIWCgZzdGF0dXMYBCABKAlSBnN0YXR1cxIjCg1pbnN0YW5jZV9uYW1lGAUg'
    'ASgJUgxpbnN0YW5jZU5hbWUSDgoCaXAYBiABKAlSAmlwEhwKCWluc3RhbGxlchgHIAEoCVIJaW'
    '5zdGFsbGVyEiAKC3BlcnNpc3RlbmNlGAggASgIUgtwZXJzaXN0ZW5jZQ==');

@$core.Deprecated('Use getAppsRequestDescriptor instead')
const GetAppsRequest$json = {
  '1': 'GetAppsRequest',
};

/// Descriptor for `GetAppsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAppsRequestDescriptor =
    $convert.base64Decode('Cg5HZXRBcHBzUmVxdWVzdA==');

@$core.Deprecated('Use getAppsResponseDescriptor instead')
const GetAppsResponse$json = {
  '1': 'GetAppsResponse',
  '2': [
    {'1': 'apps', '3': 1, '4': 3, '5': 11, '6': '.apic.App', '10': 'apps'},
  ],
};

/// Descriptor for `GetAppsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAppsResponseDescriptor = $convert.base64Decode(
    'Cg9HZXRBcHBzUmVzcG9uc2USHQoEYXBwcxgBIAMoCzIJLmFwaWMuQXBwUgRhcHBz');

@$core.Deprecated('Use createAppRequestDescriptor instead')
const CreateAppRequest$json = {
  '1': 'CreateAppRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'installer_id', '3': 2, '4': 1, '5': 9, '10': 'installerId'},
    {'1': 'instance_id', '3': 3, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'persistence', '3': 4, '4': 1, '5': 8, '10': 'persistence'},
  ],
};

/// Descriptor for `CreateAppRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createAppRequestDescriptor = $convert.base64Decode(
    'ChBDcmVhdGVBcHBSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWUSIQoMaW5zdGFsbGVyX2lkGA'
    'IgASgJUgtpbnN0YWxsZXJJZBIfCgtpbnN0YW5jZV9pZBgDIAEoCVIKaW5zdGFuY2VJZBIgCgtw'
    'ZXJzaXN0ZW5jZRgEIAEoCFILcGVyc2lzdGVuY2U=');

@$core.Deprecated('Use createAppResponseDescriptor instead')
const CreateAppResponse$json = {
  '1': 'CreateAppResponse',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `CreateAppResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createAppResponseDescriptor =
    $convert.base64Decode('ChFDcmVhdGVBcHBSZXNwb25zZRIOCgJpZBgBIAEoCVICaWQ=');

@$core.Deprecated('Use startAppRequestDescriptor instead')
const StartAppRequest$json = {
  '1': 'StartAppRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `StartAppRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startAppRequestDescriptor = $convert
    .base64Decode('Cg9TdGFydEFwcFJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use startAppResponseDescriptor instead')
const StartAppResponse$json = {
  '1': 'StartAppResponse',
};

/// Descriptor for `StartAppResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startAppResponseDescriptor =
    $convert.base64Decode('ChBTdGFydEFwcFJlc3BvbnNl');

@$core.Deprecated('Use stopAppRequestDescriptor instead')
const StopAppRequest$json = {
  '1': 'StopAppRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `StopAppRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopAppRequestDescriptor =
    $convert.base64Decode('Cg5TdG9wQXBwUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1l');

@$core.Deprecated('Use stopAppResponseDescriptor instead')
const StopAppResponse$json = {
  '1': 'StopAppResponse',
};

/// Descriptor for `StopAppResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopAppResponseDescriptor =
    $convert.base64Decode('Cg9TdG9wQXBwUmVzcG9uc2U=');

@$core.Deprecated('Use removeAppRequestDescriptor instead')
const RemoveAppRequest$json = {
  '1': 'RemoveAppRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `RemoveAppRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeAppRequestDescriptor = $convert
    .base64Decode('ChBSZW1vdmVBcHBSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use removeAppResponseDescriptor instead')
const RemoveAppResponse$json = {
  '1': 'RemoveAppResponse',
};

/// Descriptor for `RemoveAppResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeAppResponseDescriptor =
    $convert.base64Decode('ChFSZW1vdmVBcHBSZXNwb25zZQ==');

@$core.Deprecated('Use getAppLogsRequestDescriptor instead')
const GetAppLogsRequest$json = {
  '1': 'GetAppLogsRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetAppLogsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAppLogsRequestDescriptor = $convert
    .base64Decode('ChFHZXRBcHBMb2dzUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1l');

@$core.Deprecated('Use getAppLogsResponseDescriptor instead')
const GetAppLogsResponse$json = {
  '1': 'GetAppLogsResponse',
  '2': [
    {'1': 'logs', '3': 1, '4': 1, '5': 12, '10': 'logs'},
  ],
};

/// Descriptor for `GetAppLogsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getAppLogsResponseDescriptor = $convert
    .base64Decode('ChJHZXRBcHBMb2dzUmVzcG9uc2USEgoEbG9ncxgBIAEoDFIEbG9ncw==');

@$core.Deprecated('Use installerDescriptor instead')
const Installer$json = {
  '1': 'Installer',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'version', '3': 3, '4': 1, '5': 9, '10': 'version'},
    {'1': 'description', '3': 4, '4': 1, '5': 9, '10': 'description'},
    {
      '1': 'requires_resources',
      '3': 5,
      '4': 3,
      '5': 9,
      '10': 'requiresResources'
    },
    {
      '1': 'provides_resources',
      '3': 6,
      '4': 3,
      '5': 9,
      '10': 'providesResources'
    },
    {'1': 'capabilities', '3': 7, '4': 3, '5': 9, '10': 'capabilities'},
  ],
};

/// Descriptor for `Installer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List installerDescriptor = $convert.base64Decode(
    'CglJbnN0YWxsZXISDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbWUSGAoHdmVyc2'
    'lvbhgDIAEoCVIHdmVyc2lvbhIgCgtkZXNjcmlwdGlvbhgEIAEoCVILZGVzY3JpcHRpb24SLQoS'
    'cmVxdWlyZXNfcmVzb3VyY2VzGAUgAygJUhFyZXF1aXJlc1Jlc291cmNlcxItChJwcm92aWRlc1'
    '9yZXNvdXJjZXMYBiADKAlSEXByb3ZpZGVzUmVzb3VyY2VzEiIKDGNhcGFiaWxpdGllcxgHIAMo'
    'CVIMY2FwYWJpbGl0aWVz');

@$core.Deprecated('Use getInstallersRequestDescriptor instead')
const GetInstallersRequest$json = {
  '1': 'GetInstallersRequest',
};

/// Descriptor for `GetInstallersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstallersRequestDescriptor =
    $convert.base64Decode('ChRHZXRJbnN0YWxsZXJzUmVxdWVzdA==');

@$core.Deprecated('Use getInstallersResponseDescriptor instead')
const GetInstallersResponse$json = {
  '1': 'GetInstallersResponse',
  '2': [
    {
      '1': 'installers',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Installer',
      '10': 'installers'
    },
  ],
};

/// Descriptor for `GetInstallersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstallersResponseDescriptor = $convert.base64Decode(
    'ChVHZXRJbnN0YWxsZXJzUmVzcG9uc2USLwoKaW5zdGFsbGVycxgBIAMoCzIPLmFwaWMuSW5zdG'
    'FsbGVyUgppbnN0YWxsZXJz');

@$core.Deprecated('Use getInstallerRequestDescriptor instead')
const GetInstallerRequest$json = {
  '1': 'GetInstallerRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `GetInstallerRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstallerRequestDescriptor = $convert
    .base64Decode('ChNHZXRJbnN0YWxsZXJSZXF1ZXN0Eg4KAmlkGAEgASgJUgJpZA==');

@$core.Deprecated('Use getInstallerResponseDescriptor instead')
const GetInstallerResponse$json = {
  '1': 'GetInstallerResponse',
  '2': [
    {
      '1': 'installer',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.Installer',
      '10': 'installer'
    },
  ],
};

/// Descriptor for `GetInstallerResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstallerResponseDescriptor = $convert.base64Decode(
    'ChRHZXRJbnN0YWxsZXJSZXNwb25zZRItCglpbnN0YWxsZXIYASABKAsyDy5hcGljLkluc3RhbG'
    'xlclIJaW5zdGFsbGVy');

@$core.Deprecated('Use cloudMachineSpecDescriptor instead')
const CloudMachineSpec$json = {
  '1': 'CloudMachineSpec',
  '2': [
    {'1': 'cores', '3': 1, '4': 1, '5': 5, '10': 'cores'},
    {'1': 'memory', '3': 2, '4': 1, '5': 5, '10': 'memory'},
    {'1': 'default_storage', '3': 3, '4': 1, '5': 5, '10': 'defaultStorage'},
    {'1': 'bandwidth', '3': 4, '4': 1, '5': 5, '10': 'bandwidth'},
    {
      '1': 'included_data_transfer',
      '3': 5,
      '4': 1,
      '5': 5,
      '10': 'includedDataTransfer'
    },
    {'1': 'baremetal', '3': 6, '4': 1, '5': 8, '10': 'baremetal'},
    {'1': 'price_monthly', '3': 7, '4': 1, '5': 2, '10': 'priceMonthly'},
  ],
};

/// Descriptor for `CloudMachineSpec`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudMachineSpecDescriptor = $convert.base64Decode(
    'ChBDbG91ZE1hY2hpbmVTcGVjEhQKBWNvcmVzGAEgASgFUgVjb3JlcxIWCgZtZW1vcnkYAiABKA'
    'VSBm1lbW9yeRInCg9kZWZhdWx0X3N0b3JhZ2UYAyABKAVSDmRlZmF1bHRTdG9yYWdlEhwKCWJh'
    'bmR3aWR0aBgEIAEoBVIJYmFuZHdpZHRoEjQKFmluY2x1ZGVkX2RhdGFfdHJhbnNmZXIYBSABKA'
    'VSFGluY2x1ZGVkRGF0YVRyYW5zZmVyEhwKCWJhcmVtZXRhbBgGIAEoCFIJYmFyZW1ldGFsEiMK'
    'DXByaWNlX21vbnRobHkYByABKAJSDHByaWNlTW9udGhseQ==');

@$core.Deprecated('Use cloudTypeDescriptor instead')
const CloudType$json = {
  '1': 'CloudType',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'authentication_fields',
      '3': 2,
      '4': 3,
      '5': 9,
      '10': 'authenticationFields'
    },
  ],
};

/// Descriptor for `CloudType`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudTypeDescriptor = $convert.base64Decode(
    'CglDbG91ZFR5cGUSEgoEbmFtZRgBIAEoCVIEbmFtZRIzChVhdXRoZW50aWNhdGlvbl9maWVsZH'
    'MYAiADKAlSFGF1dGhlbnRpY2F0aW9uRmllbGRz');

@$core.Deprecated('Use cloudProviderDescriptor instead')
const CloudProvider$json = {
  '1': 'CloudProvider',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'type',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudType',
      '10': 'type'
    },
    {
      '1': 'supported_locations',
      '3': 3,
      '4': 3,
      '5': 9,
      '10': 'supportedLocations'
    },
    {
      '1': 'supported_machines',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.apic.CloudProvider.SupportedMachinesEntry',
      '10': 'supportedMachines'
    },
  ],
  '3': [CloudProvider_SupportedMachinesEntry$json],
};

@$core.Deprecated('Use cloudProviderDescriptor instead')
const CloudProvider_SupportedMachinesEntry$json = {
  '1': 'SupportedMachinesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudMachineSpec',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `CloudProvider`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudProviderDescriptor = $convert.base64Decode(
    'Cg1DbG91ZFByb3ZpZGVyEhIKBG5hbWUYASABKAlSBG5hbWUSIwoEdHlwZRgCIAEoCzIPLmFwaW'
    'MuQ2xvdWRUeXBlUgR0eXBlEi8KE3N1cHBvcnRlZF9sb2NhdGlvbnMYAyADKAlSEnN1cHBvcnRl'
    'ZExvY2F0aW9ucxJZChJzdXBwb3J0ZWRfbWFjaGluZXMYBCADKAsyKi5hcGljLkNsb3VkUHJvdm'
    'lkZXIuU3VwcG9ydGVkTWFjaGluZXNFbnRyeVIRc3VwcG9ydGVkTWFjaGluZXMaXAoWU3VwcG9y'
    'dGVkTWFjaGluZXNFbnRyeRIQCgNrZXkYASABKAlSA2tleRIsCgV2YWx1ZRgCIAEoCzIWLmFwaW'
    'MuQ2xvdWRNYWNoaW5lU3BlY1IFdmFsdWU6AjgB');

@$core.Deprecated('Use getSupportedCloudProvidersRequestDescriptor instead')
const GetSupportedCloudProvidersRequest$json = {
  '1': 'GetSupportedCloudProvidersRequest',
};

/// Descriptor for `GetSupportedCloudProvidersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSupportedCloudProvidersRequestDescriptor =
    $convert.base64Decode('CiFHZXRTdXBwb3J0ZWRDbG91ZFByb3ZpZGVyc1JlcXVlc3Q=');

@$core.Deprecated('Use getSupportedCloudProvidersResponseDescriptor instead')
const GetSupportedCloudProvidersResponse$json = {
  '1': 'GetSupportedCloudProvidersResponse',
  '2': [
    {
      '1': 'cloud_types',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.CloudType',
      '10': 'cloudTypes'
    },
  ],
};

/// Descriptor for `GetSupportedCloudProvidersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSupportedCloudProvidersResponseDescriptor =
    $convert.base64Decode(
        'CiJHZXRTdXBwb3J0ZWRDbG91ZFByb3ZpZGVyc1Jlc3BvbnNlEjAKC2Nsb3VkX3R5cGVzGAEgAy'
        'gLMg8uYXBpYy5DbG91ZFR5cGVSCmNsb3VkVHlwZXM=');

@$core.Deprecated('Use getCloudProvidersRequestDescriptor instead')
const GetCloudProvidersRequest$json = {
  '1': 'GetCloudProvidersRequest',
};

/// Descriptor for `GetCloudProvidersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudProvidersRequestDescriptor =
    $convert.base64Decode('ChhHZXRDbG91ZFByb3ZpZGVyc1JlcXVlc3Q=');

@$core.Deprecated('Use getCloudProvidersResponseDescriptor instead')
const GetCloudProvidersResponse$json = {
  '1': 'GetCloudProvidersResponse',
  '2': [
    {
      '1': 'cloud_providers',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.CloudProvider',
      '10': 'cloudProviders'
    },
  ],
};

/// Descriptor for `GetCloudProvidersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudProvidersResponseDescriptor =
    $convert.base64Decode(
        'ChlHZXRDbG91ZFByb3ZpZGVyc1Jlc3BvbnNlEjwKD2Nsb3VkX3Byb3ZpZGVycxgBIAMoCzITLm'
        'FwaWMuQ2xvdWRQcm92aWRlclIOY2xvdWRQcm92aWRlcnM=');

@$core.Deprecated('Use getCloudProviderRequestDescriptor instead')
const GetCloudProviderRequest$json = {
  '1': 'GetCloudProviderRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetCloudProviderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudProviderRequestDescriptor =
    $convert.base64Decode(
        'ChdHZXRDbG91ZFByb3ZpZGVyUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1l');

@$core.Deprecated('Use getCloudProviderResponseDescriptor instead')
const GetCloudProviderResponse$json = {
  '1': 'GetCloudProviderResponse',
  '2': [
    {
      '1': 'cloud_provider',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudProvider',
      '10': 'cloudProvider'
    },
  ],
};

/// Descriptor for `GetCloudProviderResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudProviderResponseDescriptor =
    $convert.base64Decode(
        'ChhHZXRDbG91ZFByb3ZpZGVyUmVzcG9uc2USOgoOY2xvdWRfcHJvdmlkZXIYASABKAsyEy5hcG'
        'ljLkNsb3VkUHJvdmlkZXJSDWNsb3VkUHJvdmlkZXI=');

@$core.Deprecated('Use addCloudProviderRequestDescriptor instead')
const AddCloudProviderRequest$json = {
  '1': 'AddCloudProviderRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 2, '4': 1, '5': 9, '10': 'type'},
    {
      '1': 'credentials',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.AddCloudProviderRequest.CredentialsEntry',
      '10': 'credentials'
    },
  ],
  '3': [AddCloudProviderRequest_CredentialsEntry$json],
};

@$core.Deprecated('Use addCloudProviderRequestDescriptor instead')
const AddCloudProviderRequest_CredentialsEntry$json = {
  '1': 'CredentialsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `AddCloudProviderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addCloudProviderRequestDescriptor = $convert.base64Decode(
    'ChdBZGRDbG91ZFByb3ZpZGVyUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEhIKBHR5cGUYAi'
    'ABKAlSBHR5cGUSUAoLY3JlZGVudGlhbHMYAyADKAsyLi5hcGljLkFkZENsb3VkUHJvdmlkZXJS'
    'ZXF1ZXN0LkNyZWRlbnRpYWxzRW50cnlSC2NyZWRlbnRpYWxzGj4KEENyZWRlbnRpYWxzRW50cn'
    'kSEAoDa2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use addCloudProviderResponseDescriptor instead')
const AddCloudProviderResponse$json = {
  '1': 'AddCloudProviderResponse',
};

/// Descriptor for `AddCloudProviderResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addCloudProviderResponseDescriptor =
    $convert.base64Decode('ChhBZGRDbG91ZFByb3ZpZGVyUmVzcG9uc2U=');

@$core.Deprecated('Use removeCloudProviderRequestDescriptor instead')
const RemoveCloudProviderRequest$json = {
  '1': 'RemoveCloudProviderRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `RemoveCloudProviderRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeCloudProviderRequestDescriptor =
    $convert.base64Decode(
        'ChpSZW1vdmVDbG91ZFByb3ZpZGVyUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1l');

@$core.Deprecated('Use removeCloudProviderResponseDescriptor instead')
const RemoveCloudProviderResponse$json = {
  '1': 'RemoveCloudProviderResponse',
};

/// Descriptor for `RemoveCloudProviderResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeCloudProviderResponseDescriptor =
    $convert.base64Decode('ChtSZW1vdmVDbG91ZFByb3ZpZGVyUmVzcG9uc2U=');

@$core.Deprecated('Use provisionerMachineSpecDescriptor instead')
const ProvisionerMachineSpec$json = {
  '1': 'ProvisionerMachineSpec',
  '2': [
    {'1': 'cores', '3': 1, '4': 1, '5': 5, '10': 'cores'},
    {'1': 'memory', '3': 2, '4': 1, '5': 5, '10': 'memory'},
    {'1': 'default_storage', '3': 3, '4': 1, '5': 5, '10': 'defaultStorage'},
    {'1': 'bandwidth', '3': 4, '4': 1, '5': 5, '10': 'bandwidth'},
    {
      '1': 'included_data_transfer',
      '3': 5,
      '4': 1,
      '5': 5,
      '10': 'includedDataTransfer'
    },
    {'1': 'baremetal', '3': 6, '4': 1, '5': 8, '10': 'baremetal'},
    {'1': 'price_monthly', '3': 7, '4': 1, '5': 2, '10': 'priceMonthly'},
  ],
};

/// Descriptor for `ProvisionerMachineSpec`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List provisionerMachineSpecDescriptor = $convert.base64Decode(
    'ChZQcm92aXNpb25lck1hY2hpbmVTcGVjEhQKBWNvcmVzGAEgASgFUgVjb3JlcxIWCgZtZW1vcn'
    'kYAiABKAVSBm1lbW9yeRInCg9kZWZhdWx0X3N0b3JhZ2UYAyABKAVSDmRlZmF1bHRTdG9yYWdl'
    'EhwKCWJhbmR3aWR0aBgEIAEoBVIJYmFuZHdpZHRoEjQKFmluY2x1ZGVkX2RhdGFfdHJhbnNmZX'
    'IYBSABKAVSFGluY2x1ZGVkRGF0YVRyYW5zZmVyEhwKCWJhcmVtZXRhbBgGIAEoCFIJYmFyZW1l'
    'dGFsEiMKDXByaWNlX21vbnRobHkYByABKAJSDHByaWNlTW9udGhseQ==');

@$core.Deprecated('Use provisionerTypeDescriptor instead')
const ProvisionerType$json = {
  '1': 'ProvisionerType',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'authentication_fields',
      '3': 2,
      '4': 3,
      '5': 9,
      '10': 'authenticationFields'
    },
  ],
};

/// Descriptor for `ProvisionerType`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List provisionerTypeDescriptor = $convert.base64Decode(
    'Cg9Qcm92aXNpb25lclR5cGUSEgoEbmFtZRgBIAEoCVIEbmFtZRIzChVhdXRoZW50aWNhdGlvbl'
    '9maWVsZHMYAiADKAlSFGF1dGhlbnRpY2F0aW9uRmllbGRz');

@$core.Deprecated('Use provisionerDescriptor instead')
const Provisioner$json = {
  '1': 'Provisioner',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'type',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.ProvisionerType',
      '10': 'type'
    },
    {
      '1': 'supported_locations',
      '3': 3,
      '4': 3,
      '5': 9,
      '10': 'supportedLocations'
    },
    {
      '1': 'supported_machines',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.apic.Provisioner.SupportedMachinesEntry',
      '10': 'supportedMachines'
    },
  ],
  '3': [Provisioner_SupportedMachinesEntry$json],
};

@$core.Deprecated('Use provisionerDescriptor instead')
const Provisioner_SupportedMachinesEntry$json = {
  '1': 'SupportedMachinesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.ProvisionerMachineSpec',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `Provisioner`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List provisionerDescriptor = $convert.base64Decode(
    'CgtQcm92aXNpb25lchISCgRuYW1lGAEgASgJUgRuYW1lEikKBHR5cGUYAiABKAsyFS5hcGljLl'
    'Byb3Zpc2lvbmVyVHlwZVIEdHlwZRIvChNzdXBwb3J0ZWRfbG9jYXRpb25zGAMgAygJUhJzdXBw'
    'b3J0ZWRMb2NhdGlvbnMSVwoSc3VwcG9ydGVkX21hY2hpbmVzGAQgAygLMiguYXBpYy5Qcm92aX'
    'Npb25lci5TdXBwb3J0ZWRNYWNoaW5lc0VudHJ5UhFzdXBwb3J0ZWRNYWNoaW5lcxpiChZTdXBw'
    'b3J0ZWRNYWNoaW5lc0VudHJ5EhAKA2tleRgBIAEoCVIDa2V5EjIKBXZhbHVlGAIgASgLMhwuYX'
    'BpYy5Qcm92aXNpb25lck1hY2hpbmVTcGVjUgV2YWx1ZToCOAE=');

@$core.Deprecated('Use getSupportedProvisionersRequestDescriptor instead')
const GetSupportedProvisionersRequest$json = {
  '1': 'GetSupportedProvisionersRequest',
};

/// Descriptor for `GetSupportedProvisionersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSupportedProvisionersRequestDescriptor =
    $convert.base64Decode('Ch9HZXRTdXBwb3J0ZWRQcm92aXNpb25lcnNSZXF1ZXN0');

@$core.Deprecated('Use getSupportedProvisionersResponseDescriptor instead')
const GetSupportedProvisionersResponse$json = {
  '1': 'GetSupportedProvisionersResponse',
  '2': [
    {
      '1': 'provisioner_types',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.ProvisionerType',
      '10': 'provisionerTypes'
    },
  ],
};

/// Descriptor for `GetSupportedProvisionersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSupportedProvisionersResponseDescriptor =
    $convert.base64Decode(
        'CiBHZXRTdXBwb3J0ZWRQcm92aXNpb25lcnNSZXNwb25zZRJCChFwcm92aXNpb25lcl90eXBlcx'
        'gBIAMoCzIVLmFwaWMuUHJvdmlzaW9uZXJUeXBlUhBwcm92aXNpb25lclR5cGVz');

@$core.Deprecated('Use getProvisionersRequestDescriptor instead')
const GetProvisionersRequest$json = {
  '1': 'GetProvisionersRequest',
};

/// Descriptor for `GetProvisionersRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionersRequestDescriptor =
    $convert.base64Decode('ChZHZXRQcm92aXNpb25lcnNSZXF1ZXN0');

@$core.Deprecated('Use getProvisionersResponseDescriptor instead')
const GetProvisionersResponse$json = {
  '1': 'GetProvisionersResponse',
  '2': [
    {
      '1': 'provisioners',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Provisioner',
      '10': 'provisioners'
    },
  ],
};

/// Descriptor for `GetProvisionersResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionersResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXRQcm92aXNpb25lcnNSZXNwb25zZRI1Cgxwcm92aXNpb25lcnMYASADKAsyES5hcGljLl'
        'Byb3Zpc2lvbmVyUgxwcm92aXNpb25lcnM=');

@$core.Deprecated('Use getProvisionerRequestDescriptor instead')
const GetProvisionerRequest$json = {
  '1': 'GetProvisionerRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetProvisionerRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionerRequestDescriptor =
    $convert.base64Decode(
        'ChVHZXRQcm92aXNpb25lclJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use getProvisionerResponseDescriptor instead')
const GetProvisionerResponse$json = {
  '1': 'GetProvisionerResponse',
  '2': [
    {
      '1': 'provisioner',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.Provisioner',
      '10': 'provisioner'
    },
  ],
};

/// Descriptor for `GetProvisionerResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionerResponseDescriptor =
    $convert.base64Decode(
        'ChZHZXRQcm92aXNpb25lclJlc3BvbnNlEjMKC3Byb3Zpc2lvbmVyGAEgASgLMhEuYXBpYy5Qcm'
        '92aXNpb25lclILcHJvdmlzaW9uZXI=');

@$core.Deprecated('Use addProvisionerRequestDescriptor instead')
const AddProvisionerRequest$json = {
  '1': 'AddProvisionerRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 2, '4': 1, '5': 9, '10': 'type'},
    {
      '1': 'credentials',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.AddProvisionerRequest.CredentialsEntry',
      '10': 'credentials'
    },
  ],
  '3': [AddProvisionerRequest_CredentialsEntry$json],
};

@$core.Deprecated('Use addProvisionerRequestDescriptor instead')
const AddProvisionerRequest_CredentialsEntry$json = {
  '1': 'CredentialsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `AddProvisionerRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addProvisionerRequestDescriptor = $convert.base64Decode(
    'ChVBZGRQcm92aXNpb25lclJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRISCgR0eXBlGAIgAS'
    'gJUgR0eXBlEk4KC2NyZWRlbnRpYWxzGAMgAygLMiwuYXBpYy5BZGRQcm92aXNpb25lclJlcXVl'
    'c3QuQ3JlZGVudGlhbHNFbnRyeVILY3JlZGVudGlhbHMaPgoQQ3JlZGVudGlhbHNFbnRyeRIQCg'
    'NrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgB');

@$core.Deprecated('Use addProvisionerResponseDescriptor instead')
const AddProvisionerResponse$json = {
  '1': 'AddProvisionerResponse',
};

/// Descriptor for `AddProvisionerResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addProvisionerResponseDescriptor =
    $convert.base64Decode('ChZBZGRQcm92aXNpb25lclJlc3BvbnNl');

@$core.Deprecated('Use removeProvisionerRequestDescriptor instead')
const RemoveProvisionerRequest$json = {
  '1': 'RemoveProvisionerRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `RemoveProvisionerRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeProvisionerRequestDescriptor =
    $convert.base64Decode(
        'ChhSZW1vdmVQcm92aXNpb25lclJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use removeProvisionerResponseDescriptor instead')
const RemoveProvisionerResponse$json = {
  '1': 'RemoveProvisionerResponse',
};

/// Descriptor for `RemoveProvisionerResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeProvisionerResponseDescriptor =
    $convert.base64Decode('ChlSZW1vdmVQcm92aXNpb25lclJlc3BvbnNl');

@$core.Deprecated('Use cloudInstanceDescriptor instead')
const CloudInstance$json = {
  '1': 'CloudInstance',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'public_ip', '3': 2, '4': 1, '5': 9, '10': 'publicIp'},
    {'1': 'internal_ip', '3': 3, '4': 1, '5': 9, '10': 'internalIp'},
    {'1': 'cloud_name', '3': 4, '4': 1, '5': 9, '10': 'cloudName'},
    {'1': 'cloud_type', '3': 5, '4': 1, '5': 9, '10': 'cloudType'},
    {'1': 'vm_id', '3': 6, '4': 1, '5': 9, '10': 'vmId'},
    {'1': 'location', '3': 7, '4': 1, '5': 9, '10': 'location'},
    {'1': 'public_key', '3': 8, '4': 1, '5': 9, '10': 'publicKey'},
    {
      '1': 'public_key_wireguard',
      '3': 9,
      '4': 1,
      '5': 9,
      '10': 'publicKeyWireguard'
    },
    {'1': 'protos_version', '3': 10, '4': 1, '5': 9, '10': 'protosVersion'},
    {'1': 'status', '3': 11, '4': 1, '5': 9, '10': 'status'},
    {'1': 'architecture', '3': 12, '4': 1, '5': 9, '10': 'architecture'},
    {
      '1': 'peers',
      '3': 13,
      '4': 3,
      '5': 11,
      '6': '.apic.CloudInstance.PeersEntry',
      '10': 'peers'
    },
    {'1': 'provider_status', '3': 14, '4': 1, '5': 9, '10': 'providerStatus'},
    {
      '1': 'admin_api_reachability',
      '3': 15,
      '4': 1,
      '5': 9,
      '10': 'adminApiReachability'
    },
    {
      '1': 'replication_connected',
      '3': 16,
      '4': 1,
      '5': 8,
      '10': 'replicationConnected'
    },
    {'1': 'admin_last_error', '3': 17, '4': 1, '5': 9, '10': 'adminLastError'},
    {'1': 'admin_last_seen', '3': 18, '4': 1, '5': 9, '10': 'adminLastSeen'},
    {'1': 'peer_id', '3': 19, '4': 1, '5': 9, '10': 'peerId'},
  ],
  '3': [CloudInstance_PeersEntry$json],
};

@$core.Deprecated('Use cloudInstanceDescriptor instead')
const CloudInstance_PeersEntry$json = {
  '1': 'PeersEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `CloudInstance`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudInstanceDescriptor = $convert.base64Decode(
    'Cg1DbG91ZEluc3RhbmNlEhIKBG5hbWUYASABKAlSBG5hbWUSGwoJcHVibGljX2lwGAIgASgJUg'
    'hwdWJsaWNJcBIfCgtpbnRlcm5hbF9pcBgDIAEoCVIKaW50ZXJuYWxJcBIdCgpjbG91ZF9uYW1l'
    'GAQgASgJUgljbG91ZE5hbWUSHQoKY2xvdWRfdHlwZRgFIAEoCVIJY2xvdWRUeXBlEhMKBXZtX2'
    'lkGAYgASgJUgR2bUlkEhoKCGxvY2F0aW9uGAcgASgJUghsb2NhdGlvbhIdCgpwdWJsaWNfa2V5'
    'GAggASgJUglwdWJsaWNLZXkSMAoUcHVibGljX2tleV93aXJlZ3VhcmQYCSABKAlSEnB1YmxpY0'
    'tleVdpcmVndWFyZBIlCg5wcm90b3NfdmVyc2lvbhgKIAEoCVINcHJvdG9zVmVyc2lvbhIWCgZz'
    'dGF0dXMYCyABKAlSBnN0YXR1cxIiCgxhcmNoaXRlY3R1cmUYDCABKAlSDGFyY2hpdGVjdHVyZR'
    'I0CgVwZWVycxgNIAMoCzIeLmFwaWMuQ2xvdWRJbnN0YW5jZS5QZWVyc0VudHJ5UgVwZWVycxIn'
    'Cg9wcm92aWRlcl9zdGF0dXMYDiABKAlSDnByb3ZpZGVyU3RhdHVzEjQKFmFkbWluX2FwaV9yZW'
    'FjaGFiaWxpdHkYDyABKAlSFGFkbWluQXBpUmVhY2hhYmlsaXR5EjMKFXJlcGxpY2F0aW9uX2Nv'
    'bm5lY3RlZBgQIAEoCFIUcmVwbGljYXRpb25Db25uZWN0ZWQSKAoQYWRtaW5fbGFzdF9lcnJvch'
    'gRIAEoCVIOYWRtaW5MYXN0RXJyb3ISJgoPYWRtaW5fbGFzdF9zZWVuGBIgASgJUg1hZG1pbkxh'
    'c3RTZWVuEhcKB3BlZXJfaWQYEyABKAlSBnBlZXJJZBo4CgpQZWVyc0VudHJ5EhAKA2tleRgBIA'
    'EoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZToCOAE=');

@$core.Deprecated('Use getInstancesRequestDescriptor instead')
const GetInstancesRequest$json = {
  '1': 'GetInstancesRequest',
};

/// Descriptor for `GetInstancesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstancesRequestDescriptor =
    $convert.base64Decode('ChNHZXRJbnN0YW5jZXNSZXF1ZXN0');

@$core.Deprecated('Use getInstancesResponseDescriptor instead')
const GetInstancesResponse$json = {
  '1': 'GetInstancesResponse',
  '2': [
    {
      '1': 'instances',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.CloudInstance',
      '10': 'instances'
    },
  ],
};

/// Descriptor for `GetInstancesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstancesResponseDescriptor = $convert.base64Decode(
    'ChRHZXRJbnN0YW5jZXNSZXNwb25zZRIxCglpbnN0YW5jZXMYASADKAsyEy5hcGljLkNsb3VkSW'
    '5zdGFuY2VSCWluc3RhbmNlcw==');

@$core.Deprecated('Use getInstanceRequestDescriptor instead')
const GetInstanceRequest$json = {
  '1': 'GetInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceRequestDescriptor = $convert
    .base64Decode('ChJHZXRJbnN0YW5jZVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use getInstanceResponseDescriptor instead')
const GetInstanceResponse$json = {
  '1': 'GetInstanceResponse',
  '2': [
    {
      '1': 'instance',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudInstance',
      '10': 'instance'
    },
  ],
};

/// Descriptor for `GetInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceResponseDescriptor = $convert.base64Decode(
    'ChNHZXRJbnN0YW5jZVJlc3BvbnNlEi8KCGluc3RhbmNlGAEgASgLMhMuYXBpYy5DbG91ZEluc3'
    'RhbmNlUghpbnN0YW5jZQ==');

@$core.Deprecated('Use instanceDeployFieldOptionDescriptor instead')
const InstanceDeployFieldOption$json = {
  '1': 'InstanceDeployFieldOption',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 9, '10': 'value'},
    {'1': 'label', '3': 2, '4': 1, '5': 9, '10': 'label'},
    {'1': 'description', '3': 3, '4': 1, '5': 9, '10': 'description'},
  ],
};

/// Descriptor for `InstanceDeployFieldOption`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List instanceDeployFieldOptionDescriptor =
    $convert.base64Decode(
        'ChlJbnN0YW5jZURlcGxveUZpZWxkT3B0aW9uEhQKBXZhbHVlGAEgASgJUgV2YWx1ZRIUCgVsYW'
        'JlbBgCIAEoCVIFbGFiZWwSIAoLZGVzY3JpcHRpb24YAyABKAlSC2Rlc2NyaXB0aW9u');

@$core.Deprecated('Use instanceDeployFieldDescriptor instead')
const InstanceDeployField$json = {
  '1': 'InstanceDeployField',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'label', '3': 2, '4': 1, '5': 9, '10': 'label'},
    {'1': 'kind', '3': 3, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'required', '3': 4, '4': 1, '5': 8, '10': 'required'},
    {'1': 'visible', '3': 5, '4': 1, '5': 8, '10': 'visible'},
    {'1': 'value', '3': 6, '4': 1, '5': 9, '10': 'value'},
    {'1': 'helper', '3': 7, '4': 1, '5': 9, '10': 'helper'},
    {
      '1': 'options',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.apic.InstanceDeployFieldOption',
      '10': 'options'
    },
  ],
};

/// Descriptor for `InstanceDeployField`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List instanceDeployFieldDescriptor = $convert.base64Decode(
    'ChNJbnN0YW5jZURlcGxveUZpZWxkEhIKBG5hbWUYASABKAlSBG5hbWUSFAoFbGFiZWwYAiABKA'
    'lSBWxhYmVsEhIKBGtpbmQYAyABKAlSBGtpbmQSGgoIcmVxdWlyZWQYBCABKAhSCHJlcXVpcmVk'
    'EhgKB3Zpc2libGUYBSABKAhSB3Zpc2libGUSFAoFdmFsdWUYBiABKAlSBXZhbHVlEhYKBmhlbH'
    'BlchgHIAEoCVIGaGVscGVyEjkKB29wdGlvbnMYCCADKAsyHy5hcGljLkluc3RhbmNlRGVwbG95'
    'RmllbGRPcHRpb25SB29wdGlvbnM=');

@$core.Deprecated('Use getInstanceDeployOptionsRequestDescriptor instead')
const GetInstanceDeployOptionsRequest$json = {
  '1': 'GetInstanceDeployOptionsRequest',
  '2': [
    {'1': 'provisioner', '3': 1, '4': 1, '5': 9, '10': 'provisioner'},
    {'1': 'location', '3': 2, '4': 1, '5': 9, '10': 'location'},
  ],
};

/// Descriptor for `GetInstanceDeployOptionsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceDeployOptionsRequestDescriptor =
    $convert.base64Decode(
        'Ch9HZXRJbnN0YW5jZURlcGxveU9wdGlvbnNSZXF1ZXN0EiAKC3Byb3Zpc2lvbmVyGAEgASgJUg'
        'twcm92aXNpb25lchIaCghsb2NhdGlvbhgCIAEoCVIIbG9jYXRpb24=');

@$core.Deprecated('Use getInstanceDeployOptionsResponseDescriptor instead')
const GetInstanceDeployOptionsResponse$json = {
  '1': 'GetInstanceDeployOptionsResponse',
  '2': [
    {
      '1': 'fields',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.InstanceDeployField',
      '10': 'fields'
    },
  ],
};

/// Descriptor for `GetInstanceDeployOptionsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceDeployOptionsResponseDescriptor =
    $convert.base64Decode(
        'CiBHZXRJbnN0YW5jZURlcGxveU9wdGlvbnNSZXNwb25zZRIxCgZmaWVsZHMYASADKAsyGS5hcG'
        'ljLkluc3RhbmNlRGVwbG95RmllbGRSBmZpZWxkcw==');

@$core.Deprecated('Use deployInstanceRequestDescriptor instead')
const DeployInstanceRequest$json = {
  '1': 'DeployInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'cloud_name', '3': 2, '4': 1, '5': 9, '10': 'cloudName'},
    {'1': 'cloud_location', '3': 3, '4': 1, '5': 9, '10': 'cloudLocation'},
    {'1': 'machine_type', '3': 4, '4': 1, '5': 9, '10': 'machineType'},
    {'1': 'protos_version', '3': 5, '4': 1, '5': 9, '10': 'protosVersion'},
    {'1': 'dev_img', '3': 6, '4': 1, '5': 9, '10': 'devImg'},
  ],
};

/// Descriptor for `DeployInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deployInstanceRequestDescriptor = $convert.base64Decode(
    'ChVEZXBsb3lJbnN0YW5jZVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRIdCgpjbG91ZF9uYW'
    '1lGAIgASgJUgljbG91ZE5hbWUSJQoOY2xvdWRfbG9jYXRpb24YAyABKAlSDWNsb3VkTG9jYXRp'
    'b24SIQoMbWFjaGluZV90eXBlGAQgASgJUgttYWNoaW5lVHlwZRIlCg5wcm90b3NfdmVyc2lvbh'
    'gFIAEoCVINcHJvdG9zVmVyc2lvbhIXCgdkZXZfaW1nGAYgASgJUgZkZXZJbWc=');

@$core.Deprecated('Use deployInstanceResponseDescriptor instead')
const DeployInstanceResponse$json = {
  '1': 'DeployInstanceResponse',
  '2': [
    {
      '1': 'instance',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudInstance',
      '10': 'instance'
    },
  ],
};

/// Descriptor for `DeployInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deployInstanceResponseDescriptor =
    $convert.base64Decode(
        'ChZEZXBsb3lJbnN0YW5jZVJlc3BvbnNlEi8KCGluc3RhbmNlGAEgASgLMhMuYXBpYy5DbG91ZE'
        'luc3RhbmNlUghpbnN0YW5jZQ==');

@$core.Deprecated('Use removeInstanceRequestDescriptor instead')
const RemoveInstanceRequest$json = {
  '1': 'RemoveInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'local_only', '3': 2, '4': 1, '5': 8, '10': 'localOnly'},
  ],
};

/// Descriptor for `RemoveInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeInstanceRequestDescriptor = $convert.base64Decode(
    'ChVSZW1vdmVJbnN0YW5jZVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZRIdCgpsb2NhbF9vbm'
    'x5GAIgASgIUglsb2NhbE9ubHk=');

@$core.Deprecated('Use removeInstanceResponseDescriptor instead')
const RemoveInstanceResponse$json = {
  '1': 'RemoveInstanceResponse',
  '2': [
    {'1': 'task_id', '3': 1, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `RemoveInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeInstanceResponseDescriptor =
    $convert.base64Decode(
        'ChZSZW1vdmVJbnN0YW5jZVJlc3BvbnNlEhcKB3Rhc2tfaWQYASABKAlSBnRhc2tJZA==');

@$core.Deprecated('Use startInstanceRequestDescriptor instead')
const StartInstanceRequest$json = {
  '1': 'StartInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `StartInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startInstanceRequestDescriptor = $convert
    .base64Decode('ChRTdGFydEluc3RhbmNlUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1l');

@$core.Deprecated('Use startInstanceResponseDescriptor instead')
const StartInstanceResponse$json = {
  '1': 'StartInstanceResponse',
  '2': [
    {'1': 'task_id', '3': 1, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `StartInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startInstanceResponseDescriptor =
    $convert.base64Decode(
        'ChVTdGFydEluc3RhbmNlUmVzcG9uc2USFwoHdGFza19pZBgBIAEoCVIGdGFza0lk');

@$core.Deprecated('Use stopInstanceRequestDescriptor instead')
const StopInstanceRequest$json = {
  '1': 'StopInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `StopInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopInstanceRequestDescriptor = $convert
    .base64Decode('ChNTdG9wSW5zdGFuY2VSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use stopInstanceResponseDescriptor instead')
const StopInstanceResponse$json = {
  '1': 'StopInstanceResponse',
  '2': [
    {'1': 'task_id', '3': 1, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `StopInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopInstanceResponseDescriptor =
    $convert.base64Decode(
        'ChRTdG9wSW5zdGFuY2VSZXNwb25zZRIXCgd0YXNrX2lkGAEgASgJUgZ0YXNrSWQ=');

@$core.Deprecated('Use getInstanceKeyRequestDescriptor instead')
const GetInstanceKeyRequest$json = {
  '1': 'GetInstanceKeyRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetInstanceKeyRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceKeyRequestDescriptor =
    $convert.base64Decode(
        'ChVHZXRJbnN0YW5jZUtleVJlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use getInstanceKeyResponseDescriptor instead')
const GetInstanceKeyResponse$json = {
  '1': 'GetInstanceKeyResponse',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
  ],
};

/// Descriptor for `GetInstanceKeyResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceKeyResponseDescriptor = $convert
    .base64Decode('ChZHZXRJbnN0YW5jZUtleVJlc3BvbnNlEhAKA2tleRgBIAEoCVIDa2V5');

@$core.Deprecated('Use getInstanceLogsRequestDescriptor instead')
const GetInstanceLogsRequest$json = {
  '1': 'GetInstanceLogsRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetInstanceLogsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceLogsRequestDescriptor =
    $convert.base64Decode(
        'ChZHZXRJbnN0YW5jZUxvZ3NSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWU=');

@$core.Deprecated('Use getInstanceLogsResponseDescriptor instead')
const GetInstanceLogsResponse$json = {
  '1': 'GetInstanceLogsResponse',
  '2': [
    {'1': 'logs', '3': 1, '4': 1, '5': 9, '10': 'logs'},
  ],
};

/// Descriptor for `GetInstanceLogsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceLogsResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXRJbnN0YW5jZUxvZ3NSZXNwb25zZRISCgRsb2dzGAEgASgJUgRsb2dz');

@$core.Deprecated('Use initInstanceRequestDescriptor instead')
const InitInstanceRequest$json = {
  '1': 'InitInstanceRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'ip', '3': 2, '4': 1, '5': 9, '10': 'ip'},
  ],
};

/// Descriptor for `InitInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List initInstanceRequestDescriptor = $convert.base64Decode(
    'ChNJbml0SW5zdGFuY2VSZXF1ZXN0EhIKBG5hbWUYASABKAlSBG5hbWUSDgoCaXAYAiABKAlSAm'
    'lw');

@$core.Deprecated('Use initInstanceResponseDescriptor instead')
const InitInstanceResponse$json = {
  '1': 'InitInstanceResponse',
};

/// Descriptor for `InitInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List initInstanceResponseDescriptor =
    $convert.base64Decode('ChRJbml0SW5zdGFuY2VSZXNwb25zZQ==');

@$core.Deprecated('Use updateInstanceRequestDescriptor instead')
const UpdateInstanceRequest$json = {
  '1': 'UpdateInstanceRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'ip', '3': 2, '4': 1, '5': 9, '10': 'ip'},
  ],
};

/// Descriptor for `UpdateInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateInstanceRequestDescriptor = $convert.base64Decode(
    'ChVVcGRhdGVJbnN0YW5jZVJlcXVlc3QSDgoCaWQYASABKAlSAmlkEg4KAmlwGAIgASgJUgJpcA'
    '==');

@$core.Deprecated('Use updateInstanceResponseDescriptor instead')
const UpdateInstanceResponse$json = {
  '1': 'UpdateInstanceResponse',
};

/// Descriptor for `UpdateInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateInstanceResponseDescriptor =
    $convert.base64Decode('ChZVcGRhdGVJbnN0YW5jZVJlc3BvbnNl');

@$core.Deprecated('Use getNetworkStateRequestDescriptor instead')
const GetNetworkStateRequest$json = {
  '1': 'GetNetworkStateRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
  ],
};

/// Descriptor for `GetNetworkStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getNetworkStateRequestDescriptor =
    $convert.base64Decode(
        'ChZHZXROZXR3b3JrU3RhdGVSZXF1ZXN0EhoKCGluc3RhbmNlGAEgASgJUghpbnN0YW5jZQ==');

@$core.Deprecated('Use getNetworkStateResponseDescriptor instead')
const GetNetworkStateResponse$json = {
  '1': 'GetNetworkStateResponse',
  '2': [
    {
      '1': 'state',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.NetworkState',
      '10': 'state'
    },
  ],
};

/// Descriptor for `GetNetworkStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getNetworkStateResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXROZXR3b3JrU3RhdGVSZXNwb25zZRIoCgVzdGF0ZRgBIAEoCzISLmFwaWMuTmV0d29ya1'
        'N0YXRlUgVzdGF0ZQ==');

@$core.Deprecated('Use setNetworkEnabledRequestDescriptor instead')
const SetNetworkEnabledRequest$json = {
  '1': 'SetNetworkEnabledRequest',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `SetNetworkEnabledRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setNetworkEnabledRequestDescriptor =
    $convert.base64Decode(
        'ChhTZXROZXR3b3JrRW5hYmxlZFJlcXVlc3QSGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZA==');

@$core.Deprecated('Use setNetworkEnabledResponseDescriptor instead')
const SetNetworkEnabledResponse$json = {
  '1': 'SetNetworkEnabledResponse',
  '2': [
    {
      '1': 'status',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.NetworkRuntimeStatus',
      '10': 'status'
    },
  ],
};

/// Descriptor for `SetNetworkEnabledResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setNetworkEnabledResponseDescriptor =
    $convert.base64Decode(
        'ChlTZXROZXR3b3JrRW5hYmxlZFJlc3BvbnNlEjIKBnN0YXR1cxgBIAEoCzIaLmFwaWMuTmV0d2'
        '9ya1J1bnRpbWVTdGF0dXNSBnN0YXR1cw==');

@$core.Deprecated('Use networkRuntimeStatusDescriptor instead')
const NetworkRuntimeStatus$json = {
  '1': 'NetworkRuntimeStatus',
  '2': [
    {'1': 'supported', '3': 1, '4': 1, '5': 8, '10': 'supported'},
    {'1': 'desired_enabled', '3': 2, '4': 1, '5': 8, '10': 'desiredEnabled'},
    {'1': 'enabled', '3': 3, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'state', '3': 4, '4': 1, '5': 9, '10': 'state'},
    {'1': 'message', '3': 5, '4': 1, '5': 9, '10': 'message'},
    {
      '1': 'network_state',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.apic.NetworkState',
      '10': 'networkState'
    },
  ],
};

/// Descriptor for `NetworkRuntimeStatus`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkRuntimeStatusDescriptor = $convert.base64Decode(
    'ChROZXR3b3JrUnVudGltZVN0YXR1cxIcCglzdXBwb3J0ZWQYASABKAhSCXN1cHBvcnRlZBInCg'
    '9kZXNpcmVkX2VuYWJsZWQYAiABKAhSDmRlc2lyZWRFbmFibGVkEhgKB2VuYWJsZWQYAyABKAhS'
    'B2VuYWJsZWQSFAoFc3RhdGUYBCABKAlSBXN0YXRlEhgKB21lc3NhZ2UYBSABKAlSB21lc3NhZ2'
    'USNwoNbmV0d29ya19zdGF0ZRgGIAEoCzISLmFwaWMuTmV0d29ya1N0YXRlUgxuZXR3b3JrU3Rh'
    'dGU=');

@$core.Deprecated('Use networkStateDescriptor instead')
const NetworkState$json = {
  '1': 'NetworkState',
  '2': [
    {'1': 'module', '3': 1, '4': 1, '5': 9, '10': 'module'},
    {'1': 'up', '3': 2, '4': 1, '5': 8, '10': 'up'},
    {'1': 'interface_name', '3': 3, '4': 1, '5': 9, '10': 'interfaceName'},
    {
      '1': 'addresses',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.apic.NetworkAddress',
      '10': 'addresses'
    },
    {
      '1': 'routes',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.NetworkRoute',
      '10': 'routes'
    },
    {
      '1': 'wireguard_peers',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.apic.WireGuardPeer',
      '10': 'wireguardPeers'
    },
    {
      '1': 'firewall_tables',
      '3': 7,
      '4': 3,
      '5': 11,
      '6': '.apic.FirewallTable',
      '10': 'firewallTables'
    },
    {'1': 'dns', '3': 8, '4': 3, '5': 11, '6': '.apic.DNSState', '10': 'dns'},
    {'1': 'messages', '3': 9, '4': 3, '5': 9, '10': 'messages'},
    {
      '1': 'interfaces',
      '3': 10,
      '4': 3,
      '5': 11,
      '6': '.apic.NetworkInterface',
      '10': 'interfaces'
    },
  ],
};

/// Descriptor for `NetworkState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkStateDescriptor = $convert.base64Decode(
    'CgxOZXR3b3JrU3RhdGUSFgoGbW9kdWxlGAEgASgJUgZtb2R1bGUSDgoCdXAYAiABKAhSAnVwEi'
    'UKDmludGVyZmFjZV9uYW1lGAMgASgJUg1pbnRlcmZhY2VOYW1lEjIKCWFkZHJlc3NlcxgEIAMo'
    'CzIULmFwaWMuTmV0d29ya0FkZHJlc3NSCWFkZHJlc3NlcxIqCgZyb3V0ZXMYBSADKAsyEi5hcG'
    'ljLk5ldHdvcmtSb3V0ZVIGcm91dGVzEjwKD3dpcmVndWFyZF9wZWVycxgGIAMoCzITLmFwaWMu'
    'V2lyZUd1YXJkUGVlclIOd2lyZWd1YXJkUGVlcnMSPAoPZmlyZXdhbGxfdGFibGVzGAcgAygLMh'
    'MuYXBpYy5GaXJld2FsbFRhYmxlUg5maXJld2FsbFRhYmxlcxIgCgNkbnMYCCADKAsyDi5hcGlj'
    'LkROU1N0YXRlUgNkbnMSGgoIbWVzc2FnZXMYCSADKAlSCG1lc3NhZ2VzEjYKCmludGVyZmFjZX'
    'MYCiADKAsyFi5hcGljLk5ldHdvcmtJbnRlcmZhY2VSCmludGVyZmFjZXM=');

@$core.Deprecated('Use networkInterfaceDescriptor instead')
const NetworkInterface$json = {
  '1': 'NetworkInterface',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 2, '4': 1, '5': 9, '10': 'type'},
    {'1': 'index', '3': 3, '4': 1, '5': 5, '10': 'index'},
    {'1': 'mtu', '3': 4, '4': 1, '5': 5, '10': 'mtu'},
    {'1': 'up', '3': 5, '4': 1, '5': 8, '10': 'up'},
    {'1': 'master', '3': 6, '4': 1, '5': 9, '10': 'master'},
    {'1': 'mac_address', '3': 7, '4': 1, '5': 9, '10': 'macAddress'},
    {'1': 'kind', '3': 8, '4': 1, '5': 9, '10': 'kind'},
  ],
};

/// Descriptor for `NetworkInterface`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkInterfaceDescriptor = $convert.base64Decode(
    'ChBOZXR3b3JrSW50ZXJmYWNlEhIKBG5hbWUYASABKAlSBG5hbWUSEgoEdHlwZRgCIAEoCVIEdH'
    'lwZRIUCgVpbmRleBgDIAEoBVIFaW5kZXgSEAoDbXR1GAQgASgFUgNtdHUSDgoCdXAYBSABKAhS'
    'AnVwEhYKBm1hc3RlchgGIAEoCVIGbWFzdGVyEh8KC21hY19hZGRyZXNzGAcgASgJUgptYWNBZG'
    'RyZXNzEhIKBGtpbmQYCCABKAlSBGtpbmQ=');

@$core.Deprecated('Use networkAddressDescriptor instead')
const NetworkAddress$json = {
  '1': 'NetworkAddress',
  '2': [
    {'1': 'interface_name', '3': 1, '4': 1, '5': 9, '10': 'interfaceName'},
    {'1': 'cidr', '3': 2, '4': 1, '5': 9, '10': 'cidr'},
    {'1': 'scope', '3': 3, '4': 1, '5': 9, '10': 'scope'},
  ],
};

/// Descriptor for `NetworkAddress`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkAddressDescriptor = $convert.base64Decode(
    'Cg5OZXR3b3JrQWRkcmVzcxIlCg5pbnRlcmZhY2VfbmFtZRgBIAEoCVINaW50ZXJmYWNlTmFtZR'
    'ISCgRjaWRyGAIgASgJUgRjaWRyEhQKBXNjb3BlGAMgASgJUgVzY29wZQ==');

@$core.Deprecated('Use networkRouteDescriptor instead')
const NetworkRoute$json = {
  '1': 'NetworkRoute',
  '2': [
    {'1': 'interface_name', '3': 1, '4': 1, '5': 9, '10': 'interfaceName'},
    {'1': 'destination', '3': 2, '4': 1, '5': 9, '10': 'destination'},
    {'1': 'gateway', '3': 3, '4': 1, '5': 9, '10': 'gateway'},
    {'1': 'source', '3': 4, '4': 1, '5': 9, '10': 'source'},
    {'1': 'family', '3': 5, '4': 1, '5': 9, '10': 'family'},
    {'1': 'table', '3': 6, '4': 1, '5': 9, '10': 'table'},
    {'1': 'protocol', '3': 7, '4': 1, '5': 9, '10': 'protocol'},
    {'1': 'scope', '3': 8, '4': 1, '5': 9, '10': 'scope'},
    {'1': 'priority', '3': 9, '4': 1, '5': 9, '10': 'priority'},
    {'1': 'kind', '3': 10, '4': 1, '5': 9, '10': 'kind'},
  ],
};

/// Descriptor for `NetworkRoute`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkRouteDescriptor = $convert.base64Decode(
    'CgxOZXR3b3JrUm91dGUSJQoOaW50ZXJmYWNlX25hbWUYASABKAlSDWludGVyZmFjZU5hbWUSIA'
    'oLZGVzdGluYXRpb24YAiABKAlSC2Rlc3RpbmF0aW9uEhgKB2dhdGV3YXkYAyABKAlSB2dhdGV3'
    'YXkSFgoGc291cmNlGAQgASgJUgZzb3VyY2USFgoGZmFtaWx5GAUgASgJUgZmYW1pbHkSFAoFdG'
    'FibGUYBiABKAlSBXRhYmxlEhoKCHByb3RvY29sGAcgASgJUghwcm90b2NvbBIUCgVzY29wZRgI'
    'IAEoCVIFc2NvcGUSGgoIcHJpb3JpdHkYCSABKAlSCHByaW9yaXR5EhIKBGtpbmQYCiABKAlSBG'
    'tpbmQ=');

@$core.Deprecated('Use wireGuardPeerDescriptor instead')
const WireGuardPeer$json = {
  '1': 'WireGuardPeer',
  '2': [
    {'1': 'public_key', '3': 1, '4': 1, '5': 9, '10': 'publicKey'},
    {'1': 'endpoint', '3': 2, '4': 1, '5': 9, '10': 'endpoint'},
    {'1': 'allowed_ips', '3': 3, '4': 3, '5': 9, '10': 'allowedIps'},
    {'1': 'latest_handshake', '3': 4, '4': 1, '5': 9, '10': 'latestHandshake'},
    {'1': 'rx_bytes', '3': 5, '4': 1, '5': 4, '10': 'rxBytes'},
    {'1': 'tx_bytes', '3': 6, '4': 1, '5': 4, '10': 'txBytes'},
  ],
};

/// Descriptor for `WireGuardPeer`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wireGuardPeerDescriptor = $convert.base64Decode(
    'Cg1XaXJlR3VhcmRQZWVyEh0KCnB1YmxpY19rZXkYASABKAlSCXB1YmxpY0tleRIaCghlbmRwb2'
    'ludBgCIAEoCVIIZW5kcG9pbnQSHwoLYWxsb3dlZF9pcHMYAyADKAlSCmFsbG93ZWRJcHMSKQoQ'
    'bGF0ZXN0X2hhbmRzaGFrZRgEIAEoCVIPbGF0ZXN0SGFuZHNoYWtlEhkKCHJ4X2J5dGVzGAUgAS'
    'gEUgdyeEJ5dGVzEhkKCHR4X2J5dGVzGAYgASgEUgd0eEJ5dGVz');

@$core.Deprecated('Use firewallTableDescriptor instead')
const FirewallTable$json = {
  '1': 'FirewallTable',
  '2': [
    {'1': 'family', '3': 1, '4': 1, '5': 9, '10': 'family'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'chains',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.FirewallChain',
      '10': 'chains'
    },
  ],
};

/// Descriptor for `FirewallTable`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List firewallTableDescriptor = $convert.base64Decode(
    'Cg1GaXJld2FsbFRhYmxlEhYKBmZhbWlseRgBIAEoCVIGZmFtaWx5EhIKBG5hbWUYAiABKAlSBG'
    '5hbWUSKwoGY2hhaW5zGAMgAygLMhMuYXBpYy5GaXJld2FsbENoYWluUgZjaGFpbnM=');

@$core.Deprecated('Use firewallChainDescriptor instead')
const FirewallChain$json = {
  '1': 'FirewallChain',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'type', '3': 2, '4': 1, '5': 9, '10': 'type'},
    {'1': 'hook', '3': 3, '4': 1, '5': 9, '10': 'hook'},
    {'1': 'priority', '3': 4, '4': 1, '5': 9, '10': 'priority'},
    {
      '1': 'rules',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.FirewallRule',
      '10': 'rules'
    },
  ],
};

/// Descriptor for `FirewallChain`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List firewallChainDescriptor = $convert.base64Decode(
    'Cg1GaXJld2FsbENoYWluEhIKBG5hbWUYASABKAlSBG5hbWUSEgoEdHlwZRgCIAEoCVIEdHlwZR'
    'ISCgRob29rGAMgASgJUgRob29rEhoKCHByaW9yaXR5GAQgASgJUghwcmlvcml0eRIoCgVydWxl'
    'cxgFIAMoCzISLmFwaWMuRmlyZXdhbGxSdWxlUgVydWxlcw==');

@$core.Deprecated('Use firewallRuleDescriptor instead')
const FirewallRule$json = {
  '1': 'FirewallRule',
  '2': [
    {'1': 'expressions', '3': 1, '4': 3, '5': 9, '10': 'expressions'},
    {'1': 'packets', '3': 2, '4': 1, '5': 4, '10': 'packets'},
    {'1': 'bytes', '3': 3, '4': 1, '5': 4, '10': 'bytes'},
  ],
};

/// Descriptor for `FirewallRule`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List firewallRuleDescriptor = $convert.base64Decode(
    'CgxGaXJld2FsbFJ1bGUSIAoLZXhwcmVzc2lvbnMYASADKAlSC2V4cHJlc3Npb25zEhgKB3BhY2'
    'tldHMYAiABKARSB3BhY2tldHMSFAoFYnl0ZXMYAyABKARSBWJ5dGVz');

@$core.Deprecated('Use dNSStateDescriptor instead')
const DNSState$json = {
  '1': 'DNSState',
  '2': [
    {'1': 'scope', '3': 1, '4': 1, '5': 9, '10': 'scope'},
    {'1': 'domain', '3': 2, '4': 1, '5': 9, '10': 'domain'},
    {'1': 'servers', '3': 3, '4': 3, '5': 9, '10': 'servers'},
    {'1': 'port', '3': 4, '4': 1, '5': 5, '10': 'port'},
    {'1': 'active', '3': 5, '4': 1, '5': 8, '10': 'active'},
    {'1': 'source', '3': 6, '4': 1, '5': 9, '10': 'source'},
  ],
};

/// Descriptor for `DNSState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List dNSStateDescriptor = $convert.base64Decode(
    'CghETlNTdGF0ZRIUCgVzY29wZRgBIAEoCVIFc2NvcGUSFgoGZG9tYWluGAIgASgJUgZkb21haW'
    '4SGAoHc2VydmVycxgDIAMoCVIHc2VydmVycxISCgRwb3J0GAQgASgFUgRwb3J0EhYKBmFjdGl2'
    'ZRgFIAEoCFIGYWN0aXZlEhYKBnNvdXJjZRgGIAEoCVIGc291cmNl');

@$core.Deprecated('Use exitRouteDescriptor instead')
const ExitRoute$json = {
  '1': 'ExitRoute',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'instance_id', '3': 3, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'instance_name', '3': 4, '4': 1, '5': 9, '10': 'instanceName'},
    {'1': 'public_ip', '3': 5, '4': 1, '5': 9, '10': 'publicIp'},
    {'1': 'location', '3': 6, '4': 1, '5': 9, '10': 'location'},
    {'1': 'status', '3': 7, '4': 1, '5': 9, '10': 'status'},
    {'1': 'dns_server', '3': 8, '4': 1, '5': 9, '10': 'dnsServer'},
    {'1': 'cidrs', '3': 9, '4': 3, '5': 9, '10': 'cidrs'},
  ],
};

/// Descriptor for `ExitRoute`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List exitRouteDescriptor = $convert.base64Decode(
    'CglFeGl0Um91dGUSDgoCaWQYASABKAlSAmlkEhsKCWRldmljZV9pZBgCIAEoCVIIZGV2aWNlSW'
    'QSHwoLaW5zdGFuY2VfaWQYAyABKAlSCmluc3RhbmNlSWQSIwoNaW5zdGFuY2VfbmFtZRgEIAEo'
    'CVIMaW5zdGFuY2VOYW1lEhsKCXB1YmxpY19pcBgFIAEoCVIIcHVibGljSXASGgoIbG9jYXRpb2'
    '4YBiABKAlSCGxvY2F0aW9uEhYKBnN0YXR1cxgHIAEoCVIGc3RhdHVzEh0KCmRuc19zZXJ2ZXIY'
    'CCABKAlSCWRuc1NlcnZlchIUCgVjaWRycxgJIAMoCVIFY2lkcnM=');

@$core.Deprecated('Use getExitRoutesRequestDescriptor instead')
const GetExitRoutesRequest$json = {
  '1': 'GetExitRoutesRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
  ],
};

/// Descriptor for `GetExitRoutesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getExitRoutesRequestDescriptor =
    $convert.base64Decode(
        'ChRHZXRFeGl0Um91dGVzUmVxdWVzdBIaCghpbnN0YW5jZRgBIAEoCVIIaW5zdGFuY2U=');

@$core.Deprecated('Use getExitRoutesResponseDescriptor instead')
const GetExitRoutesResponse$json = {
  '1': 'GetExitRoutesResponse',
  '2': [
    {
      '1': 'routes',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.ExitRoute',
      '10': 'routes'
    },
  ],
};

/// Descriptor for `GetExitRoutesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getExitRoutesResponseDescriptor = $convert.base64Decode(
    'ChVHZXRFeGl0Um91dGVzUmVzcG9uc2USJwoGcm91dGVzGAEgAygLMg8uYXBpYy5FeGl0Um91dG'
    'VSBnJvdXRlcw==');

@$core.Deprecated('Use getMobileTunnelConfigRequestDescriptor instead')
const GetMobileTunnelConfigRequest$json = {
  '1': 'GetMobileTunnelConfigRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'dns_server', '3': 3, '4': 1, '5': 9, '10': 'dnsServer'},
    {'1': 'cidrs', '3': 4, '4': 3, '5': 9, '10': 'cidrs'},
  ],
};

/// Descriptor for `GetMobileTunnelConfigRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMobileTunnelConfigRequestDescriptor =
    $convert.base64Decode(
        'ChxHZXRNb2JpbGVUdW5uZWxDb25maWdSZXF1ZXN0EhoKCGluc3RhbmNlGAEgASgJUghpbnN0YW'
        '5jZRIbCglkZXZpY2VfaWQYAiABKAlSCGRldmljZUlkEh0KCmRuc19zZXJ2ZXIYAyABKAlSCWRu'
        'c1NlcnZlchIUCgVjaWRycxgEIAMoCVIFY2lkcnM=');

@$core.Deprecated('Use mobileTunnelConfigDescriptor instead')
const MobileTunnelConfig$json = {
  '1': 'MobileTunnelConfig',
  '2': [
    {'1': 'config_id', '3': 1, '4': 1, '5': 9, '10': 'configId'},
    {'1': 'generated_at_unix', '3': 2, '4': 1, '5': 3, '10': 'generatedAtUnix'},
    {'1': 'instance_id', '3': 3, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'instance_name', '3': 4, '4': 1, '5': 9, '10': 'instanceName'},
    {'1': 'peer_public_key', '3': 5, '4': 1, '5': 9, '10': 'peerPublicKey'},
    {'1': 'peer_endpoint', '3': 6, '4': 1, '5': 9, '10': 'peerEndpoint'},
    {
      '1': 'interface_addresses',
      '3': 7,
      '4': 3,
      '5': 9,
      '10': 'interfaceAddresses'
    },
    {'1': 'dns_servers', '3': 8, '4': 3, '5': 9, '10': 'dnsServers'},
    {'1': 'included_routes', '3': 9, '4': 3, '5': 9, '10': 'includedRoutes'},
    {'1': 'excluded_routes', '3': 10, '4': 3, '5': 9, '10': 'excludedRoutes'},
    {'1': 'mtu', '3': 11, '4': 1, '5': 5, '10': 'mtu'},
    {'1': 'allowed_ips', '3': 12, '4': 3, '5': 9, '10': 'allowedIps'},
    {
      '1': 'persistent_keepalive_seconds',
      '3': 13,
      '4': 1,
      '5': 5,
      '10': 'persistentKeepaliveSeconds'
    },
    {
      '1': 'keychain_access_group',
      '3': 14,
      '4': 1,
      '5': 9,
      '10': 'keychainAccessGroup'
    },
    {'1': 'keychain_account', '3': 15, '4': 1, '5': 9, '10': 'keychainAccount'},
    {
      '1': 'wireguard_private_key',
      '3': 16,
      '4': 1,
      '5': 9,
      '10': 'wireguardPrivateKey'
    },
  ],
};

/// Descriptor for `MobileTunnelConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List mobileTunnelConfigDescriptor = $convert.base64Decode(
    'ChJNb2JpbGVUdW5uZWxDb25maWcSGwoJY29uZmlnX2lkGAEgASgJUghjb25maWdJZBIqChFnZW'
    '5lcmF0ZWRfYXRfdW5peBgCIAEoA1IPZ2VuZXJhdGVkQXRVbml4Eh8KC2luc3RhbmNlX2lkGAMg'
    'ASgJUgppbnN0YW5jZUlkEiMKDWluc3RhbmNlX25hbWUYBCABKAlSDGluc3RhbmNlTmFtZRImCg'
    '9wZWVyX3B1YmxpY19rZXkYBSABKAlSDXBlZXJQdWJsaWNLZXkSIwoNcGVlcl9lbmRwb2ludBgG'
    'IAEoCVIMcGVlckVuZHBvaW50Ei8KE2ludGVyZmFjZV9hZGRyZXNzZXMYByADKAlSEmludGVyZm'
    'FjZUFkZHJlc3NlcxIfCgtkbnNfc2VydmVycxgIIAMoCVIKZG5zU2VydmVycxInCg9pbmNsdWRl'
    'ZF9yb3V0ZXMYCSADKAlSDmluY2x1ZGVkUm91dGVzEicKD2V4Y2x1ZGVkX3JvdXRlcxgKIAMoCV'
    'IOZXhjbHVkZWRSb3V0ZXMSEAoDbXR1GAsgASgFUgNtdHUSHwoLYWxsb3dlZF9pcHMYDCADKAlS'
    'CmFsbG93ZWRJcHMSQAoccGVyc2lzdGVudF9rZWVwYWxpdmVfc2Vjb25kcxgNIAEoBVIacGVyc2'
    'lzdGVudEtlZXBhbGl2ZVNlY29uZHMSMgoVa2V5Y2hhaW5fYWNjZXNzX2dyb3VwGA4gASgJUhNr'
    'ZXljaGFpbkFjY2Vzc0dyb3VwEikKEGtleWNoYWluX2FjY291bnQYDyABKAlSD2tleWNoYWluQW'
    'Njb3VudBIyChV3aXJlZ3VhcmRfcHJpdmF0ZV9rZXkYECABKAlSE3dpcmVndWFyZFByaXZhdGVL'
    'ZXk=');

@$core.Deprecated('Use getMobileTunnelConfigResponseDescriptor instead')
const GetMobileTunnelConfigResponse$json = {
  '1': 'GetMobileTunnelConfigResponse',
  '2': [
    {
      '1': 'config',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.MobileTunnelConfig',
      '10': 'config'
    },
  ],
};

/// Descriptor for `GetMobileTunnelConfigResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMobileTunnelConfigResponseDescriptor =
    $convert.base64Decode(
        'Ch1HZXRNb2JpbGVUdW5uZWxDb25maWdSZXNwb25zZRIwCgZjb25maWcYASABKAsyGC5hcGljLk'
        '1vYmlsZVR1bm5lbENvbmZpZ1IGY29uZmln');

@$core.Deprecated('Use getRuntimeStateRequestDescriptor instead')
const GetRuntimeStateRequest$json = {
  '1': 'GetRuntimeStateRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
    {'1': 'allow_stale', '3': 2, '4': 1, '5': 8, '10': 'allowStale'},
  ],
};

/// Descriptor for `GetRuntimeStateRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getRuntimeStateRequestDescriptor =
    $convert.base64Decode(
        'ChZHZXRSdW50aW1lU3RhdGVSZXF1ZXN0EhoKCGluc3RhbmNlGAEgASgJUghpbnN0YW5jZRIfCg'
        'thbGxvd19zdGFsZRgCIAEoCFIKYWxsb3dTdGFsZQ==');

@$core.Deprecated('Use getRuntimeStateResponseDescriptor instead')
const GetRuntimeStateResponse$json = {
  '1': 'GetRuntimeStateResponse',
  '2': [
    {
      '1': 'state',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.RuntimeState',
      '10': 'state'
    },
  ],
};

/// Descriptor for `GetRuntimeStateResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getRuntimeStateResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXRSdW50aW1lU3RhdGVSZXNwb25zZRIoCgVzdGF0ZRgBIAEoCzISLmFwaWMuUnVudGltZV'
        'N0YXRlUgVzdGF0ZQ==');

@$core.Deprecated('Use watchChangesRequestDescriptor instead')
const WatchChangesRequest$json = {
  '1': 'WatchChangesRequest',
  '2': [
    {'1': 'include_snapshot', '3': 1, '4': 1, '5': 8, '10': 'includeSnapshot'},
    {
      '1': 'heartbeat_interval_ms',
      '3': 2,
      '4': 1,
      '5': 13,
      '10': 'heartbeatIntervalMs'
    },
  ],
};

/// Descriptor for `WatchChangesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List watchChangesRequestDescriptor = $convert.base64Decode(
    'ChNXYXRjaENoYW5nZXNSZXF1ZXN0EikKEGluY2x1ZGVfc25hcHNob3QYASABKAhSD2luY2x1ZG'
    'VTbmFwc2hvdBIyChVoZWFydGJlYXRfaW50ZXJ2YWxfbXMYAiABKA1SE2hlYXJ0YmVhdEludGVy'
    'dmFsTXM=');

@$core.Deprecated('Use watchChangesResponseDescriptor instead')
const WatchChangesResponse$json = {
  '1': 'WatchChangesResponse',
  '2': [
    {'1': 'sequence', '3': 1, '4': 1, '5': 4, '10': 'sequence'},
    {'1': 'table_names', '3': 2, '4': 3, '5': 9, '10': 'tableNames'},
    {'1': 'runtime_changed', '3': 3, '4': 1, '5': 8, '10': 'runtimeChanged'},
    {'1': 'reason', '3': 4, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `WatchChangesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List watchChangesResponseDescriptor = $convert.base64Decode(
    'ChRXYXRjaENoYW5nZXNSZXNwb25zZRIaCghzZXF1ZW5jZRgBIAEoBFIIc2VxdWVuY2USHwoLdG'
    'FibGVfbmFtZXMYAiADKAlSCnRhYmxlTmFtZXMSJwoPcnVudGltZV9jaGFuZ2VkGAMgASgIUg5y'
    'dW50aW1lQ2hhbmdlZBIWCgZyZWFzb24YBCABKAlSBnJlYXNvbg==');

@$core.Deprecated('Use taskDescriptor instead')
const Task$json = {
  '1': 'Task',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'stream', '3': 2, '4': 1, '5': 9, '10': 'stream'},
    {'1': 'subject_type', '3': 3, '4': 1, '5': 9, '10': 'subjectType'},
    {'1': 'subject_id', '3': 4, '4': 1, '5': 9, '10': 'subjectId'},
    {'1': 'status', '3': 5, '4': 1, '5': 9, '10': 'status'},
    {'1': 'title', '3': 6, '4': 1, '5': 9, '10': 'title'},
    {'1': 'message', '3': 7, '4': 1, '5': 9, '10': 'message'},
    {'1': 'progress', '3': 8, '4': 1, '5': 5, '10': 'progress'},
    {'1': 'payload_json', '3': 9, '4': 1, '5': 9, '10': 'payloadJson'},
    {'1': 'result_json', '3': 10, '4': 1, '5': 9, '10': 'resultJson'},
    {'1': 'error_message', '3': 11, '4': 1, '5': 9, '10': 'errorMessage'},
    {'1': 'attempts', '3': 12, '4': 1, '5': 5, '10': 'attempts'},
    {'1': 'max_attempts', '3': 13, '4': 1, '5': 5, '10': 'maxAttempts'},
    {'1': 'created_at', '3': 14, '4': 1, '5': 9, '10': 'createdAt'},
    {'1': 'updated_at', '3': 15, '4': 1, '5': 9, '10': 'updatedAt'},
    {'1': 'started_at', '3': 16, '4': 1, '5': 9, '10': 'startedAt'},
    {'1': 'finished_at', '3': 17, '4': 1, '5': 9, '10': 'finishedAt'},
    {'1': 'owner_peer_id', '3': 18, '4': 1, '5': 9, '10': 'ownerPeerId'},
  ],
};

/// Descriptor for `Task`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List taskDescriptor = $convert.base64Decode(
    'CgRUYXNrEg4KAmlkGAEgASgJUgJpZBIWCgZzdHJlYW0YAiABKAlSBnN0cmVhbRIhCgxzdWJqZW'
    'N0X3R5cGUYAyABKAlSC3N1YmplY3RUeXBlEh0KCnN1YmplY3RfaWQYBCABKAlSCXN1YmplY3RJ'
    'ZBIWCgZzdGF0dXMYBSABKAlSBnN0YXR1cxIUCgV0aXRsZRgGIAEoCVIFdGl0bGUSGAoHbWVzc2'
    'FnZRgHIAEoCVIHbWVzc2FnZRIaCghwcm9ncmVzcxgIIAEoBVIIcHJvZ3Jlc3MSIQoMcGF5bG9h'
    'ZF9qc29uGAkgASgJUgtwYXlsb2FkSnNvbhIfCgtyZXN1bHRfanNvbhgKIAEoCVIKcmVzdWx0Sn'
    'NvbhIjCg1lcnJvcl9tZXNzYWdlGAsgASgJUgxlcnJvck1lc3NhZ2USGgoIYXR0ZW1wdHMYDCAB'
    'KAVSCGF0dGVtcHRzEiEKDG1heF9hdHRlbXB0cxgNIAEoBVILbWF4QXR0ZW1wdHMSHQoKY3JlYX'
    'RlZF9hdBgOIAEoCVIJY3JlYXRlZEF0Eh0KCnVwZGF0ZWRfYXQYDyABKAlSCXVwZGF0ZWRBdBId'
    'CgpzdGFydGVkX2F0GBAgASgJUglzdGFydGVkQXQSHwoLZmluaXNoZWRfYXQYESABKAlSCmZpbm'
    'lzaGVkQXQSIgoNb3duZXJfcGVlcl9pZBgSIAEoCVILb3duZXJQZWVySWQ=');

@$core.Deprecated('Use taskEventDescriptor instead')
const TaskEvent$json = {
  '1': 'TaskEvent',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'task_id', '3': 2, '4': 1, '5': 9, '10': 'taskId'},
    {'1': 'status', '3': 3, '4': 1, '5': 9, '10': 'status'},
    {'1': 'message', '3': 4, '4': 1, '5': 9, '10': 'message'},
    {'1': 'progress', '3': 5, '4': 1, '5': 5, '10': 'progress'},
    {'1': 'details_json', '3': 6, '4': 1, '5': 9, '10': 'detailsJson'},
    {'1': 'created_at', '3': 7, '4': 1, '5': 9, '10': 'createdAt'},
  ],
};

/// Descriptor for `TaskEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List taskEventDescriptor = $convert.base64Decode(
    'CglUYXNrRXZlbnQSDgoCaWQYASABKAlSAmlkEhcKB3Rhc2tfaWQYAiABKAlSBnRhc2tJZBIWCg'
    'ZzdGF0dXMYAyABKAlSBnN0YXR1cxIYCgdtZXNzYWdlGAQgASgJUgdtZXNzYWdlEhoKCHByb2dy'
    'ZXNzGAUgASgFUghwcm9ncmVzcxIhCgxkZXRhaWxzX2pzb24YBiABKAlSC2RldGFpbHNKc29uEh'
    '0KCmNyZWF0ZWRfYXQYByABKAlSCWNyZWF0ZWRBdA==');

@$core.Deprecated('Use getTasksRequestDescriptor instead')
const GetTasksRequest$json = {
  '1': 'GetTasksRequest',
  '2': [
    {'1': 'status', '3': 1, '4': 1, '5': 9, '10': 'status'},
    {'1': 'stream', '3': 2, '4': 1, '5': 9, '10': 'stream'},
    {'1': 'subject_type', '3': 3, '4': 1, '5': 9, '10': 'subjectType'},
    {'1': 'subject_id', '3': 4, '4': 1, '5': 9, '10': 'subjectId'},
    {'1': 'max_results', '3': 5, '4': 1, '5': 5, '10': 'maxResults'},
    {'1': 'instance', '3': 6, '4': 1, '5': 9, '10': 'instance'},
  ],
};

/// Descriptor for `GetTasksRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTasksRequestDescriptor = $convert.base64Decode(
    'Cg9HZXRUYXNrc1JlcXVlc3QSFgoGc3RhdHVzGAEgASgJUgZzdGF0dXMSFgoGc3RyZWFtGAIgAS'
    'gJUgZzdHJlYW0SIQoMc3ViamVjdF90eXBlGAMgASgJUgtzdWJqZWN0VHlwZRIdCgpzdWJqZWN0'
    'X2lkGAQgASgJUglzdWJqZWN0SWQSHwoLbWF4X3Jlc3VsdHMYBSABKAVSCm1heFJlc3VsdHMSGg'
    'oIaW5zdGFuY2UYBiABKAlSCGluc3RhbmNl');

@$core.Deprecated('Use getTasksResponseDescriptor instead')
const GetTasksResponse$json = {
  '1': 'GetTasksResponse',
  '2': [
    {'1': 'tasks', '3': 1, '4': 3, '5': 11, '6': '.apic.Task', '10': 'tasks'},
    {'1': 'truncated', '3': 2, '4': 1, '5': 8, '10': 'truncated'},
  ],
};

/// Descriptor for `GetTasksResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTasksResponseDescriptor = $convert.base64Decode(
    'ChBHZXRUYXNrc1Jlc3BvbnNlEiAKBXRhc2tzGAEgAygLMgouYXBpYy5UYXNrUgV0YXNrcxIcCg'
    'l0cnVuY2F0ZWQYAiABKAhSCXRydW5jYXRlZA==');

@$core.Deprecated('Use getTaskRequestDescriptor instead')
const GetTaskRequest$json = {
  '1': 'GetTaskRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'include_events', '3': 2, '4': 1, '5': 8, '10': 'includeEvents'},
    {'1': 'instance', '3': 3, '4': 1, '5': 9, '10': 'instance'},
  ],
};

/// Descriptor for `GetTaskRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTaskRequestDescriptor = $convert.base64Decode(
    'Cg5HZXRUYXNrUmVxdWVzdBIOCgJpZBgBIAEoCVICaWQSJQoOaW5jbHVkZV9ldmVudHMYAiABKA'
    'hSDWluY2x1ZGVFdmVudHMSGgoIaW5zdGFuY2UYAyABKAlSCGluc3RhbmNl');

@$core.Deprecated('Use getTaskResponseDescriptor instead')
const GetTaskResponse$json = {
  '1': 'GetTaskResponse',
  '2': [
    {'1': 'task', '3': 1, '4': 1, '5': 11, '6': '.apic.Task', '10': 'task'},
    {
      '1': 'events',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.apic.TaskEvent',
      '10': 'events'
    },
  ],
};

/// Descriptor for `GetTaskResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getTaskResponseDescriptor = $convert.base64Decode(
    'Cg9HZXRUYXNrUmVzcG9uc2USHgoEdGFzaxgBIAEoCzIKLmFwaWMuVGFza1IEdGFzaxInCgZldm'
    'VudHMYAiADKAsyDy5hcGljLlRhc2tFdmVudFIGZXZlbnRz');

@$core.Deprecated('Use taskProgressUpdateDescriptor instead')
const TaskProgressUpdate$json = {
  '1': 'TaskProgressUpdate',
  '2': [
    {'1': 'task_id', '3': 1, '4': 1, '5': 9, '10': 'taskId'},
    {'1': 'status', '3': 2, '4': 1, '5': 9, '10': 'status'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
    {'1': 'progress', '3': 4, '4': 1, '5': 5, '10': 'progress'},
    {'1': 'details_json', '3': 5, '4': 1, '5': 9, '10': 'detailsJson'},
    {'1': 'created_at', '3': 6, '4': 1, '5': 9, '10': 'createdAt'},
    {'1': 'durable', '3': 7, '4': 1, '5': 8, '10': 'durable'},
  ],
};

/// Descriptor for `TaskProgressUpdate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List taskProgressUpdateDescriptor = $convert.base64Decode(
    'ChJUYXNrUHJvZ3Jlc3NVcGRhdGUSFwoHdGFza19pZBgBIAEoCVIGdGFza0lkEhYKBnN0YXR1cx'
    'gCIAEoCVIGc3RhdHVzEhgKB21lc3NhZ2UYAyABKAlSB21lc3NhZ2USGgoIcHJvZ3Jlc3MYBCAB'
    'KAVSCHByb2dyZXNzEiEKDGRldGFpbHNfanNvbhgFIAEoCVILZGV0YWlsc0pzb24SHQoKY3JlYX'
    'RlZF9hdBgGIAEoCVIJY3JlYXRlZEF0EhgKB2R1cmFibGUYByABKAhSB2R1cmFibGU=');

@$core.Deprecated('Use watchTaskRequestDescriptor instead')
const WatchTaskRequest$json = {
  '1': 'WatchTaskRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'include_snapshot', '3': 2, '4': 1, '5': 8, '10': 'includeSnapshot'},
    {'1': 'include_events', '3': 3, '4': 1, '5': 8, '10': 'includeEvents'},
    {
      '1': 'heartbeat_interval_ms',
      '3': 4,
      '4': 1,
      '5': 13,
      '10': 'heartbeatIntervalMs'
    },
  ],
};

/// Descriptor for `WatchTaskRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List watchTaskRequestDescriptor = $convert.base64Decode(
    'ChBXYXRjaFRhc2tSZXF1ZXN0Eg4KAmlkGAEgASgJUgJpZBIpChBpbmNsdWRlX3NuYXBzaG90GA'
    'IgASgIUg9pbmNsdWRlU25hcHNob3QSJQoOaW5jbHVkZV9ldmVudHMYAyABKAhSDWluY2x1ZGVF'
    'dmVudHMSMgoVaGVhcnRiZWF0X2ludGVydmFsX21zGAQgASgNUhNoZWFydGJlYXRJbnRlcnZhbE'
    '1z');

@$core.Deprecated('Use watchTaskResponseDescriptor instead')
const WatchTaskResponse$json = {
  '1': 'WatchTaskResponse',
  '2': [
    {'1': 'sequence', '3': 1, '4': 1, '5': 4, '10': 'sequence'},
    {'1': 'task', '3': 2, '4': 1, '5': 11, '6': '.apic.Task', '10': 'task'},
    {
      '1': 'events',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.TaskEvent',
      '10': 'events'
    },
    {
      '1': 'update',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.apic.TaskProgressUpdate',
      '10': 'update'
    },
    {'1': 'heartbeat', '3': 5, '4': 1, '5': 8, '10': 'heartbeat'},
  ],
};

/// Descriptor for `WatchTaskResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List watchTaskResponseDescriptor = $convert.base64Decode(
    'ChFXYXRjaFRhc2tSZXNwb25zZRIaCghzZXF1ZW5jZRgBIAEoBFIIc2VxdWVuY2USHgoEdGFzax'
    'gCIAEoCzIKLmFwaWMuVGFza1IEdGFzaxInCgZldmVudHMYAyADKAsyDy5hcGljLlRhc2tFdmVu'
    'dFIGZXZlbnRzEjAKBnVwZGF0ZRgEIAEoCzIYLmFwaWMuVGFza1Byb2dyZXNzVXBkYXRlUgZ1cG'
    'RhdGUSHAoJaGVhcnRiZWF0GAUgASgIUgloZWFydGJlYXQ=');

@$core.Deprecated('Use setExitRouteRequestDescriptor instead')
const SetExitRouteRequest$json = {
  '1': 'SetExitRouteRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'dns_server', '3': 3, '4': 1, '5': 9, '10': 'dnsServer'},
    {'1': 'cidrs', '3': 4, '4': 3, '5': 9, '10': 'cidrs'},
  ],
};

/// Descriptor for `SetExitRouteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setExitRouteRequestDescriptor = $convert.base64Decode(
    'ChNTZXRFeGl0Um91dGVSZXF1ZXN0EhoKCGluc3RhbmNlGAEgASgJUghpbnN0YW5jZRIbCglkZX'
    'ZpY2VfaWQYAiABKAlSCGRldmljZUlkEh0KCmRuc19zZXJ2ZXIYAyABKAlSCWRuc1NlcnZlchIU'
    'CgVjaWRycxgEIAMoCVIFY2lkcnM=');

@$core.Deprecated('Use setExitRouteResponseDescriptor instead')
const SetExitRouteResponse$json = {
  '1': 'SetExitRouteResponse',
  '2': [
    {
      '1': 'route',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.ExitRoute',
      '10': 'route'
    },
  ],
};

/// Descriptor for `SetExitRouteResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setExitRouteResponseDescriptor = $convert.base64Decode(
    'ChRTZXRFeGl0Um91dGVSZXNwb25zZRIlCgVyb3V0ZRgBIAEoCzIPLmFwaWMuRXhpdFJvdXRlUg'
    'Vyb3V0ZQ==');

@$core.Deprecated('Use clearExitRouteRequestDescriptor instead')
const ClearExitRouteRequest$json = {
  '1': 'ClearExitRouteRequest',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
  ],
};

/// Descriptor for `ClearExitRouteRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clearExitRouteRequestDescriptor = $convert.base64Decode(
    'ChVDbGVhckV4aXRSb3V0ZVJlcXVlc3QSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZA==');

@$core.Deprecated('Use clearExitRouteResponseDescriptor instead')
const ClearExitRouteResponse$json = {
  '1': 'ClearExitRouteResponse',
};

/// Descriptor for `ClearExitRouteResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clearExitRouteResponseDescriptor =
    $convert.base64Decode('ChZDbGVhckV4aXRSb3V0ZVJlc3BvbnNl');

@$core.Deprecated('Use runtimeStateDescriptor instead')
const RuntimeState$json = {
  '1': 'RuntimeState',
  '2': [
    {'1': 'peer_id', '3': 1, '4': 1, '5': 9, '10': 'peerId'},
    {'1': 'manifest_digest', '3': 2, '4': 1, '5': 9, '10': 'manifestDigest'},
    {
      '1': 'checkpoint_root_hash',
      '3': 3,
      '4': 1,
      '5': 9,
      '10': 'checkpointRootHash'
    },
    {
      '1': 'tentative_root_hash',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'tentativeRootHash'
    },
    {
      '1': 'protocol_checkpoint_root_hash',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'protocolCheckpointRootHash'
    },
    {
      '1': 'durable_main_root_hash',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'durableMainRootHash'
    },
    {'1': 'state_providers', '3': 10, '4': 3, '5': 9, '10': 'stateProviders'},
    {'1': 'connected_peers', '3': 11, '4': 3, '5': 9, '10': 'connectedPeers'},
    {'1': 'fatal_state', '3': 12, '4': 1, '5': 9, '10': 'fatalState'},
    {
      '1': 'runtime_refresh_pending',
      '3': 13,
      '4': 1,
      '5': 8,
      '10': 'runtimeRefreshPending'
    },
    {
      '1': 'runtime_refresh_last_error',
      '3': 14,
      '4': 1,
      '5': 9,
      '10': 'runtimeRefreshLastError'
    },
    {
      '1': 'runtime_checkpoint_pending',
      '3': 15,
      '4': 1,
      '5': 8,
      '10': 'runtimeCheckpointPending'
    },
    {
      '1': 'runtime_checkpoint_last_error',
      '3': 16,
      '4': 1,
      '5': 9,
      '10': 'runtimeCheckpointLastError'
    },
    {
      '1': 'runtime_materialization_policy',
      '3': 17,
      '4': 1,
      '5': 9,
      '10': 'runtimeMaterializationPolicy'
    },
    {
      '1': 'peer_statuses',
      '3': 18,
      '4': 3,
      '5': 11,
      '6': '.apic.RuntimePeerStatus',
      '10': 'peerStatuses'
    },
    {
      '1': 'compatibility',
      '3': 19,
      '4': 3,
      '5': 11,
      '6': '.apic.RuntimeCompatibility',
      '10': 'compatibility'
    },
    {
      '1': 'content_sync_trace',
      '3': 20,
      '4': 3,
      '5': 9,
      '10': 'contentSyncTrace'
    },
    {
      '1': 'protocol_checkpoint_digest',
      '3': 24,
      '4': 1,
      '5': 9,
      '10': 'protocolCheckpointDigest'
    },
    {'1': 'read_consistency', '3': 25, '4': 1, '5': 9, '10': 'readConsistency'},
    {'1': 'read_error', '3': 26, '4': 1, '5': 9, '10': 'readError'},
  ],
};

/// Descriptor for `RuntimeState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeStateDescriptor = $convert.base64Decode(
    'CgxSdW50aW1lU3RhdGUSFwoHcGVlcl9pZBgBIAEoCVIGcGVlcklkEicKD21hbmlmZXN0X2RpZ2'
    'VzdBgCIAEoCVIObWFuaWZlc3REaWdlc3QSMAoUY2hlY2twb2ludF9yb290X2hhc2gYAyABKAlS'
    'EmNoZWNrcG9pbnRSb290SGFzaBIuChN0ZW50YXRpdmVfcm9vdF9oYXNoGAQgASgJUhF0ZW50YX'
    'RpdmVSb290SGFzaBJBCh1wcm90b2NvbF9jaGVja3BvaW50X3Jvb3RfaGFzaBgFIAEoCVIacHJv'
    'dG9jb2xDaGVja3BvaW50Um9vdEhhc2gSMwoWZHVyYWJsZV9tYWluX3Jvb3RfaGFzaBgGIAEoCV'
    'ITZHVyYWJsZU1haW5Sb290SGFzaBInCg9zdGF0ZV9wcm92aWRlcnMYCiADKAlSDnN0YXRlUHJv'
    'dmlkZXJzEicKD2Nvbm5lY3RlZF9wZWVycxgLIAMoCVIOY29ubmVjdGVkUGVlcnMSHwoLZmF0YW'
    'xfc3RhdGUYDCABKAlSCmZhdGFsU3RhdGUSNgoXcnVudGltZV9yZWZyZXNoX3BlbmRpbmcYDSAB'
    'KAhSFXJ1bnRpbWVSZWZyZXNoUGVuZGluZxI7ChpydW50aW1lX3JlZnJlc2hfbGFzdF9lcnJvch'
    'gOIAEoCVIXcnVudGltZVJlZnJlc2hMYXN0RXJyb3ISPAoacnVudGltZV9jaGVja3BvaW50X3Bl'
    'bmRpbmcYDyABKAhSGHJ1bnRpbWVDaGVja3BvaW50UGVuZGluZxJBCh1ydW50aW1lX2NoZWNrcG'
    '9pbnRfbGFzdF9lcnJvchgQIAEoCVIacnVudGltZUNoZWNrcG9pbnRMYXN0RXJyb3ISRAoecnVu'
    'dGltZV9tYXRlcmlhbGl6YXRpb25fcG9saWN5GBEgASgJUhxydW50aW1lTWF0ZXJpYWxpemF0aW'
    '9uUG9saWN5EjwKDXBlZXJfc3RhdHVzZXMYEiADKAsyFy5hcGljLlJ1bnRpbWVQZWVyU3RhdHVz'
    'UgxwZWVyU3RhdHVzZXMSQAoNY29tcGF0aWJpbGl0eRgTIAMoCzIaLmFwaWMuUnVudGltZUNvbX'
    'BhdGliaWxpdHlSDWNvbXBhdGliaWxpdHkSLAoSY29udGVudF9zeW5jX3RyYWNlGBQgAygJUhBj'
    'b250ZW50U3luY1RyYWNlEjwKGnByb3RvY29sX2NoZWNrcG9pbnRfZGlnZXN0GBggASgJUhhwcm'
    '90b2NvbENoZWNrcG9pbnREaWdlc3QSKQoQcmVhZF9jb25zaXN0ZW5jeRgZIAEoCVIPcmVhZENv'
    'bnNpc3RlbmN5Eh0KCnJlYWRfZXJyb3IYGiABKAlSCXJlYWRFcnJvcg==');

@$core.Deprecated('Use runtimePeerStatusDescriptor instead')
const RuntimePeerStatus$json = {
  '1': 'RuntimePeerStatus',
  '2': [
    {'1': 'peer_id', '3': 1, '4': 1, '5': 9, '10': 'peerId'},
    {'1': 'connected', '3': 2, '4': 1, '5': 8, '10': 'connected'},
    {'1': 'dialable', '3': 3, '4': 1, '5': 8, '10': 'dialable'},
    {'1': 'state_provider', '3': 4, '4': 1, '5': 8, '10': 'stateProvider'},
    {'1': 'compatible', '3': 7, '4': 1, '5': 8, '10': 'compatible'},
    {'1': 'incompatible', '3': 8, '4': 1, '5': 8, '10': 'incompatible'},
    {'1': 'ignored', '3': 9, '4': 1, '5': 8, '10': 'ignored'},
    {'1': 'relay_only', '3': 10, '4': 1, '5': 8, '10': 'relayOnly'},
    {'1': 'addresses', '3': 11, '4': 3, '5': 9, '10': 'addresses'},
    {
      '1': 'last_dial_errors',
      '3': 12,
      '4': 3,
      '5': 11,
      '6': '.apic.RuntimePeerStatus.LastDialErrorsEntry',
      '10': 'lastDialErrors'
    },
    {'1': 'reason', '3': 13, '4': 1, '5': 9, '10': 'reason'},
    {
      '1': 'replication_priority',
      '3': 14,
      '4': 1,
      '5': 5,
      '10': 'replicationPriority'
    },
    {
      '1': 'replication_device_class',
      '3': 15,
      '4': 1,
      '5': 9,
      '10': 'replicationDeviceClass'
    },
  ],
  '3': [RuntimePeerStatus_LastDialErrorsEntry$json],
};

@$core.Deprecated('Use runtimePeerStatusDescriptor instead')
const RuntimePeerStatus_LastDialErrorsEntry$json = {
  '1': 'LastDialErrorsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `RuntimePeerStatus`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimePeerStatusDescriptor = $convert.base64Decode(
    'ChFSdW50aW1lUGVlclN0YXR1cxIXCgdwZWVyX2lkGAEgASgJUgZwZWVySWQSHAoJY29ubmVjdG'
    'VkGAIgASgIUgljb25uZWN0ZWQSGgoIZGlhbGFibGUYAyABKAhSCGRpYWxhYmxlEiUKDnN0YXRl'
    'X3Byb3ZpZGVyGAQgASgIUg1zdGF0ZVByb3ZpZGVyEh4KCmNvbXBhdGlibGUYByABKAhSCmNvbX'
    'BhdGlibGUSIgoMaW5jb21wYXRpYmxlGAggASgIUgxpbmNvbXBhdGlibGUSGAoHaWdub3JlZBgJ'
    'IAEoCFIHaWdub3JlZBIdCgpyZWxheV9vbmx5GAogASgIUglyZWxheU9ubHkSHAoJYWRkcmVzc2'
    'VzGAsgAygJUglhZGRyZXNzZXMSVQoQbGFzdF9kaWFsX2Vycm9ycxgMIAMoCzIrLmFwaWMuUnVu'
    'dGltZVBlZXJTdGF0dXMuTGFzdERpYWxFcnJvcnNFbnRyeVIObGFzdERpYWxFcnJvcnMSFgoGcm'
    'Vhc29uGA0gASgJUgZyZWFzb24SMQoUcmVwbGljYXRpb25fcHJpb3JpdHkYDiABKAVSE3JlcGxp'
    'Y2F0aW9uUHJpb3JpdHkSOAoYcmVwbGljYXRpb25fZGV2aWNlX2NsYXNzGA8gASgJUhZyZXBsaW'
    'NhdGlvbkRldmljZUNsYXNzGkEKE0xhc3REaWFsRXJyb3JzRW50cnkSEAoDa2V5GAEgASgJUgNr'
    'ZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use runtimeCompatibilityDescriptor instead')
const RuntimeCompatibility$json = {
  '1': 'RuntimeCompatibility',
  '2': [
    {'1': 'peer_id', '3': 1, '4': 1, '5': 9, '10': 'peerId'},
    {'1': 'local_digest', '3': 2, '4': 1, '5': 9, '10': 'localDigest'},
    {'1': 'remote_digest', '3': 3, '4': 1, '5': 9, '10': 'remoteDigest'},
    {'1': 'compatible', '3': 4, '4': 1, '5': 8, '10': 'compatible'},
    {'1': 'blocking', '3': 5, '4': 1, '5': 8, '10': 'blocking'},
    {'1': 'reason', '3': 6, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `RuntimeCompatibility`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeCompatibilityDescriptor = $convert.base64Decode(
    'ChRSdW50aW1lQ29tcGF0aWJpbGl0eRIXCgdwZWVyX2lkGAEgASgJUgZwZWVySWQSIQoMbG9jYW'
    'xfZGlnZXN0GAIgASgJUgtsb2NhbERpZ2VzdBIjCg1yZW1vdGVfZGlnZXN0GAMgASgJUgxyZW1v'
    'dGVEaWdlc3QSHgoKY29tcGF0aWJsZRgEIAEoCFIKY29tcGF0aWJsZRIaCghibG9ja2luZxgFIA'
    'EoCFIIYmxvY2tpbmcSFgoGcmVhc29uGAYgASgJUgZyZWFzb24=');

@$core.Deprecated('Use cloudImageDescriptor instead')
const CloudImage$json = {
  '1': 'CloudImage',
  '2': [
    {'1': 'provider', '3': 1, '4': 1, '5': 9, '10': 'provider'},
    {'1': 'url', '3': 2, '4': 1, '5': 9, '10': 'url'},
    {'1': 'digest', '3': 3, '4': 1, '5': 9, '10': 'digest'},
    {'1': 'release_date', '3': 4, '4': 1, '5': 3, '10': 'releaseDate'},
  ],
};

/// Descriptor for `CloudImage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudImageDescriptor = $convert.base64Decode(
    'CgpDbG91ZEltYWdlEhoKCHByb3ZpZGVyGAEgASgJUghwcm92aWRlchIQCgN1cmwYAiABKAlSA3'
    'VybBIWCgZkaWdlc3QYAyABKAlSBmRpZ2VzdBIhCgxyZWxlYXNlX2RhdGUYBCABKANSC3JlbGVh'
    'c2VEYXRl');

@$core.Deprecated('Use cloudSpecificImageDescriptor instead')
const CloudSpecificImage$json = {
  '1': 'CloudSpecificImage',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'location', '3': 3, '4': 1, '5': 9, '10': 'location'},
    {'1': 'logical_name', '3': 4, '4': 1, '5': 9, '10': 'logicalName'},
    {'1': 'date_suffix', '3': 5, '4': 1, '5': 9, '10': 'dateSuffix'},
    {'1': 'updated_at_unix', '3': 6, '4': 1, '5': 3, '10': 'updatedAtUnix'},
    {'1': 'canonical', '3': 7, '4': 1, '5': 8, '10': 'canonical'},
  ],
};

/// Descriptor for `CloudSpecificImage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cloudSpecificImageDescriptor = $convert.base64Decode(
    'ChJDbG91ZFNwZWNpZmljSW1hZ2USDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbW'
    'USGgoIbG9jYXRpb24YAyABKAlSCGxvY2F0aW9uEiEKDGxvZ2ljYWxfbmFtZRgEIAEoCVILbG9n'
    'aWNhbE5hbWUSHwoLZGF0ZV9zdWZmaXgYBSABKAlSCmRhdGVTdWZmaXgSJgoPdXBkYXRlZF9hdF'
    '91bml4GAYgASgDUg11cGRhdGVkQXRVbml4EhwKCWNhbm9uaWNhbBgHIAEoCFIJY2Fub25pY2Fs');

@$core.Deprecated('Use releaseDescriptor instead')
const Release$json = {
  '1': 'Release',
  '2': [
    {
      '1': 'cloud_images',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Release.CloudImagesEntry',
      '10': 'cloudImages'
    },
    {'1': 'version', '3': 2, '4': 1, '5': 9, '10': 'version'},
    {'1': 'description', '3': 3, '4': 1, '5': 9, '10': 'description'},
    {'1': 'release_date', '3': 4, '4': 1, '5': 3, '10': 'releaseDate'},
  ],
  '3': [Release_CloudImagesEntry$json],
};

@$core.Deprecated('Use releaseDescriptor instead')
const Release_CloudImagesEntry$json = {
  '1': 'CloudImagesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudImage',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `Release`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List releaseDescriptor = $convert.base64Decode(
    'CgdSZWxlYXNlEkEKDGNsb3VkX2ltYWdlcxgBIAMoCzIeLmFwaWMuUmVsZWFzZS5DbG91ZEltYW'
    'dlc0VudHJ5UgtjbG91ZEltYWdlcxIYCgd2ZXJzaW9uGAIgASgJUgd2ZXJzaW9uEiAKC2Rlc2Ny'
    'aXB0aW9uGAMgASgJUgtkZXNjcmlwdGlvbhIhCgxyZWxlYXNlX2RhdGUYBCABKANSC3JlbGVhc2'
    'VEYXRlGlAKEENsb3VkSW1hZ2VzRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSJgoFdmFsdWUYAiAB'
    'KAsyEC5hcGljLkNsb3VkSW1hZ2VSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use getProtosdReleasesRequestDescriptor instead')
const GetProtosdReleasesRequest$json = {
  '1': 'GetProtosdReleasesRequest',
};

/// Descriptor for `GetProtosdReleasesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProtosdReleasesRequestDescriptor =
    $convert.base64Decode('ChlHZXRQcm90b3NkUmVsZWFzZXNSZXF1ZXN0');

@$core.Deprecated('Use getProtosdReleasesResponseDescriptor instead')
const GetProtosdReleasesResponse$json = {
  '1': 'GetProtosdReleasesResponse',
  '2': [
    {
      '1': 'releases',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Release',
      '10': 'releases'
    },
  ],
};

/// Descriptor for `GetProtosdReleasesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProtosdReleasesResponseDescriptor =
    $convert.base64Decode(
        'ChpHZXRQcm90b3NkUmVsZWFzZXNSZXNwb25zZRIpCghyZWxlYXNlcxgBIAMoCzINLmFwaWMuUm'
        'VsZWFzZVIIcmVsZWFzZXM=');

@$core.Deprecated('Use getCloudImagesRequestDescriptor instead')
const GetCloudImagesRequest$json = {
  '1': 'GetCloudImagesRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetCloudImagesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudImagesRequestDescriptor =
    $convert.base64Decode(
        'ChVHZXRDbG91ZEltYWdlc1JlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use getCloudImagesResponseDescriptor instead')
const GetCloudImagesResponse$json = {
  '1': 'GetCloudImagesResponse',
  '2': [
    {
      '1': 'cloud_images',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.GetCloudImagesResponse.CloudImagesEntry',
      '10': 'cloudImages'
    },
  ],
  '3': [GetCloudImagesResponse_CloudImagesEntry$json],
};

@$core.Deprecated('Use getCloudImagesResponseDescriptor instead')
const GetCloudImagesResponse_CloudImagesEntry$json = {
  '1': 'CloudImagesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudSpecificImage',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `GetCloudImagesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCloudImagesResponseDescriptor = $convert.base64Decode(
    'ChZHZXRDbG91ZEltYWdlc1Jlc3BvbnNlElAKDGNsb3VkX2ltYWdlcxgBIAMoCzItLmFwaWMuR2'
    'V0Q2xvdWRJbWFnZXNSZXNwb25zZS5DbG91ZEltYWdlc0VudHJ5UgtjbG91ZEltYWdlcxpYChBD'
    'bG91ZEltYWdlc0VudHJ5EhAKA2tleRgBIAEoCVIDa2V5Ei4KBXZhbHVlGAIgASgLMhguYXBpYy'
    '5DbG91ZFNwZWNpZmljSW1hZ2VSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use getProvisionerImagesRequestDescriptor instead')
const GetProvisionerImagesRequest$json = {
  '1': 'GetProvisionerImagesRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `GetProvisionerImagesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionerImagesRequestDescriptor =
    $convert.base64Decode(
        'ChtHZXRQcm92aXNpb25lckltYWdlc1JlcXVlc3QSEgoEbmFtZRgBIAEoCVIEbmFtZQ==');

@$core.Deprecated('Use getProvisionerImagesResponseDescriptor instead')
const GetProvisionerImagesResponse$json = {
  '1': 'GetProvisionerImagesResponse',
  '2': [
    {
      '1': 'images',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.GetProvisionerImagesResponse.ImagesEntry',
      '10': 'images'
    },
  ],
  '3': [GetProvisionerImagesResponse_ImagesEntry$json],
};

@$core.Deprecated('Use getProvisionerImagesResponseDescriptor instead')
const GetProvisionerImagesResponse_ImagesEntry$json = {
  '1': 'ImagesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CloudSpecificImage',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `GetProvisionerImagesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getProvisionerImagesResponseDescriptor = $convert.base64Decode(
    'ChxHZXRQcm92aXNpb25lckltYWdlc1Jlc3BvbnNlEkYKBmltYWdlcxgBIAMoCzIuLmFwaWMuR2'
    'V0UHJvdmlzaW9uZXJJbWFnZXNSZXNwb25zZS5JbWFnZXNFbnRyeVIGaW1hZ2VzGlMKC0ltYWdl'
    'c0VudHJ5EhAKA2tleRgBIAEoCVIDa2V5Ei4KBXZhbHVlGAIgASgLMhguYXBpYy5DbG91ZFNwZW'
    'NpZmljSW1hZ2VSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use uploadCloudImageRequestDescriptor instead')
const UploadCloudImageRequest$json = {
  '1': 'UploadCloudImageRequest',
  '2': [
    {'1': 'image_path', '3': 1, '4': 1, '5': 9, '10': 'imagePath'},
    {'1': 'image_name', '3': 2, '4': 1, '5': 9, '10': 'imageName'},
    {'1': 'cloud_name', '3': 3, '4': 1, '5': 9, '10': 'cloudName'},
    {'1': 'cloud_location', '3': 4, '4': 1, '5': 9, '10': 'cloudLocation'},
    {'1': 'timeout', '3': 5, '4': 1, '5': 5, '10': 'timeout'},
  ],
};

/// Descriptor for `UploadCloudImageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadCloudImageRequestDescriptor = $convert.base64Decode(
    'ChdVcGxvYWRDbG91ZEltYWdlUmVxdWVzdBIdCgppbWFnZV9wYXRoGAEgASgJUglpbWFnZVBhdG'
    'gSHQoKaW1hZ2VfbmFtZRgCIAEoCVIJaW1hZ2VOYW1lEh0KCmNsb3VkX25hbWUYAyABKAlSCWNs'
    'b3VkTmFtZRIlCg5jbG91ZF9sb2NhdGlvbhgEIAEoCVINY2xvdWRMb2NhdGlvbhIYCgd0aW1lb3'
    'V0GAUgASgFUgd0aW1lb3V0');

@$core.Deprecated('Use uploadCloudImageResponseDescriptor instead')
const UploadCloudImageResponse$json = {
  '1': 'UploadCloudImageResponse',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'task_id', '3': 2, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `UploadCloudImageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadCloudImageResponseDescriptor =
    $convert.base64Decode(
        'ChhVcGxvYWRDbG91ZEltYWdlUmVzcG9uc2USDgoCaWQYASABKAlSAmlkEhcKB3Rhc2tfaWQYAi'
        'ABKAlSBnRhc2tJZA==');

@$core.Deprecated('Use uploadProvisionerImageRequestDescriptor instead')
const UploadProvisionerImageRequest$json = {
  '1': 'UploadProvisionerImageRequest',
  '2': [
    {'1': 'image_path', '3': 1, '4': 1, '5': 9, '10': 'imagePath'},
    {'1': 'image_name', '3': 2, '4': 1, '5': 9, '10': 'imageName'},
    {'1': 'provisioner_name', '3': 3, '4': 1, '5': 9, '10': 'provisionerName'},
    {'1': 'location', '3': 4, '4': 1, '5': 9, '10': 'location'},
    {'1': 'timeout', '3': 5, '4': 1, '5': 5, '10': 'timeout'},
  ],
};

/// Descriptor for `UploadProvisionerImageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadProvisionerImageRequestDescriptor = $convert.base64Decode(
    'Ch1VcGxvYWRQcm92aXNpb25lckltYWdlUmVxdWVzdBIdCgppbWFnZV9wYXRoGAEgASgJUglpbW'
    'FnZVBhdGgSHQoKaW1hZ2VfbmFtZRgCIAEoCVIJaW1hZ2VOYW1lEikKEHByb3Zpc2lvbmVyX25h'
    'bWUYAyABKAlSD3Byb3Zpc2lvbmVyTmFtZRIaCghsb2NhdGlvbhgEIAEoCVIIbG9jYXRpb24SGA'
    'oHdGltZW91dBgFIAEoBVIHdGltZW91dA==');

@$core.Deprecated('Use uploadProvisionerImageResponseDescriptor instead')
const UploadProvisionerImageResponse$json = {
  '1': 'UploadProvisionerImageResponse',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'task_id', '3': 2, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `UploadProvisionerImageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadProvisionerImageResponseDescriptor =
    $convert.base64Decode(
        'Ch5VcGxvYWRQcm92aXNpb25lckltYWdlUmVzcG9uc2USDgoCaWQYASABKAlSAmlkEhcKB3Rhc2'
        'tfaWQYAiABKAlSBnRhc2tJZA==');

@$core.Deprecated('Use removeCloudImageRequestDescriptor instead')
const RemoveCloudImageRequest$json = {
  '1': 'RemoveCloudImageRequest',
  '2': [
    {'1': 'image_name', '3': 2, '4': 1, '5': 9, '10': 'imageName'},
    {'1': 'cloud_name', '3': 3, '4': 1, '5': 9, '10': 'cloudName'},
    {'1': 'cloud_location', '3': 4, '4': 1, '5': 9, '10': 'cloudLocation'},
  ],
};

/// Descriptor for `RemoveCloudImageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeCloudImageRequestDescriptor = $convert.base64Decode(
    'ChdSZW1vdmVDbG91ZEltYWdlUmVxdWVzdBIdCgppbWFnZV9uYW1lGAIgASgJUglpbWFnZU5hbW'
    'USHQoKY2xvdWRfbmFtZRgDIAEoCVIJY2xvdWROYW1lEiUKDmNsb3VkX2xvY2F0aW9uGAQgASgJ'
    'Ug1jbG91ZExvY2F0aW9u');

@$core.Deprecated('Use removeCloudImageResponseDescriptor instead')
const RemoveCloudImageResponse$json = {
  '1': 'RemoveCloudImageResponse',
};

/// Descriptor for `RemoveCloudImageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeCloudImageResponseDescriptor =
    $convert.base64Decode('ChhSZW1vdmVDbG91ZEltYWdlUmVzcG9uc2U=');

@$core.Deprecated('Use removeProvisionerImageRequestDescriptor instead')
const RemoveProvisionerImageRequest$json = {
  '1': 'RemoveProvisionerImageRequest',
  '2': [
    {'1': 'image_name', '3': 1, '4': 1, '5': 9, '10': 'imageName'},
    {'1': 'provisioner_name', '3': 2, '4': 1, '5': 9, '10': 'provisionerName'},
    {'1': 'location', '3': 3, '4': 1, '5': 9, '10': 'location'},
  ],
};

/// Descriptor for `RemoveProvisionerImageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeProvisionerImageRequestDescriptor =
    $convert.base64Decode(
        'Ch1SZW1vdmVQcm92aXNpb25lckltYWdlUmVxdWVzdBIdCgppbWFnZV9uYW1lGAEgASgJUglpbW'
        'FnZU5hbWUSKQoQcHJvdmlzaW9uZXJfbmFtZRgCIAEoCVIPcHJvdmlzaW9uZXJOYW1lEhoKCGxv'
        'Y2F0aW9uGAMgASgJUghsb2NhdGlvbg==');

@$core.Deprecated('Use removeProvisionerImageResponseDescriptor instead')
const RemoveProvisionerImageResponse$json = {
  '1': 'RemoveProvisionerImageResponse',
};

/// Descriptor for `RemoveProvisionerImageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeProvisionerImageResponseDescriptor =
    $convert.base64Decode('Ch5SZW1vdmVQcm92aXNpb25lckltYWdlUmVzcG9uc2U=');

@$core.Deprecated('Use imageContentDescriptorDescriptor instead')
const ImageContentDescriptor$json = {
  '1': 'ImageContentDescriptor',
  '2': [
    {'1': 'media_type', '3': 1, '4': 1, '5': 9, '10': 'mediaType'},
    {'1': 'digest', '3': 2, '4': 1, '5': 9, '10': 'digest'},
    {'1': 'size_bytes', '3': 3, '4': 1, '5': 4, '10': 'sizeBytes'},
    {'1': 'platform', '3': 4, '4': 1, '5': 9, '10': 'platform'},
    {
      '1': 'annotations',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.ImageContentDescriptor.AnnotationsEntry',
      '10': 'annotations'
    },
  ],
  '3': [ImageContentDescriptor_AnnotationsEntry$json],
};

@$core.Deprecated('Use imageContentDescriptorDescriptor instead')
const ImageContentDescriptor_AnnotationsEntry$json = {
  '1': 'AnnotationsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `ImageContentDescriptor`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List imageContentDescriptorDescriptor = $convert.base64Decode(
    'ChZJbWFnZUNvbnRlbnREZXNjcmlwdG9yEh0KCm1lZGlhX3R5cGUYASABKAlSCW1lZGlhVHlwZR'
    'IWCgZkaWdlc3QYAiABKAlSBmRpZ2VzdBIdCgpzaXplX2J5dGVzGAMgASgEUglzaXplQnl0ZXMS'
    'GgoIcGxhdGZvcm0YBCABKAlSCHBsYXRmb3JtEk8KC2Fubm90YXRpb25zGAUgAygLMi0uYXBpYy'
    '5JbWFnZUNvbnRlbnREZXNjcmlwdG9yLkFubm90YXRpb25zRW50cnlSC2Fubm90YXRpb25zGj4K'
    'EEFubm90YXRpb25zRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbH'
    'VlOgI4AQ==');

@$core.Deprecated('Use getInstanceImageRequestDescriptor instead')
const GetInstanceImageRequest$json = {
  '1': 'GetInstanceImageRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
    {'1': 'image_ref', '3': 2, '4': 1, '5': 9, '10': 'imageRef'},
    {'1': 'include_content', '3': 3, '4': 1, '5': 8, '10': 'includeContent'},
  ],
};

/// Descriptor for `GetInstanceImageRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceImageRequestDescriptor = $convert.base64Decode(
    'ChdHZXRJbnN0YW5jZUltYWdlUmVxdWVzdBIaCghpbnN0YW5jZRgBIAEoCVIIaW5zdGFuY2USGw'
    'oJaW1hZ2VfcmVmGAIgASgJUghpbWFnZVJlZhInCg9pbmNsdWRlX2NvbnRlbnQYAyABKAhSDmlu'
    'Y2x1ZGVDb250ZW50');

@$core.Deprecated('Use getInstanceImageResponseDescriptor instead')
const GetInstanceImageResponse$json = {
  '1': 'GetInstanceImageResponse',
  '2': [
    {'1': 'found', '3': 1, '4': 1, '5': 8, '10': 'found'},
    {'1': 'image_ref', '3': 2, '4': 1, '5': 9, '10': 'imageRef'},
    {'1': 'target_digest', '3': 3, '4': 1, '5': 9, '10': 'targetDigest'},
    {'1': 'platform', '3': 4, '4': 1, '5': 9, '10': 'platform'},
    {
      '1': 'labels',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.GetInstanceImageResponse.LabelsEntry',
      '10': 'labels'
    },
    {'1': 'has_content', '3': 6, '4': 1, '5': 8, '10': 'hasContent'},
    {
      '1': 'target',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.apic.ImageContentDescriptor',
      '10': 'target'
    },
    {
      '1': 'descriptors',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.apic.ImageContentDescriptor',
      '10': 'descriptors'
    },
  ],
  '3': [GetInstanceImageResponse_LabelsEntry$json],
};

@$core.Deprecated('Use getInstanceImageResponseDescriptor instead')
const GetInstanceImageResponse_LabelsEntry$json = {
  '1': 'LabelsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `GetInstanceImageResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceImageResponseDescriptor = $convert.base64Decode(
    'ChhHZXRJbnN0YW5jZUltYWdlUmVzcG9uc2USFAoFZm91bmQYASABKAhSBWZvdW5kEhsKCWltYW'
    'dlX3JlZhgCIAEoCVIIaW1hZ2VSZWYSIwoNdGFyZ2V0X2RpZ2VzdBgDIAEoCVIMdGFyZ2V0RGln'
    'ZXN0EhoKCHBsYXRmb3JtGAQgASgJUghwbGF0Zm9ybRJCCgZsYWJlbHMYBSADKAsyKi5hcGljLk'
    'dldEluc3RhbmNlSW1hZ2VSZXNwb25zZS5MYWJlbHNFbnRyeVIGbGFiZWxzEh8KC2hhc19jb250'
    'ZW50GAYgASgIUgpoYXNDb250ZW50EjQKBnRhcmdldBgHIAEoCzIcLmFwaWMuSW1hZ2VDb250ZW'
    '50RGVzY3JpcHRvclIGdGFyZ2V0Ej4KC2Rlc2NyaXB0b3JzGAggAygLMhwuYXBpYy5JbWFnZUNv'
    'bnRlbnREZXNjcmlwdG9yUgtkZXNjcmlwdG9ycxo5CgtMYWJlbHNFbnRyeRIQCgNrZXkYASABKA'
    'lSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgB');

@$core.Deprecated('Use uploadInstanceImageArchiveRequestDescriptor instead')
const UploadInstanceImageArchiveRequest$json = {
  '1': 'UploadInstanceImageArchiveRequest',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 9, '10': 'instance'},
    {'1': 'archive_path', '3': 2, '4': 1, '5': 9, '10': 'archivePath'},
    {'1': 'image_ref', '3': 3, '4': 1, '5': 9, '10': 'imageRef'},
  ],
};

/// Descriptor for `UploadInstanceImageArchiveRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadInstanceImageArchiveRequestDescriptor =
    $convert.base64Decode(
        'CiFVcGxvYWRJbnN0YW5jZUltYWdlQXJjaGl2ZVJlcXVlc3QSGgoIaW5zdGFuY2UYASABKAlSCG'
        'luc3RhbmNlEiEKDGFyY2hpdmVfcGF0aBgCIAEoCVILYXJjaGl2ZVBhdGgSGwoJaW1hZ2VfcmVm'
        'GAMgASgJUghpbWFnZVJlZg==');

@$core.Deprecated('Use uploadInstanceImageArchiveResponseDescriptor instead')
const UploadInstanceImageArchiveResponse$json = {
  '1': 'UploadInstanceImageArchiveResponse',
  '2': [
    {'1': 'task_id', '3': 1, '4': 1, '5': 9, '10': 'taskId'},
  ],
};

/// Descriptor for `UploadInstanceImageArchiveResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uploadInstanceImageArchiveResponseDescriptor =
    $convert.base64Decode(
        'CiJVcGxvYWRJbnN0YW5jZUltYWdlQXJjaGl2ZVJlc3BvbnNlEhcKB3Rhc2tfaWQYASABKAlSBn'
        'Rhc2tJZA==');

@$core.Deprecated('Use coreEndpointDescriptor instead')
const CoreEndpoint$json = {
  '1': 'CoreEndpoint',
  '2': [
    {'1': 'kind', '3': 1, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'address', '3': 2, '4': 1, '5': 9, '10': 'address'},
    {'1': 'active', '3': 3, '4': 1, '5': 8, '10': 'active'},
    {'1': 'message', '3': 4, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `CoreEndpoint`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List coreEndpointDescriptor = $convert.base64Decode(
    'CgxDb3JlRW5kcG9pbnQSEgoEa2luZBgBIAEoCVIEa2luZBIYCgdhZGRyZXNzGAIgASgJUgdhZG'
    'RyZXNzEhYKBmFjdGl2ZRgDIAEoCFIGYWN0aXZlEhgKB21lc3NhZ2UYBCABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use hostAgentConnectionStatusDescriptor instead')
const HostAgentConnectionStatus$json = {
  '1': 'HostAgentConnectionStatus',
  '2': [
    {'1': 'connected', '3': 1, '4': 1, '5': 8, '10': 'connected'},
    {'1': 'socket', '3': 2, '4': 1, '5': 9, '10': 'socket'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `HostAgentConnectionStatus`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List hostAgentConnectionStatusDescriptor =
    $convert.base64Decode(
        'ChlIb3N0QWdlbnRDb25uZWN0aW9uU3RhdHVzEhwKCWNvbm5lY3RlZBgBIAEoCFIJY29ubmVjdG'
        'VkEhYKBnNvY2tldBgCIAEoCVIGc29ja2V0EhgKB21lc3NhZ2UYAyABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use systemStatusDescriptor instead')
const SystemStatus$json = {
  '1': 'SystemStatus',
  '2': [
    {'1': 'core_status', '3': 1, '4': 1, '5': 9, '10': 'coreStatus'},
    {'1': 'work_dir', '3': 2, '4': 1, '5': 9, '10': 'workDir'},
    {'1': 'capabilities', '3': 3, '4': 1, '5': 9, '10': 'capabilities'},
    {'1': 'p2p_port', '3': 4, '4': 1, '5': 5, '10': 'p2pPort'},
    {
      '1': 'endpoints',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.CoreEndpoint',
      '10': 'endpoints'
    },
    {
      '1': 'host_agent',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.apic.HostAgentConnectionStatus',
      '10': 'hostAgent'
    },
    {'1': 'network_enabled', '3': 7, '4': 1, '5': 8, '10': 'networkEnabled'},
    {
      '1': 'host_agent_supported',
      '3': 8,
      '4': 1,
      '5': 8,
      '10': 'hostAgentSupported'
    },
    {
      '1': 'network',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.apic.NetworkRuntimeStatus',
      '10': 'network'
    },
  ],
};

/// Descriptor for `SystemStatus`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List systemStatusDescriptor = $convert.base64Decode(
    'CgxTeXN0ZW1TdGF0dXMSHwoLY29yZV9zdGF0dXMYASABKAlSCmNvcmVTdGF0dXMSGQoId29ya1'
    '9kaXIYAiABKAlSB3dvcmtEaXISIgoMY2FwYWJpbGl0aWVzGAMgASgJUgxjYXBhYmlsaXRpZXMS'
    'GQoIcDJwX3BvcnQYBCABKAVSB3AycFBvcnQSMAoJZW5kcG9pbnRzGAUgAygLMhIuYXBpYy5Db3'
    'JlRW5kcG9pbnRSCWVuZHBvaW50cxI+Cgpob3N0X2FnZW50GAYgASgLMh8uYXBpYy5Ib3N0QWdl'
    'bnRDb25uZWN0aW9uU3RhdHVzUglob3N0QWdlbnQSJwoPbmV0d29ya19lbmFibGVkGAcgASgIUg'
    '5uZXR3b3JrRW5hYmxlZBIwChRob3N0X2FnZW50X3N1cHBvcnRlZBgIIAEoCFISaG9zdEFnZW50'
    'U3VwcG9ydGVkEjQKB25ldHdvcmsYCSABKAsyGi5hcGljLk5ldHdvcmtSdW50aW1lU3RhdHVzUg'
    'duZXR3b3Jr');

@$core.Deprecated('Use getSystemStatusRequestDescriptor instead')
const GetSystemStatusRequest$json = {
  '1': 'GetSystemStatusRequest',
};

/// Descriptor for `GetSystemStatusRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSystemStatusRequestDescriptor =
    $convert.base64Decode('ChZHZXRTeXN0ZW1TdGF0dXNSZXF1ZXN0');

@$core.Deprecated('Use getSystemStatusResponseDescriptor instead')
const GetSystemStatusResponse$json = {
  '1': 'GetSystemStatusResponse',
  '2': [
    {
      '1': 'status',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.SystemStatus',
      '10': 'status'
    },
  ],
};

/// Descriptor for `GetSystemStatusResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getSystemStatusResponseDescriptor =
    $convert.base64Decode(
        'ChdHZXRTeXN0ZW1TdGF0dXNSZXNwb25zZRIqCgZzdGF0dXMYASABKAsyEi5hcGljLlN5c3RlbV'
        'N0YXR1c1IGc3RhdHVz');

@$core.Deprecated('Use startHostAgentRequestDescriptor instead')
const StartHostAgentRequest$json = {
  '1': 'StartHostAgentRequest',
};

/// Descriptor for `StartHostAgentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startHostAgentRequestDescriptor =
    $convert.base64Decode('ChVTdGFydEhvc3RBZ2VudFJlcXVlc3Q=');

@$core.Deprecated('Use startHostAgentResponseDescriptor instead')
const StartHostAgentResponse$json = {
  '1': 'StartHostAgentResponse',
  '2': [
    {
      '1': 'status',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.HostAgentConnectionStatus',
      '10': 'status'
    },
  ],
};

/// Descriptor for `StartHostAgentResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startHostAgentResponseDescriptor =
    $convert.base64Decode(
        'ChZTdGFydEhvc3RBZ2VudFJlc3BvbnNlEjcKBnN0YXR1cxgBIAEoCzIfLmFwaWMuSG9zdEFnZW'
        '50Q29ubmVjdGlvblN0YXR1c1IGc3RhdHVz');

@$core.Deprecated('Use stopHostAgentRequestDescriptor instead')
const StopHostAgentRequest$json = {
  '1': 'StopHostAgentRequest',
};

/// Descriptor for `StopHostAgentRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopHostAgentRequestDescriptor =
    $convert.base64Decode('ChRTdG9wSG9zdEFnZW50UmVxdWVzdA==');

@$core.Deprecated('Use stopHostAgentResponseDescriptor instead')
const StopHostAgentResponse$json = {
  '1': 'StopHostAgentResponse',
  '2': [
    {
      '1': 'status',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.HostAgentConnectionStatus',
      '10': 'status'
    },
  ],
};

/// Descriptor for `StopHostAgentResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopHostAgentResponseDescriptor = $convert.base64Decode(
    'ChVTdG9wSG9zdEFnZW50UmVzcG9uc2USNwoGc3RhdHVzGAEgASgLMh8uYXBpYy5Ib3N0QWdlbn'
    'RDb25uZWN0aW9uU3RhdHVzUgZzdGF0dXM=');

@$core.Deprecated('Use commitDescriptor instead')
const Commit$json = {
  '1': 'Commit',
  '2': [
    {'1': 'hash', '3': 1, '4': 1, '5': 9, '10': 'hash'},
    {'1': 'committer', '3': 2, '4': 1, '5': 9, '10': 'committer'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
    {'1': 'states', '3': 4, '4': 3, '5': 9, '10': 'states'},
    {'1': 'date_unix', '3': 5, '4': 1, '5': 3, '10': 'dateUnix'},
    {'1': 'parent_hashes', '3': 6, '4': 3, '5': 9, '10': 'parentHashes'},
    {'1': 'refs', '3': 7, '4': 3, '5': 9, '10': 'refs'},
  ],
};

/// Descriptor for `Commit`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDescriptor = $convert.base64Decode(
    'CgZDb21taXQSEgoEaGFzaBgBIAEoCVIEaGFzaBIcCgljb21taXR0ZXIYAiABKAlSCWNvbW1pdH'
    'RlchIYCgdtZXNzYWdlGAMgASgJUgdtZXNzYWdlEhYKBnN0YXRlcxgEIAMoCVIGc3RhdGVzEhsK'
    'CWRhdGVfdW5peBgFIAEoA1IIZGF0ZVVuaXgSIwoNcGFyZW50X2hhc2hlcxgGIAMoCVIMcGFyZW'
    '50SGFzaGVzEhIKBHJlZnMYByADKAlSBHJlZnM=');

@$core.Deprecated('Use commitGraphRelationDescriptor instead')
const CommitGraphRelation$json = {
  '1': 'CommitGraphRelation',
  '2': [
    {'1': 'parent_hash', '3': 1, '4': 1, '5': 9, '10': 'parentHash'},
    {'1': 'parent_row', '3': 2, '4': 1, '5': 5, '10': 'parentRow'},
    {'1': 'from_lane', '3': 3, '4': 1, '5': 5, '10': 'fromLane'},
    {'1': 'to_lane', '3': 4, '4': 1, '5': 5, '10': 'toLane'},
    {'1': 'visible', '3': 5, '4': 1, '5': 8, '10': 'visible'},
  ],
};

/// Descriptor for `CommitGraphRelation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitGraphRelationDescriptor = $convert.base64Decode(
    'ChNDb21taXRHcmFwaFJlbGF0aW9uEh8KC3BhcmVudF9oYXNoGAEgASgJUgpwYXJlbnRIYXNoEh'
    '0KCnBhcmVudF9yb3cYAiABKAVSCXBhcmVudFJvdxIbCglmcm9tX2xhbmUYAyABKAVSCGZyb21M'
    'YW5lEhcKB3RvX2xhbmUYBCABKAVSBnRvTGFuZRIYCgd2aXNpYmxlGAUgASgIUgd2aXNpYmxl');

@$core.Deprecated('Use commitGraphItemDescriptor instead')
const CommitGraphItem$json = {
  '1': 'CommitGraphItem',
  '2': [
    {
      '1': 'commit',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.Commit',
      '10': 'commit'
    },
    {'1': 'row', '3': 2, '4': 1, '5': 5, '10': 'row'},
    {'1': 'lane', '3': 3, '4': 1, '5': 5, '10': 'lane'},
    {'1': 'active_lanes', '3': 4, '4': 3, '5': 5, '10': 'activeLanes'},
    {
      '1': 'relations',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitGraphRelation',
      '10': 'relations'
    },
  ],
};

/// Descriptor for `CommitGraphItem`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitGraphItemDescriptor = $convert.base64Decode(
    'Cg9Db21taXRHcmFwaEl0ZW0SJAoGY29tbWl0GAEgASgLMgwuYXBpYy5Db21taXRSBmNvbW1pdB'
    'IQCgNyb3cYAiABKAVSA3JvdxISCgRsYW5lGAMgASgFUgRsYW5lEiEKDGFjdGl2ZV9sYW5lcxgE'
    'IAMoBVILYWN0aXZlTGFuZXMSNwoJcmVsYXRpb25zGAUgAygLMhkuYXBpYy5Db21taXRHcmFwaF'
    'JlbGF0aW9uUglyZWxhdGlvbnM=');

@$core.Deprecated('Use commitGraphDescriptor instead')
const CommitGraph$json = {
  '1': 'CommitGraph',
  '2': [
    {
      '1': 'items',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitGraphItem',
      '10': 'items'
    },
    {'1': 'lane_count', '3': 2, '4': 1, '5': 5, '10': 'laneCount'},
  ],
};

/// Descriptor for `CommitGraph`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitGraphDescriptor = $convert.base64Decode(
    'CgtDb21taXRHcmFwaBIrCgVpdGVtcxgBIAMoCzIVLmFwaWMuQ29tbWl0R3JhcGhJdGVtUgVpdG'
    'VtcxIdCgpsYW5lX2NvdW50GAIgASgFUglsYW5lQ291bnQ=');

@$core.Deprecated('Use getLocalCommitsRequestDescriptor instead')
const GetLocalCommitsRequest$json = {
  '1': 'GetLocalCommitsRequest',
};

/// Descriptor for `GetLocalCommitsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLocalCommitsRequestDescriptor =
    $convert.base64Decode('ChZHZXRMb2NhbENvbW1pdHNSZXF1ZXN0');

@$core.Deprecated('Use getLocalCommitsResponseDescriptor instead')
const GetLocalCommitsResponse$json = {
  '1': 'GetLocalCommitsResponse',
  '2': [
    {
      '1': 'commits',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Commit',
      '10': 'commits'
    },
    {
      '1': 'graph',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CommitGraph',
      '10': 'graph'
    },
  ],
};

/// Descriptor for `GetLocalCommitsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getLocalCommitsResponseDescriptor = $convert.base64Decode(
    'ChdHZXRMb2NhbENvbW1pdHNSZXNwb25zZRImCgdjb21taXRzGAEgAygLMgwuYXBpYy5Db21taX'
    'RSB2NvbW1pdHMSJwoFZ3JhcGgYAiABKAsyES5hcGljLkNvbW1pdEdyYXBoUgVncmFwaA==');

@$core.Deprecated('Use getRemoteCommitsRequestDescriptor instead')
const GetRemoteCommitsRequest$json = {
  '1': 'GetRemoteCommitsRequest',
  '2': [
    {'1': 'remote', '3': 1, '4': 1, '5': 9, '10': 'remote'},
  ],
};

/// Descriptor for `GetRemoteCommitsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getRemoteCommitsRequestDescriptor =
    $convert.base64Decode(
        'ChdHZXRSZW1vdGVDb21taXRzUmVxdWVzdBIWCgZyZW1vdGUYASABKAlSBnJlbW90ZQ==');

@$core.Deprecated('Use getRemoteCommitsResponseDescriptor instead')
const GetRemoteCommitsResponse$json = {
  '1': 'GetRemoteCommitsResponse',
  '2': [
    {
      '1': 'commits',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.Commit',
      '10': 'commits'
    },
    {
      '1': 'graph',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CommitGraph',
      '10': 'graph'
    },
  ],
};

/// Descriptor for `GetRemoteCommitsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getRemoteCommitsResponseDescriptor =
    $convert.base64Decode(
        'ChhHZXRSZW1vdGVDb21taXRzUmVzcG9uc2USJgoHY29tbWl0cxgBIAMoCzIMLmFwaWMuQ29tbW'
        'l0Ugdjb21taXRzEicKBWdyYXBoGAIgASgLMhEuYXBpYy5Db21taXRHcmFwaFIFZ3JhcGg=');

@$core.Deprecated('Use commitDiffValueDescriptor instead')
const CommitDiffValue$json = {
  '1': 'CommitDiffValue',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 9, '10': 'value'},
    {'1': 'is_null', '3': 2, '4': 1, '5': 8, '10': 'isNull'},
  ],
};

/// Descriptor for `CommitDiffValue`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffValueDescriptor = $convert.base64Decode(
    'Cg9Db21taXREaWZmVmFsdWUSFAoFdmFsdWUYASABKAlSBXZhbHVlEhcKB2lzX251bGwYAiABKA'
    'hSBmlzTnVsbA==');

@$core.Deprecated('Use commitDiffFieldDescriptor instead')
const CommitDiffField$json = {
  '1': 'CommitDiffField',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'before',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.apic.CommitDiffValue',
      '10': 'before'
    },
    {
      '1': 'after',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.apic.CommitDiffValue',
      '10': 'after'
    },
    {'1': 'before_cue', '3': 4, '4': 1, '5': 9, '10': 'beforeCue'},
    {'1': 'after_cue', '3': 5, '4': 1, '5': 9, '10': 'afterCue'},
    {'1': 'changed', '3': 6, '4': 1, '5': 8, '10': 'changed'},
  ],
};

/// Descriptor for `CommitDiffField`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffFieldDescriptor = $convert.base64Decode(
    'Cg9Db21taXREaWZmRmllbGQSEgoEbmFtZRgBIAEoCVIEbmFtZRItCgZiZWZvcmUYAiABKAsyFS'
    '5hcGljLkNvbW1pdERpZmZWYWx1ZVIGYmVmb3JlEisKBWFmdGVyGAMgASgLMhUuYXBpYy5Db21t'
    'aXREaWZmVmFsdWVSBWFmdGVyEh0KCmJlZm9yZV9jdWUYBCABKAlSCWJlZm9yZUN1ZRIbCglhZn'
    'Rlcl9jdWUYBSABKAlSCGFmdGVyQ3VlEhgKB2NoYW5nZWQYBiABKAhSB2NoYW5nZWQ=');

@$core.Deprecated('Use commitDiffRowDescriptor instead')
const CommitDiffRow$json = {
  '1': 'CommitDiffRow',
  '2': [
    {'1': 'change_type', '3': 1, '4': 1, '5': 9, '10': 'changeType'},
    {'1': 'key', '3': 2, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'fields',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitDiffField',
      '10': 'fields'
    },
    {'1': 'before_cue', '3': 4, '4': 1, '5': 9, '10': 'beforeCue'},
    {'1': 'after_cue', '3': 5, '4': 1, '5': 9, '10': 'afterCue'},
    {'1': 'cue', '3': 6, '4': 1, '5': 9, '10': 'cue'},
  ],
};

/// Descriptor for `CommitDiffRow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffRowDescriptor = $convert.base64Decode(
    'Cg1Db21taXREaWZmUm93Eh8KC2NoYW5nZV90eXBlGAEgASgJUgpjaGFuZ2VUeXBlEhAKA2tleR'
    'gCIAEoCVIDa2V5Ei0KBmZpZWxkcxgDIAMoCzIVLmFwaWMuQ29tbWl0RGlmZkZpZWxkUgZmaWVs'
    'ZHMSHQoKYmVmb3JlX2N1ZRgEIAEoCVIJYmVmb3JlQ3VlEhsKCWFmdGVyX2N1ZRgFIAEoCVIIYW'
    'Z0ZXJDdWUSEAoDY3VlGAYgASgJUgNjdWU=');

@$core.Deprecated('Use commitDiffTableDescriptor instead')
const CommitDiffTable$json = {
  '1': 'CommitDiffTable',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {
      '1': 'rows',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitDiffRow',
      '10': 'rows'
    },
    {'1': 'cue', '3': 3, '4': 1, '5': 9, '10': 'cue'},
  ],
};

/// Descriptor for `CommitDiffTable`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffTableDescriptor = $convert.base64Decode(
    'Cg9Db21taXREaWZmVGFibGUSEgoEbmFtZRgBIAEoCVIEbmFtZRInCgRyb3dzGAIgAygLMhMuYX'
    'BpYy5Db21taXREaWZmUm93UgRyb3dzEhAKA2N1ZRgDIAEoCVIDY3Vl');

@$core.Deprecated('Use commitDiffTaskContextDescriptor instead')
const CommitDiffTaskContext$json = {
  '1': 'CommitDiffTaskContext',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'stream', '3': 2, '4': 1, '5': 9, '10': 'stream'},
    {'1': 'subject_type', '3': 3, '4': 1, '5': 9, '10': 'subjectType'},
    {'1': 'subject_id', '3': 4, '4': 1, '5': 9, '10': 'subjectId'},
    {'1': 'owner_peer_id', '3': 5, '4': 1, '5': 9, '10': 'ownerPeerId'},
    {'1': 'status', '3': 6, '4': 1, '5': 9, '10': 'status'},
    {'1': 'title', '3': 7, '4': 1, '5': 9, '10': 'title'},
    {'1': 'message', '3': 8, '4': 1, '5': 9, '10': 'message'},
    {'1': 'progress', '3': 9, '4': 1, '5': 5, '10': 'progress'},
    {'1': 'change_sources', '3': 10, '4': 3, '5': 9, '10': 'changeSources'},
    {'1': 'event_count', '3': 11, '4': 1, '5': 5, '10': 'eventCount'},
    {'1': 'summary', '3': 12, '4': 1, '5': 9, '10': 'summary'},
  ],
};

/// Descriptor for `CommitDiffTaskContext`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffTaskContextDescriptor = $convert.base64Decode(
    'ChVDb21taXREaWZmVGFza0NvbnRleHQSDgoCaWQYASABKAlSAmlkEhYKBnN0cmVhbRgCIAEoCV'
    'IGc3RyZWFtEiEKDHN1YmplY3RfdHlwZRgDIAEoCVILc3ViamVjdFR5cGUSHQoKc3ViamVjdF9p'
    'ZBgEIAEoCVIJc3ViamVjdElkEiIKDW93bmVyX3BlZXJfaWQYBSABKAlSC293bmVyUGVlcklkEh'
    'YKBnN0YXR1cxgGIAEoCVIGc3RhdHVzEhQKBXRpdGxlGAcgASgJUgV0aXRsZRIYCgdtZXNzYWdl'
    'GAggASgJUgdtZXNzYWdlEhoKCHByb2dyZXNzGAkgASgFUghwcm9ncmVzcxIlCg5jaGFuZ2Vfc2'
    '91cmNlcxgKIAMoCVINY2hhbmdlU291cmNlcxIfCgtldmVudF9jb3VudBgLIAEoBVIKZXZlbnRD'
    'b3VudBIYCgdzdW1tYXJ5GAwgASgJUgdzdW1tYXJ5');

@$core.Deprecated('Use commitDiffDescriptor instead')
const CommitDiff$json = {
  '1': 'CommitDiff',
  '2': [
    {'1': 'base_hash', '3': 1, '4': 1, '5': 9, '10': 'baseHash'},
    {'1': 'target_hash', '3': 2, '4': 1, '5': 9, '10': 'targetHash'},
    {
      '1': 'tables',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitDiffTable',
      '10': 'tables'
    },
    {'1': 'cue', '3': 4, '4': 1, '5': 9, '10': 'cue'},
    {'1': 'truncated', '3': 5, '4': 1, '5': 8, '10': 'truncated'},
    {'1': 'message', '3': 6, '4': 1, '5': 9, '10': 'message'},
    {'1': 'unified_diff', '3': 7, '4': 1, '5': 9, '10': 'unifiedDiff'},
    {
      '1': 'related_tasks',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.apic.CommitDiffTaskContext',
      '10': 'relatedTasks'
    },
    {'1': 'sql', '3': 9, '4': 1, '5': 9, '10': 'sql'},
  ],
};

/// Descriptor for `CommitDiff`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commitDiffDescriptor = $convert.base64Decode(
    'CgpDb21taXREaWZmEhsKCWJhc2VfaGFzaBgBIAEoCVIIYmFzZUhhc2gSHwoLdGFyZ2V0X2hhc2'
    'gYAiABKAlSCnRhcmdldEhhc2gSLQoGdGFibGVzGAMgAygLMhUuYXBpYy5Db21taXREaWZmVGFi'
    'bGVSBnRhYmxlcxIQCgNjdWUYBCABKAlSA2N1ZRIcCgl0cnVuY2F0ZWQYBSABKAhSCXRydW5jYX'
    'RlZBIYCgdtZXNzYWdlGAYgASgJUgdtZXNzYWdlEiEKDHVuaWZpZWRfZGlmZhgHIAEoCVILdW5p'
    'ZmllZERpZmYSQAoNcmVsYXRlZF90YXNrcxgIIAMoCzIbLmFwaWMuQ29tbWl0RGlmZlRhc2tDb2'
    '50ZXh0UgxyZWxhdGVkVGFza3MSEAoDc3FsGAkgASgJUgNzcWw=');

@$core.Deprecated('Use getCommitDiffRequestDescriptor instead')
const GetCommitDiffRequest$json = {
  '1': 'GetCommitDiffRequest',
  '2': [
    {'1': 'commit_hash', '3': 1, '4': 1, '5': 9, '10': 'commitHash'},
    {'1': 'base_hash', '3': 2, '4': 1, '5': 9, '10': 'baseHash'},
    {'1': 'remote', '3': 3, '4': 1, '5': 9, '10': 'remote'},
  ],
};

/// Descriptor for `GetCommitDiffRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCommitDiffRequestDescriptor = $convert.base64Decode(
    'ChRHZXRDb21taXREaWZmUmVxdWVzdBIfCgtjb21taXRfaGFzaBgBIAEoCVIKY29tbWl0SGFzaB'
    'IbCgliYXNlX2hhc2gYAiABKAlSCGJhc2VIYXNoEhYKBnJlbW90ZRgDIAEoCVIGcmVtb3Rl');

@$core.Deprecated('Use getCommitDiffResponseDescriptor instead')
const GetCommitDiffResponse$json = {
  '1': 'GetCommitDiffResponse',
  '2': [
    {
      '1': 'diff',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.apic.CommitDiff',
      '10': 'diff'
    },
  ],
};

/// Descriptor for `GetCommitDiffResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getCommitDiffResponseDescriptor = $convert.base64Decode(
    'ChVHZXRDb21taXREaWZmUmVzcG9uc2USJAoEZGlmZhgBIAEoCzIQLmFwaWMuQ29tbWl0RGlmZl'
    'IEZGlmZg==');

@$core.Deprecated('Use sqlCellDescriptor instead')
const SqlCell$json = {
  '1': 'SqlCell',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 9, '10': 'value'},
    {'1': 'is_null', '3': 2, '4': 1, '5': 8, '10': 'isNull'},
  ],
};

/// Descriptor for `SqlCell`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sqlCellDescriptor = $convert.base64Decode(
    'CgdTcWxDZWxsEhQKBXZhbHVlGAEgASgJUgV2YWx1ZRIXCgdpc19udWxsGAIgASgIUgZpc051bG'
    'w=');

@$core.Deprecated('Use sqlRowDescriptor instead')
const SqlRow$json = {
  '1': 'SqlRow',
  '2': [
    {
      '1': 'cells',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.apic.SqlCell',
      '10': 'cells'
    },
  ],
};

/// Descriptor for `SqlRow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sqlRowDescriptor = $convert.base64Decode(
    'CgZTcWxSb3cSIwoFY2VsbHMYASADKAsyDS5hcGljLlNxbENlbGxSBWNlbGxz');

@$core.Deprecated('Use executeSqlRequestDescriptor instead')
const ExecuteSqlRequest$json = {
  '1': 'ExecuteSqlRequest',
  '2': [
    {'1': 'sql', '3': 1, '4': 1, '5': 9, '10': 'sql'},
    {'1': 'max_rows', '3': 2, '4': 1, '5': 5, '10': 'maxRows'},
  ],
};

/// Descriptor for `ExecuteSqlRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List executeSqlRequestDescriptor = $convert.base64Decode(
    'ChFFeGVjdXRlU3FsUmVxdWVzdBIQCgNzcWwYASABKAlSA3NxbBIZCghtYXhfcm93cxgCIAEoBV'
    'IHbWF4Um93cw==');

@$core.Deprecated('Use executeSqlResponseDescriptor instead')
const ExecuteSqlResponse$json = {
  '1': 'ExecuteSqlResponse',
  '2': [
    {'1': 'columns', '3': 1, '4': 3, '5': 9, '10': 'columns'},
    {'1': 'rows', '3': 2, '4': 3, '5': 11, '6': '.apic.SqlRow', '10': 'rows'},
    {'1': 'rows_affected', '3': 3, '4': 1, '5': 3, '10': 'rowsAffected'},
    {'1': 'truncated', '3': 4, '4': 1, '5': 8, '10': 'truncated'},
    {'1': 'message', '3': 5, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ExecuteSqlResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List executeSqlResponseDescriptor = $convert.base64Decode(
    'ChJFeGVjdXRlU3FsUmVzcG9uc2USGAoHY29sdW1ucxgBIAMoCVIHY29sdW1ucxIgCgRyb3dzGA'
    'IgAygLMgwuYXBpYy5TcWxSb3dSBHJvd3MSIwoNcm93c19hZmZlY3RlZBgDIAEoA1IMcm93c0Fm'
    'ZmVjdGVkEhwKCXRydW5jYXRlZBgEIAEoCFIJdHJ1bmNhdGVkEhgKB21lc3NhZ2UYBSABKAlSB2'
    '1lc3NhZ2U=');

const $core.Map<$core.String, $core.dynamic> ProtosClientApiServiceBase$json = {
  '1': 'ProtosClientApi',
  '2': [
    {'1': 'Init', '2': '.apic.InitRequest', '3': '.apic.InitResponse', '4': {}},
    {
      '1': 'GetUserDevices',
      '2': '.apic.GetUserDevicesRequest',
      '3': '.apic.GetUserDevicesResponse',
      '4': {}
    },
    {
      '1': 'GetUserInfo',
      '2': '.apic.GetUserInfoRequest',
      '3': '.apic.GetUserInfoResponse',
      '4': {}
    },
    {
      '1': 'ListOrganisations',
      '2': '.apic.ListOrganisationsRequest',
      '3': '.apic.ListOrganisationsResponse',
      '4': {}
    },
    {
      '1': 'StartDeviceInvite',
      '2': '.apic.StartDeviceInviteRequest',
      '3': '.apic.StartDeviceInviteResponse',
      '4': {}
    },
    {
      '1': 'ListNearbyOrganisations',
      '2': '.apic.ListNearbyOrganisationsRequest',
      '3': '.apic.ListNearbyOrganisationsResponse',
      '4': {}
    },
    {
      '1': 'JoinOrganisation',
      '2': '.apic.JoinOrganisationRequest',
      '3': '.apic.JoinOrganisationResponse',
      '4': {}
    },
    {
      '1': 'GetLocalSSHKey',
      '2': '.apic.GetLocalSSHKeyRequest',
      '3': '.apic.GetLocalSSHKeyResponse',
      '4': {}
    },
    {
      '1': 'GetApps',
      '2': '.apic.GetAppsRequest',
      '3': '.apic.GetAppsResponse',
      '4': {}
    },
    {
      '1': 'CreateApp',
      '2': '.apic.CreateAppRequest',
      '3': '.apic.CreateAppResponse',
      '4': {}
    },
    {
      '1': 'StartApp',
      '2': '.apic.StartAppRequest',
      '3': '.apic.StartAppResponse',
      '4': {}
    },
    {
      '1': 'StopApp',
      '2': '.apic.StopAppRequest',
      '3': '.apic.StopAppResponse',
      '4': {}
    },
    {
      '1': 'RemoveApp',
      '2': '.apic.RemoveAppRequest',
      '3': '.apic.RemoveAppResponse',
      '4': {}
    },
    {
      '1': 'GetAppLogs',
      '2': '.apic.GetAppLogsRequest',
      '3': '.apic.GetAppLogsResponse',
      '4': {}
    },
    {
      '1': 'GetSupportedCloudProviders',
      '2': '.apic.GetSupportedCloudProvidersRequest',
      '3': '.apic.GetSupportedCloudProvidersResponse',
      '4': {}
    },
    {
      '1': 'GetCloudProviders',
      '2': '.apic.GetCloudProvidersRequest',
      '3': '.apic.GetCloudProvidersResponse',
      '4': {}
    },
    {
      '1': 'GetCloudProvider',
      '2': '.apic.GetCloudProviderRequest',
      '3': '.apic.GetCloudProviderResponse',
      '4': {}
    },
    {
      '1': 'AddCloudProvider',
      '2': '.apic.AddCloudProviderRequest',
      '3': '.apic.AddCloudProviderResponse',
      '4': {}
    },
    {
      '1': 'RemoveCloudProvider',
      '2': '.apic.RemoveCloudProviderRequest',
      '3': '.apic.RemoveCloudProviderResponse',
      '4': {}
    },
    {
      '1': 'GetSupportedProvisioners',
      '2': '.apic.GetSupportedProvisionersRequest',
      '3': '.apic.GetSupportedProvisionersResponse',
      '4': {}
    },
    {
      '1': 'GetProvisioners',
      '2': '.apic.GetProvisionersRequest',
      '3': '.apic.GetProvisionersResponse',
      '4': {}
    },
    {
      '1': 'GetProvisioner',
      '2': '.apic.GetProvisionerRequest',
      '3': '.apic.GetProvisionerResponse',
      '4': {}
    },
    {
      '1': 'AddProvisioner',
      '2': '.apic.AddProvisionerRequest',
      '3': '.apic.AddProvisionerResponse',
      '4': {}
    },
    {
      '1': 'RemoveProvisioner',
      '2': '.apic.RemoveProvisionerRequest',
      '3': '.apic.RemoveProvisionerResponse',
      '4': {}
    },
    {
      '1': 'GetInstances',
      '2': '.apic.GetInstancesRequest',
      '3': '.apic.GetInstancesResponse',
      '4': {}
    },
    {
      '1': 'GetInstance',
      '2': '.apic.GetInstanceRequest',
      '3': '.apic.GetInstanceResponse',
      '4': {}
    },
    {
      '1': 'GetInstanceDeployOptions',
      '2': '.apic.GetInstanceDeployOptionsRequest',
      '3': '.apic.GetInstanceDeployOptionsResponse',
      '4': {}
    },
    {
      '1': 'DeployInstance',
      '2': '.apic.DeployInstanceRequest',
      '3': '.apic.DeployInstanceResponse',
      '4': {}
    },
    {
      '1': 'RemoveInstance',
      '2': '.apic.RemoveInstanceRequest',
      '3': '.apic.RemoveInstanceResponse',
      '4': {}
    },
    {
      '1': 'StartInstance',
      '2': '.apic.StartInstanceRequest',
      '3': '.apic.StartInstanceResponse',
      '4': {}
    },
    {
      '1': 'StopInstance',
      '2': '.apic.StopInstanceRequest',
      '3': '.apic.StopInstanceResponse',
      '4': {}
    },
    {
      '1': 'GetInstanceKey',
      '2': '.apic.GetInstanceKeyRequest',
      '3': '.apic.GetInstanceKeyResponse',
      '4': {}
    },
    {
      '1': 'GetInstanceLogs',
      '2': '.apic.GetInstanceLogsRequest',
      '3': '.apic.GetInstanceLogsResponse',
      '4': {}
    },
    {
      '1': 'InitInstance',
      '2': '.apic.InitInstanceRequest',
      '3': '.apic.InitInstanceResponse',
      '4': {}
    },
    {
      '1': 'UpdateInstance',
      '2': '.apic.UpdateInstanceRequest',
      '3': '.apic.UpdateInstanceResponse',
      '4': {}
    },
    {
      '1': 'GetNetworkState',
      '2': '.apic.GetNetworkStateRequest',
      '3': '.apic.GetNetworkStateResponse',
      '4': {}
    },
    {
      '1': 'SetNetworkEnabled',
      '2': '.apic.SetNetworkEnabledRequest',
      '3': '.apic.SetNetworkEnabledResponse',
      '4': {}
    },
    {
      '1': 'GetExitRoutes',
      '2': '.apic.GetExitRoutesRequest',
      '3': '.apic.GetExitRoutesResponse',
      '4': {}
    },
    {
      '1': 'GetMobileTunnelConfig',
      '2': '.apic.GetMobileTunnelConfigRequest',
      '3': '.apic.GetMobileTunnelConfigResponse',
      '4': {}
    },
    {
      '1': 'GetRuntimeState',
      '2': '.apic.GetRuntimeStateRequest',
      '3': '.apic.GetRuntimeStateResponse',
      '4': {}
    },
    {
      '1': 'WatchChanges',
      '2': '.apic.WatchChangesRequest',
      '3': '.apic.WatchChangesResponse',
      '4': {},
      '6': true
    },
    {
      '1': 'GetTasks',
      '2': '.apic.GetTasksRequest',
      '3': '.apic.GetTasksResponse',
      '4': {}
    },
    {
      '1': 'GetTask',
      '2': '.apic.GetTaskRequest',
      '3': '.apic.GetTaskResponse',
      '4': {}
    },
    {
      '1': 'WatchTask',
      '2': '.apic.WatchTaskRequest',
      '3': '.apic.WatchTaskResponse',
      '4': {},
      '6': true
    },
    {
      '1': 'SetExitRoute',
      '2': '.apic.SetExitRouteRequest',
      '3': '.apic.SetExitRouteResponse',
      '4': {}
    },
    {
      '1': 'ClearExitRoute',
      '2': '.apic.ClearExitRouteRequest',
      '3': '.apic.ClearExitRouteResponse',
      '4': {}
    },
    {
      '1': 'GetProtosdReleases',
      '2': '.apic.GetProtosdReleasesRequest',
      '3': '.apic.GetProtosdReleasesResponse',
      '4': {}
    },
    {
      '1': 'GetCloudImages',
      '2': '.apic.GetCloudImagesRequest',
      '3': '.apic.GetCloudImagesResponse',
      '4': {}
    },
    {
      '1': 'UploadCloudImage',
      '2': '.apic.UploadCloudImageRequest',
      '3': '.apic.UploadCloudImageResponse',
      '4': {}
    },
    {
      '1': 'RemoveCloudImage',
      '2': '.apic.RemoveCloudImageRequest',
      '3': '.apic.RemoveCloudImageResponse',
      '4': {}
    },
    {
      '1': 'GetProvisionerImages',
      '2': '.apic.GetProvisionerImagesRequest',
      '3': '.apic.GetProvisionerImagesResponse',
      '4': {}
    },
    {
      '1': 'UploadProvisionerImage',
      '2': '.apic.UploadProvisionerImageRequest',
      '3': '.apic.UploadProvisionerImageResponse',
      '4': {}
    },
    {
      '1': 'RemoveProvisionerImage',
      '2': '.apic.RemoveProvisionerImageRequest',
      '3': '.apic.RemoveProvisionerImageResponse',
      '4': {}
    },
    {
      '1': 'GetInstanceImage',
      '2': '.apic.GetInstanceImageRequest',
      '3': '.apic.GetInstanceImageResponse',
      '4': {}
    },
    {
      '1': 'UploadInstanceImageArchive',
      '2': '.apic.UploadInstanceImageArchiveRequest',
      '3': '.apic.UploadInstanceImageArchiveResponse',
      '4': {}
    },
    {
      '1': 'GetSystemStatus',
      '2': '.apic.GetSystemStatusRequest',
      '3': '.apic.GetSystemStatusResponse',
      '4': {}
    },
    {
      '1': 'StartHostAgent',
      '2': '.apic.StartHostAgentRequest',
      '3': '.apic.StartHostAgentResponse',
      '4': {}
    },
    {
      '1': 'StopHostAgent',
      '2': '.apic.StopHostAgentRequest',
      '3': '.apic.StopHostAgentResponse',
      '4': {}
    },
    {
      '1': 'GetLocalCommits',
      '2': '.apic.GetLocalCommitsRequest',
      '3': '.apic.GetLocalCommitsResponse',
      '4': {}
    },
    {
      '1': 'GetRemoteCommits',
      '2': '.apic.GetRemoteCommitsRequest',
      '3': '.apic.GetRemoteCommitsResponse',
      '4': {}
    },
    {
      '1': 'GetCommitDiff',
      '2': '.apic.GetCommitDiffRequest',
      '3': '.apic.GetCommitDiffResponse',
      '4': {}
    },
    {
      '1': 'ExecuteSql',
      '2': '.apic.ExecuteSqlRequest',
      '3': '.apic.ExecuteSqlResponse',
      '4': {}
    },
  ],
};

@$core.Deprecated('Use protosClientApiServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    ProtosClientApiServiceBase$messageJson = {
  '.apic.InitRequest': InitRequest$json,
  '.apic.InitResponse': InitResponse$json,
  '.apic.GetUserDevicesRequest': GetUserDevicesRequest$json,
  '.apic.GetUserDevicesResponse': GetUserDevicesResponse$json,
  '.apic.UserDevice': UserDevice$json,
  '.apic.GetUserInfoRequest': GetUserInfoRequest$json,
  '.apic.GetUserInfoResponse': GetUserInfoResponse$json,
  '.apic.ListOrganisationsRequest': ListOrganisationsRequest$json,
  '.apic.ListOrganisationsResponse': ListOrganisationsResponse$json,
  '.apic.Organisation': Organisation$json,
  '.apic.StartDeviceInviteRequest': StartDeviceInviteRequest$json,
  '.apic.StartDeviceInviteResponse': StartDeviceInviteResponse$json,
  '.apic.ListNearbyOrganisationsRequest': ListNearbyOrganisationsRequest$json,
  '.apic.ListNearbyOrganisationsResponse': ListNearbyOrganisationsResponse$json,
  '.apic.NearbyOrganisation': NearbyOrganisation$json,
  '.apic.JoinOrganisationRequest': JoinOrganisationRequest$json,
  '.apic.JoinOrganisationResponse': JoinOrganisationResponse$json,
  '.apic.GetLocalSSHKeyRequest': GetLocalSSHKeyRequest$json,
  '.apic.GetLocalSSHKeyResponse': GetLocalSSHKeyResponse$json,
  '.apic.GetAppsRequest': GetAppsRequest$json,
  '.apic.GetAppsResponse': GetAppsResponse$json,
  '.apic.App': App$json,
  '.apic.CreateAppRequest': CreateAppRequest$json,
  '.apic.CreateAppResponse': CreateAppResponse$json,
  '.apic.StartAppRequest': StartAppRequest$json,
  '.apic.StartAppResponse': StartAppResponse$json,
  '.apic.StopAppRequest': StopAppRequest$json,
  '.apic.StopAppResponse': StopAppResponse$json,
  '.apic.RemoveAppRequest': RemoveAppRequest$json,
  '.apic.RemoveAppResponse': RemoveAppResponse$json,
  '.apic.GetAppLogsRequest': GetAppLogsRequest$json,
  '.apic.GetAppLogsResponse': GetAppLogsResponse$json,
  '.apic.GetSupportedCloudProvidersRequest':
      GetSupportedCloudProvidersRequest$json,
  '.apic.GetSupportedCloudProvidersResponse':
      GetSupportedCloudProvidersResponse$json,
  '.apic.CloudType': CloudType$json,
  '.apic.GetCloudProvidersRequest': GetCloudProvidersRequest$json,
  '.apic.GetCloudProvidersResponse': GetCloudProvidersResponse$json,
  '.apic.CloudProvider': CloudProvider$json,
  '.apic.CloudProvider.SupportedMachinesEntry':
      CloudProvider_SupportedMachinesEntry$json,
  '.apic.CloudMachineSpec': CloudMachineSpec$json,
  '.apic.GetCloudProviderRequest': GetCloudProviderRequest$json,
  '.apic.GetCloudProviderResponse': GetCloudProviderResponse$json,
  '.apic.AddCloudProviderRequest': AddCloudProviderRequest$json,
  '.apic.AddCloudProviderRequest.CredentialsEntry':
      AddCloudProviderRequest_CredentialsEntry$json,
  '.apic.AddCloudProviderResponse': AddCloudProviderResponse$json,
  '.apic.RemoveCloudProviderRequest': RemoveCloudProviderRequest$json,
  '.apic.RemoveCloudProviderResponse': RemoveCloudProviderResponse$json,
  '.apic.GetSupportedProvisionersRequest': GetSupportedProvisionersRequest$json,
  '.apic.GetSupportedProvisionersResponse':
      GetSupportedProvisionersResponse$json,
  '.apic.ProvisionerType': ProvisionerType$json,
  '.apic.GetProvisionersRequest': GetProvisionersRequest$json,
  '.apic.GetProvisionersResponse': GetProvisionersResponse$json,
  '.apic.Provisioner': Provisioner$json,
  '.apic.Provisioner.SupportedMachinesEntry':
      Provisioner_SupportedMachinesEntry$json,
  '.apic.ProvisionerMachineSpec': ProvisionerMachineSpec$json,
  '.apic.GetProvisionerRequest': GetProvisionerRequest$json,
  '.apic.GetProvisionerResponse': GetProvisionerResponse$json,
  '.apic.AddProvisionerRequest': AddProvisionerRequest$json,
  '.apic.AddProvisionerRequest.CredentialsEntry':
      AddProvisionerRequest_CredentialsEntry$json,
  '.apic.AddProvisionerResponse': AddProvisionerResponse$json,
  '.apic.RemoveProvisionerRequest': RemoveProvisionerRequest$json,
  '.apic.RemoveProvisionerResponse': RemoveProvisionerResponse$json,
  '.apic.GetInstancesRequest': GetInstancesRequest$json,
  '.apic.GetInstancesResponse': GetInstancesResponse$json,
  '.apic.CloudInstance': CloudInstance$json,
  '.apic.CloudInstance.PeersEntry': CloudInstance_PeersEntry$json,
  '.apic.GetInstanceRequest': GetInstanceRequest$json,
  '.apic.GetInstanceResponse': GetInstanceResponse$json,
  '.apic.GetInstanceDeployOptionsRequest': GetInstanceDeployOptionsRequest$json,
  '.apic.GetInstanceDeployOptionsResponse':
      GetInstanceDeployOptionsResponse$json,
  '.apic.InstanceDeployField': InstanceDeployField$json,
  '.apic.InstanceDeployFieldOption': InstanceDeployFieldOption$json,
  '.apic.DeployInstanceRequest': DeployInstanceRequest$json,
  '.apic.DeployInstanceResponse': DeployInstanceResponse$json,
  '.apic.RemoveInstanceRequest': RemoveInstanceRequest$json,
  '.apic.RemoveInstanceResponse': RemoveInstanceResponse$json,
  '.apic.StartInstanceRequest': StartInstanceRequest$json,
  '.apic.StartInstanceResponse': StartInstanceResponse$json,
  '.apic.StopInstanceRequest': StopInstanceRequest$json,
  '.apic.StopInstanceResponse': StopInstanceResponse$json,
  '.apic.GetInstanceKeyRequest': GetInstanceKeyRequest$json,
  '.apic.GetInstanceKeyResponse': GetInstanceKeyResponse$json,
  '.apic.GetInstanceLogsRequest': GetInstanceLogsRequest$json,
  '.apic.GetInstanceLogsResponse': GetInstanceLogsResponse$json,
  '.apic.InitInstanceRequest': InitInstanceRequest$json,
  '.apic.InitInstanceResponse': InitInstanceResponse$json,
  '.apic.UpdateInstanceRequest': UpdateInstanceRequest$json,
  '.apic.UpdateInstanceResponse': UpdateInstanceResponse$json,
  '.apic.GetNetworkStateRequest': GetNetworkStateRequest$json,
  '.apic.GetNetworkStateResponse': GetNetworkStateResponse$json,
  '.apic.NetworkState': NetworkState$json,
  '.apic.NetworkAddress': NetworkAddress$json,
  '.apic.NetworkRoute': NetworkRoute$json,
  '.apic.WireGuardPeer': WireGuardPeer$json,
  '.apic.FirewallTable': FirewallTable$json,
  '.apic.FirewallChain': FirewallChain$json,
  '.apic.FirewallRule': FirewallRule$json,
  '.apic.DNSState': DNSState$json,
  '.apic.NetworkInterface': NetworkInterface$json,
  '.apic.SetNetworkEnabledRequest': SetNetworkEnabledRequest$json,
  '.apic.SetNetworkEnabledResponse': SetNetworkEnabledResponse$json,
  '.apic.NetworkRuntimeStatus': NetworkRuntimeStatus$json,
  '.apic.GetExitRoutesRequest': GetExitRoutesRequest$json,
  '.apic.GetExitRoutesResponse': GetExitRoutesResponse$json,
  '.apic.ExitRoute': ExitRoute$json,
  '.apic.GetMobileTunnelConfigRequest': GetMobileTunnelConfigRequest$json,
  '.apic.GetMobileTunnelConfigResponse': GetMobileTunnelConfigResponse$json,
  '.apic.MobileTunnelConfig': MobileTunnelConfig$json,
  '.apic.GetRuntimeStateRequest': GetRuntimeStateRequest$json,
  '.apic.GetRuntimeStateResponse': GetRuntimeStateResponse$json,
  '.apic.RuntimeState': RuntimeState$json,
  '.apic.RuntimePeerStatus': RuntimePeerStatus$json,
  '.apic.RuntimePeerStatus.LastDialErrorsEntry':
      RuntimePeerStatus_LastDialErrorsEntry$json,
  '.apic.RuntimeCompatibility': RuntimeCompatibility$json,
  '.apic.WatchChangesRequest': WatchChangesRequest$json,
  '.apic.WatchChangesResponse': WatchChangesResponse$json,
  '.apic.GetTasksRequest': GetTasksRequest$json,
  '.apic.GetTasksResponse': GetTasksResponse$json,
  '.apic.Task': Task$json,
  '.apic.GetTaskRequest': GetTaskRequest$json,
  '.apic.GetTaskResponse': GetTaskResponse$json,
  '.apic.TaskEvent': TaskEvent$json,
  '.apic.WatchTaskRequest': WatchTaskRequest$json,
  '.apic.WatchTaskResponse': WatchTaskResponse$json,
  '.apic.TaskProgressUpdate': TaskProgressUpdate$json,
  '.apic.SetExitRouteRequest': SetExitRouteRequest$json,
  '.apic.SetExitRouteResponse': SetExitRouteResponse$json,
  '.apic.ClearExitRouteRequest': ClearExitRouteRequest$json,
  '.apic.ClearExitRouteResponse': ClearExitRouteResponse$json,
  '.apic.GetProtosdReleasesRequest': GetProtosdReleasesRequest$json,
  '.apic.GetProtosdReleasesResponse': GetProtosdReleasesResponse$json,
  '.apic.Release': Release$json,
  '.apic.Release.CloudImagesEntry': Release_CloudImagesEntry$json,
  '.apic.CloudImage': CloudImage$json,
  '.apic.GetCloudImagesRequest': GetCloudImagesRequest$json,
  '.apic.GetCloudImagesResponse': GetCloudImagesResponse$json,
  '.apic.GetCloudImagesResponse.CloudImagesEntry':
      GetCloudImagesResponse_CloudImagesEntry$json,
  '.apic.CloudSpecificImage': CloudSpecificImage$json,
  '.apic.UploadCloudImageRequest': UploadCloudImageRequest$json,
  '.apic.UploadCloudImageResponse': UploadCloudImageResponse$json,
  '.apic.RemoveCloudImageRequest': RemoveCloudImageRequest$json,
  '.apic.RemoveCloudImageResponse': RemoveCloudImageResponse$json,
  '.apic.GetProvisionerImagesRequest': GetProvisionerImagesRequest$json,
  '.apic.GetProvisionerImagesResponse': GetProvisionerImagesResponse$json,
  '.apic.GetProvisionerImagesResponse.ImagesEntry':
      GetProvisionerImagesResponse_ImagesEntry$json,
  '.apic.UploadProvisionerImageRequest': UploadProvisionerImageRequest$json,
  '.apic.UploadProvisionerImageResponse': UploadProvisionerImageResponse$json,
  '.apic.RemoveProvisionerImageRequest': RemoveProvisionerImageRequest$json,
  '.apic.RemoveProvisionerImageResponse': RemoveProvisionerImageResponse$json,
  '.apic.GetInstanceImageRequest': GetInstanceImageRequest$json,
  '.apic.GetInstanceImageResponse': GetInstanceImageResponse$json,
  '.apic.GetInstanceImageResponse.LabelsEntry':
      GetInstanceImageResponse_LabelsEntry$json,
  '.apic.ImageContentDescriptor': ImageContentDescriptor$json,
  '.apic.ImageContentDescriptor.AnnotationsEntry':
      ImageContentDescriptor_AnnotationsEntry$json,
  '.apic.UploadInstanceImageArchiveRequest':
      UploadInstanceImageArchiveRequest$json,
  '.apic.UploadInstanceImageArchiveResponse':
      UploadInstanceImageArchiveResponse$json,
  '.apic.GetSystemStatusRequest': GetSystemStatusRequest$json,
  '.apic.GetSystemStatusResponse': GetSystemStatusResponse$json,
  '.apic.SystemStatus': SystemStatus$json,
  '.apic.CoreEndpoint': CoreEndpoint$json,
  '.apic.HostAgentConnectionStatus': HostAgentConnectionStatus$json,
  '.apic.StartHostAgentRequest': StartHostAgentRequest$json,
  '.apic.StartHostAgentResponse': StartHostAgentResponse$json,
  '.apic.StopHostAgentRequest': StopHostAgentRequest$json,
  '.apic.StopHostAgentResponse': StopHostAgentResponse$json,
  '.apic.GetLocalCommitsRequest': GetLocalCommitsRequest$json,
  '.apic.GetLocalCommitsResponse': GetLocalCommitsResponse$json,
  '.apic.Commit': Commit$json,
  '.apic.CommitGraph': CommitGraph$json,
  '.apic.CommitGraphItem': CommitGraphItem$json,
  '.apic.CommitGraphRelation': CommitGraphRelation$json,
  '.apic.GetRemoteCommitsRequest': GetRemoteCommitsRequest$json,
  '.apic.GetRemoteCommitsResponse': GetRemoteCommitsResponse$json,
  '.apic.GetCommitDiffRequest': GetCommitDiffRequest$json,
  '.apic.GetCommitDiffResponse': GetCommitDiffResponse$json,
  '.apic.CommitDiff': CommitDiff$json,
  '.apic.CommitDiffTable': CommitDiffTable$json,
  '.apic.CommitDiffRow': CommitDiffRow$json,
  '.apic.CommitDiffField': CommitDiffField$json,
  '.apic.CommitDiffValue': CommitDiffValue$json,
  '.apic.CommitDiffTaskContext': CommitDiffTaskContext$json,
  '.apic.ExecuteSqlRequest': ExecuteSqlRequest$json,
  '.apic.ExecuteSqlResponse': ExecuteSqlResponse$json,
  '.apic.SqlRow': SqlRow$json,
  '.apic.SqlCell': SqlCell$json,
};

/// Descriptor for `ProtosClientApi`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List protosClientApiServiceDescriptor = $convert.base64Decode(
    'Cg9Qcm90b3NDbGllbnRBcGkSLwoESW5pdBIRLmFwaWMuSW5pdFJlcXVlc3QaEi5hcGljLkluaX'
    'RSZXNwb25zZSIAEk0KDkdldFVzZXJEZXZpY2VzEhsuYXBpYy5HZXRVc2VyRGV2aWNlc1JlcXVl'
    'c3QaHC5hcGljLkdldFVzZXJEZXZpY2VzUmVzcG9uc2UiABJECgtHZXRVc2VySW5mbxIYLmFwaW'
    'MuR2V0VXNlckluZm9SZXF1ZXN0GhkuYXBpYy5HZXRVc2VySW5mb1Jlc3BvbnNlIgASVgoRTGlz'
    'dE9yZ2FuaXNhdGlvbnMSHi5hcGljLkxpc3RPcmdhbmlzYXRpb25zUmVxdWVzdBofLmFwaWMuTG'
    'lzdE9yZ2FuaXNhdGlvbnNSZXNwb25zZSIAElYKEVN0YXJ0RGV2aWNlSW52aXRlEh4uYXBpYy5T'
    'dGFydERldmljZUludml0ZVJlcXVlc3QaHy5hcGljLlN0YXJ0RGV2aWNlSW52aXRlUmVzcG9uc2'
    'UiABJoChdMaXN0TmVhcmJ5T3JnYW5pc2F0aW9ucxIkLmFwaWMuTGlzdE5lYXJieU9yZ2FuaXNh'
    'dGlvbnNSZXF1ZXN0GiUuYXBpYy5MaXN0TmVhcmJ5T3JnYW5pc2F0aW9uc1Jlc3BvbnNlIgASUw'
    'oQSm9pbk9yZ2FuaXNhdGlvbhIdLmFwaWMuSm9pbk9yZ2FuaXNhdGlvblJlcXVlc3QaHi5hcGlj'
    'LkpvaW5PcmdhbmlzYXRpb25SZXNwb25zZSIAEk0KDkdldExvY2FsU1NIS2V5EhsuYXBpYy5HZX'
    'RMb2NhbFNTSEtleVJlcXVlc3QaHC5hcGljLkdldExvY2FsU1NIS2V5UmVzcG9uc2UiABI4CgdH'
    'ZXRBcHBzEhQuYXBpYy5HZXRBcHBzUmVxdWVzdBoVLmFwaWMuR2V0QXBwc1Jlc3BvbnNlIgASPg'
    'oJQ3JlYXRlQXBwEhYuYXBpYy5DcmVhdGVBcHBSZXF1ZXN0GhcuYXBpYy5DcmVhdGVBcHBSZXNw'
    'b25zZSIAEjsKCFN0YXJ0QXBwEhUuYXBpYy5TdGFydEFwcFJlcXVlc3QaFi5hcGljLlN0YXJ0QX'
    'BwUmVzcG9uc2UiABI4CgdTdG9wQXBwEhQuYXBpYy5TdG9wQXBwUmVxdWVzdBoVLmFwaWMuU3Rv'
    'cEFwcFJlc3BvbnNlIgASPgoJUmVtb3ZlQXBwEhYuYXBpYy5SZW1vdmVBcHBSZXF1ZXN0GhcuYX'
    'BpYy5SZW1vdmVBcHBSZXNwb25zZSIAEkEKCkdldEFwcExvZ3MSFy5hcGljLkdldEFwcExvZ3NS'
    'ZXF1ZXN0GhguYXBpYy5HZXRBcHBMb2dzUmVzcG9uc2UiABJxChpHZXRTdXBwb3J0ZWRDbG91ZF'
    'Byb3ZpZGVycxInLmFwaWMuR2V0U3VwcG9ydGVkQ2xvdWRQcm92aWRlcnNSZXF1ZXN0GiguYXBp'
    'Yy5HZXRTdXBwb3J0ZWRDbG91ZFByb3ZpZGVyc1Jlc3BvbnNlIgASVgoRR2V0Q2xvdWRQcm92aW'
    'RlcnMSHi5hcGljLkdldENsb3VkUHJvdmlkZXJzUmVxdWVzdBofLmFwaWMuR2V0Q2xvdWRQcm92'
    'aWRlcnNSZXNwb25zZSIAElMKEEdldENsb3VkUHJvdmlkZXISHS5hcGljLkdldENsb3VkUHJvdm'
    'lkZXJSZXF1ZXN0Gh4uYXBpYy5HZXRDbG91ZFByb3ZpZGVyUmVzcG9uc2UiABJTChBBZGRDbG91'
    'ZFByb3ZpZGVyEh0uYXBpYy5BZGRDbG91ZFByb3ZpZGVyUmVxdWVzdBoeLmFwaWMuQWRkQ2xvdW'
    'RQcm92aWRlclJlc3BvbnNlIgASXAoTUmVtb3ZlQ2xvdWRQcm92aWRlchIgLmFwaWMuUmVtb3Zl'
    'Q2xvdWRQcm92aWRlclJlcXVlc3QaIS5hcGljLlJlbW92ZUNsb3VkUHJvdmlkZXJSZXNwb25zZS'
    'IAEmsKGEdldFN1cHBvcnRlZFByb3Zpc2lvbmVycxIlLmFwaWMuR2V0U3VwcG9ydGVkUHJvdmlz'
    'aW9uZXJzUmVxdWVzdBomLmFwaWMuR2V0U3VwcG9ydGVkUHJvdmlzaW9uZXJzUmVzcG9uc2UiAB'
    'JQCg9HZXRQcm92aXNpb25lcnMSHC5hcGljLkdldFByb3Zpc2lvbmVyc1JlcXVlc3QaHS5hcGlj'
    'LkdldFByb3Zpc2lvbmVyc1Jlc3BvbnNlIgASTQoOR2V0UHJvdmlzaW9uZXISGy5hcGljLkdldF'
    'Byb3Zpc2lvbmVyUmVxdWVzdBocLmFwaWMuR2V0UHJvdmlzaW9uZXJSZXNwb25zZSIAEk0KDkFk'
    'ZFByb3Zpc2lvbmVyEhsuYXBpYy5BZGRQcm92aXNpb25lclJlcXVlc3QaHC5hcGljLkFkZFByb3'
    'Zpc2lvbmVyUmVzcG9uc2UiABJWChFSZW1vdmVQcm92aXNpb25lchIeLmFwaWMuUmVtb3ZlUHJv'
    'dmlzaW9uZXJSZXF1ZXN0Gh8uYXBpYy5SZW1vdmVQcm92aXNpb25lclJlc3BvbnNlIgASRwoMR2'
    'V0SW5zdGFuY2VzEhkuYXBpYy5HZXRJbnN0YW5jZXNSZXF1ZXN0GhouYXBpYy5HZXRJbnN0YW5j'
    'ZXNSZXNwb25zZSIAEkQKC0dldEluc3RhbmNlEhguYXBpYy5HZXRJbnN0YW5jZVJlcXVlc3QaGS'
    '5hcGljLkdldEluc3RhbmNlUmVzcG9uc2UiABJrChhHZXRJbnN0YW5jZURlcGxveU9wdGlvbnMS'
    'JS5hcGljLkdldEluc3RhbmNlRGVwbG95T3B0aW9uc1JlcXVlc3QaJi5hcGljLkdldEluc3Rhbm'
    'NlRGVwbG95T3B0aW9uc1Jlc3BvbnNlIgASTQoORGVwbG95SW5zdGFuY2USGy5hcGljLkRlcGxv'
    'eUluc3RhbmNlUmVxdWVzdBocLmFwaWMuRGVwbG95SW5zdGFuY2VSZXNwb25zZSIAEk0KDlJlbW'
    '92ZUluc3RhbmNlEhsuYXBpYy5SZW1vdmVJbnN0YW5jZVJlcXVlc3QaHC5hcGljLlJlbW92ZUlu'
    'c3RhbmNlUmVzcG9uc2UiABJKCg1TdGFydEluc3RhbmNlEhouYXBpYy5TdGFydEluc3RhbmNlUm'
    'VxdWVzdBobLmFwaWMuU3RhcnRJbnN0YW5jZVJlc3BvbnNlIgASRwoMU3RvcEluc3RhbmNlEhku'
    'YXBpYy5TdG9wSW5zdGFuY2VSZXF1ZXN0GhouYXBpYy5TdG9wSW5zdGFuY2VSZXNwb25zZSIAEk'
    '0KDkdldEluc3RhbmNlS2V5EhsuYXBpYy5HZXRJbnN0YW5jZUtleVJlcXVlc3QaHC5hcGljLkdl'
    'dEluc3RhbmNlS2V5UmVzcG9uc2UiABJQCg9HZXRJbnN0YW5jZUxvZ3MSHC5hcGljLkdldEluc3'
    'RhbmNlTG9nc1JlcXVlc3QaHS5hcGljLkdldEluc3RhbmNlTG9nc1Jlc3BvbnNlIgASRwoMSW5p'
    'dEluc3RhbmNlEhkuYXBpYy5Jbml0SW5zdGFuY2VSZXF1ZXN0GhouYXBpYy5Jbml0SW5zdGFuY2'
    'VSZXNwb25zZSIAEk0KDlVwZGF0ZUluc3RhbmNlEhsuYXBpYy5VcGRhdGVJbnN0YW5jZVJlcXVl'
    'c3QaHC5hcGljLlVwZGF0ZUluc3RhbmNlUmVzcG9uc2UiABJQCg9HZXROZXR3b3JrU3RhdGUSHC'
    '5hcGljLkdldE5ldHdvcmtTdGF0ZVJlcXVlc3QaHS5hcGljLkdldE5ldHdvcmtTdGF0ZVJlc3Bv'
    'bnNlIgASVgoRU2V0TmV0d29ya0VuYWJsZWQSHi5hcGljLlNldE5ldHdvcmtFbmFibGVkUmVxdW'
    'VzdBofLmFwaWMuU2V0TmV0d29ya0VuYWJsZWRSZXNwb25zZSIAEkoKDUdldEV4aXRSb3V0ZXMS'
    'Gi5hcGljLkdldEV4aXRSb3V0ZXNSZXF1ZXN0GhsuYXBpYy5HZXRFeGl0Um91dGVzUmVzcG9uc2'
    'UiABJiChVHZXRNb2JpbGVUdW5uZWxDb25maWcSIi5hcGljLkdldE1vYmlsZVR1bm5lbENvbmZp'
    'Z1JlcXVlc3QaIy5hcGljLkdldE1vYmlsZVR1bm5lbENvbmZpZ1Jlc3BvbnNlIgASUAoPR2V0Un'
    'VudGltZVN0YXRlEhwuYXBpYy5HZXRSdW50aW1lU3RhdGVSZXF1ZXN0Gh0uYXBpYy5HZXRSdW50'
    'aW1lU3RhdGVSZXNwb25zZSIAEkkKDFdhdGNoQ2hhbmdlcxIZLmFwaWMuV2F0Y2hDaGFuZ2VzUm'
    'VxdWVzdBoaLmFwaWMuV2F0Y2hDaGFuZ2VzUmVzcG9uc2UiADABEjsKCEdldFRhc2tzEhUuYXBp'
    'Yy5HZXRUYXNrc1JlcXVlc3QaFi5hcGljLkdldFRhc2tzUmVzcG9uc2UiABI4CgdHZXRUYXNrEh'
    'QuYXBpYy5HZXRUYXNrUmVxdWVzdBoVLmFwaWMuR2V0VGFza1Jlc3BvbnNlIgASQAoJV2F0Y2hU'
    'YXNrEhYuYXBpYy5XYXRjaFRhc2tSZXF1ZXN0GhcuYXBpYy5XYXRjaFRhc2tSZXNwb25zZSIAMA'
    'ESRwoMU2V0RXhpdFJvdXRlEhkuYXBpYy5TZXRFeGl0Um91dGVSZXF1ZXN0GhouYXBpYy5TZXRF'
    'eGl0Um91dGVSZXNwb25zZSIAEk0KDkNsZWFyRXhpdFJvdXRlEhsuYXBpYy5DbGVhckV4aXRSb3'
    'V0ZVJlcXVlc3QaHC5hcGljLkNsZWFyRXhpdFJvdXRlUmVzcG9uc2UiABJZChJHZXRQcm90b3Nk'
    'UmVsZWFzZXMSHy5hcGljLkdldFByb3Rvc2RSZWxlYXNlc1JlcXVlc3QaIC5hcGljLkdldFByb3'
    'Rvc2RSZWxlYXNlc1Jlc3BvbnNlIgASTQoOR2V0Q2xvdWRJbWFnZXMSGy5hcGljLkdldENsb3Vk'
    'SW1hZ2VzUmVxdWVzdBocLmFwaWMuR2V0Q2xvdWRJbWFnZXNSZXNwb25zZSIAElMKEFVwbG9hZE'
    'Nsb3VkSW1hZ2USHS5hcGljLlVwbG9hZENsb3VkSW1hZ2VSZXF1ZXN0Gh4uYXBpYy5VcGxvYWRD'
    'bG91ZEltYWdlUmVzcG9uc2UiABJTChBSZW1vdmVDbG91ZEltYWdlEh0uYXBpYy5SZW1vdmVDbG'
    '91ZEltYWdlUmVxdWVzdBoeLmFwaWMuUmVtb3ZlQ2xvdWRJbWFnZVJlc3BvbnNlIgASXwoUR2V0'
    'UHJvdmlzaW9uZXJJbWFnZXMSIS5hcGljLkdldFByb3Zpc2lvbmVySW1hZ2VzUmVxdWVzdBoiLm'
    'FwaWMuR2V0UHJvdmlzaW9uZXJJbWFnZXNSZXNwb25zZSIAEmUKFlVwbG9hZFByb3Zpc2lvbmVy'
    'SW1hZ2USIy5hcGljLlVwbG9hZFByb3Zpc2lvbmVySW1hZ2VSZXF1ZXN0GiQuYXBpYy5VcGxvYW'
    'RQcm92aXNpb25lckltYWdlUmVzcG9uc2UiABJlChZSZW1vdmVQcm92aXNpb25lckltYWdlEiMu'
    'YXBpYy5SZW1vdmVQcm92aXNpb25lckltYWdlUmVxdWVzdBokLmFwaWMuUmVtb3ZlUHJvdmlzaW'
    '9uZXJJbWFnZVJlc3BvbnNlIgASUwoQR2V0SW5zdGFuY2VJbWFnZRIdLmFwaWMuR2V0SW5zdGFu'
    'Y2VJbWFnZVJlcXVlc3QaHi5hcGljLkdldEluc3RhbmNlSW1hZ2VSZXNwb25zZSIAEnEKGlVwbG'
    '9hZEluc3RhbmNlSW1hZ2VBcmNoaXZlEicuYXBpYy5VcGxvYWRJbnN0YW5jZUltYWdlQXJjaGl2'
    'ZVJlcXVlc3QaKC5hcGljLlVwbG9hZEluc3RhbmNlSW1hZ2VBcmNoaXZlUmVzcG9uc2UiABJQCg'
    '9HZXRTeXN0ZW1TdGF0dXMSHC5hcGljLkdldFN5c3RlbVN0YXR1c1JlcXVlc3QaHS5hcGljLkdl'
    'dFN5c3RlbVN0YXR1c1Jlc3BvbnNlIgASTQoOU3RhcnRIb3N0QWdlbnQSGy5hcGljLlN0YXJ0SG'
    '9zdEFnZW50UmVxdWVzdBocLmFwaWMuU3RhcnRIb3N0QWdlbnRSZXNwb25zZSIAEkoKDVN0b3BI'
    'b3N0QWdlbnQSGi5hcGljLlN0b3BIb3N0QWdlbnRSZXF1ZXN0GhsuYXBpYy5TdG9wSG9zdEFnZW'
    '50UmVzcG9uc2UiABJQCg9HZXRMb2NhbENvbW1pdHMSHC5hcGljLkdldExvY2FsQ29tbWl0c1Jl'
    'cXVlc3QaHS5hcGljLkdldExvY2FsQ29tbWl0c1Jlc3BvbnNlIgASUwoQR2V0UmVtb3RlQ29tbW'
    'l0cxIdLmFwaWMuR2V0UmVtb3RlQ29tbWl0c1JlcXVlc3QaHi5hcGljLkdldFJlbW90ZUNvbW1p'
    'dHNSZXNwb25zZSIAEkoKDUdldENvbW1pdERpZmYSGi5hcGljLkdldENvbW1pdERpZmZSZXF1ZX'
    'N0GhsuYXBpYy5HZXRDb21taXREaWZmUmVzcG9uc2UiABJBCgpFeGVjdXRlU3FsEhcuYXBpYy5F'
    'eGVjdXRlU3FsUmVxdWVzdBoYLmFwaWMuRXhlY3V0ZVNxbFJlc3BvbnNlIgA=');
