// This is a generated file - do not edit.
//
// Generated from apic/proto/apic.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class InitRequest extends $pb.GeneratedMessage {
  factory InitRequest({
    $core.String? username,
    $core.String? name,
    $core.String? organisation,
  }) {
    final result = create();
    if (username != null) result.username = username;
    if (name != null) result.name = name;
    if (organisation != null) result.organisation = organisation;
    return result;
  }

  InitRequest._();

  factory InitRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InitRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InitRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'username')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'organisation')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitRequest copyWith(void Function(InitRequest) updates) =>
      super.copyWith((message) => updates(message as InitRequest))
          as InitRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InitRequest create() => InitRequest._();
  @$core.override
  InitRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InitRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InitRequest>(create);
  static InitRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get username => $_getSZ(0);
  @$pb.TagNumber(1)
  set username($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUsername() => $_has(0);
  @$pb.TagNumber(1)
  void clearUsername() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get organisation => $_getSZ(2);
  @$pb.TagNumber(3)
  set organisation($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOrganisation() => $_has(2);
  @$pb.TagNumber(3)
  void clearOrganisation() => $_clearField(3);
}

class InitResponse extends $pb.GeneratedMessage {
  factory InitResponse() => create();

  InitResponse._();

  factory InitResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InitResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InitResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitResponse copyWith(void Function(InitResponse) updates) =>
      super.copyWith((message) => updates(message as InitResponse))
          as InitResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InitResponse create() => InitResponse._();
  @$core.override
  InitResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InitResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InitResponse>(create);
  static InitResponse? _defaultInstance;
}

class UserDevice extends $pb.GeneratedMessage {
  factory UserDevice({
    $core.String? id,
    $core.String? name,
    $core.String? publicKey,
    $core.String? publicKeyWireguard,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (publicKey != null) result.publicKey = publicKey;
    if (publicKeyWireguard != null)
      result.publicKeyWireguard = publicKeyWireguard;
    return result;
  }

  UserDevice._();

  factory UserDevice.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UserDevice.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UserDevice',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'publicKey')
    ..aOS(4, _omitFieldNames ? '' : 'publicKeyWireguard')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UserDevice clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UserDevice copyWith(void Function(UserDevice) updates) =>
      super.copyWith((message) => updates(message as UserDevice)) as UserDevice;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UserDevice create() => UserDevice._();
  @$core.override
  UserDevice createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UserDevice getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UserDevice>(create);
  static UserDevice? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get publicKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set publicKey($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublicKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublicKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get publicKeyWireguard => $_getSZ(3);
  @$pb.TagNumber(4)
  set publicKeyWireguard($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPublicKeyWireguard() => $_has(3);
  @$pb.TagNumber(4)
  void clearPublicKeyWireguard() => $_clearField(4);
}

class GetUserDevicesRequest extends $pb.GeneratedMessage {
  factory GetUserDevicesRequest() => create();

  GetUserDevicesRequest._();

  factory GetUserDevicesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserDevicesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserDevicesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserDevicesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserDevicesRequest copyWith(
          void Function(GetUserDevicesRequest) updates) =>
      super.copyWith((message) => updates(message as GetUserDevicesRequest))
          as GetUserDevicesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserDevicesRequest create() => GetUserDevicesRequest._();
  @$core.override
  GetUserDevicesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserDevicesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserDevicesRequest>(create);
  static GetUserDevicesRequest? _defaultInstance;
}

class GetUserDevicesResponse extends $pb.GeneratedMessage {
  factory GetUserDevicesResponse({
    $core.Iterable<UserDevice>? devices,
  }) {
    final result = create();
    if (devices != null) result.devices.addAll(devices);
    return result;
  }

  GetUserDevicesResponse._();

  factory GetUserDevicesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserDevicesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserDevicesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<UserDevice>(1, _omitFieldNames ? '' : 'devices',
        subBuilder: UserDevice.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserDevicesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserDevicesResponse copyWith(
          void Function(GetUserDevicesResponse) updates) =>
      super.copyWith((message) => updates(message as GetUserDevicesResponse))
          as GetUserDevicesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserDevicesResponse create() => GetUserDevicesResponse._();
  @$core.override
  GetUserDevicesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserDevicesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserDevicesResponse>(create);
  static GetUserDevicesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<UserDevice> get devices => $_getList(0);
}

class GetUserInfoRequest extends $pb.GeneratedMessage {
  factory GetUserInfoRequest() => create();

  GetUserInfoRequest._();

  factory GetUserInfoRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserInfoRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserInfoRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoRequest copyWith(void Function(GetUserInfoRequest) updates) =>
      super.copyWith((message) => updates(message as GetUserInfoRequest))
          as GetUserInfoRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserInfoRequest create() => GetUserInfoRequest._();
  @$core.override
  GetUserInfoRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserInfoRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserInfoRequest>(create);
  static GetUserInfoRequest? _defaultInstance;
}

class GetUserInfoResponse extends $pb.GeneratedMessage {
  factory GetUserInfoResponse({
    $core.String? username,
    $core.String? name,
    $core.bool? isAdmin,
    $core.String? organisationId,
    $core.String? organisationName,
  }) {
    final result = create();
    if (username != null) result.username = username;
    if (name != null) result.name = name;
    if (isAdmin != null) result.isAdmin = isAdmin;
    if (organisationId != null) result.organisationId = organisationId;
    if (organisationName != null) result.organisationName = organisationName;
    return result;
  }

  GetUserInfoResponse._();

  factory GetUserInfoResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetUserInfoResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetUserInfoResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'username')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOB(3, _omitFieldNames ? '' : 'isAdmin')
    ..aOS(4, _omitFieldNames ? '' : 'organisationId')
    ..aOS(5, _omitFieldNames ? '' : 'organisationName')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetUserInfoResponse copyWith(void Function(GetUserInfoResponse) updates) =>
      super.copyWith((message) => updates(message as GetUserInfoResponse))
          as GetUserInfoResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetUserInfoResponse create() => GetUserInfoResponse._();
  @$core.override
  GetUserInfoResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetUserInfoResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetUserInfoResponse>(create);
  static GetUserInfoResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get username => $_getSZ(0);
  @$pb.TagNumber(1)
  set username($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUsername() => $_has(0);
  @$pb.TagNumber(1)
  void clearUsername() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get isAdmin => $_getBF(2);
  @$pb.TagNumber(3)
  set isAdmin($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIsAdmin() => $_has(2);
  @$pb.TagNumber(3)
  void clearIsAdmin() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get organisationId => $_getSZ(3);
  @$pb.TagNumber(4)
  set organisationId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOrganisationId() => $_has(3);
  @$pb.TagNumber(4)
  void clearOrganisationId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get organisationName => $_getSZ(4);
  @$pb.TagNumber(5)
  set organisationName($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOrganisationName() => $_has(4);
  @$pb.TagNumber(5)
  void clearOrganisationName() => $_clearField(5);
}

class Organisation extends $pb.GeneratedMessage {
  factory Organisation({
    $core.String? id,
    $core.String? name,
    $core.String? createdAt,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (createdAt != null) result.createdAt = createdAt;
    return result;
  }

  Organisation._();

  factory Organisation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Organisation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Organisation',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'createdAt')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Organisation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Organisation copyWith(void Function(Organisation) updates) =>
      super.copyWith((message) => updates(message as Organisation))
          as Organisation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Organisation create() => Organisation._();
  @$core.override
  Organisation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Organisation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Organisation>(create);
  static Organisation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get createdAt => $_getSZ(2);
  @$pb.TagNumber(3)
  set createdAt($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCreatedAt() => $_has(2);
  @$pb.TagNumber(3)
  void clearCreatedAt() => $_clearField(3);
}

class ListOrganisationsRequest extends $pb.GeneratedMessage {
  factory ListOrganisationsRequest() => create();

  ListOrganisationsRequest._();

  factory ListOrganisationsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOrganisationsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOrganisationsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOrganisationsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOrganisationsRequest copyWith(
          void Function(ListOrganisationsRequest) updates) =>
      super.copyWith((message) => updates(message as ListOrganisationsRequest))
          as ListOrganisationsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOrganisationsRequest create() => ListOrganisationsRequest._();
  @$core.override
  ListOrganisationsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOrganisationsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOrganisationsRequest>(create);
  static ListOrganisationsRequest? _defaultInstance;
}

class ListOrganisationsResponse extends $pb.GeneratedMessage {
  factory ListOrganisationsResponse({
    $core.Iterable<Organisation>? organisations,
  }) {
    final result = create();
    if (organisations != null) result.organisations.addAll(organisations);
    return result;
  }

  ListOrganisationsResponse._();

  factory ListOrganisationsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListOrganisationsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListOrganisationsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Organisation>(1, _omitFieldNames ? '' : 'organisations',
        subBuilder: Organisation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOrganisationsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListOrganisationsResponse copyWith(
          void Function(ListOrganisationsResponse) updates) =>
      super.copyWith((message) => updates(message as ListOrganisationsResponse))
          as ListOrganisationsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListOrganisationsResponse create() => ListOrganisationsResponse._();
  @$core.override
  ListOrganisationsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListOrganisationsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListOrganisationsResponse>(create);
  static ListOrganisationsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Organisation> get organisations => $_getList(0);
}

class StartDeviceInviteRequest extends $pb.GeneratedMessage {
  factory StartDeviceInviteRequest({
    $core.String? organisationId,
    $core.String? channel,
    $core.String? joinMode,
    $core.String? username,
  }) {
    final result = create();
    if (organisationId != null) result.organisationId = organisationId;
    if (channel != null) result.channel = channel;
    if (joinMode != null) result.joinMode = joinMode;
    if (username != null) result.username = username;
    return result;
  }

  StartDeviceInviteRequest._();

  factory StartDeviceInviteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartDeviceInviteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartDeviceInviteRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'organisationId')
    ..aOS(2, _omitFieldNames ? '' : 'channel')
    ..aOS(3, _omitFieldNames ? '' : 'joinMode')
    ..aOS(4, _omitFieldNames ? '' : 'username')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartDeviceInviteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartDeviceInviteRequest copyWith(
          void Function(StartDeviceInviteRequest) updates) =>
      super.copyWith((message) => updates(message as StartDeviceInviteRequest))
          as StartDeviceInviteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartDeviceInviteRequest create() => StartDeviceInviteRequest._();
  @$core.override
  StartDeviceInviteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartDeviceInviteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartDeviceInviteRequest>(create);
  static StartDeviceInviteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get organisationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set organisationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOrganisationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrganisationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get channel => $_getSZ(1);
  @$pb.TagNumber(2)
  set channel($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasChannel() => $_has(1);
  @$pb.TagNumber(2)
  void clearChannel() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get joinMode => $_getSZ(2);
  @$pb.TagNumber(3)
  set joinMode($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasJoinMode() => $_has(2);
  @$pb.TagNumber(3)
  void clearJoinMode() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get username => $_getSZ(3);
  @$pb.TagNumber(4)
  set username($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsername() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsername() => $_clearField(4);
}

class StartDeviceInviteResponse extends $pb.GeneratedMessage {
  factory StartDeviceInviteResponse({
    $core.String? inviteId,
    $fixnum.Int64? expiresAtUnix,
    $core.String? advertiseName,
    $core.String? advertiseService,
    $core.String? channel,
    $core.String? verificationCode,
    $core.String? joinMode,
  }) {
    final result = create();
    if (inviteId != null) result.inviteId = inviteId;
    if (expiresAtUnix != null) result.expiresAtUnix = expiresAtUnix;
    if (advertiseName != null) result.advertiseName = advertiseName;
    if (advertiseService != null) result.advertiseService = advertiseService;
    if (channel != null) result.channel = channel;
    if (verificationCode != null) result.verificationCode = verificationCode;
    if (joinMode != null) result.joinMode = joinMode;
    return result;
  }

  StartDeviceInviteResponse._();

  factory StartDeviceInviteResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartDeviceInviteResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartDeviceInviteResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'inviteId')
    ..aInt64(2, _omitFieldNames ? '' : 'expiresAtUnix')
    ..aOS(3, _omitFieldNames ? '' : 'advertiseName')
    ..aOS(4, _omitFieldNames ? '' : 'advertiseService')
    ..aOS(5, _omitFieldNames ? '' : 'channel')
    ..aOS(6, _omitFieldNames ? '' : 'verificationCode')
    ..aOS(7, _omitFieldNames ? '' : 'joinMode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartDeviceInviteResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartDeviceInviteResponse copyWith(
          void Function(StartDeviceInviteResponse) updates) =>
      super.copyWith((message) => updates(message as StartDeviceInviteResponse))
          as StartDeviceInviteResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartDeviceInviteResponse create() => StartDeviceInviteResponse._();
  @$core.override
  StartDeviceInviteResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartDeviceInviteResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartDeviceInviteResponse>(create);
  static StartDeviceInviteResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get inviteId => $_getSZ(0);
  @$pb.TagNumber(1)
  set inviteId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInviteId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInviteId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get expiresAtUnix => $_getI64(1);
  @$pb.TagNumber(2)
  set expiresAtUnix($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExpiresAtUnix() => $_has(1);
  @$pb.TagNumber(2)
  void clearExpiresAtUnix() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get advertiseName => $_getSZ(2);
  @$pb.TagNumber(3)
  set advertiseName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAdvertiseName() => $_has(2);
  @$pb.TagNumber(3)
  void clearAdvertiseName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get advertiseService => $_getSZ(3);
  @$pb.TagNumber(4)
  set advertiseService($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAdvertiseService() => $_has(3);
  @$pb.TagNumber(4)
  void clearAdvertiseService() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get channel => $_getSZ(4);
  @$pb.TagNumber(5)
  set channel($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasChannel() => $_has(4);
  @$pb.TagNumber(5)
  void clearChannel() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get verificationCode => $_getSZ(5);
  @$pb.TagNumber(6)
  set verificationCode($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasVerificationCode() => $_has(5);
  @$pb.TagNumber(6)
  void clearVerificationCode() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get joinMode => $_getSZ(6);
  @$pb.TagNumber(7)
  set joinMode($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasJoinMode() => $_has(6);
  @$pb.TagNumber(7)
  void clearJoinMode() => $_clearField(7);
}

class NearbyOrganisation extends $pb.GeneratedMessage {
  factory NearbyOrganisation({
    $core.String? organisationId,
    $core.String? organisationName,
    $core.String? deviceName,
    $core.String? peerId,
    $core.String? inviteId,
    $core.String? channel,
    $core.String? joinMode,
  }) {
    final result = create();
    if (organisationId != null) result.organisationId = organisationId;
    if (organisationName != null) result.organisationName = organisationName;
    if (deviceName != null) result.deviceName = deviceName;
    if (peerId != null) result.peerId = peerId;
    if (inviteId != null) result.inviteId = inviteId;
    if (channel != null) result.channel = channel;
    if (joinMode != null) result.joinMode = joinMode;
    return result;
  }

  NearbyOrganisation._();

  factory NearbyOrganisation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NearbyOrganisation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NearbyOrganisation',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'organisationId')
    ..aOS(2, _omitFieldNames ? '' : 'organisationName')
    ..aOS(3, _omitFieldNames ? '' : 'deviceName')
    ..aOS(4, _omitFieldNames ? '' : 'peerId')
    ..aOS(5, _omitFieldNames ? '' : 'inviteId')
    ..aOS(6, _omitFieldNames ? '' : 'channel')
    ..aOS(7, _omitFieldNames ? '' : 'joinMode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NearbyOrganisation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NearbyOrganisation copyWith(void Function(NearbyOrganisation) updates) =>
      super.copyWith((message) => updates(message as NearbyOrganisation))
          as NearbyOrganisation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NearbyOrganisation create() => NearbyOrganisation._();
  @$core.override
  NearbyOrganisation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NearbyOrganisation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NearbyOrganisation>(create);
  static NearbyOrganisation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get organisationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set organisationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOrganisationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrganisationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get organisationName => $_getSZ(1);
  @$pb.TagNumber(2)
  set organisationName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOrganisationName() => $_has(1);
  @$pb.TagNumber(2)
  void clearOrganisationName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceName => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceName() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get peerId => $_getSZ(3);
  @$pb.TagNumber(4)
  set peerId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPeerId() => $_has(3);
  @$pb.TagNumber(4)
  void clearPeerId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get inviteId => $_getSZ(4);
  @$pb.TagNumber(5)
  set inviteId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasInviteId() => $_has(4);
  @$pb.TagNumber(5)
  void clearInviteId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get channel => $_getSZ(5);
  @$pb.TagNumber(6)
  set channel($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasChannel() => $_has(5);
  @$pb.TagNumber(6)
  void clearChannel() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get joinMode => $_getSZ(6);
  @$pb.TagNumber(7)
  set joinMode($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasJoinMode() => $_has(6);
  @$pb.TagNumber(7)
  void clearJoinMode() => $_clearField(7);
}

class ListNearbyOrganisationsRequest extends $pb.GeneratedMessage {
  factory ListNearbyOrganisationsRequest({
    $core.String? channel,
  }) {
    final result = create();
    if (channel != null) result.channel = channel;
    return result;
  }

  ListNearbyOrganisationsRequest._();

  factory ListNearbyOrganisationsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListNearbyOrganisationsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListNearbyOrganisationsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'channel')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListNearbyOrganisationsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListNearbyOrganisationsRequest copyWith(
          void Function(ListNearbyOrganisationsRequest) updates) =>
      super.copyWith(
              (message) => updates(message as ListNearbyOrganisationsRequest))
          as ListNearbyOrganisationsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListNearbyOrganisationsRequest create() =>
      ListNearbyOrganisationsRequest._();
  @$core.override
  ListNearbyOrganisationsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListNearbyOrganisationsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListNearbyOrganisationsRequest>(create);
  static ListNearbyOrganisationsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get channel => $_getSZ(0);
  @$pb.TagNumber(1)
  set channel($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChannel() => $_has(0);
  @$pb.TagNumber(1)
  void clearChannel() => $_clearField(1);
}

class ListNearbyOrganisationsResponse extends $pb.GeneratedMessage {
  factory ListNearbyOrganisationsResponse({
    $core.Iterable<NearbyOrganisation>? organisations,
  }) {
    final result = create();
    if (organisations != null) result.organisations.addAll(organisations);
    return result;
  }

  ListNearbyOrganisationsResponse._();

  factory ListNearbyOrganisationsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ListNearbyOrganisationsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListNearbyOrganisationsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<NearbyOrganisation>(1, _omitFieldNames ? '' : 'organisations',
        subBuilder: NearbyOrganisation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListNearbyOrganisationsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListNearbyOrganisationsResponse copyWith(
          void Function(ListNearbyOrganisationsResponse) updates) =>
      super.copyWith(
              (message) => updates(message as ListNearbyOrganisationsResponse))
          as ListNearbyOrganisationsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListNearbyOrganisationsResponse create() =>
      ListNearbyOrganisationsResponse._();
  @$core.override
  ListNearbyOrganisationsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ListNearbyOrganisationsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListNearbyOrganisationsResponse>(
          create);
  static ListNearbyOrganisationsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<NearbyOrganisation> get organisations => $_getList(0);
}

class JoinOrganisationRequest extends $pb.GeneratedMessage {
  factory JoinOrganisationRequest({
    $core.String? organisationId,
    $core.String? peerId,
    $core.String? inviteId,
    $core.String? username,
    $core.String? name,
    $core.String? channel,
    $core.String? verificationCode,
    $core.String? joinMode,
  }) {
    final result = create();
    if (organisationId != null) result.organisationId = organisationId;
    if (peerId != null) result.peerId = peerId;
    if (inviteId != null) result.inviteId = inviteId;
    if (username != null) result.username = username;
    if (name != null) result.name = name;
    if (channel != null) result.channel = channel;
    if (verificationCode != null) result.verificationCode = verificationCode;
    if (joinMode != null) result.joinMode = joinMode;
    return result;
  }

  JoinOrganisationRequest._();

  factory JoinOrganisationRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory JoinOrganisationRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'JoinOrganisationRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'organisationId')
    ..aOS(2, _omitFieldNames ? '' : 'peerId')
    ..aOS(3, _omitFieldNames ? '' : 'inviteId')
    ..aOS(4, _omitFieldNames ? '' : 'username')
    ..aOS(5, _omitFieldNames ? '' : 'name')
    ..aOS(6, _omitFieldNames ? '' : 'channel')
    ..aOS(7, _omitFieldNames ? '' : 'verificationCode')
    ..aOS(8, _omitFieldNames ? '' : 'joinMode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinOrganisationRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinOrganisationRequest copyWith(
          void Function(JoinOrganisationRequest) updates) =>
      super.copyWith((message) => updates(message as JoinOrganisationRequest))
          as JoinOrganisationRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinOrganisationRequest create() => JoinOrganisationRequest._();
  @$core.override
  JoinOrganisationRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static JoinOrganisationRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<JoinOrganisationRequest>(create);
  static JoinOrganisationRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get organisationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set organisationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOrganisationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearOrganisationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get peerId => $_getSZ(1);
  @$pb.TagNumber(2)
  set peerId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPeerId() => $_has(1);
  @$pb.TagNumber(2)
  void clearPeerId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get inviteId => $_getSZ(2);
  @$pb.TagNumber(3)
  set inviteId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInviteId() => $_has(2);
  @$pb.TagNumber(3)
  void clearInviteId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get username => $_getSZ(3);
  @$pb.TagNumber(4)
  set username($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsername() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsername() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get name => $_getSZ(4);
  @$pb.TagNumber(5)
  set name($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasName() => $_has(4);
  @$pb.TagNumber(5)
  void clearName() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get channel => $_getSZ(5);
  @$pb.TagNumber(6)
  set channel($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasChannel() => $_has(5);
  @$pb.TagNumber(6)
  void clearChannel() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get verificationCode => $_getSZ(6);
  @$pb.TagNumber(7)
  set verificationCode($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasVerificationCode() => $_has(6);
  @$pb.TagNumber(7)
  void clearVerificationCode() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get joinMode => $_getSZ(7);
  @$pb.TagNumber(8)
  set joinMode($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasJoinMode() => $_has(7);
  @$pb.TagNumber(8)
  void clearJoinMode() => $_clearField(8);
}

class JoinOrganisationResponse extends $pb.GeneratedMessage {
  factory JoinOrganisationResponse() => create();

  JoinOrganisationResponse._();

  factory JoinOrganisationResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory JoinOrganisationResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'JoinOrganisationResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinOrganisationResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  JoinOrganisationResponse copyWith(
          void Function(JoinOrganisationResponse) updates) =>
      super.copyWith((message) => updates(message as JoinOrganisationResponse))
          as JoinOrganisationResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static JoinOrganisationResponse create() => JoinOrganisationResponse._();
  @$core.override
  JoinOrganisationResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static JoinOrganisationResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<JoinOrganisationResponse>(create);
  static JoinOrganisationResponse? _defaultInstance;
}

class GetLocalSSHKeyRequest extends $pb.GeneratedMessage {
  factory GetLocalSSHKeyRequest() => create();

  GetLocalSSHKeyRequest._();

  factory GetLocalSSHKeyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLocalSSHKeyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLocalSSHKeyRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalSSHKeyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalSSHKeyRequest copyWith(
          void Function(GetLocalSSHKeyRequest) updates) =>
      super.copyWith((message) => updates(message as GetLocalSSHKeyRequest))
          as GetLocalSSHKeyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLocalSSHKeyRequest create() => GetLocalSSHKeyRequest._();
  @$core.override
  GetLocalSSHKeyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLocalSSHKeyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLocalSSHKeyRequest>(create);
  static GetLocalSSHKeyRequest? _defaultInstance;
}

class GetLocalSSHKeyResponse extends $pb.GeneratedMessage {
  factory GetLocalSSHKeyResponse({
    $core.String? public,
    $core.String? private,
  }) {
    final result = create();
    if (public != null) result.public = public;
    if (private != null) result.private = private;
    return result;
  }

  GetLocalSSHKeyResponse._();

  factory GetLocalSSHKeyResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLocalSSHKeyResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLocalSSHKeyResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'public')
    ..aOS(2, _omitFieldNames ? '' : 'private')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalSSHKeyResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalSSHKeyResponse copyWith(
          void Function(GetLocalSSHKeyResponse) updates) =>
      super.copyWith((message) => updates(message as GetLocalSSHKeyResponse))
          as GetLocalSSHKeyResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLocalSSHKeyResponse create() => GetLocalSSHKeyResponse._();
  @$core.override
  GetLocalSSHKeyResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLocalSSHKeyResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLocalSSHKeyResponse>(create);
  static GetLocalSSHKeyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get public => $_getSZ(0);
  @$pb.TagNumber(1)
  set public($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPublic() => $_has(0);
  @$pb.TagNumber(1)
  void clearPublic() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get private => $_getSZ(1);
  @$pb.TagNumber(2)
  set private($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPrivate() => $_has(1);
  @$pb.TagNumber(2)
  void clearPrivate() => $_clearField(2);
}

class App extends $pb.GeneratedMessage {
  factory App({
    $core.String? id,
    $core.String? name,
    $core.String? version,
    $core.String? status,
    $core.String? instanceName,
    $core.String? ip,
    $core.String? installer,
    $core.bool? persistence,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (version != null) result.version = version;
    if (status != null) result.status = status;
    if (instanceName != null) result.instanceName = instanceName;
    if (ip != null) result.ip = ip;
    if (installer != null) result.installer = installer;
    if (persistence != null) result.persistence = persistence;
    return result;
  }

  App._();

  factory App.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory App.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'App',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'version')
    ..aOS(4, _omitFieldNames ? '' : 'status')
    ..aOS(5, _omitFieldNames ? '' : 'instanceName')
    ..aOS(6, _omitFieldNames ? '' : 'ip')
    ..aOS(7, _omitFieldNames ? '' : 'installer')
    ..aOB(8, _omitFieldNames ? '' : 'persistence')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  App clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  App copyWith(void Function(App) updates) =>
      super.copyWith((message) => updates(message as App)) as App;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static App create() => App._();
  @$core.override
  App createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static App getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<App>(create);
  static App? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get version => $_getSZ(2);
  @$pb.TagNumber(3)
  set version($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get status => $_getSZ(3);
  @$pb.TagNumber(4)
  set status($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get instanceName => $_getSZ(4);
  @$pb.TagNumber(5)
  set instanceName($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasInstanceName() => $_has(4);
  @$pb.TagNumber(5)
  void clearInstanceName() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get ip => $_getSZ(5);
  @$pb.TagNumber(6)
  set ip($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIp() => $_has(5);
  @$pb.TagNumber(6)
  void clearIp() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get installer => $_getSZ(6);
  @$pb.TagNumber(7)
  set installer($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasInstaller() => $_has(6);
  @$pb.TagNumber(7)
  void clearInstaller() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get persistence => $_getBF(7);
  @$pb.TagNumber(8)
  set persistence($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasPersistence() => $_has(7);
  @$pb.TagNumber(8)
  void clearPersistence() => $_clearField(8);
}

class GetAppsRequest extends $pb.GeneratedMessage {
  factory GetAppsRequest() => create();

  GetAppsRequest._();

  factory GetAppsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAppsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAppsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppsRequest copyWith(void Function(GetAppsRequest) updates) =>
      super.copyWith((message) => updates(message as GetAppsRequest))
          as GetAppsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAppsRequest create() => GetAppsRequest._();
  @$core.override
  GetAppsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAppsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAppsRequest>(create);
  static GetAppsRequest? _defaultInstance;
}

class GetAppsResponse extends $pb.GeneratedMessage {
  factory GetAppsResponse({
    $core.Iterable<App>? apps,
  }) {
    final result = create();
    if (apps != null) result.apps.addAll(apps);
    return result;
  }

  GetAppsResponse._();

  factory GetAppsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAppsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAppsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<App>(1, _omitFieldNames ? '' : 'apps', subBuilder: App.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppsResponse copyWith(void Function(GetAppsResponse) updates) =>
      super.copyWith((message) => updates(message as GetAppsResponse))
          as GetAppsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAppsResponse create() => GetAppsResponse._();
  @$core.override
  GetAppsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAppsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAppsResponse>(create);
  static GetAppsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<App> get apps => $_getList(0);
}

class CreateAppRequest extends $pb.GeneratedMessage {
  factory CreateAppRequest({
    $core.String? name,
    $core.String? installerId,
    $core.String? instanceId,
    $core.bool? persistence,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (installerId != null) result.installerId = installerId;
    if (instanceId != null) result.instanceId = instanceId;
    if (persistence != null) result.persistence = persistence;
    return result;
  }

  CreateAppRequest._();

  factory CreateAppRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateAppRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateAppRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'installerId')
    ..aOS(3, _omitFieldNames ? '' : 'instanceId')
    ..aOB(4, _omitFieldNames ? '' : 'persistence')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateAppRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateAppRequest copyWith(void Function(CreateAppRequest) updates) =>
      super.copyWith((message) => updates(message as CreateAppRequest))
          as CreateAppRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateAppRequest create() => CreateAppRequest._();
  @$core.override
  CreateAppRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateAppRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateAppRequest>(create);
  static CreateAppRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get installerId => $_getSZ(1);
  @$pb.TagNumber(2)
  set installerId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasInstallerId() => $_has(1);
  @$pb.TagNumber(2)
  void clearInstallerId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get instanceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set instanceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInstanceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearInstanceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get persistence => $_getBF(3);
  @$pb.TagNumber(4)
  set persistence($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPersistence() => $_has(3);
  @$pb.TagNumber(4)
  void clearPersistence() => $_clearField(4);
}

class CreateAppResponse extends $pb.GeneratedMessage {
  factory CreateAppResponse({
    $core.String? id,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  CreateAppResponse._();

  factory CreateAppResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CreateAppResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateAppResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOM<WriteConfirmation>(2, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateAppResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateAppResponse copyWith(void Function(CreateAppResponse) updates) =>
      super.copyWith((message) => updates(message as CreateAppResponse))
          as CreateAppResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateAppResponse create() => CreateAppResponse._();
  @$core.override
  CreateAppResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CreateAppResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateAppResponse>(create);
  static CreateAppResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  WriteConfirmation get confirmation => $_getN(1);
  @$pb.TagNumber(2)
  set confirmation(WriteConfirmation value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasConfirmation() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfirmation() => $_clearField(2);
  @$pb.TagNumber(2)
  WriteConfirmation ensureConfirmation() => $_ensure(1);
}

class StartAppRequest extends $pb.GeneratedMessage {
  factory StartAppRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  StartAppRequest._();

  factory StartAppRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartAppRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartAppRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartAppRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartAppRequest copyWith(void Function(StartAppRequest) updates) =>
      super.copyWith((message) => updates(message as StartAppRequest))
          as StartAppRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartAppRequest create() => StartAppRequest._();
  @$core.override
  StartAppRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartAppRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartAppRequest>(create);
  static StartAppRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class StartAppResponse extends $pb.GeneratedMessage {
  factory StartAppResponse({
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  StartAppResponse._();

  factory StartAppResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartAppResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartAppResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<WriteConfirmation>(1, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartAppResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartAppResponse copyWith(void Function(StartAppResponse) updates) =>
      super.copyWith((message) => updates(message as StartAppResponse))
          as StartAppResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartAppResponse create() => StartAppResponse._();
  @$core.override
  StartAppResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartAppResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartAppResponse>(create);
  static StartAppResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WriteConfirmation get confirmation => $_getN(0);
  @$pb.TagNumber(1)
  set confirmation(WriteConfirmation value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfirmation() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfirmation() => $_clearField(1);
  @$pb.TagNumber(1)
  WriteConfirmation ensureConfirmation() => $_ensure(0);
}

class StopAppRequest extends $pb.GeneratedMessage {
  factory StopAppRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  StopAppRequest._();

  factory StopAppRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopAppRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopAppRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopAppRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopAppRequest copyWith(void Function(StopAppRequest) updates) =>
      super.copyWith((message) => updates(message as StopAppRequest))
          as StopAppRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopAppRequest create() => StopAppRequest._();
  @$core.override
  StopAppRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopAppRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopAppRequest>(create);
  static StopAppRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class StopAppResponse extends $pb.GeneratedMessage {
  factory StopAppResponse({
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  StopAppResponse._();

  factory StopAppResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopAppResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopAppResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<WriteConfirmation>(1, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopAppResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopAppResponse copyWith(void Function(StopAppResponse) updates) =>
      super.copyWith((message) => updates(message as StopAppResponse))
          as StopAppResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopAppResponse create() => StopAppResponse._();
  @$core.override
  StopAppResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopAppResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopAppResponse>(create);
  static StopAppResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WriteConfirmation get confirmation => $_getN(0);
  @$pb.TagNumber(1)
  set confirmation(WriteConfirmation value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfirmation() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfirmation() => $_clearField(1);
  @$pb.TagNumber(1)
  WriteConfirmation ensureConfirmation() => $_ensure(0);
}

class RemoveAppRequest extends $pb.GeneratedMessage {
  factory RemoveAppRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  RemoveAppRequest._();

  factory RemoveAppRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveAppRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveAppRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveAppRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveAppRequest copyWith(void Function(RemoveAppRequest) updates) =>
      super.copyWith((message) => updates(message as RemoveAppRequest))
          as RemoveAppRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveAppRequest create() => RemoveAppRequest._();
  @$core.override
  RemoveAppRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveAppRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveAppRequest>(create);
  static RemoveAppRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class RemoveAppResponse extends $pb.GeneratedMessage {
  factory RemoveAppResponse({
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  RemoveAppResponse._();

  factory RemoveAppResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveAppResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveAppResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<WriteConfirmation>(1, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveAppResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveAppResponse copyWith(void Function(RemoveAppResponse) updates) =>
      super.copyWith((message) => updates(message as RemoveAppResponse))
          as RemoveAppResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveAppResponse create() => RemoveAppResponse._();
  @$core.override
  RemoveAppResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveAppResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveAppResponse>(create);
  static RemoveAppResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WriteConfirmation get confirmation => $_getN(0);
  @$pb.TagNumber(1)
  set confirmation(WriteConfirmation value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfirmation() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfirmation() => $_clearField(1);
  @$pb.TagNumber(1)
  WriteConfirmation ensureConfirmation() => $_ensure(0);
}

class GetAppLogsRequest extends $pb.GeneratedMessage {
  factory GetAppLogsRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetAppLogsRequest._();

  factory GetAppLogsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAppLogsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAppLogsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppLogsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppLogsRequest copyWith(void Function(GetAppLogsRequest) updates) =>
      super.copyWith((message) => updates(message as GetAppLogsRequest))
          as GetAppLogsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAppLogsRequest create() => GetAppLogsRequest._();
  @$core.override
  GetAppLogsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAppLogsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAppLogsRequest>(create);
  static GetAppLogsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetAppLogsResponse extends $pb.GeneratedMessage {
  factory GetAppLogsResponse({
    $core.List<$core.int>? logs,
  }) {
    final result = create();
    if (logs != null) result.logs = logs;
    return result;
  }

  GetAppLogsResponse._();

  factory GetAppLogsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetAppLogsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetAppLogsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'logs', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppLogsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetAppLogsResponse copyWith(void Function(GetAppLogsResponse) updates) =>
      super.copyWith((message) => updates(message as GetAppLogsResponse))
          as GetAppLogsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetAppLogsResponse create() => GetAppLogsResponse._();
  @$core.override
  GetAppLogsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetAppLogsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetAppLogsResponse>(create);
  static GetAppLogsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get logs => $_getN(0);
  @$pb.TagNumber(1)
  set logs($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLogs() => $_has(0);
  @$pb.TagNumber(1)
  void clearLogs() => $_clearField(1);
}

class ProvisionerMachineSpec extends $pb.GeneratedMessage {
  factory ProvisionerMachineSpec({
    $core.int? cores,
    $core.int? memory,
    $core.int? defaultStorage,
    $core.int? bandwidth,
    $core.int? includedDataTransfer,
    $core.bool? baremetal,
    $core.double? priceMonthly,
  }) {
    final result = create();
    if (cores != null) result.cores = cores;
    if (memory != null) result.memory = memory;
    if (defaultStorage != null) result.defaultStorage = defaultStorage;
    if (bandwidth != null) result.bandwidth = bandwidth;
    if (includedDataTransfer != null)
      result.includedDataTransfer = includedDataTransfer;
    if (baremetal != null) result.baremetal = baremetal;
    if (priceMonthly != null) result.priceMonthly = priceMonthly;
    return result;
  }

  ProvisionerMachineSpec._();

  factory ProvisionerMachineSpec.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvisionerMachineSpec.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvisionerMachineSpec',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'cores')
    ..aI(2, _omitFieldNames ? '' : 'memory')
    ..aI(3, _omitFieldNames ? '' : 'defaultStorage')
    ..aI(4, _omitFieldNames ? '' : 'bandwidth')
    ..aI(5, _omitFieldNames ? '' : 'includedDataTransfer')
    ..aOB(6, _omitFieldNames ? '' : 'baremetal')
    ..aD(7, _omitFieldNames ? '' : 'priceMonthly',
        fieldType: $pb.PbFieldType.OF)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerMachineSpec clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerMachineSpec copyWith(
          void Function(ProvisionerMachineSpec) updates) =>
      super.copyWith((message) => updates(message as ProvisionerMachineSpec))
          as ProvisionerMachineSpec;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvisionerMachineSpec create() => ProvisionerMachineSpec._();
  @$core.override
  ProvisionerMachineSpec createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvisionerMachineSpec getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvisionerMachineSpec>(create);
  static ProvisionerMachineSpec? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get cores => $_getIZ(0);
  @$pb.TagNumber(1)
  set cores($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCores() => $_has(0);
  @$pb.TagNumber(1)
  void clearCores() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get memory => $_getIZ(1);
  @$pb.TagNumber(2)
  set memory($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMemory() => $_has(1);
  @$pb.TagNumber(2)
  void clearMemory() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get defaultStorage => $_getIZ(2);
  @$pb.TagNumber(3)
  set defaultStorage($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDefaultStorage() => $_has(2);
  @$pb.TagNumber(3)
  void clearDefaultStorage() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get bandwidth => $_getIZ(3);
  @$pb.TagNumber(4)
  set bandwidth($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBandwidth() => $_has(3);
  @$pb.TagNumber(4)
  void clearBandwidth() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get includedDataTransfer => $_getIZ(4);
  @$pb.TagNumber(5)
  set includedDataTransfer($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasIncludedDataTransfer() => $_has(4);
  @$pb.TagNumber(5)
  void clearIncludedDataTransfer() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get baremetal => $_getBF(5);
  @$pb.TagNumber(6)
  set baremetal($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBaremetal() => $_has(5);
  @$pb.TagNumber(6)
  void clearBaremetal() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.double get priceMonthly => $_getN(6);
  @$pb.TagNumber(7)
  set priceMonthly($core.double value) => $_setFloat(6, value);
  @$pb.TagNumber(7)
  $core.bool hasPriceMonthly() => $_has(6);
  @$pb.TagNumber(7)
  void clearPriceMonthly() => $_clearField(7);
}

class ProvisionerType extends $pb.GeneratedMessage {
  factory ProvisionerType({
    $core.String? name,
    $core.Iterable<$core.String>? authenticationFields,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (authenticationFields != null)
      result.authenticationFields.addAll(authenticationFields);
    return result;
  }

  ProvisionerType._();

  factory ProvisionerType.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvisionerType.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvisionerType',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..pPS(2, _omitFieldNames ? '' : 'authenticationFields')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerType clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerType copyWith(void Function(ProvisionerType) updates) =>
      super.copyWith((message) => updates(message as ProvisionerType))
          as ProvisionerType;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvisionerType create() => ProvisionerType._();
  @$core.override
  ProvisionerType createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvisionerType getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvisionerType>(create);
  static ProvisionerType? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get authenticationFields => $_getList(1);
}

class Provisioner extends $pb.GeneratedMessage {
  factory Provisioner({
    $core.String? name,
    ProvisionerType? type,
    $core.Iterable<$core.String>? supportedLocations,
    $core.Iterable<$core.MapEntry<$core.String, ProvisionerMachineSpec>>?
        supportedMachines,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (type != null) result.type = type;
    if (supportedLocations != null)
      result.supportedLocations.addAll(supportedLocations);
    if (supportedMachines != null)
      result.supportedMachines.addEntries(supportedMachines);
    return result;
  }

  Provisioner._();

  factory Provisioner.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Provisioner.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Provisioner',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOM<ProvisionerType>(2, _omitFieldNames ? '' : 'type',
        subBuilder: ProvisionerType.create)
    ..pPS(3, _omitFieldNames ? '' : 'supportedLocations')
    ..m<$core.String, ProvisionerMachineSpec>(
        4, _omitFieldNames ? '' : 'supportedMachines',
        entryClassName: 'Provisioner.SupportedMachinesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: ProvisionerMachineSpec.create,
        valueDefaultOrMaker: ProvisionerMachineSpec.getDefault,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Provisioner clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Provisioner copyWith(void Function(Provisioner) updates) =>
      super.copyWith((message) => updates(message as Provisioner))
          as Provisioner;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Provisioner create() => Provisioner._();
  @$core.override
  Provisioner createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Provisioner getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Provisioner>(create);
  static Provisioner? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  ProvisionerType get type => $_getN(1);
  @$pb.TagNumber(2)
  set type(ProvisionerType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);
  @$pb.TagNumber(2)
  ProvisionerType ensureType() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get supportedLocations => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, ProvisionerMachineSpec> get supportedMachines =>
      $_getMap(3);
}

class GetSupportedProvisionersRequest extends $pb.GeneratedMessage {
  factory GetSupportedProvisionersRequest() => create();

  GetSupportedProvisionersRequest._();

  factory GetSupportedProvisionersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSupportedProvisionersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSupportedProvisionersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedProvisionersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedProvisionersRequest copyWith(
          void Function(GetSupportedProvisionersRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetSupportedProvisionersRequest))
          as GetSupportedProvisionersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSupportedProvisionersRequest create() =>
      GetSupportedProvisionersRequest._();
  @$core.override
  GetSupportedProvisionersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSupportedProvisionersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSupportedProvisionersRequest>(
          create);
  static GetSupportedProvisionersRequest? _defaultInstance;
}

class GetSupportedProvisionersResponse extends $pb.GeneratedMessage {
  factory GetSupportedProvisionersResponse({
    $core.Iterable<ProvisionerType>? provisionerTypes,
  }) {
    final result = create();
    if (provisionerTypes != null)
      result.provisionerTypes.addAll(provisionerTypes);
    return result;
  }

  GetSupportedProvisionersResponse._();

  factory GetSupportedProvisionersResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSupportedProvisionersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSupportedProvisionersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<ProvisionerType>(1, _omitFieldNames ? '' : 'provisionerTypes',
        subBuilder: ProvisionerType.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedProvisionersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedProvisionersResponse copyWith(
          void Function(GetSupportedProvisionersResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetSupportedProvisionersResponse))
          as GetSupportedProvisionersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSupportedProvisionersResponse create() =>
      GetSupportedProvisionersResponse._();
  @$core.override
  GetSupportedProvisionersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSupportedProvisionersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSupportedProvisionersResponse>(
          create);
  static GetSupportedProvisionersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ProvisionerType> get provisionerTypes => $_getList(0);
}

class GetProvisionersRequest extends $pb.GeneratedMessage {
  factory GetProvisionersRequest() => create();

  GetProvisionersRequest._();

  factory GetProvisionersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionersRequest copyWith(
          void Function(GetProvisionersRequest) updates) =>
      super.copyWith((message) => updates(message as GetProvisionersRequest))
          as GetProvisionersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionersRequest create() => GetProvisionersRequest._();
  @$core.override
  GetProvisionersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionersRequest>(create);
  static GetProvisionersRequest? _defaultInstance;
}

class GetProvisionersResponse extends $pb.GeneratedMessage {
  factory GetProvisionersResponse({
    $core.Iterable<Provisioner>? provisioners,
  }) {
    final result = create();
    if (provisioners != null) result.provisioners.addAll(provisioners);
    return result;
  }

  GetProvisionersResponse._();

  factory GetProvisionersResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Provisioner>(1, _omitFieldNames ? '' : 'provisioners',
        subBuilder: Provisioner.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionersResponse copyWith(
          void Function(GetProvisionersResponse) updates) =>
      super.copyWith((message) => updates(message as GetProvisionersResponse))
          as GetProvisionersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionersResponse create() => GetProvisionersResponse._();
  @$core.override
  GetProvisionersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionersResponse>(create);
  static GetProvisionersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Provisioner> get provisioners => $_getList(0);
}

class GetProvisionerRequest extends $pb.GeneratedMessage {
  factory GetProvisionerRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetProvisionerRequest._();

  factory GetProvisionerRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionerRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionerRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerRequest copyWith(
          void Function(GetProvisionerRequest) updates) =>
      super.copyWith((message) => updates(message as GetProvisionerRequest))
          as GetProvisionerRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionerRequest create() => GetProvisionerRequest._();
  @$core.override
  GetProvisionerRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionerRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionerRequest>(create);
  static GetProvisionerRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetProvisionerResponse extends $pb.GeneratedMessage {
  factory GetProvisionerResponse({
    Provisioner? provisioner,
  }) {
    final result = create();
    if (provisioner != null) result.provisioner = provisioner;
    return result;
  }

  GetProvisionerResponse._();

  factory GetProvisionerResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionerResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionerResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<Provisioner>(1, _omitFieldNames ? '' : 'provisioner',
        subBuilder: Provisioner.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerResponse copyWith(
          void Function(GetProvisionerResponse) updates) =>
      super.copyWith((message) => updates(message as GetProvisionerResponse))
          as GetProvisionerResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionerResponse create() => GetProvisionerResponse._();
  @$core.override
  GetProvisionerResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionerResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionerResponse>(create);
  static GetProvisionerResponse? _defaultInstance;

  @$pb.TagNumber(1)
  Provisioner get provisioner => $_getN(0);
  @$pb.TagNumber(1)
  set provisioner(Provisioner value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasProvisioner() => $_has(0);
  @$pb.TagNumber(1)
  void clearProvisioner() => $_clearField(1);
  @$pb.TagNumber(1)
  Provisioner ensureProvisioner() => $_ensure(0);
}

class AddProvisionerRequest extends $pb.GeneratedMessage {
  factory AddProvisionerRequest({
    $core.String? name,
    $core.String? type,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? credentials,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (type != null) result.type = type;
    if (credentials != null) result.credentials.addEntries(credentials);
    return result;
  }

  AddProvisionerRequest._();

  factory AddProvisionerRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AddProvisionerRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AddProvisionerRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'type')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'credentials',
        entryClassName: 'AddProvisionerRequest.CredentialsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddProvisionerRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddProvisionerRequest copyWith(
          void Function(AddProvisionerRequest) updates) =>
      super.copyWith((message) => updates(message as AddProvisionerRequest))
          as AddProvisionerRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AddProvisionerRequest create() => AddProvisionerRequest._();
  @$core.override
  AddProvisionerRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AddProvisionerRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AddProvisionerRequest>(create);
  static AddProvisionerRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get type => $_getSZ(1);
  @$pb.TagNumber(2)
  set type($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get credentials => $_getMap(2);
}

class AddProvisionerResponse extends $pb.GeneratedMessage {
  factory AddProvisionerResponse() => create();

  AddProvisionerResponse._();

  factory AddProvisionerResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AddProvisionerResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AddProvisionerResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddProvisionerResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddProvisionerResponse copyWith(
          void Function(AddProvisionerResponse) updates) =>
      super.copyWith((message) => updates(message as AddProvisionerResponse))
          as AddProvisionerResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AddProvisionerResponse create() => AddProvisionerResponse._();
  @$core.override
  AddProvisionerResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AddProvisionerResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AddProvisionerResponse>(create);
  static AddProvisionerResponse? _defaultInstance;
}

class RemoveProvisionerRequest extends $pb.GeneratedMessage {
  factory RemoveProvisionerRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  RemoveProvisionerRequest._();

  factory RemoveProvisionerRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveProvisionerRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveProvisionerRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerRequest copyWith(
          void Function(RemoveProvisionerRequest) updates) =>
      super.copyWith((message) => updates(message as RemoveProvisionerRequest))
          as RemoveProvisionerRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerRequest create() => RemoveProvisionerRequest._();
  @$core.override
  RemoveProvisionerRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveProvisionerRequest>(create);
  static RemoveProvisionerRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class RemoveProvisionerResponse extends $pb.GeneratedMessage {
  factory RemoveProvisionerResponse() => create();

  RemoveProvisionerResponse._();

  factory RemoveProvisionerResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveProvisionerResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveProvisionerResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerResponse copyWith(
          void Function(RemoveProvisionerResponse) updates) =>
      super.copyWith((message) => updates(message as RemoveProvisionerResponse))
          as RemoveProvisionerResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerResponse create() => RemoveProvisionerResponse._();
  @$core.override
  RemoveProvisionerResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveProvisionerResponse>(create);
  static RemoveProvisionerResponse? _defaultInstance;
}

class CloudInstance extends $pb.GeneratedMessage {
  factory CloudInstance({
    $core.String? name,
    $core.String? publicIp,
    $core.String? internalIp,
    $core.String? cloudName,
    $core.String? cloudType,
    $core.String? vmId,
    $core.String? location,
    $core.String? publicKey,
    $core.String? publicKeyWireguard,
    $core.String? protosVersion,
    $core.String? status,
    $core.String? architecture,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? peers,
    $core.String? providerStatus,
    $core.String? adminApiReachability,
    $core.bool? replicationRouted,
    $core.String? adminLastError,
    $core.String? adminLastSeen,
    $core.String? peerId,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (publicIp != null) result.publicIp = publicIp;
    if (internalIp != null) result.internalIp = internalIp;
    if (cloudName != null) result.cloudName = cloudName;
    if (cloudType != null) result.cloudType = cloudType;
    if (vmId != null) result.vmId = vmId;
    if (location != null) result.location = location;
    if (publicKey != null) result.publicKey = publicKey;
    if (publicKeyWireguard != null)
      result.publicKeyWireguard = publicKeyWireguard;
    if (protosVersion != null) result.protosVersion = protosVersion;
    if (status != null) result.status = status;
    if (architecture != null) result.architecture = architecture;
    if (peers != null) result.peers.addEntries(peers);
    if (providerStatus != null) result.providerStatus = providerStatus;
    if (adminApiReachability != null)
      result.adminApiReachability = adminApiReachability;
    if (replicationRouted != null) result.replicationRouted = replicationRouted;
    if (adminLastError != null) result.adminLastError = adminLastError;
    if (adminLastSeen != null) result.adminLastSeen = adminLastSeen;
    if (peerId != null) result.peerId = peerId;
    return result;
  }

  CloudInstance._();

  factory CloudInstance.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudInstance.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudInstance',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'publicIp')
    ..aOS(3, _omitFieldNames ? '' : 'internalIp')
    ..aOS(4, _omitFieldNames ? '' : 'cloudName')
    ..aOS(5, _omitFieldNames ? '' : 'cloudType')
    ..aOS(6, _omitFieldNames ? '' : 'vmId')
    ..aOS(7, _omitFieldNames ? '' : 'location')
    ..aOS(8, _omitFieldNames ? '' : 'publicKey')
    ..aOS(9, _omitFieldNames ? '' : 'publicKeyWireguard')
    ..aOS(10, _omitFieldNames ? '' : 'protosVersion')
    ..aOS(11, _omitFieldNames ? '' : 'status')
    ..aOS(12, _omitFieldNames ? '' : 'architecture')
    ..m<$core.String, $core.String>(13, _omitFieldNames ? '' : 'peers',
        entryClassName: 'CloudInstance.PeersEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..aOS(14, _omitFieldNames ? '' : 'providerStatus')
    ..aOS(15, _omitFieldNames ? '' : 'adminApiReachability')
    ..aOB(16, _omitFieldNames ? '' : 'replicationRouted')
    ..aOS(17, _omitFieldNames ? '' : 'adminLastError')
    ..aOS(18, _omitFieldNames ? '' : 'adminLastSeen')
    ..aOS(19, _omitFieldNames ? '' : 'peerId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudInstance clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudInstance copyWith(void Function(CloudInstance) updates) =>
      super.copyWith((message) => updates(message as CloudInstance))
          as CloudInstance;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudInstance create() => CloudInstance._();
  @$core.override
  CloudInstance createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudInstance getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudInstance>(create);
  static CloudInstance? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get publicIp => $_getSZ(1);
  @$pb.TagNumber(2)
  set publicIp($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPublicIp() => $_has(1);
  @$pb.TagNumber(2)
  void clearPublicIp() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get internalIp => $_getSZ(2);
  @$pb.TagNumber(3)
  set internalIp($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInternalIp() => $_has(2);
  @$pb.TagNumber(3)
  void clearInternalIp() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get cloudName => $_getSZ(3);
  @$pb.TagNumber(4)
  set cloudName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCloudName() => $_has(3);
  @$pb.TagNumber(4)
  void clearCloudName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get cloudType => $_getSZ(4);
  @$pb.TagNumber(5)
  set cloudType($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCloudType() => $_has(4);
  @$pb.TagNumber(5)
  void clearCloudType() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get vmId => $_getSZ(5);
  @$pb.TagNumber(6)
  set vmId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasVmId() => $_has(5);
  @$pb.TagNumber(6)
  void clearVmId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get location => $_getSZ(6);
  @$pb.TagNumber(7)
  set location($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasLocation() => $_has(6);
  @$pb.TagNumber(7)
  void clearLocation() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get publicKey => $_getSZ(7);
  @$pb.TagNumber(8)
  set publicKey($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasPublicKey() => $_has(7);
  @$pb.TagNumber(8)
  void clearPublicKey() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get publicKeyWireguard => $_getSZ(8);
  @$pb.TagNumber(9)
  set publicKeyWireguard($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPublicKeyWireguard() => $_has(8);
  @$pb.TagNumber(9)
  void clearPublicKeyWireguard() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get protosVersion => $_getSZ(9);
  @$pb.TagNumber(10)
  set protosVersion($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasProtosVersion() => $_has(9);
  @$pb.TagNumber(10)
  void clearProtosVersion() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get status => $_getSZ(10);
  @$pb.TagNumber(11)
  set status($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasStatus() => $_has(10);
  @$pb.TagNumber(11)
  void clearStatus() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get architecture => $_getSZ(11);
  @$pb.TagNumber(12)
  set architecture($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasArchitecture() => $_has(11);
  @$pb.TagNumber(12)
  void clearArchitecture() => $_clearField(12);

  @$pb.TagNumber(13)
  $pb.PbMap<$core.String, $core.String> get peers => $_getMap(12);

  @$pb.TagNumber(14)
  $core.String get providerStatus => $_getSZ(13);
  @$pb.TagNumber(14)
  set providerStatus($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasProviderStatus() => $_has(13);
  @$pb.TagNumber(14)
  void clearProviderStatus() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get adminApiReachability => $_getSZ(14);
  @$pb.TagNumber(15)
  set adminApiReachability($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasAdminApiReachability() => $_has(14);
  @$pb.TagNumber(15)
  void clearAdminApiReachability() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.bool get replicationRouted => $_getBF(15);
  @$pb.TagNumber(16)
  set replicationRouted($core.bool value) => $_setBool(15, value);
  @$pb.TagNumber(16)
  $core.bool hasReplicationRouted() => $_has(15);
  @$pb.TagNumber(16)
  void clearReplicationRouted() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.String get adminLastError => $_getSZ(16);
  @$pb.TagNumber(17)
  set adminLastError($core.String value) => $_setString(16, value);
  @$pb.TagNumber(17)
  $core.bool hasAdminLastError() => $_has(16);
  @$pb.TagNumber(17)
  void clearAdminLastError() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.String get adminLastSeen => $_getSZ(17);
  @$pb.TagNumber(18)
  set adminLastSeen($core.String value) => $_setString(17, value);
  @$pb.TagNumber(18)
  $core.bool hasAdminLastSeen() => $_has(17);
  @$pb.TagNumber(18)
  void clearAdminLastSeen() => $_clearField(18);

  @$pb.TagNumber(19)
  $core.String get peerId => $_getSZ(18);
  @$pb.TagNumber(19)
  set peerId($core.String value) => $_setString(18, value);
  @$pb.TagNumber(19)
  $core.bool hasPeerId() => $_has(18);
  @$pb.TagNumber(19)
  void clearPeerId() => $_clearField(19);
}

class GetInstancesRequest extends $pb.GeneratedMessage {
  factory GetInstancesRequest() => create();

  GetInstancesRequest._();

  factory GetInstancesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstancesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstancesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstancesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstancesRequest copyWith(void Function(GetInstancesRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstancesRequest))
          as GetInstancesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstancesRequest create() => GetInstancesRequest._();
  @$core.override
  GetInstancesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstancesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstancesRequest>(create);
  static GetInstancesRequest? _defaultInstance;
}

class GetInstancesResponse extends $pb.GeneratedMessage {
  factory GetInstancesResponse({
    $core.Iterable<CloudInstance>? instances,
  }) {
    final result = create();
    if (instances != null) result.instances.addAll(instances);
    return result;
  }

  GetInstancesResponse._();

  factory GetInstancesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstancesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstancesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<CloudInstance>(1, _omitFieldNames ? '' : 'instances',
        subBuilder: CloudInstance.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstancesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstancesResponse copyWith(void Function(GetInstancesResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstancesResponse))
          as GetInstancesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstancesResponse create() => GetInstancesResponse._();
  @$core.override
  GetInstancesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstancesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstancesResponse>(create);
  static GetInstancesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<CloudInstance> get instances => $_getList(0);
}

class GetInstanceRequest extends $pb.GeneratedMessage {
  factory GetInstanceRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetInstanceRequest._();

  factory GetInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRequest copyWith(void Function(GetInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstanceRequest))
          as GetInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceRequest create() => GetInstanceRequest._();
  @$core.override
  GetInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceRequest>(create);
  static GetInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetInstanceResponse extends $pb.GeneratedMessage {
  factory GetInstanceResponse({
    CloudInstance? instance,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    return result;
  }

  GetInstanceResponse._();

  factory GetInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<CloudInstance>(1, _omitFieldNames ? '' : 'instance',
        subBuilder: CloudInstance.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceResponse copyWith(void Function(GetInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstanceResponse))
          as GetInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceResponse create() => GetInstanceResponse._();
  @$core.override
  GetInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceResponse>(create);
  static GetInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CloudInstance get instance => $_getN(0);
  @$pb.TagNumber(1)
  set instance(CloudInstance value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);
  @$pb.TagNumber(1)
  CloudInstance ensureInstance() => $_ensure(0);
}

class InstanceDeployFieldOption extends $pb.GeneratedMessage {
  factory InstanceDeployFieldOption({
    $core.String? value,
    $core.String? label,
    $core.String? description,
  }) {
    final result = create();
    if (value != null) result.value = value;
    if (label != null) result.label = label;
    if (description != null) result.description = description;
    return result;
  }

  InstanceDeployFieldOption._();

  factory InstanceDeployFieldOption.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InstanceDeployFieldOption.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InstanceDeployFieldOption',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'value')
    ..aOS(2, _omitFieldNames ? '' : 'label')
    ..aOS(3, _omitFieldNames ? '' : 'description')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstanceDeployFieldOption clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstanceDeployFieldOption copyWith(
          void Function(InstanceDeployFieldOption) updates) =>
      super.copyWith((message) => updates(message as InstanceDeployFieldOption))
          as InstanceDeployFieldOption;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InstanceDeployFieldOption create() => InstanceDeployFieldOption._();
  @$core.override
  InstanceDeployFieldOption createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InstanceDeployFieldOption getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InstanceDeployFieldOption>(create);
  static InstanceDeployFieldOption? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get value => $_getSZ(0);
  @$pb.TagNumber(1)
  set value($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get label => $_getSZ(1);
  @$pb.TagNumber(2)
  set label($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLabel() => $_has(1);
  @$pb.TagNumber(2)
  void clearLabel() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get description => $_getSZ(2);
  @$pb.TagNumber(3)
  set description($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDescription() => $_has(2);
  @$pb.TagNumber(3)
  void clearDescription() => $_clearField(3);
}

class InstanceDeployField extends $pb.GeneratedMessage {
  factory InstanceDeployField({
    $core.String? name,
    $core.String? label,
    $core.String? kind,
    $core.bool? required,
    $core.bool? visible,
    $core.String? value,
    $core.String? helper,
    $core.Iterable<InstanceDeployFieldOption>? options,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (label != null) result.label = label;
    if (kind != null) result.kind = kind;
    if (required != null) result.required = required;
    if (visible != null) result.visible = visible;
    if (value != null) result.value = value;
    if (helper != null) result.helper = helper;
    if (options != null) result.options.addAll(options);
    return result;
  }

  InstanceDeployField._();

  factory InstanceDeployField.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InstanceDeployField.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InstanceDeployField',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'label')
    ..aOS(3, _omitFieldNames ? '' : 'kind')
    ..aOB(4, _omitFieldNames ? '' : 'required')
    ..aOB(5, _omitFieldNames ? '' : 'visible')
    ..aOS(6, _omitFieldNames ? '' : 'value')
    ..aOS(7, _omitFieldNames ? '' : 'helper')
    ..pPM<InstanceDeployFieldOption>(8, _omitFieldNames ? '' : 'options',
        subBuilder: InstanceDeployFieldOption.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstanceDeployField clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstanceDeployField copyWith(void Function(InstanceDeployField) updates) =>
      super.copyWith((message) => updates(message as InstanceDeployField))
          as InstanceDeployField;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InstanceDeployField create() => InstanceDeployField._();
  @$core.override
  InstanceDeployField createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InstanceDeployField getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InstanceDeployField>(create);
  static InstanceDeployField? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get label => $_getSZ(1);
  @$pb.TagNumber(2)
  set label($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLabel() => $_has(1);
  @$pb.TagNumber(2)
  void clearLabel() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get kind => $_getSZ(2);
  @$pb.TagNumber(3)
  set kind($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasKind() => $_has(2);
  @$pb.TagNumber(3)
  void clearKind() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get required => $_getBF(3);
  @$pb.TagNumber(4)
  set required($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRequired() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequired() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get visible => $_getBF(4);
  @$pb.TagNumber(5)
  set visible($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasVisible() => $_has(4);
  @$pb.TagNumber(5)
  void clearVisible() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get value => $_getSZ(5);
  @$pb.TagNumber(6)
  set value($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasValue() => $_has(5);
  @$pb.TagNumber(6)
  void clearValue() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get helper => $_getSZ(6);
  @$pb.TagNumber(7)
  set helper($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasHelper() => $_has(6);
  @$pb.TagNumber(7)
  void clearHelper() => $_clearField(7);

  @$pb.TagNumber(8)
  $pb.PbList<InstanceDeployFieldOption> get options => $_getList(7);
}

class GetInstanceDeployOptionsRequest extends $pb.GeneratedMessage {
  factory GetInstanceDeployOptionsRequest({
    $core.String? provisioner,
    $core.String? location,
  }) {
    final result = create();
    if (provisioner != null) result.provisioner = provisioner;
    if (location != null) result.location = location;
    return result;
  }

  GetInstanceDeployOptionsRequest._();

  factory GetInstanceDeployOptionsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceDeployOptionsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceDeployOptionsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'provisioner')
    ..aOS(2, _omitFieldNames ? '' : 'location')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceDeployOptionsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceDeployOptionsRequest copyWith(
          void Function(GetInstanceDeployOptionsRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetInstanceDeployOptionsRequest))
          as GetInstanceDeployOptionsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceDeployOptionsRequest create() =>
      GetInstanceDeployOptionsRequest._();
  @$core.override
  GetInstanceDeployOptionsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceDeployOptionsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceDeployOptionsRequest>(
          create);
  static GetInstanceDeployOptionsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get provisioner => $_getSZ(0);
  @$pb.TagNumber(1)
  set provisioner($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProvisioner() => $_has(0);
  @$pb.TagNumber(1)
  void clearProvisioner() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get location => $_getSZ(1);
  @$pb.TagNumber(2)
  set location($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLocation() => $_has(1);
  @$pb.TagNumber(2)
  void clearLocation() => $_clearField(2);
}

class GetInstanceDeployOptionsResponse extends $pb.GeneratedMessage {
  factory GetInstanceDeployOptionsResponse({
    $core.Iterable<InstanceDeployField>? fields,
  }) {
    final result = create();
    if (fields != null) result.fields.addAll(fields);
    return result;
  }

  GetInstanceDeployOptionsResponse._();

  factory GetInstanceDeployOptionsResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceDeployOptionsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceDeployOptionsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<InstanceDeployField>(1, _omitFieldNames ? '' : 'fields',
        subBuilder: InstanceDeployField.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceDeployOptionsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceDeployOptionsResponse copyWith(
          void Function(GetInstanceDeployOptionsResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetInstanceDeployOptionsResponse))
          as GetInstanceDeployOptionsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceDeployOptionsResponse create() =>
      GetInstanceDeployOptionsResponse._();
  @$core.override
  GetInstanceDeployOptionsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceDeployOptionsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceDeployOptionsResponse>(
          create);
  static GetInstanceDeployOptionsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<InstanceDeployField> get fields => $_getList(0);
}

class DeployInstanceRequest extends $pb.GeneratedMessage {
  factory DeployInstanceRequest({
    $core.String? name,
    $core.String? cloudName,
    $core.String? cloudLocation,
    $core.String? machineType,
    $core.String? protosVersion,
    $core.String? devImg,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (cloudName != null) result.cloudName = cloudName;
    if (cloudLocation != null) result.cloudLocation = cloudLocation;
    if (machineType != null) result.machineType = machineType;
    if (protosVersion != null) result.protosVersion = protosVersion;
    if (devImg != null) result.devImg = devImg;
    return result;
  }

  DeployInstanceRequest._();

  factory DeployInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeployInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeployInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'cloudName')
    ..aOS(3, _omitFieldNames ? '' : 'cloudLocation')
    ..aOS(4, _omitFieldNames ? '' : 'machineType')
    ..aOS(5, _omitFieldNames ? '' : 'protosVersion')
    ..aOS(6, _omitFieldNames ? '' : 'devImg')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeployInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeployInstanceRequest copyWith(
          void Function(DeployInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as DeployInstanceRequest))
          as DeployInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeployInstanceRequest create() => DeployInstanceRequest._();
  @$core.override
  DeployInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeployInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeployInstanceRequest>(create);
  static DeployInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cloudName => $_getSZ(1);
  @$pb.TagNumber(2)
  set cloudName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCloudName() => $_has(1);
  @$pb.TagNumber(2)
  void clearCloudName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get cloudLocation => $_getSZ(2);
  @$pb.TagNumber(3)
  set cloudLocation($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCloudLocation() => $_has(2);
  @$pb.TagNumber(3)
  void clearCloudLocation() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get machineType => $_getSZ(3);
  @$pb.TagNumber(4)
  set machineType($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMachineType() => $_has(3);
  @$pb.TagNumber(4)
  void clearMachineType() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get protosVersion => $_getSZ(4);
  @$pb.TagNumber(5)
  set protosVersion($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProtosVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearProtosVersion() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get devImg => $_getSZ(5);
  @$pb.TagNumber(6)
  set devImg($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDevImg() => $_has(5);
  @$pb.TagNumber(6)
  void clearDevImg() => $_clearField(6);
}

class DeployInstanceResponse extends $pb.GeneratedMessage {
  factory DeployInstanceResponse({
    CloudInstance? instance,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  DeployInstanceResponse._();

  factory DeployInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeployInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeployInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<CloudInstance>(1, _omitFieldNames ? '' : 'instance',
        subBuilder: CloudInstance.create)
    ..aOM<WriteConfirmation>(2, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeployInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeployInstanceResponse copyWith(
          void Function(DeployInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as DeployInstanceResponse))
          as DeployInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeployInstanceResponse create() => DeployInstanceResponse._();
  @$core.override
  DeployInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeployInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeployInstanceResponse>(create);
  static DeployInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CloudInstance get instance => $_getN(0);
  @$pb.TagNumber(1)
  set instance(CloudInstance value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);
  @$pb.TagNumber(1)
  CloudInstance ensureInstance() => $_ensure(0);

  @$pb.TagNumber(2)
  WriteConfirmation get confirmation => $_getN(1);
  @$pb.TagNumber(2)
  set confirmation(WriteConfirmation value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasConfirmation() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfirmation() => $_clearField(2);
  @$pb.TagNumber(2)
  WriteConfirmation ensureConfirmation() => $_ensure(1);
}

class RemoveInstanceRequest extends $pb.GeneratedMessage {
  factory RemoveInstanceRequest({
    $core.String? name,
    $core.bool? localOnly,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (localOnly != null) result.localOnly = localOnly;
    return result;
  }

  RemoveInstanceRequest._();

  factory RemoveInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOB(2, _omitFieldNames ? '' : 'localOnly')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveInstanceRequest copyWith(
          void Function(RemoveInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as RemoveInstanceRequest))
          as RemoveInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveInstanceRequest create() => RemoveInstanceRequest._();
  @$core.override
  RemoveInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveInstanceRequest>(create);
  static RemoveInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get localOnly => $_getBF(1);
  @$pb.TagNumber(2)
  set localOnly($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLocalOnly() => $_has(1);
  @$pb.TagNumber(2)
  void clearLocalOnly() => $_clearField(2);
}

class RemoveInstanceResponse extends $pb.GeneratedMessage {
  factory RemoveInstanceResponse({
    $core.String? taskId,
  }) {
    final result = create();
    if (taskId != null) result.taskId = taskId;
    return result;
  }

  RemoveInstanceResponse._();

  factory RemoveInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'taskId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveInstanceResponse copyWith(
          void Function(RemoveInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as RemoveInstanceResponse))
          as RemoveInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveInstanceResponse create() => RemoveInstanceResponse._();
  @$core.override
  RemoveInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveInstanceResponse>(create);
  static RemoveInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get taskId => $_getSZ(0);
  @$pb.TagNumber(1)
  set taskId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTaskId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTaskId() => $_clearField(1);
}

class StartInstanceRequest extends $pb.GeneratedMessage {
  factory StartInstanceRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  StartInstanceRequest._();

  factory StartInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartInstanceRequest copyWith(void Function(StartInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as StartInstanceRequest))
          as StartInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartInstanceRequest create() => StartInstanceRequest._();
  @$core.override
  StartInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartInstanceRequest>(create);
  static StartInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class StartInstanceResponse extends $pb.GeneratedMessage {
  factory StartInstanceResponse({
    $core.String? taskId,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (taskId != null) result.taskId = taskId;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  StartInstanceResponse._();

  factory StartInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'taskId')
    ..aOM<WriteConfirmation>(2, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartInstanceResponse copyWith(
          void Function(StartInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as StartInstanceResponse))
          as StartInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartInstanceResponse create() => StartInstanceResponse._();
  @$core.override
  StartInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartInstanceResponse>(create);
  static StartInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get taskId => $_getSZ(0);
  @$pb.TagNumber(1)
  set taskId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTaskId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTaskId() => $_clearField(1);

  @$pb.TagNumber(2)
  WriteConfirmation get confirmation => $_getN(1);
  @$pb.TagNumber(2)
  set confirmation(WriteConfirmation value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasConfirmation() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfirmation() => $_clearField(2);
  @$pb.TagNumber(2)
  WriteConfirmation ensureConfirmation() => $_ensure(1);
}

class StopInstanceRequest extends $pb.GeneratedMessage {
  factory StopInstanceRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  StopInstanceRequest._();

  factory StopInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopInstanceRequest copyWith(void Function(StopInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as StopInstanceRequest))
          as StopInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopInstanceRequest create() => StopInstanceRequest._();
  @$core.override
  StopInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopInstanceRequest>(create);
  static StopInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class StopInstanceResponse extends $pb.GeneratedMessage {
  factory StopInstanceResponse({
    $core.String? taskId,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (taskId != null) result.taskId = taskId;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  StopInstanceResponse._();

  factory StopInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'taskId')
    ..aOM<WriteConfirmation>(2, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopInstanceResponse copyWith(void Function(StopInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as StopInstanceResponse))
          as StopInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopInstanceResponse create() => StopInstanceResponse._();
  @$core.override
  StopInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopInstanceResponse>(create);
  static StopInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get taskId => $_getSZ(0);
  @$pb.TagNumber(1)
  set taskId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTaskId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTaskId() => $_clearField(1);

  @$pb.TagNumber(2)
  WriteConfirmation get confirmation => $_getN(1);
  @$pb.TagNumber(2)
  set confirmation(WriteConfirmation value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasConfirmation() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfirmation() => $_clearField(2);
  @$pb.TagNumber(2)
  WriteConfirmation ensureConfirmation() => $_ensure(1);
}

class GetInstanceKeyRequest extends $pb.GeneratedMessage {
  factory GetInstanceKeyRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetInstanceKeyRequest._();

  factory GetInstanceKeyRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceKeyRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceKeyRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceKeyRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceKeyRequest copyWith(
          void Function(GetInstanceKeyRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstanceKeyRequest))
          as GetInstanceKeyRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceKeyRequest create() => GetInstanceKeyRequest._();
  @$core.override
  GetInstanceKeyRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceKeyRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceKeyRequest>(create);
  static GetInstanceKeyRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetInstanceKeyResponse extends $pb.GeneratedMessage {
  factory GetInstanceKeyResponse({
    $core.String? key,
  }) {
    final result = create();
    if (key != null) result.key = key;
    return result;
  }

  GetInstanceKeyResponse._();

  factory GetInstanceKeyResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceKeyResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceKeyResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'key')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceKeyResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceKeyResponse copyWith(
          void Function(GetInstanceKeyResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstanceKeyResponse))
          as GetInstanceKeyResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceKeyResponse create() => GetInstanceKeyResponse._();
  @$core.override
  GetInstanceKeyResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceKeyResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceKeyResponse>(create);
  static GetInstanceKeyResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get key => $_getSZ(0);
  @$pb.TagNumber(1)
  set key($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearKey() => $_clearField(1);
}

class GetInstanceLogsRequest extends $pb.GeneratedMessage {
  factory GetInstanceLogsRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetInstanceLogsRequest._();

  factory GetInstanceLogsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceLogsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceLogsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceLogsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceLogsRequest copyWith(
          void Function(GetInstanceLogsRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstanceLogsRequest))
          as GetInstanceLogsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceLogsRequest create() => GetInstanceLogsRequest._();
  @$core.override
  GetInstanceLogsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceLogsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceLogsRequest>(create);
  static GetInstanceLogsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetInstanceLogsResponse extends $pb.GeneratedMessage {
  factory GetInstanceLogsResponse({
    $core.String? logs,
  }) {
    final result = create();
    if (logs != null) result.logs = logs;
    return result;
  }

  GetInstanceLogsResponse._();

  factory GetInstanceLogsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceLogsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceLogsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'logs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceLogsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceLogsResponse copyWith(
          void Function(GetInstanceLogsResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstanceLogsResponse))
          as GetInstanceLogsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceLogsResponse create() => GetInstanceLogsResponse._();
  @$core.override
  GetInstanceLogsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceLogsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceLogsResponse>(create);
  static GetInstanceLogsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get logs => $_getSZ(0);
  @$pb.TagNumber(1)
  set logs($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLogs() => $_has(0);
  @$pb.TagNumber(1)
  void clearLogs() => $_clearField(1);
}

class InitInstanceRequest extends $pb.GeneratedMessage {
  factory InitInstanceRequest({
    $core.String? name,
    $core.String? ip,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (ip != null) result.ip = ip;
    return result;
  }

  InitInstanceRequest._();

  factory InitInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InitInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InitInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'ip')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitInstanceRequest copyWith(void Function(InitInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as InitInstanceRequest))
          as InitInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InitInstanceRequest create() => InitInstanceRequest._();
  @$core.override
  InitInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InitInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InitInstanceRequest>(create);
  static InitInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get ip => $_getSZ(1);
  @$pb.TagNumber(2)
  set ip($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIp() => $_has(1);
  @$pb.TagNumber(2)
  void clearIp() => $_clearField(2);
}

class InitInstanceResponse extends $pb.GeneratedMessage {
  factory InitInstanceResponse() => create();

  InitInstanceResponse._();

  factory InitInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InitInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InitInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InitInstanceResponse copyWith(void Function(InitInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as InitInstanceResponse))
          as InitInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InitInstanceResponse create() => InitInstanceResponse._();
  @$core.override
  InitInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InitInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InitInstanceResponse>(create);
  static InitInstanceResponse? _defaultInstance;
}

class UpdateInstanceRequest extends $pb.GeneratedMessage {
  factory UpdateInstanceRequest({
    $core.String? id,
    $core.String? ip,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (ip != null) result.ip = ip;
    return result;
  }

  UpdateInstanceRequest._();

  factory UpdateInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UpdateInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'ip')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateInstanceRequest copyWith(
          void Function(UpdateInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as UpdateInstanceRequest))
          as UpdateInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateInstanceRequest create() => UpdateInstanceRequest._();
  @$core.override
  UpdateInstanceRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UpdateInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UpdateInstanceRequest>(create);
  static UpdateInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get ip => $_getSZ(1);
  @$pb.TagNumber(2)
  set ip($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIp() => $_has(1);
  @$pb.TagNumber(2)
  void clearIp() => $_clearField(2);
}

class UpdateInstanceResponse extends $pb.GeneratedMessage {
  factory UpdateInstanceResponse() => create();

  UpdateInstanceResponse._();

  factory UpdateInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UpdateInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateInstanceResponse copyWith(
          void Function(UpdateInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as UpdateInstanceResponse))
          as UpdateInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateInstanceResponse create() => UpdateInstanceResponse._();
  @$core.override
  UpdateInstanceResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UpdateInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UpdateInstanceResponse>(create);
  static UpdateInstanceResponse? _defaultInstance;
}

class GetNetworkStateRequest extends $pb.GeneratedMessage {
  factory GetNetworkStateRequest({
    $core.String? instance,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    return result;
  }

  GetNetworkStateRequest._();

  factory GetNetworkStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetNetworkStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetNetworkStateRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNetworkStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNetworkStateRequest copyWith(
          void Function(GetNetworkStateRequest) updates) =>
      super.copyWith((message) => updates(message as GetNetworkStateRequest))
          as GetNetworkStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetNetworkStateRequest create() => GetNetworkStateRequest._();
  @$core.override
  GetNetworkStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetNetworkStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetNetworkStateRequest>(create);
  static GetNetworkStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);
}

class GetNetworkStateResponse extends $pb.GeneratedMessage {
  factory GetNetworkStateResponse({
    NetworkState? state,
  }) {
    final result = create();
    if (state != null) result.state = state;
    return result;
  }

  GetNetworkStateResponse._();

  factory GetNetworkStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetNetworkStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetNetworkStateResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<NetworkState>(1, _omitFieldNames ? '' : 'state',
        subBuilder: NetworkState.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNetworkStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNetworkStateResponse copyWith(
          void Function(GetNetworkStateResponse) updates) =>
      super.copyWith((message) => updates(message as GetNetworkStateResponse))
          as GetNetworkStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetNetworkStateResponse create() => GetNetworkStateResponse._();
  @$core.override
  GetNetworkStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetNetworkStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetNetworkStateResponse>(create);
  static GetNetworkStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  NetworkState get state => $_getN(0);
  @$pb.TagNumber(1)
  set state(NetworkState value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasState() => $_has(0);
  @$pb.TagNumber(1)
  void clearState() => $_clearField(1);
  @$pb.TagNumber(1)
  NetworkState ensureState() => $_ensure(0);
}

class SetNetworkEnabledRequest extends $pb.GeneratedMessage {
  factory SetNetworkEnabledRequest({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  SetNetworkEnabledRequest._();

  factory SetNetworkEnabledRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetNetworkEnabledRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetNetworkEnabledRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetNetworkEnabledRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetNetworkEnabledRequest copyWith(
          void Function(SetNetworkEnabledRequest) updates) =>
      super.copyWith((message) => updates(message as SetNetworkEnabledRequest))
          as SetNetworkEnabledRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetNetworkEnabledRequest create() => SetNetworkEnabledRequest._();
  @$core.override
  SetNetworkEnabledRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetNetworkEnabledRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetNetworkEnabledRequest>(create);
  static SetNetworkEnabledRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

class SetNetworkEnabledResponse extends $pb.GeneratedMessage {
  factory SetNetworkEnabledResponse({
    NetworkRuntimeStatus? status,
  }) {
    final result = create();
    if (status != null) result.status = status;
    return result;
  }

  SetNetworkEnabledResponse._();

  factory SetNetworkEnabledResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetNetworkEnabledResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetNetworkEnabledResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<NetworkRuntimeStatus>(1, _omitFieldNames ? '' : 'status',
        subBuilder: NetworkRuntimeStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetNetworkEnabledResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetNetworkEnabledResponse copyWith(
          void Function(SetNetworkEnabledResponse) updates) =>
      super.copyWith((message) => updates(message as SetNetworkEnabledResponse))
          as SetNetworkEnabledResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetNetworkEnabledResponse create() => SetNetworkEnabledResponse._();
  @$core.override
  SetNetworkEnabledResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetNetworkEnabledResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetNetworkEnabledResponse>(create);
  static SetNetworkEnabledResponse? _defaultInstance;

  @$pb.TagNumber(1)
  NetworkRuntimeStatus get status => $_getN(0);
  @$pb.TagNumber(1)
  set status(NetworkRuntimeStatus value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearStatus() => $_clearField(1);
  @$pb.TagNumber(1)
  NetworkRuntimeStatus ensureStatus() => $_ensure(0);
}

class NetworkRuntimeStatus extends $pb.GeneratedMessage {
  factory NetworkRuntimeStatus({
    $core.bool? supported,
    $core.bool? desiredEnabled,
    $core.bool? enabled,
    $core.String? state,
    $core.String? message,
    NetworkState? networkState,
  }) {
    final result = create();
    if (supported != null) result.supported = supported;
    if (desiredEnabled != null) result.desiredEnabled = desiredEnabled;
    if (enabled != null) result.enabled = enabled;
    if (state != null) result.state = state;
    if (message != null) result.message = message;
    if (networkState != null) result.networkState = networkState;
    return result;
  }

  NetworkRuntimeStatus._();

  factory NetworkRuntimeStatus.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NetworkRuntimeStatus.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkRuntimeStatus',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'supported')
    ..aOB(2, _omitFieldNames ? '' : 'desiredEnabled')
    ..aOB(3, _omitFieldNames ? '' : 'enabled')
    ..aOS(4, _omitFieldNames ? '' : 'state')
    ..aOS(5, _omitFieldNames ? '' : 'message')
    ..aOM<NetworkState>(6, _omitFieldNames ? '' : 'networkState',
        subBuilder: NetworkState.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkRuntimeStatus clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkRuntimeStatus copyWith(void Function(NetworkRuntimeStatus) updates) =>
      super.copyWith((message) => updates(message as NetworkRuntimeStatus))
          as NetworkRuntimeStatus;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NetworkRuntimeStatus create() => NetworkRuntimeStatus._();
  @$core.override
  NetworkRuntimeStatus createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NetworkRuntimeStatus getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkRuntimeStatus>(create);
  static NetworkRuntimeStatus? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get supported => $_getBF(0);
  @$pb.TagNumber(1)
  set supported($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSupported() => $_has(0);
  @$pb.TagNumber(1)
  void clearSupported() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get desiredEnabled => $_getBF(1);
  @$pb.TagNumber(2)
  set desiredEnabled($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDesiredEnabled() => $_has(1);
  @$pb.TagNumber(2)
  void clearDesiredEnabled() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get enabled => $_getBF(2);
  @$pb.TagNumber(3)
  set enabled($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEnabled() => $_has(2);
  @$pb.TagNumber(3)
  void clearEnabled() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get state => $_getSZ(3);
  @$pb.TagNumber(4)
  set state($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasState() => $_has(3);
  @$pb.TagNumber(4)
  void clearState() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get message => $_getSZ(4);
  @$pb.TagNumber(5)
  set message($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearMessage() => $_clearField(5);

  @$pb.TagNumber(6)
  NetworkState get networkState => $_getN(5);
  @$pb.TagNumber(6)
  set networkState(NetworkState value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasNetworkState() => $_has(5);
  @$pb.TagNumber(6)
  void clearNetworkState() => $_clearField(6);
  @$pb.TagNumber(6)
  NetworkState ensureNetworkState() => $_ensure(5);
}

class NetworkState extends $pb.GeneratedMessage {
  factory NetworkState({
    $core.String? module,
    $core.bool? up,
    $core.String? interfaceName,
    $core.Iterable<NetworkAddress>? addresses,
    $core.Iterable<NetworkRoute>? routes,
    $core.Iterable<WireGuardPeer>? wireguardPeers,
    $core.Iterable<FirewallTable>? firewallTables,
    $core.Iterable<DNSState>? dns,
    $core.Iterable<$core.String>? messages,
    $core.Iterable<NetworkInterface>? interfaces,
  }) {
    final result = create();
    if (module != null) result.module = module;
    if (up != null) result.up = up;
    if (interfaceName != null) result.interfaceName = interfaceName;
    if (addresses != null) result.addresses.addAll(addresses);
    if (routes != null) result.routes.addAll(routes);
    if (wireguardPeers != null) result.wireguardPeers.addAll(wireguardPeers);
    if (firewallTables != null) result.firewallTables.addAll(firewallTables);
    if (dns != null) result.dns.addAll(dns);
    if (messages != null) result.messages.addAll(messages);
    if (interfaces != null) result.interfaces.addAll(interfaces);
    return result;
  }

  NetworkState._();

  factory NetworkState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NetworkState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkState',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'module')
    ..aOB(2, _omitFieldNames ? '' : 'up')
    ..aOS(3, _omitFieldNames ? '' : 'interfaceName')
    ..pPM<NetworkAddress>(4, _omitFieldNames ? '' : 'addresses',
        subBuilder: NetworkAddress.create)
    ..pPM<NetworkRoute>(5, _omitFieldNames ? '' : 'routes',
        subBuilder: NetworkRoute.create)
    ..pPM<WireGuardPeer>(6, _omitFieldNames ? '' : 'wireguardPeers',
        subBuilder: WireGuardPeer.create)
    ..pPM<FirewallTable>(7, _omitFieldNames ? '' : 'firewallTables',
        subBuilder: FirewallTable.create)
    ..pPM<DNSState>(8, _omitFieldNames ? '' : 'dns',
        subBuilder: DNSState.create)
    ..pPS(9, _omitFieldNames ? '' : 'messages')
    ..pPM<NetworkInterface>(10, _omitFieldNames ? '' : 'interfaces',
        subBuilder: NetworkInterface.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkState copyWith(void Function(NetworkState) updates) =>
      super.copyWith((message) => updates(message as NetworkState))
          as NetworkState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NetworkState create() => NetworkState._();
  @$core.override
  NetworkState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NetworkState getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkState>(create);
  static NetworkState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get module => $_getSZ(0);
  @$pb.TagNumber(1)
  set module($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasModule() => $_has(0);
  @$pb.TagNumber(1)
  void clearModule() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get up => $_getBF(1);
  @$pb.TagNumber(2)
  set up($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasUp() => $_has(1);
  @$pb.TagNumber(2)
  void clearUp() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get interfaceName => $_getSZ(2);
  @$pb.TagNumber(3)
  set interfaceName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInterfaceName() => $_has(2);
  @$pb.TagNumber(3)
  void clearInterfaceName() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<NetworkAddress> get addresses => $_getList(3);

  @$pb.TagNumber(5)
  $pb.PbList<NetworkRoute> get routes => $_getList(4);

  @$pb.TagNumber(6)
  $pb.PbList<WireGuardPeer> get wireguardPeers => $_getList(5);

  @$pb.TagNumber(7)
  $pb.PbList<FirewallTable> get firewallTables => $_getList(6);

  @$pb.TagNumber(8)
  $pb.PbList<DNSState> get dns => $_getList(7);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get messages => $_getList(8);

  @$pb.TagNumber(10)
  $pb.PbList<NetworkInterface> get interfaces => $_getList(9);
}

class NetworkInterface extends $pb.GeneratedMessage {
  factory NetworkInterface({
    $core.String? name,
    $core.String? type,
    $core.int? index,
    $core.int? mtu,
    $core.bool? up,
    $core.String? master,
    $core.String? macAddress,
    $core.String? kind,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (type != null) result.type = type;
    if (index != null) result.index = index;
    if (mtu != null) result.mtu = mtu;
    if (up != null) result.up = up;
    if (master != null) result.master = master;
    if (macAddress != null) result.macAddress = macAddress;
    if (kind != null) result.kind = kind;
    return result;
  }

  NetworkInterface._();

  factory NetworkInterface.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NetworkInterface.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkInterface',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'type')
    ..aI(3, _omitFieldNames ? '' : 'index')
    ..aI(4, _omitFieldNames ? '' : 'mtu')
    ..aOB(5, _omitFieldNames ? '' : 'up')
    ..aOS(6, _omitFieldNames ? '' : 'master')
    ..aOS(7, _omitFieldNames ? '' : 'macAddress')
    ..aOS(8, _omitFieldNames ? '' : 'kind')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkInterface clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkInterface copyWith(void Function(NetworkInterface) updates) =>
      super.copyWith((message) => updates(message as NetworkInterface))
          as NetworkInterface;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NetworkInterface create() => NetworkInterface._();
  @$core.override
  NetworkInterface createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NetworkInterface getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkInterface>(create);
  static NetworkInterface? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get type => $_getSZ(1);
  @$pb.TagNumber(2)
  set type($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get index => $_getIZ(2);
  @$pb.TagNumber(3)
  set index($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIndex() => $_has(2);
  @$pb.TagNumber(3)
  void clearIndex() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get mtu => $_getIZ(3);
  @$pb.TagNumber(4)
  set mtu($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMtu() => $_has(3);
  @$pb.TagNumber(4)
  void clearMtu() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get up => $_getBF(4);
  @$pb.TagNumber(5)
  set up($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasUp() => $_has(4);
  @$pb.TagNumber(5)
  void clearUp() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get master => $_getSZ(5);
  @$pb.TagNumber(6)
  set master($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMaster() => $_has(5);
  @$pb.TagNumber(6)
  void clearMaster() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get macAddress => $_getSZ(6);
  @$pb.TagNumber(7)
  set macAddress($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMacAddress() => $_has(6);
  @$pb.TagNumber(7)
  void clearMacAddress() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get kind => $_getSZ(7);
  @$pb.TagNumber(8)
  set kind($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasKind() => $_has(7);
  @$pb.TagNumber(8)
  void clearKind() => $_clearField(8);
}

class NetworkAddress extends $pb.GeneratedMessage {
  factory NetworkAddress({
    $core.String? interfaceName,
    $core.String? cidr,
    $core.String? scope,
  }) {
    final result = create();
    if (interfaceName != null) result.interfaceName = interfaceName;
    if (cidr != null) result.cidr = cidr;
    if (scope != null) result.scope = scope;
    return result;
  }

  NetworkAddress._();

  factory NetworkAddress.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NetworkAddress.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkAddress',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'interfaceName')
    ..aOS(2, _omitFieldNames ? '' : 'cidr')
    ..aOS(3, _omitFieldNames ? '' : 'scope')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkAddress clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkAddress copyWith(void Function(NetworkAddress) updates) =>
      super.copyWith((message) => updates(message as NetworkAddress))
          as NetworkAddress;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NetworkAddress create() => NetworkAddress._();
  @$core.override
  NetworkAddress createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NetworkAddress getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkAddress>(create);
  static NetworkAddress? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get interfaceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set interfaceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInterfaceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearInterfaceName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cidr => $_getSZ(1);
  @$pb.TagNumber(2)
  set cidr($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCidr() => $_has(1);
  @$pb.TagNumber(2)
  void clearCidr() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get scope => $_getSZ(2);
  @$pb.TagNumber(3)
  set scope($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasScope() => $_has(2);
  @$pb.TagNumber(3)
  void clearScope() => $_clearField(3);
}

class NetworkRoute extends $pb.GeneratedMessage {
  factory NetworkRoute({
    $core.String? interfaceName,
    $core.String? destination,
    $core.String? gateway,
    $core.String? source,
    $core.String? family,
    $core.String? table,
    $core.String? protocol,
    $core.String? scope,
    $core.String? priority,
    $core.String? kind,
  }) {
    final result = create();
    if (interfaceName != null) result.interfaceName = interfaceName;
    if (destination != null) result.destination = destination;
    if (gateway != null) result.gateway = gateway;
    if (source != null) result.source = source;
    if (family != null) result.family = family;
    if (table != null) result.table = table;
    if (protocol != null) result.protocol = protocol;
    if (scope != null) result.scope = scope;
    if (priority != null) result.priority = priority;
    if (kind != null) result.kind = kind;
    return result;
  }

  NetworkRoute._();

  factory NetworkRoute.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory NetworkRoute.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkRoute',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'interfaceName')
    ..aOS(2, _omitFieldNames ? '' : 'destination')
    ..aOS(3, _omitFieldNames ? '' : 'gateway')
    ..aOS(4, _omitFieldNames ? '' : 'source')
    ..aOS(5, _omitFieldNames ? '' : 'family')
    ..aOS(6, _omitFieldNames ? '' : 'table')
    ..aOS(7, _omitFieldNames ? '' : 'protocol')
    ..aOS(8, _omitFieldNames ? '' : 'scope')
    ..aOS(9, _omitFieldNames ? '' : 'priority')
    ..aOS(10, _omitFieldNames ? '' : 'kind')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkRoute clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkRoute copyWith(void Function(NetworkRoute) updates) =>
      super.copyWith((message) => updates(message as NetworkRoute))
          as NetworkRoute;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NetworkRoute create() => NetworkRoute._();
  @$core.override
  NetworkRoute createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static NetworkRoute getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkRoute>(create);
  static NetworkRoute? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get interfaceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set interfaceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInterfaceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearInterfaceName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get destination => $_getSZ(1);
  @$pb.TagNumber(2)
  set destination($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDestination() => $_has(1);
  @$pb.TagNumber(2)
  void clearDestination() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get gateway => $_getSZ(2);
  @$pb.TagNumber(3)
  set gateway($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasGateway() => $_has(2);
  @$pb.TagNumber(3)
  void clearGateway() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get source => $_getSZ(3);
  @$pb.TagNumber(4)
  set source($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSource() => $_has(3);
  @$pb.TagNumber(4)
  void clearSource() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get family => $_getSZ(4);
  @$pb.TagNumber(5)
  set family($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasFamily() => $_has(4);
  @$pb.TagNumber(5)
  void clearFamily() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get table => $_getSZ(5);
  @$pb.TagNumber(6)
  set table($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTable() => $_has(5);
  @$pb.TagNumber(6)
  void clearTable() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get protocol => $_getSZ(6);
  @$pb.TagNumber(7)
  set protocol($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasProtocol() => $_has(6);
  @$pb.TagNumber(7)
  void clearProtocol() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get scope => $_getSZ(7);
  @$pb.TagNumber(8)
  set scope($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasScope() => $_has(7);
  @$pb.TagNumber(8)
  void clearScope() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get priority => $_getSZ(8);
  @$pb.TagNumber(9)
  set priority($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPriority() => $_has(8);
  @$pb.TagNumber(9)
  void clearPriority() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get kind => $_getSZ(9);
  @$pb.TagNumber(10)
  set kind($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasKind() => $_has(9);
  @$pb.TagNumber(10)
  void clearKind() => $_clearField(10);
}

class WireGuardPeer extends $pb.GeneratedMessage {
  factory WireGuardPeer({
    $core.String? publicKey,
    $core.String? endpoint,
    $core.Iterable<$core.String>? allowedIps,
    $core.String? latestHandshake,
    $fixnum.Int64? rxBytes,
    $fixnum.Int64? txBytes,
  }) {
    final result = create();
    if (publicKey != null) result.publicKey = publicKey;
    if (endpoint != null) result.endpoint = endpoint;
    if (allowedIps != null) result.allowedIps.addAll(allowedIps);
    if (latestHandshake != null) result.latestHandshake = latestHandshake;
    if (rxBytes != null) result.rxBytes = rxBytes;
    if (txBytes != null) result.txBytes = txBytes;
    return result;
  }

  WireGuardPeer._();

  factory WireGuardPeer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WireGuardPeer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WireGuardPeer',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'publicKey')
    ..aOS(2, _omitFieldNames ? '' : 'endpoint')
    ..pPS(3, _omitFieldNames ? '' : 'allowedIps')
    ..aOS(4, _omitFieldNames ? '' : 'latestHandshake')
    ..a<$fixnum.Int64>(5, _omitFieldNames ? '' : 'rxBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'txBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireGuardPeer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireGuardPeer copyWith(void Function(WireGuardPeer) updates) =>
      super.copyWith((message) => updates(message as WireGuardPeer))
          as WireGuardPeer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WireGuardPeer create() => WireGuardPeer._();
  @$core.override
  WireGuardPeer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WireGuardPeer getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WireGuardPeer>(create);
  static WireGuardPeer? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get publicKey => $_getSZ(0);
  @$pb.TagNumber(1)
  set publicKey($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPublicKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearPublicKey() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpoint => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpoint($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpoint() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpoint() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get allowedIps => $_getList(2);

  @$pb.TagNumber(4)
  $core.String get latestHandshake => $_getSZ(3);
  @$pb.TagNumber(4)
  set latestHandshake($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLatestHandshake() => $_has(3);
  @$pb.TagNumber(4)
  void clearLatestHandshake() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get rxBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set rxBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRxBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearRxBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get txBytes => $_getI64(5);
  @$pb.TagNumber(6)
  set txBytes($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTxBytes() => $_has(5);
  @$pb.TagNumber(6)
  void clearTxBytes() => $_clearField(6);
}

class FirewallTable extends $pb.GeneratedMessage {
  factory FirewallTable({
    $core.String? family,
    $core.String? name,
    $core.Iterable<FirewallChain>? chains,
  }) {
    final result = create();
    if (family != null) result.family = family;
    if (name != null) result.name = name;
    if (chains != null) result.chains.addAll(chains);
    return result;
  }

  FirewallTable._();

  factory FirewallTable.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FirewallTable.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FirewallTable',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'family')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..pPM<FirewallChain>(3, _omitFieldNames ? '' : 'chains',
        subBuilder: FirewallChain.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallTable clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallTable copyWith(void Function(FirewallTable) updates) =>
      super.copyWith((message) => updates(message as FirewallTable))
          as FirewallTable;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FirewallTable create() => FirewallTable._();
  @$core.override
  FirewallTable createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FirewallTable getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FirewallTable>(create);
  static FirewallTable? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get family => $_getSZ(0);
  @$pb.TagNumber(1)
  set family($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFamily() => $_has(0);
  @$pb.TagNumber(1)
  void clearFamily() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<FirewallChain> get chains => $_getList(2);
}

class FirewallChain extends $pb.GeneratedMessage {
  factory FirewallChain({
    $core.String? name,
    $core.String? type,
    $core.String? hook,
    $core.String? priority,
    $core.Iterable<FirewallRule>? rules,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (type != null) result.type = type;
    if (hook != null) result.hook = hook;
    if (priority != null) result.priority = priority;
    if (rules != null) result.rules.addAll(rules);
    return result;
  }

  FirewallChain._();

  factory FirewallChain.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FirewallChain.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FirewallChain',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'type')
    ..aOS(3, _omitFieldNames ? '' : 'hook')
    ..aOS(4, _omitFieldNames ? '' : 'priority')
    ..pPM<FirewallRule>(5, _omitFieldNames ? '' : 'rules',
        subBuilder: FirewallRule.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallChain clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallChain copyWith(void Function(FirewallChain) updates) =>
      super.copyWith((message) => updates(message as FirewallChain))
          as FirewallChain;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FirewallChain create() => FirewallChain._();
  @$core.override
  FirewallChain createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FirewallChain getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FirewallChain>(create);
  static FirewallChain? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get type => $_getSZ(1);
  @$pb.TagNumber(2)
  set type($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get hook => $_getSZ(2);
  @$pb.TagNumber(3)
  set hook($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasHook() => $_has(2);
  @$pb.TagNumber(3)
  void clearHook() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get priority => $_getSZ(3);
  @$pb.TagNumber(4)
  set priority($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPriority() => $_has(3);
  @$pb.TagNumber(4)
  void clearPriority() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<FirewallRule> get rules => $_getList(4);
}

class FirewallRule extends $pb.GeneratedMessage {
  factory FirewallRule({
    $core.Iterable<$core.String>? expressions,
    $fixnum.Int64? packets,
    $fixnum.Int64? bytes,
  }) {
    final result = create();
    if (expressions != null) result.expressions.addAll(expressions);
    if (packets != null) result.packets = packets;
    if (bytes != null) result.bytes = bytes;
    return result;
  }

  FirewallRule._();

  factory FirewallRule.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FirewallRule.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FirewallRule',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'expressions')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'packets', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'bytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallRule clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FirewallRule copyWith(void Function(FirewallRule) updates) =>
      super.copyWith((message) => updates(message as FirewallRule))
          as FirewallRule;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FirewallRule create() => FirewallRule._();
  @$core.override
  FirewallRule createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FirewallRule getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FirewallRule>(create);
  static FirewallRule? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get expressions => $_getList(0);

  @$pb.TagNumber(2)
  $fixnum.Int64 get packets => $_getI64(1);
  @$pb.TagNumber(2)
  set packets($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPackets() => $_has(1);
  @$pb.TagNumber(2)
  void clearPackets() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get bytes => $_getI64(2);
  @$pb.TagNumber(3)
  set bytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearBytes() => $_clearField(3);
}

class DNSState extends $pb.GeneratedMessage {
  factory DNSState({
    $core.String? scope,
    $core.String? domain,
    $core.Iterable<$core.String>? servers,
    $core.int? port,
    $core.bool? active,
    $core.String? source,
  }) {
    final result = create();
    if (scope != null) result.scope = scope;
    if (domain != null) result.domain = domain;
    if (servers != null) result.servers.addAll(servers);
    if (port != null) result.port = port;
    if (active != null) result.active = active;
    if (source != null) result.source = source;
    return result;
  }

  DNSState._();

  factory DNSState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DNSState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DNSState',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'scope')
    ..aOS(2, _omitFieldNames ? '' : 'domain')
    ..pPS(3, _omitFieldNames ? '' : 'servers')
    ..aI(4, _omitFieldNames ? '' : 'port')
    ..aOB(5, _omitFieldNames ? '' : 'active')
    ..aOS(6, _omitFieldNames ? '' : 'source')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DNSState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DNSState copyWith(void Function(DNSState) updates) =>
      super.copyWith((message) => updates(message as DNSState)) as DNSState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DNSState create() => DNSState._();
  @$core.override
  DNSState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DNSState getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DNSState>(create);
  static DNSState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get scope => $_getSZ(0);
  @$pb.TagNumber(1)
  set scope($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasScope() => $_has(0);
  @$pb.TagNumber(1)
  void clearScope() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get domain => $_getSZ(1);
  @$pb.TagNumber(2)
  set domain($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDomain() => $_has(1);
  @$pb.TagNumber(2)
  void clearDomain() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get servers => $_getList(2);

  @$pb.TagNumber(4)
  $core.int get port => $_getIZ(3);
  @$pb.TagNumber(4)
  set port($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPort() => $_has(3);
  @$pb.TagNumber(4)
  void clearPort() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get active => $_getBF(4);
  @$pb.TagNumber(5)
  set active($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasActive() => $_has(4);
  @$pb.TagNumber(5)
  void clearActive() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get source => $_getSZ(5);
  @$pb.TagNumber(6)
  set source($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasSource() => $_has(5);
  @$pb.TagNumber(6)
  void clearSource() => $_clearField(6);
}

class ExitRoute extends $pb.GeneratedMessage {
  factory ExitRoute({
    $core.String? id,
    $core.String? deviceId,
    $core.String? instanceId,
    $core.String? instanceName,
    $core.String? publicIp,
    $core.String? location,
    $core.String? status,
    $core.String? dnsServer,
    $core.Iterable<$core.String>? cidrs,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (deviceId != null) result.deviceId = deviceId;
    if (instanceId != null) result.instanceId = instanceId;
    if (instanceName != null) result.instanceName = instanceName;
    if (publicIp != null) result.publicIp = publicIp;
    if (location != null) result.location = location;
    if (status != null) result.status = status;
    if (dnsServer != null) result.dnsServer = dnsServer;
    if (cidrs != null) result.cidrs.addAll(cidrs);
    return result;
  }

  ExitRoute._();

  factory ExitRoute.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ExitRoute.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ExitRoute',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'instanceId')
    ..aOS(4, _omitFieldNames ? '' : 'instanceName')
    ..aOS(5, _omitFieldNames ? '' : 'publicIp')
    ..aOS(6, _omitFieldNames ? '' : 'location')
    ..aOS(7, _omitFieldNames ? '' : 'status')
    ..aOS(8, _omitFieldNames ? '' : 'dnsServer')
    ..pPS(9, _omitFieldNames ? '' : 'cidrs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExitRoute clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExitRoute copyWith(void Function(ExitRoute) updates) =>
      super.copyWith((message) => updates(message as ExitRoute)) as ExitRoute;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ExitRoute create() => ExitRoute._();
  @$core.override
  ExitRoute createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ExitRoute getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ExitRoute>(create);
  static ExitRoute? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get instanceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set instanceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInstanceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearInstanceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get instanceName => $_getSZ(3);
  @$pb.TagNumber(4)
  set instanceName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasInstanceName() => $_has(3);
  @$pb.TagNumber(4)
  void clearInstanceName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get publicIp => $_getSZ(4);
  @$pb.TagNumber(5)
  set publicIp($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPublicIp() => $_has(4);
  @$pb.TagNumber(5)
  void clearPublicIp() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get location => $_getSZ(5);
  @$pb.TagNumber(6)
  set location($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasLocation() => $_has(5);
  @$pb.TagNumber(6)
  void clearLocation() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get status => $_getSZ(6);
  @$pb.TagNumber(7)
  set status($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasStatus() => $_has(6);
  @$pb.TagNumber(7)
  void clearStatus() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get dnsServer => $_getSZ(7);
  @$pb.TagNumber(8)
  set dnsServer($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDnsServer() => $_has(7);
  @$pb.TagNumber(8)
  void clearDnsServer() => $_clearField(8);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get cidrs => $_getList(8);
}

class GetExitRoutesRequest extends $pb.GeneratedMessage {
  factory GetExitRoutesRequest({
    $core.String? instance,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    return result;
  }

  GetExitRoutesRequest._();

  factory GetExitRoutesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetExitRoutesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetExitRoutesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExitRoutesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExitRoutesRequest copyWith(void Function(GetExitRoutesRequest) updates) =>
      super.copyWith((message) => updates(message as GetExitRoutesRequest))
          as GetExitRoutesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetExitRoutesRequest create() => GetExitRoutesRequest._();
  @$core.override
  GetExitRoutesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetExitRoutesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetExitRoutesRequest>(create);
  static GetExitRoutesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);
}

class GetExitRoutesResponse extends $pb.GeneratedMessage {
  factory GetExitRoutesResponse({
    $core.Iterable<ExitRoute>? routes,
  }) {
    final result = create();
    if (routes != null) result.routes.addAll(routes);
    return result;
  }

  GetExitRoutesResponse._();

  factory GetExitRoutesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetExitRoutesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetExitRoutesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<ExitRoute>(1, _omitFieldNames ? '' : 'routes',
        subBuilder: ExitRoute.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExitRoutesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExitRoutesResponse copyWith(
          void Function(GetExitRoutesResponse) updates) =>
      super.copyWith((message) => updates(message as GetExitRoutesResponse))
          as GetExitRoutesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetExitRoutesResponse create() => GetExitRoutesResponse._();
  @$core.override
  GetExitRoutesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetExitRoutesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetExitRoutesResponse>(create);
  static GetExitRoutesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ExitRoute> get routes => $_getList(0);
}

class GetMobileTunnelConfigRequest extends $pb.GeneratedMessage {
  factory GetMobileTunnelConfigRequest({
    $core.String? instance,
    $core.String? deviceId,
    $core.String? dnsServer,
    $core.Iterable<$core.String>? cidrs,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (deviceId != null) result.deviceId = deviceId;
    if (dnsServer != null) result.dnsServer = dnsServer;
    if (cidrs != null) result.cidrs.addAll(cidrs);
    return result;
  }

  GetMobileTunnelConfigRequest._();

  factory GetMobileTunnelConfigRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetMobileTunnelConfigRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetMobileTunnelConfigRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'dnsServer')
    ..pPS(4, _omitFieldNames ? '' : 'cidrs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMobileTunnelConfigRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMobileTunnelConfigRequest copyWith(
          void Function(GetMobileTunnelConfigRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetMobileTunnelConfigRequest))
          as GetMobileTunnelConfigRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetMobileTunnelConfigRequest create() =>
      GetMobileTunnelConfigRequest._();
  @$core.override
  GetMobileTunnelConfigRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetMobileTunnelConfigRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetMobileTunnelConfigRequest>(create);
  static GetMobileTunnelConfigRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get dnsServer => $_getSZ(2);
  @$pb.TagNumber(3)
  set dnsServer($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDnsServer() => $_has(2);
  @$pb.TagNumber(3)
  void clearDnsServer() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get cidrs => $_getList(3);
}

class MobileTunnelConfig extends $pb.GeneratedMessage {
  factory MobileTunnelConfig({
    $core.String? configId,
    $fixnum.Int64? generatedAtUnix,
    $core.String? instanceId,
    $core.String? instanceName,
    $core.String? peerPublicKey,
    $core.String? peerEndpoint,
    $core.Iterable<$core.String>? interfaceAddresses,
    $core.Iterable<$core.String>? dnsServers,
    $core.Iterable<$core.String>? includedRoutes,
    $core.Iterable<$core.String>? excludedRoutes,
    $core.int? mtu,
    $core.Iterable<$core.String>? allowedIps,
    $core.int? persistentKeepaliveSeconds,
    $core.String? keychainAccessGroup,
    $core.String? keychainAccount,
    $core.String? wireguardPrivateKey,
  }) {
    final result = create();
    if (configId != null) result.configId = configId;
    if (generatedAtUnix != null) result.generatedAtUnix = generatedAtUnix;
    if (instanceId != null) result.instanceId = instanceId;
    if (instanceName != null) result.instanceName = instanceName;
    if (peerPublicKey != null) result.peerPublicKey = peerPublicKey;
    if (peerEndpoint != null) result.peerEndpoint = peerEndpoint;
    if (interfaceAddresses != null)
      result.interfaceAddresses.addAll(interfaceAddresses);
    if (dnsServers != null) result.dnsServers.addAll(dnsServers);
    if (includedRoutes != null) result.includedRoutes.addAll(includedRoutes);
    if (excludedRoutes != null) result.excludedRoutes.addAll(excludedRoutes);
    if (mtu != null) result.mtu = mtu;
    if (allowedIps != null) result.allowedIps.addAll(allowedIps);
    if (persistentKeepaliveSeconds != null)
      result.persistentKeepaliveSeconds = persistentKeepaliveSeconds;
    if (keychainAccessGroup != null)
      result.keychainAccessGroup = keychainAccessGroup;
    if (keychainAccount != null) result.keychainAccount = keychainAccount;
    if (wireguardPrivateKey != null)
      result.wireguardPrivateKey = wireguardPrivateKey;
    return result;
  }

  MobileTunnelConfig._();

  factory MobileTunnelConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory MobileTunnelConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MobileTunnelConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'configId')
    ..aInt64(2, _omitFieldNames ? '' : 'generatedAtUnix')
    ..aOS(3, _omitFieldNames ? '' : 'instanceId')
    ..aOS(4, _omitFieldNames ? '' : 'instanceName')
    ..aOS(5, _omitFieldNames ? '' : 'peerPublicKey')
    ..aOS(6, _omitFieldNames ? '' : 'peerEndpoint')
    ..pPS(7, _omitFieldNames ? '' : 'interfaceAddresses')
    ..pPS(8, _omitFieldNames ? '' : 'dnsServers')
    ..pPS(9, _omitFieldNames ? '' : 'includedRoutes')
    ..pPS(10, _omitFieldNames ? '' : 'excludedRoutes')
    ..aI(11, _omitFieldNames ? '' : 'mtu')
    ..pPS(12, _omitFieldNames ? '' : 'allowedIps')
    ..aI(13, _omitFieldNames ? '' : 'persistentKeepaliveSeconds')
    ..aOS(14, _omitFieldNames ? '' : 'keychainAccessGroup')
    ..aOS(15, _omitFieldNames ? '' : 'keychainAccount')
    ..aOS(16, _omitFieldNames ? '' : 'wireguardPrivateKey')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MobileTunnelConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MobileTunnelConfig copyWith(void Function(MobileTunnelConfig) updates) =>
      super.copyWith((message) => updates(message as MobileTunnelConfig))
          as MobileTunnelConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MobileTunnelConfig create() => MobileTunnelConfig._();
  @$core.override
  MobileTunnelConfig createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static MobileTunnelConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MobileTunnelConfig>(create);
  static MobileTunnelConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get configId => $_getSZ(0);
  @$pb.TagNumber(1)
  set configId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasConfigId() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfigId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get generatedAtUnix => $_getI64(1);
  @$pb.TagNumber(2)
  set generatedAtUnix($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGeneratedAtUnix() => $_has(1);
  @$pb.TagNumber(2)
  void clearGeneratedAtUnix() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get instanceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set instanceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInstanceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearInstanceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get instanceName => $_getSZ(3);
  @$pb.TagNumber(4)
  set instanceName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasInstanceName() => $_has(3);
  @$pb.TagNumber(4)
  void clearInstanceName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get peerPublicKey => $_getSZ(4);
  @$pb.TagNumber(5)
  set peerPublicKey($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPeerPublicKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearPeerPublicKey() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get peerEndpoint => $_getSZ(5);
  @$pb.TagNumber(6)
  set peerEndpoint($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPeerEndpoint() => $_has(5);
  @$pb.TagNumber(6)
  void clearPeerEndpoint() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get interfaceAddresses => $_getList(6);

  @$pb.TagNumber(8)
  $pb.PbList<$core.String> get dnsServers => $_getList(7);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get includedRoutes => $_getList(8);

  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get excludedRoutes => $_getList(9);

  @$pb.TagNumber(11)
  $core.int get mtu => $_getIZ(10);
  @$pb.TagNumber(11)
  set mtu($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasMtu() => $_has(10);
  @$pb.TagNumber(11)
  void clearMtu() => $_clearField(11);

  @$pb.TagNumber(12)
  $pb.PbList<$core.String> get allowedIps => $_getList(11);

  @$pb.TagNumber(13)
  $core.int get persistentKeepaliveSeconds => $_getIZ(12);
  @$pb.TagNumber(13)
  set persistentKeepaliveSeconds($core.int value) =>
      $_setSignedInt32(12, value);
  @$pb.TagNumber(13)
  $core.bool hasPersistentKeepaliveSeconds() => $_has(12);
  @$pb.TagNumber(13)
  void clearPersistentKeepaliveSeconds() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get keychainAccessGroup => $_getSZ(13);
  @$pb.TagNumber(14)
  set keychainAccessGroup($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasKeychainAccessGroup() => $_has(13);
  @$pb.TagNumber(14)
  void clearKeychainAccessGroup() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get keychainAccount => $_getSZ(14);
  @$pb.TagNumber(15)
  set keychainAccount($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasKeychainAccount() => $_has(14);
  @$pb.TagNumber(15)
  void clearKeychainAccount() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get wireguardPrivateKey => $_getSZ(15);
  @$pb.TagNumber(16)
  set wireguardPrivateKey($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasWireguardPrivateKey() => $_has(15);
  @$pb.TagNumber(16)
  void clearWireguardPrivateKey() => $_clearField(16);
}

class GetMobileTunnelConfigResponse extends $pb.GeneratedMessage {
  factory GetMobileTunnelConfigResponse({
    MobileTunnelConfig? config,
  }) {
    final result = create();
    if (config != null) result.config = config;
    return result;
  }

  GetMobileTunnelConfigResponse._();

  factory GetMobileTunnelConfigResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetMobileTunnelConfigResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetMobileTunnelConfigResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<MobileTunnelConfig>(1, _omitFieldNames ? '' : 'config',
        subBuilder: MobileTunnelConfig.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMobileTunnelConfigResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMobileTunnelConfigResponse copyWith(
          void Function(GetMobileTunnelConfigResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetMobileTunnelConfigResponse))
          as GetMobileTunnelConfigResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetMobileTunnelConfigResponse create() =>
      GetMobileTunnelConfigResponse._();
  @$core.override
  GetMobileTunnelConfigResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetMobileTunnelConfigResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetMobileTunnelConfigResponse>(create);
  static GetMobileTunnelConfigResponse? _defaultInstance;

  @$pb.TagNumber(1)
  MobileTunnelConfig get config => $_getN(0);
  @$pb.TagNumber(1)
  set config(MobileTunnelConfig value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfig() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfig() => $_clearField(1);
  @$pb.TagNumber(1)
  MobileTunnelConfig ensureConfig() => $_ensure(0);
}

class GetRuntimeStateRequest extends $pb.GeneratedMessage {
  factory GetRuntimeStateRequest({
    $core.String? instance,
    $core.bool? allowStale,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (allowStale != null) result.allowStale = allowStale;
    return result;
  }

  GetRuntimeStateRequest._();

  factory GetRuntimeStateRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetRuntimeStateRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetRuntimeStateRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..aOB(2, _omitFieldNames ? '' : 'allowStale')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRuntimeStateRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRuntimeStateRequest copyWith(
          void Function(GetRuntimeStateRequest) updates) =>
      super.copyWith((message) => updates(message as GetRuntimeStateRequest))
          as GetRuntimeStateRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetRuntimeStateRequest create() => GetRuntimeStateRequest._();
  @$core.override
  GetRuntimeStateRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetRuntimeStateRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetRuntimeStateRequest>(create);
  static GetRuntimeStateRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get allowStale => $_getBF(1);
  @$pb.TagNumber(2)
  set allowStale($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAllowStale() => $_has(1);
  @$pb.TagNumber(2)
  void clearAllowStale() => $_clearField(2);
}

class GetRuntimeStateResponse extends $pb.GeneratedMessage {
  factory GetRuntimeStateResponse({
    RuntimeState? state,
  }) {
    final result = create();
    if (state != null) result.state = state;
    return result;
  }

  GetRuntimeStateResponse._();

  factory GetRuntimeStateResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetRuntimeStateResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetRuntimeStateResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<RuntimeState>(1, _omitFieldNames ? '' : 'state',
        subBuilder: RuntimeState.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRuntimeStateResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRuntimeStateResponse copyWith(
          void Function(GetRuntimeStateResponse) updates) =>
      super.copyWith((message) => updates(message as GetRuntimeStateResponse))
          as GetRuntimeStateResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetRuntimeStateResponse create() => GetRuntimeStateResponse._();
  @$core.override
  GetRuntimeStateResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetRuntimeStateResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetRuntimeStateResponse>(create);
  static GetRuntimeStateResponse? _defaultInstance;

  @$pb.TagNumber(1)
  RuntimeState get state => $_getN(0);
  @$pb.TagNumber(1)
  set state(RuntimeState value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasState() => $_has(0);
  @$pb.TagNumber(1)
  void clearState() => $_clearField(1);
  @$pb.TagNumber(1)
  RuntimeState ensureState() => $_ensure(0);
}

class WatchChangesRequest extends $pb.GeneratedMessage {
  factory WatchChangesRequest({
    $core.bool? includeSnapshot,
    $core.int? heartbeatIntervalMs,
  }) {
    final result = create();
    if (includeSnapshot != null) result.includeSnapshot = includeSnapshot;
    if (heartbeatIntervalMs != null)
      result.heartbeatIntervalMs = heartbeatIntervalMs;
    return result;
  }

  WatchChangesRequest._();

  factory WatchChangesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WatchChangesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WatchChangesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'includeSnapshot')
    ..aI(2, _omitFieldNames ? '' : 'heartbeatIntervalMs',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchChangesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchChangesRequest copyWith(void Function(WatchChangesRequest) updates) =>
      super.copyWith((message) => updates(message as WatchChangesRequest))
          as WatchChangesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WatchChangesRequest create() => WatchChangesRequest._();
  @$core.override
  WatchChangesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WatchChangesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WatchChangesRequest>(create);
  static WatchChangesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get includeSnapshot => $_getBF(0);
  @$pb.TagNumber(1)
  set includeSnapshot($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIncludeSnapshot() => $_has(0);
  @$pb.TagNumber(1)
  void clearIncludeSnapshot() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get heartbeatIntervalMs => $_getIZ(1);
  @$pb.TagNumber(2)
  set heartbeatIntervalMs($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHeartbeatIntervalMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeartbeatIntervalMs() => $_clearField(2);
}

class WatchChangesResponse extends $pb.GeneratedMessage {
  factory WatchChangesResponse({
    $fixnum.Int64? sequence,
    $core.Iterable<$core.String>? tableNames,
    $core.bool? runtimeChanged,
    $core.String? reason,
  }) {
    final result = create();
    if (sequence != null) result.sequence = sequence;
    if (tableNames != null) result.tableNames.addAll(tableNames);
    if (runtimeChanged != null) result.runtimeChanged = runtimeChanged;
    if (reason != null) result.reason = reason;
    return result;
  }

  WatchChangesResponse._();

  factory WatchChangesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WatchChangesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WatchChangesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPS(2, _omitFieldNames ? '' : 'tableNames')
    ..aOB(3, _omitFieldNames ? '' : 'runtimeChanged')
    ..aOS(4, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchChangesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchChangesResponse copyWith(void Function(WatchChangesResponse) updates) =>
      super.copyWith((message) => updates(message as WatchChangesResponse))
          as WatchChangesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WatchChangesResponse create() => WatchChangesResponse._();
  @$core.override
  WatchChangesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WatchChangesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WatchChangesResponse>(create);
  static WatchChangesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sequence => $_getI64(0);
  @$pb.TagNumber(1)
  set sequence($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSequence() => $_has(0);
  @$pb.TagNumber(1)
  void clearSequence() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get tableNames => $_getList(1);

  @$pb.TagNumber(3)
  $core.bool get runtimeChanged => $_getBF(2);
  @$pb.TagNumber(3)
  set runtimeChanged($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRuntimeChanged() => $_has(2);
  @$pb.TagNumber(3)
  void clearRuntimeChanged() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get reason => $_getSZ(3);
  @$pb.TagNumber(4)
  set reason($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReason() => $_has(3);
  @$pb.TagNumber(4)
  void clearReason() => $_clearField(4);
}

class Task extends $pb.GeneratedMessage {
  factory Task({
    $core.String? id,
    $core.String? stream,
    $core.String? subjectType,
    $core.String? subjectId,
    $core.String? status,
    $core.String? title,
    $core.String? message,
    $core.int? progress,
    $core.String? payloadJson,
    $core.String? resultJson,
    $core.String? errorMessage,
    $core.int? attempts,
    $core.int? maxAttempts,
    $core.String? createdAt,
    $core.String? updatedAt,
    $core.String? startedAt,
    $core.String? finishedAt,
    $core.String? ownerPeerId,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (stream != null) result.stream = stream;
    if (subjectType != null) result.subjectType = subjectType;
    if (subjectId != null) result.subjectId = subjectId;
    if (status != null) result.status = status;
    if (title != null) result.title = title;
    if (message != null) result.message = message;
    if (progress != null) result.progress = progress;
    if (payloadJson != null) result.payloadJson = payloadJson;
    if (resultJson != null) result.resultJson = resultJson;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (attempts != null) result.attempts = attempts;
    if (maxAttempts != null) result.maxAttempts = maxAttempts;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (startedAt != null) result.startedAt = startedAt;
    if (finishedAt != null) result.finishedAt = finishedAt;
    if (ownerPeerId != null) result.ownerPeerId = ownerPeerId;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  Task._();

  factory Task.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Task.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Task',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'stream')
    ..aOS(3, _omitFieldNames ? '' : 'subjectType')
    ..aOS(4, _omitFieldNames ? '' : 'subjectId')
    ..aOS(5, _omitFieldNames ? '' : 'status')
    ..aOS(6, _omitFieldNames ? '' : 'title')
    ..aOS(7, _omitFieldNames ? '' : 'message')
    ..aI(8, _omitFieldNames ? '' : 'progress')
    ..aOS(9, _omitFieldNames ? '' : 'payloadJson')
    ..aOS(10, _omitFieldNames ? '' : 'resultJson')
    ..aOS(11, _omitFieldNames ? '' : 'errorMessage')
    ..aI(12, _omitFieldNames ? '' : 'attempts')
    ..aI(13, _omitFieldNames ? '' : 'maxAttempts')
    ..aOS(14, _omitFieldNames ? '' : 'createdAt')
    ..aOS(15, _omitFieldNames ? '' : 'updatedAt')
    ..aOS(16, _omitFieldNames ? '' : 'startedAt')
    ..aOS(17, _omitFieldNames ? '' : 'finishedAt')
    ..aOS(18, _omitFieldNames ? '' : 'ownerPeerId')
    ..aOM<WriteConfirmation>(19, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Task clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Task copyWith(void Function(Task) updates) =>
      super.copyWith((message) => updates(message as Task)) as Task;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Task create() => Task._();
  @$core.override
  Task createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Task getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Task>(create);
  static Task? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get stream => $_getSZ(1);
  @$pb.TagNumber(2)
  set stream($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStream() => $_has(1);
  @$pb.TagNumber(2)
  void clearStream() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subjectType => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectType() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get subjectId => $_getSZ(3);
  @$pb.TagNumber(4)
  set subjectId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSubjectId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSubjectId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get status => $_getSZ(4);
  @$pb.TagNumber(5)
  set status($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get title => $_getSZ(5);
  @$pb.TagNumber(6)
  set title($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTitle() => $_has(5);
  @$pb.TagNumber(6)
  void clearTitle() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get message => $_getSZ(6);
  @$pb.TagNumber(7)
  set message($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMessage() => $_has(6);
  @$pb.TagNumber(7)
  void clearMessage() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get progress => $_getIZ(7);
  @$pb.TagNumber(8)
  set progress($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasProgress() => $_has(7);
  @$pb.TagNumber(8)
  void clearProgress() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get payloadJson => $_getSZ(8);
  @$pb.TagNumber(9)
  set payloadJson($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasPayloadJson() => $_has(8);
  @$pb.TagNumber(9)
  void clearPayloadJson() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get resultJson => $_getSZ(9);
  @$pb.TagNumber(10)
  set resultJson($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasResultJson() => $_has(9);
  @$pb.TagNumber(10)
  void clearResultJson() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.String get errorMessage => $_getSZ(10);
  @$pb.TagNumber(11)
  set errorMessage($core.String value) => $_setString(10, value);
  @$pb.TagNumber(11)
  $core.bool hasErrorMessage() => $_has(10);
  @$pb.TagNumber(11)
  void clearErrorMessage() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.int get attempts => $_getIZ(11);
  @$pb.TagNumber(12)
  set attempts($core.int value) => $_setSignedInt32(11, value);
  @$pb.TagNumber(12)
  $core.bool hasAttempts() => $_has(11);
  @$pb.TagNumber(12)
  void clearAttempts() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.int get maxAttempts => $_getIZ(12);
  @$pb.TagNumber(13)
  set maxAttempts($core.int value) => $_setSignedInt32(12, value);
  @$pb.TagNumber(13)
  $core.bool hasMaxAttempts() => $_has(12);
  @$pb.TagNumber(13)
  void clearMaxAttempts() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get createdAt => $_getSZ(13);
  @$pb.TagNumber(14)
  set createdAt($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasCreatedAt() => $_has(13);
  @$pb.TagNumber(14)
  void clearCreatedAt() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get updatedAt => $_getSZ(14);
  @$pb.TagNumber(15)
  set updatedAt($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasUpdatedAt() => $_has(14);
  @$pb.TagNumber(15)
  void clearUpdatedAt() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get startedAt => $_getSZ(15);
  @$pb.TagNumber(16)
  set startedAt($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasStartedAt() => $_has(15);
  @$pb.TagNumber(16)
  void clearStartedAt() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.String get finishedAt => $_getSZ(16);
  @$pb.TagNumber(17)
  set finishedAt($core.String value) => $_setString(16, value);
  @$pb.TagNumber(17)
  $core.bool hasFinishedAt() => $_has(16);
  @$pb.TagNumber(17)
  void clearFinishedAt() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.String get ownerPeerId => $_getSZ(17);
  @$pb.TagNumber(18)
  set ownerPeerId($core.String value) => $_setString(17, value);
  @$pb.TagNumber(18)
  $core.bool hasOwnerPeerId() => $_has(17);
  @$pb.TagNumber(18)
  void clearOwnerPeerId() => $_clearField(18);

  @$pb.TagNumber(19)
  WriteConfirmation get confirmation => $_getN(18);
  @$pb.TagNumber(19)
  set confirmation(WriteConfirmation value) => $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasConfirmation() => $_has(18);
  @$pb.TagNumber(19)
  void clearConfirmation() => $_clearField(19);
  @$pb.TagNumber(19)
  WriteConfirmation ensureConfirmation() => $_ensure(18);
}

class TaskEvent extends $pb.GeneratedMessage {
  factory TaskEvent({
    $core.String? id,
    $core.String? taskId,
    $core.String? status,
    $core.String? message,
    $core.int? progress,
    $core.String? detailsJson,
    $core.String? createdAt,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (taskId != null) result.taskId = taskId;
    if (status != null) result.status = status;
    if (message != null) result.message = message;
    if (progress != null) result.progress = progress;
    if (detailsJson != null) result.detailsJson = detailsJson;
    if (createdAt != null) result.createdAt = createdAt;
    return result;
  }

  TaskEvent._();

  factory TaskEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TaskEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TaskEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'taskId')
    ..aOS(3, _omitFieldNames ? '' : 'status')
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..aI(5, _omitFieldNames ? '' : 'progress')
    ..aOS(6, _omitFieldNames ? '' : 'detailsJson')
    ..aOS(7, _omitFieldNames ? '' : 'createdAt')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TaskEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TaskEvent copyWith(void Function(TaskEvent) updates) =>
      super.copyWith((message) => updates(message as TaskEvent)) as TaskEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TaskEvent create() => TaskEvent._();
  @$core.override
  TaskEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TaskEvent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TaskEvent>(create);
  static TaskEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get taskId => $_getSZ(1);
  @$pb.TagNumber(2)
  set taskId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTaskId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTaskId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get status => $_getSZ(2);
  @$pb.TagNumber(3)
  set status($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasStatus() => $_has(2);
  @$pb.TagNumber(3)
  void clearStatus() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get message => $_getSZ(3);
  @$pb.TagNumber(4)
  set message($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessage() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get progress => $_getIZ(4);
  @$pb.TagNumber(5)
  set progress($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProgress() => $_has(4);
  @$pb.TagNumber(5)
  void clearProgress() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get detailsJson => $_getSZ(5);
  @$pb.TagNumber(6)
  set detailsJson($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDetailsJson() => $_has(5);
  @$pb.TagNumber(6)
  void clearDetailsJson() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get createdAt => $_getSZ(6);
  @$pb.TagNumber(7)
  set createdAt($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCreatedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearCreatedAt() => $_clearField(7);
}

class GetTasksRequest extends $pb.GeneratedMessage {
  factory GetTasksRequest({
    $core.String? status,
    $core.String? stream,
    $core.String? subjectType,
    $core.String? subjectId,
    $core.int? maxResults,
    $core.String? instance,
  }) {
    final result = create();
    if (status != null) result.status = status;
    if (stream != null) result.stream = stream;
    if (subjectType != null) result.subjectType = subjectType;
    if (subjectId != null) result.subjectId = subjectId;
    if (maxResults != null) result.maxResults = maxResults;
    if (instance != null) result.instance = instance;
    return result;
  }

  GetTasksRequest._();

  factory GetTasksRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTasksRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTasksRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'status')
    ..aOS(2, _omitFieldNames ? '' : 'stream')
    ..aOS(3, _omitFieldNames ? '' : 'subjectType')
    ..aOS(4, _omitFieldNames ? '' : 'subjectId')
    ..aI(5, _omitFieldNames ? '' : 'maxResults')
    ..aOS(6, _omitFieldNames ? '' : 'instance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTasksRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTasksRequest copyWith(void Function(GetTasksRequest) updates) =>
      super.copyWith((message) => updates(message as GetTasksRequest))
          as GetTasksRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTasksRequest create() => GetTasksRequest._();
  @$core.override
  GetTasksRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTasksRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTasksRequest>(create);
  static GetTasksRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get status => $_getSZ(0);
  @$pb.TagNumber(1)
  set status($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearStatus() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get stream => $_getSZ(1);
  @$pb.TagNumber(2)
  set stream($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStream() => $_has(1);
  @$pb.TagNumber(2)
  void clearStream() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subjectType => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectType() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get subjectId => $_getSZ(3);
  @$pb.TagNumber(4)
  set subjectId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSubjectId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSubjectId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get maxResults => $_getIZ(4);
  @$pb.TagNumber(5)
  set maxResults($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMaxResults() => $_has(4);
  @$pb.TagNumber(5)
  void clearMaxResults() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get instance => $_getSZ(5);
  @$pb.TagNumber(6)
  set instance($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasInstance() => $_has(5);
  @$pb.TagNumber(6)
  void clearInstance() => $_clearField(6);
}

class GetTasksResponse extends $pb.GeneratedMessage {
  factory GetTasksResponse({
    $core.Iterable<Task>? tasks,
    $core.bool? truncated,
  }) {
    final result = create();
    if (tasks != null) result.tasks.addAll(tasks);
    if (truncated != null) result.truncated = truncated;
    return result;
  }

  GetTasksResponse._();

  factory GetTasksResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTasksResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTasksResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Task>(1, _omitFieldNames ? '' : 'tasks', subBuilder: Task.create)
    ..aOB(2, _omitFieldNames ? '' : 'truncated')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTasksResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTasksResponse copyWith(void Function(GetTasksResponse) updates) =>
      super.copyWith((message) => updates(message as GetTasksResponse))
          as GetTasksResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTasksResponse create() => GetTasksResponse._();
  @$core.override
  GetTasksResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTasksResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTasksResponse>(create);
  static GetTasksResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Task> get tasks => $_getList(0);

  @$pb.TagNumber(2)
  $core.bool get truncated => $_getBF(1);
  @$pb.TagNumber(2)
  set truncated($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTruncated() => $_has(1);
  @$pb.TagNumber(2)
  void clearTruncated() => $_clearField(2);
}

class GetTaskRequest extends $pb.GeneratedMessage {
  factory GetTaskRequest({
    $core.String? id,
    $core.bool? includeEvents,
    $core.String? instance,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (includeEvents != null) result.includeEvents = includeEvents;
    if (instance != null) result.instance = instance;
    return result;
  }

  GetTaskRequest._();

  factory GetTaskRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTaskRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTaskRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOB(2, _omitFieldNames ? '' : 'includeEvents')
    ..aOS(3, _omitFieldNames ? '' : 'instance')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTaskRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTaskRequest copyWith(void Function(GetTaskRequest) updates) =>
      super.copyWith((message) => updates(message as GetTaskRequest))
          as GetTaskRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTaskRequest create() => GetTaskRequest._();
  @$core.override
  GetTaskRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTaskRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTaskRequest>(create);
  static GetTaskRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includeEvents => $_getBF(1);
  @$pb.TagNumber(2)
  set includeEvents($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIncludeEvents() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludeEvents() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get instance => $_getSZ(2);
  @$pb.TagNumber(3)
  set instance($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasInstance() => $_has(2);
  @$pb.TagNumber(3)
  void clearInstance() => $_clearField(3);
}

class GetTaskResponse extends $pb.GeneratedMessage {
  factory GetTaskResponse({
    Task? task,
    $core.Iterable<TaskEvent>? events,
  }) {
    final result = create();
    if (task != null) result.task = task;
    if (events != null) result.events.addAll(events);
    return result;
  }

  GetTaskResponse._();

  factory GetTaskResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetTaskResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetTaskResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<Task>(1, _omitFieldNames ? '' : 'task', subBuilder: Task.create)
    ..pPM<TaskEvent>(2, _omitFieldNames ? '' : 'events',
        subBuilder: TaskEvent.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTaskResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetTaskResponse copyWith(void Function(GetTaskResponse) updates) =>
      super.copyWith((message) => updates(message as GetTaskResponse))
          as GetTaskResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetTaskResponse create() => GetTaskResponse._();
  @$core.override
  GetTaskResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetTaskResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetTaskResponse>(create);
  static GetTaskResponse? _defaultInstance;

  @$pb.TagNumber(1)
  Task get task => $_getN(0);
  @$pb.TagNumber(1)
  set task(Task value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasTask() => $_has(0);
  @$pb.TagNumber(1)
  void clearTask() => $_clearField(1);
  @$pb.TagNumber(1)
  Task ensureTask() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<TaskEvent> get events => $_getList(1);
}

class TaskProgressUpdate extends $pb.GeneratedMessage {
  factory TaskProgressUpdate({
    $core.String? taskId,
    $core.String? status,
    $core.String? message,
    $core.int? progress,
    $core.String? detailsJson,
    $core.String? createdAt,
    $core.bool? durable,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (taskId != null) result.taskId = taskId;
    if (status != null) result.status = status;
    if (message != null) result.message = message;
    if (progress != null) result.progress = progress;
    if (detailsJson != null) result.detailsJson = detailsJson;
    if (createdAt != null) result.createdAt = createdAt;
    if (durable != null) result.durable = durable;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  TaskProgressUpdate._();

  factory TaskProgressUpdate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TaskProgressUpdate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TaskProgressUpdate',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'taskId')
    ..aOS(2, _omitFieldNames ? '' : 'status')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..aI(4, _omitFieldNames ? '' : 'progress')
    ..aOS(5, _omitFieldNames ? '' : 'detailsJson')
    ..aOS(6, _omitFieldNames ? '' : 'createdAt')
    ..aOB(7, _omitFieldNames ? '' : 'durable')
    ..aOM<WriteConfirmation>(8, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TaskProgressUpdate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TaskProgressUpdate copyWith(void Function(TaskProgressUpdate) updates) =>
      super.copyWith((message) => updates(message as TaskProgressUpdate))
          as TaskProgressUpdate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TaskProgressUpdate create() => TaskProgressUpdate._();
  @$core.override
  TaskProgressUpdate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TaskProgressUpdate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TaskProgressUpdate>(create);
  static TaskProgressUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get taskId => $_getSZ(0);
  @$pb.TagNumber(1)
  set taskId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTaskId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTaskId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get status => $_getSZ(1);
  @$pb.TagNumber(2)
  set status($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get progress => $_getIZ(3);
  @$pb.TagNumber(4)
  set progress($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProgress() => $_has(3);
  @$pb.TagNumber(4)
  void clearProgress() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get detailsJson => $_getSZ(4);
  @$pb.TagNumber(5)
  set detailsJson($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDetailsJson() => $_has(4);
  @$pb.TagNumber(5)
  void clearDetailsJson() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get createdAt => $_getSZ(5);
  @$pb.TagNumber(6)
  set createdAt($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCreatedAt() => $_has(5);
  @$pb.TagNumber(6)
  void clearCreatedAt() => $_clearField(6);

  /// True when the task state was saved and its local root published; this is not Swarmion event applied_durably or content durability.
  @$pb.TagNumber(7)
  $core.bool get durable => $_getBF(6);
  @$pb.TagNumber(7)
  set durable($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasDurable() => $_has(6);
  @$pb.TagNumber(7)
  void clearDurable() => $_clearField(7);

  @$pb.TagNumber(8)
  WriteConfirmation get confirmation => $_getN(7);
  @$pb.TagNumber(8)
  set confirmation(WriteConfirmation value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasConfirmation() => $_has(7);
  @$pb.TagNumber(8)
  void clearConfirmation() => $_clearField(8);
  @$pb.TagNumber(8)
  WriteConfirmation ensureConfirmation() => $_ensure(7);
}

class WatchTaskRequest extends $pb.GeneratedMessage {
  factory WatchTaskRequest({
    $core.String? id,
    $core.bool? includeSnapshot,
    $core.bool? includeEvents,
    $core.int? heartbeatIntervalMs,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (includeSnapshot != null) result.includeSnapshot = includeSnapshot;
    if (includeEvents != null) result.includeEvents = includeEvents;
    if (heartbeatIntervalMs != null)
      result.heartbeatIntervalMs = heartbeatIntervalMs;
    return result;
  }

  WatchTaskRequest._();

  factory WatchTaskRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WatchTaskRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WatchTaskRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOB(2, _omitFieldNames ? '' : 'includeSnapshot')
    ..aOB(3, _omitFieldNames ? '' : 'includeEvents')
    ..aI(4, _omitFieldNames ? '' : 'heartbeatIntervalMs',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchTaskRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchTaskRequest copyWith(void Function(WatchTaskRequest) updates) =>
      super.copyWith((message) => updates(message as WatchTaskRequest))
          as WatchTaskRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WatchTaskRequest create() => WatchTaskRequest._();
  @$core.override
  WatchTaskRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WatchTaskRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WatchTaskRequest>(create);
  static WatchTaskRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includeSnapshot => $_getBF(1);
  @$pb.TagNumber(2)
  set includeSnapshot($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIncludeSnapshot() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludeSnapshot() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get includeEvents => $_getBF(2);
  @$pb.TagNumber(3)
  set includeEvents($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIncludeEvents() => $_has(2);
  @$pb.TagNumber(3)
  void clearIncludeEvents() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get heartbeatIntervalMs => $_getIZ(3);
  @$pb.TagNumber(4)
  set heartbeatIntervalMs($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasHeartbeatIntervalMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearHeartbeatIntervalMs() => $_clearField(4);
}

class WatchTaskResponse extends $pb.GeneratedMessage {
  factory WatchTaskResponse({
    $fixnum.Int64? sequence,
    Task? task,
    $core.Iterable<TaskEvent>? events,
    TaskProgressUpdate? update,
    $core.bool? heartbeat,
  }) {
    final result = create();
    if (sequence != null) result.sequence = sequence;
    if (task != null) result.task = task;
    if (events != null) result.events.addAll(events);
    if (update != null) result.update = update;
    if (heartbeat != null) result.heartbeat = heartbeat;
    return result;
  }

  WatchTaskResponse._();

  factory WatchTaskResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WatchTaskResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WatchTaskResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<Task>(2, _omitFieldNames ? '' : 'task', subBuilder: Task.create)
    ..pPM<TaskEvent>(3, _omitFieldNames ? '' : 'events',
        subBuilder: TaskEvent.create)
    ..aOM<TaskProgressUpdate>(4, _omitFieldNames ? '' : 'update',
        subBuilder: TaskProgressUpdate.create)
    ..aOB(5, _omitFieldNames ? '' : 'heartbeat')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchTaskResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WatchTaskResponse copyWith(void Function(WatchTaskResponse) updates) =>
      super.copyWith((message) => updates(message as WatchTaskResponse))
          as WatchTaskResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WatchTaskResponse create() => WatchTaskResponse._();
  @$core.override
  WatchTaskResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WatchTaskResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WatchTaskResponse>(create);
  static WatchTaskResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get sequence => $_getI64(0);
  @$pb.TagNumber(1)
  set sequence($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSequence() => $_has(0);
  @$pb.TagNumber(1)
  void clearSequence() => $_clearField(1);

  @$pb.TagNumber(2)
  Task get task => $_getN(1);
  @$pb.TagNumber(2)
  set task(Task value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTask() => $_has(1);
  @$pb.TagNumber(2)
  void clearTask() => $_clearField(2);
  @$pb.TagNumber(2)
  Task ensureTask() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<TaskEvent> get events => $_getList(2);

  @$pb.TagNumber(4)
  TaskProgressUpdate get update => $_getN(3);
  @$pb.TagNumber(4)
  set update(TaskProgressUpdate value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasUpdate() => $_has(3);
  @$pb.TagNumber(4)
  void clearUpdate() => $_clearField(4);
  @$pb.TagNumber(4)
  TaskProgressUpdate ensureUpdate() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get heartbeat => $_getBF(4);
  @$pb.TagNumber(5)
  set heartbeat($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasHeartbeat() => $_has(4);
  @$pb.TagNumber(5)
  void clearHeartbeat() => $_clearField(5);
}

class SetExitRouteRequest extends $pb.GeneratedMessage {
  factory SetExitRouteRequest({
    $core.String? instance,
    $core.String? deviceId,
    $core.String? dnsServer,
    $core.Iterable<$core.String>? cidrs,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (deviceId != null) result.deviceId = deviceId;
    if (dnsServer != null) result.dnsServer = dnsServer;
    if (cidrs != null) result.cidrs.addAll(cidrs);
    return result;
  }

  SetExitRouteRequest._();

  factory SetExitRouteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetExitRouteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetExitRouteRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'dnsServer')
    ..pPS(4, _omitFieldNames ? '' : 'cidrs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetExitRouteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetExitRouteRequest copyWith(void Function(SetExitRouteRequest) updates) =>
      super.copyWith((message) => updates(message as SetExitRouteRequest))
          as SetExitRouteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetExitRouteRequest create() => SetExitRouteRequest._();
  @$core.override
  SetExitRouteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetExitRouteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetExitRouteRequest>(create);
  static SetExitRouteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get dnsServer => $_getSZ(2);
  @$pb.TagNumber(3)
  set dnsServer($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDnsServer() => $_has(2);
  @$pb.TagNumber(3)
  void clearDnsServer() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get cidrs => $_getList(3);
}

class SetExitRouteResponse extends $pb.GeneratedMessage {
  factory SetExitRouteResponse({
    ExitRoute? route,
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (route != null) result.route = route;
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  SetExitRouteResponse._();

  factory SetExitRouteResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetExitRouteResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetExitRouteResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<ExitRoute>(1, _omitFieldNames ? '' : 'route',
        subBuilder: ExitRoute.create)
    ..aOM<WriteConfirmation>(2, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetExitRouteResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetExitRouteResponse copyWith(void Function(SetExitRouteResponse) updates) =>
      super.copyWith((message) => updates(message as SetExitRouteResponse))
          as SetExitRouteResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetExitRouteResponse create() => SetExitRouteResponse._();
  @$core.override
  SetExitRouteResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetExitRouteResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SetExitRouteResponse>(create);
  static SetExitRouteResponse? _defaultInstance;

  @$pb.TagNumber(1)
  ExitRoute get route => $_getN(0);
  @$pb.TagNumber(1)
  set route(ExitRoute value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRoute() => $_has(0);
  @$pb.TagNumber(1)
  void clearRoute() => $_clearField(1);
  @$pb.TagNumber(1)
  ExitRoute ensureRoute() => $_ensure(0);

  @$pb.TagNumber(2)
  WriteConfirmation get confirmation => $_getN(1);
  @$pb.TagNumber(2)
  set confirmation(WriteConfirmation value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasConfirmation() => $_has(1);
  @$pb.TagNumber(2)
  void clearConfirmation() => $_clearField(2);
  @$pb.TagNumber(2)
  WriteConfirmation ensureConfirmation() => $_ensure(1);
}

class ClearExitRouteRequest extends $pb.GeneratedMessage {
  factory ClearExitRouteRequest({
    $core.String? deviceId,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    return result;
  }

  ClearExitRouteRequest._();

  factory ClearExitRouteRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClearExitRouteRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClearExitRouteRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClearExitRouteRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClearExitRouteRequest copyWith(
          void Function(ClearExitRouteRequest) updates) =>
      super.copyWith((message) => updates(message as ClearExitRouteRequest))
          as ClearExitRouteRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClearExitRouteRequest create() => ClearExitRouteRequest._();
  @$core.override
  ClearExitRouteRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClearExitRouteRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClearExitRouteRequest>(create);
  static ClearExitRouteRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);
}

class ClearExitRouteResponse extends $pb.GeneratedMessage {
  factory ClearExitRouteResponse({
    WriteConfirmation? confirmation,
  }) {
    final result = create();
    if (confirmation != null) result.confirmation = confirmation;
    return result;
  }

  ClearExitRouteResponse._();

  factory ClearExitRouteResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClearExitRouteResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClearExitRouteResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<WriteConfirmation>(1, _omitFieldNames ? '' : 'confirmation',
        subBuilder: WriteConfirmation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClearExitRouteResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClearExitRouteResponse copyWith(
          void Function(ClearExitRouteResponse) updates) =>
      super.copyWith((message) => updates(message as ClearExitRouteResponse))
          as ClearExitRouteResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClearExitRouteResponse create() => ClearExitRouteResponse._();
  @$core.override
  ClearExitRouteResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClearExitRouteResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClearExitRouteResponse>(create);
  static ClearExitRouteResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WriteConfirmation get confirmation => $_getN(0);
  @$pb.TagNumber(1)
  set confirmation(WriteConfirmation value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfirmation() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfirmation() => $_clearField(1);
  @$pb.TagNumber(1)
  WriteConfirmation ensureConfirmation() => $_ensure(0);
}

class RuntimeState extends $pb.GeneratedMessage {
  factory RuntimeState({
    $core.String? peerId,
    $core.String? manifestDigest,
    $core.String? checkpointRootHash,
    $core.String? tentativeRootHash,
    $core.String? protocolCheckpointRootHash,
    $core.String? durableMainRootHash,
    $core.Iterable<$core.String>? stateProviders,
    $core.String? fatalState,
    $core.bool? runtimeRefreshPending,
    $core.String? runtimeRefreshLastError,
    $core.bool? runtimeCheckpointPending,
    $core.String? runtimeCheckpointLastError,
    $core.String? runtimeMaterializationPolicy,
    $core.Iterable<RuntimePeerStatus>? peerStatuses,
    $core.Iterable<RuntimeCompatibility>? compatibility,
    $core.Iterable<$core.String>? contentSyncTrace,
    $core.String? protocolCheckpointDigest,
    $core.String? readConsistency,
    $core.String? readError,
    $fixnum.Int64? eventReceiptContentDissentObservations,
    $core.Iterable<$core.String>? routedPeers,
    $core.Iterable<$core.String>? participatingPeers,
    $core.Iterable<$core.String>? logicalPeers,
    $core.int? logicalPeerTarget,
    $core.Iterable<$core.String>? physicalConnectedPeers,
  }) {
    final result = create();
    if (peerId != null) result.peerId = peerId;
    if (manifestDigest != null) result.manifestDigest = manifestDigest;
    if (checkpointRootHash != null)
      result.checkpointRootHash = checkpointRootHash;
    if (tentativeRootHash != null) result.tentativeRootHash = tentativeRootHash;
    if (protocolCheckpointRootHash != null)
      result.protocolCheckpointRootHash = protocolCheckpointRootHash;
    if (durableMainRootHash != null)
      result.durableMainRootHash = durableMainRootHash;
    if (stateProviders != null) result.stateProviders.addAll(stateProviders);
    if (fatalState != null) result.fatalState = fatalState;
    if (runtimeRefreshPending != null)
      result.runtimeRefreshPending = runtimeRefreshPending;
    if (runtimeRefreshLastError != null)
      result.runtimeRefreshLastError = runtimeRefreshLastError;
    if (runtimeCheckpointPending != null)
      result.runtimeCheckpointPending = runtimeCheckpointPending;
    if (runtimeCheckpointLastError != null)
      result.runtimeCheckpointLastError = runtimeCheckpointLastError;
    if (runtimeMaterializationPolicy != null)
      result.runtimeMaterializationPolicy = runtimeMaterializationPolicy;
    if (peerStatuses != null) result.peerStatuses.addAll(peerStatuses);
    if (compatibility != null) result.compatibility.addAll(compatibility);
    if (contentSyncTrace != null)
      result.contentSyncTrace.addAll(contentSyncTrace);
    if (protocolCheckpointDigest != null)
      result.protocolCheckpointDigest = protocolCheckpointDigest;
    if (readConsistency != null) result.readConsistency = readConsistency;
    if (readError != null) result.readError = readError;
    if (eventReceiptContentDissentObservations != null)
      result.eventReceiptContentDissentObservations =
          eventReceiptContentDissentObservations;
    if (routedPeers != null) result.routedPeers.addAll(routedPeers);
    if (participatingPeers != null)
      result.participatingPeers.addAll(participatingPeers);
    if (logicalPeers != null) result.logicalPeers.addAll(logicalPeers);
    if (logicalPeerTarget != null) result.logicalPeerTarget = logicalPeerTarget;
    if (physicalConnectedPeers != null)
      result.physicalConnectedPeers.addAll(physicalConnectedPeers);
    return result;
  }

  RuntimeState._();

  factory RuntimeState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeState',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'peerId')
    ..aOS(2, _omitFieldNames ? '' : 'manifestDigest')
    ..aOS(3, _omitFieldNames ? '' : 'checkpointRootHash')
    ..aOS(4, _omitFieldNames ? '' : 'tentativeRootHash')
    ..aOS(5, _omitFieldNames ? '' : 'protocolCheckpointRootHash')
    ..aOS(6, _omitFieldNames ? '' : 'durableMainRootHash')
    ..pPS(10, _omitFieldNames ? '' : 'stateProviders')
    ..aOS(12, _omitFieldNames ? '' : 'fatalState')
    ..aOB(13, _omitFieldNames ? '' : 'runtimeRefreshPending')
    ..aOS(14, _omitFieldNames ? '' : 'runtimeRefreshLastError')
    ..aOB(15, _omitFieldNames ? '' : 'runtimeCheckpointPending')
    ..aOS(16, _omitFieldNames ? '' : 'runtimeCheckpointLastError')
    ..aOS(17, _omitFieldNames ? '' : 'runtimeMaterializationPolicy')
    ..pPM<RuntimePeerStatus>(18, _omitFieldNames ? '' : 'peerStatuses',
        subBuilder: RuntimePeerStatus.create)
    ..pPM<RuntimeCompatibility>(19, _omitFieldNames ? '' : 'compatibility',
        subBuilder: RuntimeCompatibility.create)
    ..pPS(20, _omitFieldNames ? '' : 'contentSyncTrace')
    ..aOS(24, _omitFieldNames ? '' : 'protocolCheckpointDigest')
    ..aOS(25, _omitFieldNames ? '' : 'readConsistency')
    ..aOS(26, _omitFieldNames ? '' : 'readError')
    ..a<$fixnum.Int64>(
        27,
        _omitFieldNames ? '' : 'eventReceiptContentDissentObservations',
        $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..pPS(28, _omitFieldNames ? '' : 'routedPeers')
    ..pPS(29, _omitFieldNames ? '' : 'participatingPeers')
    ..pPS(30, _omitFieldNames ? '' : 'logicalPeers')
    ..aI(31, _omitFieldNames ? '' : 'logicalPeerTarget')
    ..pPS(32, _omitFieldNames ? '' : 'physicalConnectedPeers')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeState copyWith(void Function(RuntimeState) updates) =>
      super.copyWith((message) => updates(message as RuntimeState))
          as RuntimeState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeState create() => RuntimeState._();
  @$core.override
  RuntimeState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeState getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeState>(create);
  static RuntimeState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get peerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set peerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPeerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPeerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get manifestDigest => $_getSZ(1);
  @$pb.TagNumber(2)
  set manifestDigest($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasManifestDigest() => $_has(1);
  @$pb.TagNumber(2)
  void clearManifestDigest() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get checkpointRootHash => $_getSZ(2);
  @$pb.TagNumber(3)
  set checkpointRootHash($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCheckpointRootHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearCheckpointRootHash() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get tentativeRootHash => $_getSZ(3);
  @$pb.TagNumber(4)
  set tentativeRootHash($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTentativeRootHash() => $_has(3);
  @$pb.TagNumber(4)
  void clearTentativeRootHash() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get protocolCheckpointRootHash => $_getSZ(4);
  @$pb.TagNumber(5)
  set protocolCheckpointRootHash($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProtocolCheckpointRootHash() => $_has(4);
  @$pb.TagNumber(5)
  void clearProtocolCheckpointRootHash() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get durableMainRootHash => $_getSZ(5);
  @$pb.TagNumber(6)
  set durableMainRootHash($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDurableMainRootHash() => $_has(5);
  @$pb.TagNumber(6)
  void clearDurableMainRootHash() => $_clearField(6);

  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get stateProviders => $_getList(6);

  @$pb.TagNumber(12)
  $core.String get fatalState => $_getSZ(7);
  @$pb.TagNumber(12)
  set fatalState($core.String value) => $_setString(7, value);
  @$pb.TagNumber(12)
  $core.bool hasFatalState() => $_has(7);
  @$pb.TagNumber(12)
  void clearFatalState() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.bool get runtimeRefreshPending => $_getBF(8);
  @$pb.TagNumber(13)
  set runtimeRefreshPending($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(13)
  $core.bool hasRuntimeRefreshPending() => $_has(8);
  @$pb.TagNumber(13)
  void clearRuntimeRefreshPending() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get runtimeRefreshLastError => $_getSZ(9);
  @$pb.TagNumber(14)
  set runtimeRefreshLastError($core.String value) => $_setString(9, value);
  @$pb.TagNumber(14)
  $core.bool hasRuntimeRefreshLastError() => $_has(9);
  @$pb.TagNumber(14)
  void clearRuntimeRefreshLastError() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get runtimeCheckpointPending => $_getBF(10);
  @$pb.TagNumber(15)
  set runtimeCheckpointPending($core.bool value) => $_setBool(10, value);
  @$pb.TagNumber(15)
  $core.bool hasRuntimeCheckpointPending() => $_has(10);
  @$pb.TagNumber(15)
  void clearRuntimeCheckpointPending() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get runtimeCheckpointLastError => $_getSZ(11);
  @$pb.TagNumber(16)
  set runtimeCheckpointLastError($core.String value) => $_setString(11, value);
  @$pb.TagNumber(16)
  $core.bool hasRuntimeCheckpointLastError() => $_has(11);
  @$pb.TagNumber(16)
  void clearRuntimeCheckpointLastError() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.String get runtimeMaterializationPolicy => $_getSZ(12);
  @$pb.TagNumber(17)
  set runtimeMaterializationPolicy($core.String value) =>
      $_setString(12, value);
  @$pb.TagNumber(17)
  $core.bool hasRuntimeMaterializationPolicy() => $_has(12);
  @$pb.TagNumber(17)
  void clearRuntimeMaterializationPolicy() => $_clearField(17);

  @$pb.TagNumber(18)
  $pb.PbList<RuntimePeerStatus> get peerStatuses => $_getList(13);

  @$pb.TagNumber(19)
  $pb.PbList<RuntimeCompatibility> get compatibility => $_getList(14);

  @$pb.TagNumber(20)
  $pb.PbList<$core.String> get contentSyncTrace => $_getList(15);

  @$pb.TagNumber(24)
  $core.String get protocolCheckpointDigest => $_getSZ(16);
  @$pb.TagNumber(24)
  set protocolCheckpointDigest($core.String value) => $_setString(16, value);
  @$pb.TagNumber(24)
  $core.bool hasProtocolCheckpointDigest() => $_has(16);
  @$pb.TagNumber(24)
  void clearProtocolCheckpointDigest() => $_clearField(24);

  @$pb.TagNumber(25)
  $core.String get readConsistency => $_getSZ(17);
  @$pb.TagNumber(25)
  set readConsistency($core.String value) => $_setString(17, value);
  @$pb.TagNumber(25)
  $core.bool hasReadConsistency() => $_has(17);
  @$pb.TagNumber(25)
  void clearReadConsistency() => $_clearField(25);

  @$pb.TagNumber(26)
  $core.String get readError => $_getSZ(18);
  @$pb.TagNumber(26)
  set readError($core.String value) => $_setString(18, value);
  @$pb.TagNumber(26)
  $core.bool hasReadError() => $_has(18);
  @$pb.TagNumber(26)
  void clearReadError() => $_clearField(26);

  /// Exact-event content_dissent observations recorded by this backend process.
  @$pb.TagNumber(27)
  $fixnum.Int64 get eventReceiptContentDissentObservations => $_getI64(19);
  @$pb.TagNumber(27)
  set eventReceiptContentDissentObservations($fixnum.Int64 value) =>
      $_setInt64(19, value);
  @$pb.TagNumber(27)
  $core.bool hasEventReceiptContentDissentObservations() => $_has(19);
  @$pb.TagNumber(27)
  void clearEventReceiptContentDissentObservations() => $_clearField(27);

  /// Swarmion peers for which the application-owned transport has a route.
  @$pb.TagNumber(28)
  $pb.PbList<$core.String> get routedPeers => $_getList(20);

  /// Routed peers participating in this Swarmion database scope.
  @$pb.TagNumber(29)
  $pb.PbList<$core.String> get participatingPeers => $_getList(21);

  /// The bounded Swarmion messaging overlay. This is not physical connectivity.
  @$pb.TagNumber(30)
  $pb.PbList<$core.String> get logicalPeers => $_getList(22);

  @$pb.TagNumber(31)
  $core.int get logicalPeerTarget => $_getIZ(23);
  @$pb.TagNumber(31)
  set logicalPeerTarget($core.int value) => $_setSignedInt32(23, value);
  @$pb.TagNumber(31)
  $core.bool hasLogicalPeerTarget() => $_has(23);
  @$pb.TagNumber(31)
  void clearLogicalPeerTarget() => $_clearField(31);

  /// Peers with a live connection on the application-owned physical host.
  @$pb.TagNumber(32)
  $pb.PbList<$core.String> get physicalConnectedPeers => $_getList(24);
}

class RuntimePeerStatus extends $pb.GeneratedMessage {
  factory RuntimePeerStatus({
    $core.String? peerId,
    $core.bool? stateProvider,
    $core.bool? compatible,
    $core.bool? incompatible,
    $core.bool? ignored,
    $core.bool? relayOnly,
    $core.Iterable<$core.String>? addresses,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? lastDialErrors,
    $core.String? reason,
    $core.int? replicationPriority,
    $core.String? replicationDeviceClass,
    $core.bool? routed,
    $core.bool? participating,
    $core.bool? logical,
    $core.bool? physicalConnected,
    $fixnum.Int64? lastRoutedAtUnixNano,
  }) {
    final result = create();
    if (peerId != null) result.peerId = peerId;
    if (stateProvider != null) result.stateProvider = stateProvider;
    if (compatible != null) result.compatible = compatible;
    if (incompatible != null) result.incompatible = incompatible;
    if (ignored != null) result.ignored = ignored;
    if (relayOnly != null) result.relayOnly = relayOnly;
    if (addresses != null) result.addresses.addAll(addresses);
    if (lastDialErrors != null)
      result.lastDialErrors.addEntries(lastDialErrors);
    if (reason != null) result.reason = reason;
    if (replicationPriority != null)
      result.replicationPriority = replicationPriority;
    if (replicationDeviceClass != null)
      result.replicationDeviceClass = replicationDeviceClass;
    if (routed != null) result.routed = routed;
    if (participating != null) result.participating = participating;
    if (logical != null) result.logical = logical;
    if (physicalConnected != null) result.physicalConnected = physicalConnected;
    if (lastRoutedAtUnixNano != null)
      result.lastRoutedAtUnixNano = lastRoutedAtUnixNano;
    return result;
  }

  RuntimePeerStatus._();

  factory RuntimePeerStatus.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimePeerStatus.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimePeerStatus',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'peerId')
    ..aOB(4, _omitFieldNames ? '' : 'stateProvider')
    ..aOB(7, _omitFieldNames ? '' : 'compatible')
    ..aOB(8, _omitFieldNames ? '' : 'incompatible')
    ..aOB(9, _omitFieldNames ? '' : 'ignored')
    ..aOB(10, _omitFieldNames ? '' : 'relayOnly')
    ..pPS(11, _omitFieldNames ? '' : 'addresses')
    ..m<$core.String, $core.String>(12, _omitFieldNames ? '' : 'lastDialErrors',
        entryClassName: 'RuntimePeerStatus.LastDialErrorsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..aOS(13, _omitFieldNames ? '' : 'reason')
    ..aI(14, _omitFieldNames ? '' : 'replicationPriority')
    ..aOS(15, _omitFieldNames ? '' : 'replicationDeviceClass')
    ..aOB(16, _omitFieldNames ? '' : 'routed')
    ..aOB(17, _omitFieldNames ? '' : 'participating')
    ..aOB(18, _omitFieldNames ? '' : 'logical')
    ..aOB(19, _omitFieldNames ? '' : 'physicalConnected')
    ..aInt64(20, _omitFieldNames ? '' : 'lastRoutedAtUnixNano')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimePeerStatus clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimePeerStatus copyWith(void Function(RuntimePeerStatus) updates) =>
      super.copyWith((message) => updates(message as RuntimePeerStatus))
          as RuntimePeerStatus;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimePeerStatus create() => RuntimePeerStatus._();
  @$core.override
  RuntimePeerStatus createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimePeerStatus getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimePeerStatus>(create);
  static RuntimePeerStatus? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get peerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set peerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPeerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPeerId() => $_clearField(1);

  @$pb.TagNumber(4)
  $core.bool get stateProvider => $_getBF(1);
  @$pb.TagNumber(4)
  set stateProvider($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(4)
  $core.bool hasStateProvider() => $_has(1);
  @$pb.TagNumber(4)
  void clearStateProvider() => $_clearField(4);

  @$pb.TagNumber(7)
  $core.bool get compatible => $_getBF(2);
  @$pb.TagNumber(7)
  set compatible($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(7)
  $core.bool hasCompatible() => $_has(2);
  @$pb.TagNumber(7)
  void clearCompatible() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get incompatible => $_getBF(3);
  @$pb.TagNumber(8)
  set incompatible($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(8)
  $core.bool hasIncompatible() => $_has(3);
  @$pb.TagNumber(8)
  void clearIncompatible() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get ignored => $_getBF(4);
  @$pb.TagNumber(9)
  set ignored($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(9)
  $core.bool hasIgnored() => $_has(4);
  @$pb.TagNumber(9)
  void clearIgnored() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get relayOnly => $_getBF(5);
  @$pb.TagNumber(10)
  set relayOnly($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(10)
  $core.bool hasRelayOnly() => $_has(5);
  @$pb.TagNumber(10)
  void clearRelayOnly() => $_clearField(10);

  @$pb.TagNumber(11)
  $pb.PbList<$core.String> get addresses => $_getList(6);

  @$pb.TagNumber(12)
  $pb.PbMap<$core.String, $core.String> get lastDialErrors => $_getMap(7);

  @$pb.TagNumber(13)
  $core.String get reason => $_getSZ(8);
  @$pb.TagNumber(13)
  set reason($core.String value) => $_setString(8, value);
  @$pb.TagNumber(13)
  $core.bool hasReason() => $_has(8);
  @$pb.TagNumber(13)
  void clearReason() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.int get replicationPriority => $_getIZ(9);
  @$pb.TagNumber(14)
  set replicationPriority($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(14)
  $core.bool hasReplicationPriority() => $_has(9);
  @$pb.TagNumber(14)
  void clearReplicationPriority() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get replicationDeviceClass => $_getSZ(10);
  @$pb.TagNumber(15)
  set replicationDeviceClass($core.String value) => $_setString(10, value);
  @$pb.TagNumber(15)
  $core.bool hasReplicationDeviceClass() => $_has(10);
  @$pb.TagNumber(15)
  void clearReplicationDeviceClass() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.bool get routed => $_getBF(11);
  @$pb.TagNumber(16)
  set routed($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(16)
  $core.bool hasRouted() => $_has(11);
  @$pb.TagNumber(16)
  void clearRouted() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.bool get participating => $_getBF(12);
  @$pb.TagNumber(17)
  set participating($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(17)
  $core.bool hasParticipating() => $_has(12);
  @$pb.TagNumber(17)
  void clearParticipating() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.bool get logical => $_getBF(13);
  @$pb.TagNumber(18)
  set logical($core.bool value) => $_setBool(13, value);
  @$pb.TagNumber(18)
  $core.bool hasLogical() => $_has(13);
  @$pb.TagNumber(18)
  void clearLogical() => $_clearField(18);

  @$pb.TagNumber(19)
  $core.bool get physicalConnected => $_getBF(14);
  @$pb.TagNumber(19)
  set physicalConnected($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(19)
  $core.bool hasPhysicalConnected() => $_has(14);
  @$pb.TagNumber(19)
  void clearPhysicalConnected() => $_clearField(19);

  @$pb.TagNumber(20)
  $fixnum.Int64 get lastRoutedAtUnixNano => $_getI64(15);
  @$pb.TagNumber(20)
  set lastRoutedAtUnixNano($fixnum.Int64 value) => $_setInt64(15, value);
  @$pb.TagNumber(20)
  $core.bool hasLastRoutedAtUnixNano() => $_has(15);
  @$pb.TagNumber(20)
  void clearLastRoutedAtUnixNano() => $_clearField(20);
}

class RuntimeCompatibility extends $pb.GeneratedMessage {
  factory RuntimeCompatibility({
    $core.String? peerId,
    $core.String? localDigest,
    $core.String? remoteDigest,
    $core.bool? compatible,
    $core.bool? blocking,
    $core.String? reason,
  }) {
    final result = create();
    if (peerId != null) result.peerId = peerId;
    if (localDigest != null) result.localDigest = localDigest;
    if (remoteDigest != null) result.remoteDigest = remoteDigest;
    if (compatible != null) result.compatible = compatible;
    if (blocking != null) result.blocking = blocking;
    if (reason != null) result.reason = reason;
    return result;
  }

  RuntimeCompatibility._();

  factory RuntimeCompatibility.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeCompatibility.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeCompatibility',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'peerId')
    ..aOS(2, _omitFieldNames ? '' : 'localDigest')
    ..aOS(3, _omitFieldNames ? '' : 'remoteDigest')
    ..aOB(4, _omitFieldNames ? '' : 'compatible')
    ..aOB(5, _omitFieldNames ? '' : 'blocking')
    ..aOS(6, _omitFieldNames ? '' : 'reason')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeCompatibility clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeCompatibility copyWith(void Function(RuntimeCompatibility) updates) =>
      super.copyWith((message) => updates(message as RuntimeCompatibility))
          as RuntimeCompatibility;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeCompatibility create() => RuntimeCompatibility._();
  @$core.override
  RuntimeCompatibility createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeCompatibility getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeCompatibility>(create);
  static RuntimeCompatibility? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get peerId => $_getSZ(0);
  @$pb.TagNumber(1)
  set peerId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPeerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearPeerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get localDigest => $_getSZ(1);
  @$pb.TagNumber(2)
  set localDigest($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLocalDigest() => $_has(1);
  @$pb.TagNumber(2)
  void clearLocalDigest() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get remoteDigest => $_getSZ(2);
  @$pb.TagNumber(3)
  set remoteDigest($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRemoteDigest() => $_has(2);
  @$pb.TagNumber(3)
  void clearRemoteDigest() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get compatible => $_getBF(3);
  @$pb.TagNumber(4)
  set compatible($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCompatible() => $_has(3);
  @$pb.TagNumber(4)
  void clearCompatible() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get blocking => $_getBF(4);
  @$pb.TagNumber(5)
  set blocking($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBlocking() => $_has(4);
  @$pb.TagNumber(5)
  void clearBlocking() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get reason => $_getSZ(5);
  @$pb.TagNumber(6)
  set reason($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasReason() => $_has(5);
  @$pb.TagNumber(6)
  void clearReason() => $_clearField(6);
}

class CloudImage extends $pb.GeneratedMessage {
  factory CloudImage({
    $core.String? provider,
    $core.String? url,
    $core.String? digest,
    $fixnum.Int64? releaseDate,
  }) {
    final result = create();
    if (provider != null) result.provider = provider;
    if (url != null) result.url = url;
    if (digest != null) result.digest = digest;
    if (releaseDate != null) result.releaseDate = releaseDate;
    return result;
  }

  CloudImage._();

  factory CloudImage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudImage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudImage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'provider')
    ..aOS(2, _omitFieldNames ? '' : 'url')
    ..aOS(3, _omitFieldNames ? '' : 'digest')
    ..aInt64(4, _omitFieldNames ? '' : 'releaseDate')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudImage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudImage copyWith(void Function(CloudImage) updates) =>
      super.copyWith((message) => updates(message as CloudImage)) as CloudImage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudImage create() => CloudImage._();
  @$core.override
  CloudImage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudImage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudImage>(create);
  static CloudImage? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get provider => $_getSZ(0);
  @$pb.TagNumber(1)
  set provider($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProvider() => $_has(0);
  @$pb.TagNumber(1)
  void clearProvider() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get url => $_getSZ(1);
  @$pb.TagNumber(2)
  set url($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasUrl() => $_has(1);
  @$pb.TagNumber(2)
  void clearUrl() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get digest => $_getSZ(2);
  @$pb.TagNumber(3)
  set digest($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDigest() => $_has(2);
  @$pb.TagNumber(3)
  void clearDigest() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get releaseDate => $_getI64(3);
  @$pb.TagNumber(4)
  set releaseDate($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReleaseDate() => $_has(3);
  @$pb.TagNumber(4)
  void clearReleaseDate() => $_clearField(4);
}

class ProvisionerImage extends $pb.GeneratedMessage {
  factory ProvisionerImage({
    $core.String? id,
    $core.String? name,
    $core.String? location,
    $core.String? logicalName,
    $core.String? dateSuffix,
    $fixnum.Int64? updatedAtUnix,
    $core.bool? canonical,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (location != null) result.location = location;
    if (logicalName != null) result.logicalName = logicalName;
    if (dateSuffix != null) result.dateSuffix = dateSuffix;
    if (updatedAtUnix != null) result.updatedAtUnix = updatedAtUnix;
    if (canonical != null) result.canonical = canonical;
    return result;
  }

  ProvisionerImage._();

  factory ProvisionerImage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProvisionerImage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProvisionerImage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'location')
    ..aOS(4, _omitFieldNames ? '' : 'logicalName')
    ..aOS(5, _omitFieldNames ? '' : 'dateSuffix')
    ..aInt64(6, _omitFieldNames ? '' : 'updatedAtUnix')
    ..aOB(7, _omitFieldNames ? '' : 'canonical')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerImage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProvisionerImage copyWith(void Function(ProvisionerImage) updates) =>
      super.copyWith((message) => updates(message as ProvisionerImage))
          as ProvisionerImage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProvisionerImage create() => ProvisionerImage._();
  @$core.override
  ProvisionerImage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProvisionerImage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProvisionerImage>(create);
  static ProvisionerImage? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get location => $_getSZ(2);
  @$pb.TagNumber(3)
  set location($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLocation() => $_has(2);
  @$pb.TagNumber(3)
  void clearLocation() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get logicalName => $_getSZ(3);
  @$pb.TagNumber(4)
  set logicalName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLogicalName() => $_has(3);
  @$pb.TagNumber(4)
  void clearLogicalName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get dateSuffix => $_getSZ(4);
  @$pb.TagNumber(5)
  set dateSuffix($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDateSuffix() => $_has(4);
  @$pb.TagNumber(5)
  void clearDateSuffix() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get updatedAtUnix => $_getI64(5);
  @$pb.TagNumber(6)
  set updatedAtUnix($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasUpdatedAtUnix() => $_has(5);
  @$pb.TagNumber(6)
  void clearUpdatedAtUnix() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get canonical => $_getBF(6);
  @$pb.TagNumber(7)
  set canonical($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCanonical() => $_has(6);
  @$pb.TagNumber(7)
  void clearCanonical() => $_clearField(7);
}

class Release extends $pb.GeneratedMessage {
  factory Release({
    $core.Iterable<$core.MapEntry<$core.String, CloudImage>>? cloudImages,
    $core.String? version,
    $core.String? description,
    $fixnum.Int64? releaseDate,
  }) {
    final result = create();
    if (cloudImages != null) result.cloudImages.addEntries(cloudImages);
    if (version != null) result.version = version;
    if (description != null) result.description = description;
    if (releaseDate != null) result.releaseDate = releaseDate;
    return result;
  }

  Release._();

  factory Release.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Release.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Release',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..m<$core.String, CloudImage>(1, _omitFieldNames ? '' : 'cloudImages',
        entryClassName: 'Release.CloudImagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: CloudImage.create,
        valueDefaultOrMaker: CloudImage.getDefault,
        packageName: const $pb.PackageName('apic'))
    ..aOS(2, _omitFieldNames ? '' : 'version')
    ..aOS(3, _omitFieldNames ? '' : 'description')
    ..aInt64(4, _omitFieldNames ? '' : 'releaseDate')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Release clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Release copyWith(void Function(Release) updates) =>
      super.copyWith((message) => updates(message as Release)) as Release;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Release create() => Release._();
  @$core.override
  Release createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Release getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Release>(create);
  static Release? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbMap<$core.String, CloudImage> get cloudImages => $_getMap(0);

  @$pb.TagNumber(2)
  $core.String get version => $_getSZ(1);
  @$pb.TagNumber(2)
  set version($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get description => $_getSZ(2);
  @$pb.TagNumber(3)
  set description($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDescription() => $_has(2);
  @$pb.TagNumber(3)
  void clearDescription() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get releaseDate => $_getI64(3);
  @$pb.TagNumber(4)
  set releaseDate($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReleaseDate() => $_has(3);
  @$pb.TagNumber(4)
  void clearReleaseDate() => $_clearField(4);
}

class GetProtosdReleasesRequest extends $pb.GeneratedMessage {
  factory GetProtosdReleasesRequest() => create();

  GetProtosdReleasesRequest._();

  factory GetProtosdReleasesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProtosdReleasesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProtosdReleasesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProtosdReleasesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProtosdReleasesRequest copyWith(
          void Function(GetProtosdReleasesRequest) updates) =>
      super.copyWith((message) => updates(message as GetProtosdReleasesRequest))
          as GetProtosdReleasesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProtosdReleasesRequest create() => GetProtosdReleasesRequest._();
  @$core.override
  GetProtosdReleasesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProtosdReleasesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProtosdReleasesRequest>(create);
  static GetProtosdReleasesRequest? _defaultInstance;
}

class GetProtosdReleasesResponse extends $pb.GeneratedMessage {
  factory GetProtosdReleasesResponse({
    $core.Iterable<Release>? releases,
  }) {
    final result = create();
    if (releases != null) result.releases.addAll(releases);
    return result;
  }

  GetProtosdReleasesResponse._();

  factory GetProtosdReleasesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProtosdReleasesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProtosdReleasesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Release>(1, _omitFieldNames ? '' : 'releases',
        subBuilder: Release.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProtosdReleasesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProtosdReleasesResponse copyWith(
          void Function(GetProtosdReleasesResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetProtosdReleasesResponse))
          as GetProtosdReleasesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProtosdReleasesResponse create() => GetProtosdReleasesResponse._();
  @$core.override
  GetProtosdReleasesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProtosdReleasesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProtosdReleasesResponse>(create);
  static GetProtosdReleasesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Release> get releases => $_getList(0);
}

class GetProvisionerImagesRequest extends $pb.GeneratedMessage {
  factory GetProvisionerImagesRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetProvisionerImagesRequest._();

  factory GetProvisionerImagesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionerImagesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionerImagesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerImagesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerImagesRequest copyWith(
          void Function(GetProvisionerImagesRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetProvisionerImagesRequest))
          as GetProvisionerImagesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionerImagesRequest create() =>
      GetProvisionerImagesRequest._();
  @$core.override
  GetProvisionerImagesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionerImagesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionerImagesRequest>(create);
  static GetProvisionerImagesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetProvisionerImagesResponse extends $pb.GeneratedMessage {
  factory GetProvisionerImagesResponse({
    $core.Iterable<$core.MapEntry<$core.String, ProvisionerImage>>? images,
  }) {
    final result = create();
    if (images != null) result.images.addEntries(images);
    return result;
  }

  GetProvisionerImagesResponse._();

  factory GetProvisionerImagesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetProvisionerImagesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetProvisionerImagesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..m<$core.String, ProvisionerImage>(1, _omitFieldNames ? '' : 'images',
        entryClassName: 'GetProvisionerImagesResponse.ImagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: ProvisionerImage.create,
        valueDefaultOrMaker: ProvisionerImage.getDefault,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerImagesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetProvisionerImagesResponse copyWith(
          void Function(GetProvisionerImagesResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetProvisionerImagesResponse))
          as GetProvisionerImagesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetProvisionerImagesResponse create() =>
      GetProvisionerImagesResponse._();
  @$core.override
  GetProvisionerImagesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetProvisionerImagesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetProvisionerImagesResponse>(create);
  static GetProvisionerImagesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbMap<$core.String, ProvisionerImage> get images => $_getMap(0);
}

class UploadProvisionerImageRequest extends $pb.GeneratedMessage {
  factory UploadProvisionerImageRequest({
    $core.String? imagePath,
    $core.String? imageName,
    $core.String? provisionerName,
    $core.String? location,
    $core.int? timeout,
  }) {
    final result = create();
    if (imagePath != null) result.imagePath = imagePath;
    if (imageName != null) result.imageName = imageName;
    if (provisionerName != null) result.provisionerName = provisionerName;
    if (location != null) result.location = location;
    if (timeout != null) result.timeout = timeout;
    return result;
  }

  UploadProvisionerImageRequest._();

  factory UploadProvisionerImageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadProvisionerImageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadProvisionerImageRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'imagePath')
    ..aOS(2, _omitFieldNames ? '' : 'imageName')
    ..aOS(3, _omitFieldNames ? '' : 'provisionerName')
    ..aOS(4, _omitFieldNames ? '' : 'location')
    ..aI(5, _omitFieldNames ? '' : 'timeout')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadProvisionerImageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadProvisionerImageRequest copyWith(
          void Function(UploadProvisionerImageRequest) updates) =>
      super.copyWith(
              (message) => updates(message as UploadProvisionerImageRequest))
          as UploadProvisionerImageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadProvisionerImageRequest create() =>
      UploadProvisionerImageRequest._();
  @$core.override
  UploadProvisionerImageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadProvisionerImageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadProvisionerImageRequest>(create);
  static UploadProvisionerImageRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get imagePath => $_getSZ(0);
  @$pb.TagNumber(1)
  set imagePath($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasImagePath() => $_has(0);
  @$pb.TagNumber(1)
  void clearImagePath() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get imageName => $_getSZ(1);
  @$pb.TagNumber(2)
  set imageName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasImageName() => $_has(1);
  @$pb.TagNumber(2)
  void clearImageName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get provisionerName => $_getSZ(2);
  @$pb.TagNumber(3)
  set provisionerName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasProvisionerName() => $_has(2);
  @$pb.TagNumber(3)
  void clearProvisionerName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get location => $_getSZ(3);
  @$pb.TagNumber(4)
  set location($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLocation() => $_has(3);
  @$pb.TagNumber(4)
  void clearLocation() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get timeout => $_getIZ(4);
  @$pb.TagNumber(5)
  set timeout($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTimeout() => $_has(4);
  @$pb.TagNumber(5)
  void clearTimeout() => $_clearField(5);
}

class UploadProvisionerImageResponse extends $pb.GeneratedMessage {
  factory UploadProvisionerImageResponse({
    $core.String? id,
    $core.String? taskId,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (taskId != null) result.taskId = taskId;
    return result;
  }

  UploadProvisionerImageResponse._();

  factory UploadProvisionerImageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadProvisionerImageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadProvisionerImageResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'taskId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadProvisionerImageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadProvisionerImageResponse copyWith(
          void Function(UploadProvisionerImageResponse) updates) =>
      super.copyWith(
              (message) => updates(message as UploadProvisionerImageResponse))
          as UploadProvisionerImageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadProvisionerImageResponse create() =>
      UploadProvisionerImageResponse._();
  @$core.override
  UploadProvisionerImageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadProvisionerImageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadProvisionerImageResponse>(create);
  static UploadProvisionerImageResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get taskId => $_getSZ(1);
  @$pb.TagNumber(2)
  set taskId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTaskId() => $_has(1);
  @$pb.TagNumber(2)
  void clearTaskId() => $_clearField(2);
}

class RemoveProvisionerImageRequest extends $pb.GeneratedMessage {
  factory RemoveProvisionerImageRequest({
    $core.String? imageName,
    $core.String? provisionerName,
    $core.String? location,
  }) {
    final result = create();
    if (imageName != null) result.imageName = imageName;
    if (provisionerName != null) result.provisionerName = provisionerName;
    if (location != null) result.location = location;
    return result;
  }

  RemoveProvisionerImageRequest._();

  factory RemoveProvisionerImageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveProvisionerImageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveProvisionerImageRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'imageName')
    ..aOS(2, _omitFieldNames ? '' : 'provisionerName')
    ..aOS(3, _omitFieldNames ? '' : 'location')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerImageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerImageRequest copyWith(
          void Function(RemoveProvisionerImageRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RemoveProvisionerImageRequest))
          as RemoveProvisionerImageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerImageRequest create() =>
      RemoveProvisionerImageRequest._();
  @$core.override
  RemoveProvisionerImageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerImageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveProvisionerImageRequest>(create);
  static RemoveProvisionerImageRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get imageName => $_getSZ(0);
  @$pb.TagNumber(1)
  set imageName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasImageName() => $_has(0);
  @$pb.TagNumber(1)
  void clearImageName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get provisionerName => $_getSZ(1);
  @$pb.TagNumber(2)
  set provisionerName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasProvisionerName() => $_has(1);
  @$pb.TagNumber(2)
  void clearProvisionerName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get location => $_getSZ(2);
  @$pb.TagNumber(3)
  set location($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLocation() => $_has(2);
  @$pb.TagNumber(3)
  void clearLocation() => $_clearField(3);
}

class RemoveProvisionerImageResponse extends $pb.GeneratedMessage {
  factory RemoveProvisionerImageResponse() => create();

  RemoveProvisionerImageResponse._();

  factory RemoveProvisionerImageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveProvisionerImageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveProvisionerImageResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerImageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveProvisionerImageResponse copyWith(
          void Function(RemoveProvisionerImageResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RemoveProvisionerImageResponse))
          as RemoveProvisionerImageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerImageResponse create() =>
      RemoveProvisionerImageResponse._();
  @$core.override
  RemoveProvisionerImageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveProvisionerImageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveProvisionerImageResponse>(create);
  static RemoveProvisionerImageResponse? _defaultInstance;
}

class ImageContentDescriptor extends $pb.GeneratedMessage {
  factory ImageContentDescriptor({
    $core.String? mediaType,
    $core.String? digest,
    $fixnum.Int64? sizeBytes,
    $core.String? platform,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? annotations,
  }) {
    final result = create();
    if (mediaType != null) result.mediaType = mediaType;
    if (digest != null) result.digest = digest;
    if (sizeBytes != null) result.sizeBytes = sizeBytes;
    if (platform != null) result.platform = platform;
    if (annotations != null) result.annotations.addEntries(annotations);
    return result;
  }

  ImageContentDescriptor._();

  factory ImageContentDescriptor.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ImageContentDescriptor.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ImageContentDescriptor',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'mediaType')
    ..aOS(2, _omitFieldNames ? '' : 'digest')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'sizeBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(4, _omitFieldNames ? '' : 'platform')
    ..m<$core.String, $core.String>(5, _omitFieldNames ? '' : 'annotations',
        entryClassName: 'ImageContentDescriptor.AnnotationsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImageContentDescriptor clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImageContentDescriptor copyWith(
          void Function(ImageContentDescriptor) updates) =>
      super.copyWith((message) => updates(message as ImageContentDescriptor))
          as ImageContentDescriptor;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ImageContentDescriptor create() => ImageContentDescriptor._();
  @$core.override
  ImageContentDescriptor createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ImageContentDescriptor getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ImageContentDescriptor>(create);
  static ImageContentDescriptor? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get mediaType => $_getSZ(0);
  @$pb.TagNumber(1)
  set mediaType($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMediaType() => $_has(0);
  @$pb.TagNumber(1)
  void clearMediaType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get digest => $_getSZ(1);
  @$pb.TagNumber(2)
  set digest($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDigest() => $_has(1);
  @$pb.TagNumber(2)
  void clearDigest() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get sizeBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set sizeBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSizeBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearSizeBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get platform => $_getSZ(3);
  @$pb.TagNumber(4)
  set platform($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlatform() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlatform() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbMap<$core.String, $core.String> get annotations => $_getMap(4);
}

class GetInstanceImageRequest extends $pb.GeneratedMessage {
  factory GetInstanceImageRequest({
    $core.String? instance,
    $core.String? imageRef,
    $core.bool? includeContent,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (imageRef != null) result.imageRef = imageRef;
    if (includeContent != null) result.includeContent = includeContent;
    return result;
  }

  GetInstanceImageRequest._();

  factory GetInstanceImageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceImageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceImageRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..aOS(2, _omitFieldNames ? '' : 'imageRef')
    ..aOB(3, _omitFieldNames ? '' : 'includeContent')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceImageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceImageRequest copyWith(
          void Function(GetInstanceImageRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstanceImageRequest))
          as GetInstanceImageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceImageRequest create() => GetInstanceImageRequest._();
  @$core.override
  GetInstanceImageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceImageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceImageRequest>(create);
  static GetInstanceImageRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get imageRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set imageRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasImageRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearImageRef() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get includeContent => $_getBF(2);
  @$pb.TagNumber(3)
  set includeContent($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIncludeContent() => $_has(2);
  @$pb.TagNumber(3)
  void clearIncludeContent() => $_clearField(3);
}

class GetInstanceImageResponse extends $pb.GeneratedMessage {
  factory GetInstanceImageResponse({
    $core.bool? found,
    $core.String? imageRef,
    $core.String? targetDigest,
    $core.String? platform,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? labels,
    $core.bool? hasContent,
    ImageContentDescriptor? target,
    $core.Iterable<ImageContentDescriptor>? descriptors,
  }) {
    final result = create();
    if (found != null) result.found = found;
    if (imageRef != null) result.imageRef = imageRef;
    if (targetDigest != null) result.targetDigest = targetDigest;
    if (platform != null) result.platform = platform;
    if (labels != null) result.labels.addEntries(labels);
    if (hasContent != null) result.hasContent = hasContent;
    if (target != null) result.target = target;
    if (descriptors != null) result.descriptors.addAll(descriptors);
    return result;
  }

  GetInstanceImageResponse._();

  factory GetInstanceImageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstanceImageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceImageResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'found')
    ..aOS(2, _omitFieldNames ? '' : 'imageRef')
    ..aOS(3, _omitFieldNames ? '' : 'targetDigest')
    ..aOS(4, _omitFieldNames ? '' : 'platform')
    ..m<$core.String, $core.String>(5, _omitFieldNames ? '' : 'labels',
        entryClassName: 'GetInstanceImageResponse.LabelsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..aOB(6, _omitFieldNames ? '' : 'hasContent')
    ..aOM<ImageContentDescriptor>(7, _omitFieldNames ? '' : 'target',
        subBuilder: ImageContentDescriptor.create)
    ..pPM<ImageContentDescriptor>(8, _omitFieldNames ? '' : 'descriptors',
        subBuilder: ImageContentDescriptor.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceImageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceImageResponse copyWith(
          void Function(GetInstanceImageResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstanceImageResponse))
          as GetInstanceImageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceImageResponse create() => GetInstanceImageResponse._();
  @$core.override
  GetInstanceImageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstanceImageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceImageResponse>(create);
  static GetInstanceImageResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get found => $_getBF(0);
  @$pb.TagNumber(1)
  set found($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFound() => $_has(0);
  @$pb.TagNumber(1)
  void clearFound() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get imageRef => $_getSZ(1);
  @$pb.TagNumber(2)
  set imageRef($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasImageRef() => $_has(1);
  @$pb.TagNumber(2)
  void clearImageRef() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get targetDigest => $_getSZ(2);
  @$pb.TagNumber(3)
  set targetDigest($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTargetDigest() => $_has(2);
  @$pb.TagNumber(3)
  void clearTargetDigest() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get platform => $_getSZ(3);
  @$pb.TagNumber(4)
  set platform($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlatform() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlatform() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbMap<$core.String, $core.String> get labels => $_getMap(4);

  @$pb.TagNumber(6)
  $core.bool get hasContent => $_getBF(5);
  @$pb.TagNumber(6)
  set hasContent($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasHasContent() => $_has(5);
  @$pb.TagNumber(6)
  void clearHasContent() => $_clearField(6);

  @$pb.TagNumber(7)
  ImageContentDescriptor get target => $_getN(6);
  @$pb.TagNumber(7)
  set target(ImageContentDescriptor value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasTarget() => $_has(6);
  @$pb.TagNumber(7)
  void clearTarget() => $_clearField(7);
  @$pb.TagNumber(7)
  ImageContentDescriptor ensureTarget() => $_ensure(6);

  @$pb.TagNumber(8)
  $pb.PbList<ImageContentDescriptor> get descriptors => $_getList(7);
}

class UploadInstanceImageArchiveRequest extends $pb.GeneratedMessage {
  factory UploadInstanceImageArchiveRequest({
    $core.String? instance,
    $core.String? archivePath,
    $core.String? imageRef,
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
    if (archivePath != null) result.archivePath = archivePath;
    if (imageRef != null) result.imageRef = imageRef;
    return result;
  }

  UploadInstanceImageArchiveRequest._();

  factory UploadInstanceImageArchiveRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadInstanceImageArchiveRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadInstanceImageArchiveRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instance')
    ..aOS(2, _omitFieldNames ? '' : 'archivePath')
    ..aOS(3, _omitFieldNames ? '' : 'imageRef')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadInstanceImageArchiveRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadInstanceImageArchiveRequest copyWith(
          void Function(UploadInstanceImageArchiveRequest) updates) =>
      super.copyWith((message) =>
              updates(message as UploadInstanceImageArchiveRequest))
          as UploadInstanceImageArchiveRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadInstanceImageArchiveRequest create() =>
      UploadInstanceImageArchiveRequest._();
  @$core.override
  UploadInstanceImageArchiveRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadInstanceImageArchiveRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadInstanceImageArchiveRequest>(
          create);
  static UploadInstanceImageArchiveRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instance => $_getSZ(0);
  @$pb.TagNumber(1)
  set instance($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get archivePath => $_getSZ(1);
  @$pb.TagNumber(2)
  set archivePath($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasArchivePath() => $_has(1);
  @$pb.TagNumber(2)
  void clearArchivePath() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get imageRef => $_getSZ(2);
  @$pb.TagNumber(3)
  set imageRef($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasImageRef() => $_has(2);
  @$pb.TagNumber(3)
  void clearImageRef() => $_clearField(3);
}

class UploadInstanceImageArchiveResponse extends $pb.GeneratedMessage {
  factory UploadInstanceImageArchiveResponse({
    $core.String? taskId,
  }) {
    final result = create();
    if (taskId != null) result.taskId = taskId;
    return result;
  }

  UploadInstanceImageArchiveResponse._();

  factory UploadInstanceImageArchiveResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadInstanceImageArchiveResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadInstanceImageArchiveResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'taskId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadInstanceImageArchiveResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadInstanceImageArchiveResponse copyWith(
          void Function(UploadInstanceImageArchiveResponse) updates) =>
      super.copyWith((message) =>
              updates(message as UploadInstanceImageArchiveResponse))
          as UploadInstanceImageArchiveResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadInstanceImageArchiveResponse create() =>
      UploadInstanceImageArchiveResponse._();
  @$core.override
  UploadInstanceImageArchiveResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadInstanceImageArchiveResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadInstanceImageArchiveResponse>(
          create);
  static UploadInstanceImageArchiveResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get taskId => $_getSZ(0);
  @$pb.TagNumber(1)
  set taskId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTaskId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTaskId() => $_clearField(1);
}

class CoreEndpoint extends $pb.GeneratedMessage {
  factory CoreEndpoint({
    $core.String? kind,
    $core.String? address,
    $core.bool? active,
    $core.String? message,
  }) {
    final result = create();
    if (kind != null) result.kind = kind;
    if (address != null) result.address = address;
    if (active != null) result.active = active;
    if (message != null) result.message = message;
    return result;
  }

  CoreEndpoint._();

  factory CoreEndpoint.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CoreEndpoint.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CoreEndpoint',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'kind')
    ..aOS(2, _omitFieldNames ? '' : 'address')
    ..aOB(3, _omitFieldNames ? '' : 'active')
    ..aOS(4, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreEndpoint clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreEndpoint copyWith(void Function(CoreEndpoint) updates) =>
      super.copyWith((message) => updates(message as CoreEndpoint))
          as CoreEndpoint;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CoreEndpoint create() => CoreEndpoint._();
  @$core.override
  CoreEndpoint createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CoreEndpoint getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CoreEndpoint>(create);
  static CoreEndpoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get kind => $_getSZ(0);
  @$pb.TagNumber(1)
  set kind($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKind() => $_has(0);
  @$pb.TagNumber(1)
  void clearKind() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get address => $_getSZ(1);
  @$pb.TagNumber(2)
  set address($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAddress() => $_has(1);
  @$pb.TagNumber(2)
  void clearAddress() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get active => $_getBF(2);
  @$pb.TagNumber(3)
  set active($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActive() => $_has(2);
  @$pb.TagNumber(3)
  void clearActive() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get message => $_getSZ(3);
  @$pb.TagNumber(4)
  set message($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessage() => $_clearField(4);
}

class HostAgentConnectionStatus extends $pb.GeneratedMessage {
  factory HostAgentConnectionStatus({
    $core.bool? connected,
    $core.String? socket,
    $core.String? message,
  }) {
    final result = create();
    if (connected != null) result.connected = connected;
    if (socket != null) result.socket = socket;
    if (message != null) result.message = message;
    return result;
  }

  HostAgentConnectionStatus._();

  factory HostAgentConnectionStatus.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HostAgentConnectionStatus.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HostAgentConnectionStatus',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'connected')
    ..aOS(2, _omitFieldNames ? '' : 'socket')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HostAgentConnectionStatus clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HostAgentConnectionStatus copyWith(
          void Function(HostAgentConnectionStatus) updates) =>
      super.copyWith((message) => updates(message as HostAgentConnectionStatus))
          as HostAgentConnectionStatus;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HostAgentConnectionStatus create() => HostAgentConnectionStatus._();
  @$core.override
  HostAgentConnectionStatus createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HostAgentConnectionStatus getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HostAgentConnectionStatus>(create);
  static HostAgentConnectionStatus? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get connected => $_getBF(0);
  @$pb.TagNumber(1)
  set connected($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasConnected() => $_has(0);
  @$pb.TagNumber(1)
  void clearConnected() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get socket => $_getSZ(1);
  @$pb.TagNumber(2)
  set socket($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSocket() => $_has(1);
  @$pb.TagNumber(2)
  void clearSocket() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

class SystemStatus extends $pb.GeneratedMessage {
  factory SystemStatus({
    $core.String? coreStatus,
    $core.String? workDir,
    $core.String? capabilities,
    $core.int? p2pPort,
    $core.Iterable<CoreEndpoint>? endpoints,
    HostAgentConnectionStatus? hostAgent,
    $core.bool? networkEnabled,
    $core.bool? hostAgentSupported,
    NetworkRuntimeStatus? network,
  }) {
    final result = create();
    if (coreStatus != null) result.coreStatus = coreStatus;
    if (workDir != null) result.workDir = workDir;
    if (capabilities != null) result.capabilities = capabilities;
    if (p2pPort != null) result.p2pPort = p2pPort;
    if (endpoints != null) result.endpoints.addAll(endpoints);
    if (hostAgent != null) result.hostAgent = hostAgent;
    if (networkEnabled != null) result.networkEnabled = networkEnabled;
    if (hostAgentSupported != null)
      result.hostAgentSupported = hostAgentSupported;
    if (network != null) result.network = network;
    return result;
  }

  SystemStatus._();

  factory SystemStatus.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SystemStatus.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SystemStatus',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'coreStatus')
    ..aOS(2, _omitFieldNames ? '' : 'workDir')
    ..aOS(3, _omitFieldNames ? '' : 'capabilities')
    ..aI(4, _omitFieldNames ? '' : 'p2pPort')
    ..pPM<CoreEndpoint>(5, _omitFieldNames ? '' : 'endpoints',
        subBuilder: CoreEndpoint.create)
    ..aOM<HostAgentConnectionStatus>(6, _omitFieldNames ? '' : 'hostAgent',
        subBuilder: HostAgentConnectionStatus.create)
    ..aOB(7, _omitFieldNames ? '' : 'networkEnabled')
    ..aOB(8, _omitFieldNames ? '' : 'hostAgentSupported')
    ..aOM<NetworkRuntimeStatus>(9, _omitFieldNames ? '' : 'network',
        subBuilder: NetworkRuntimeStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SystemStatus clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SystemStatus copyWith(void Function(SystemStatus) updates) =>
      super.copyWith((message) => updates(message as SystemStatus))
          as SystemStatus;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SystemStatus create() => SystemStatus._();
  @$core.override
  SystemStatus createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SystemStatus getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SystemStatus>(create);
  static SystemStatus? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get coreStatus => $_getSZ(0);
  @$pb.TagNumber(1)
  set coreStatus($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCoreStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearCoreStatus() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get workDir => $_getSZ(1);
  @$pb.TagNumber(2)
  set workDir($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWorkDir() => $_has(1);
  @$pb.TagNumber(2)
  void clearWorkDir() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get capabilities => $_getSZ(2);
  @$pb.TagNumber(3)
  set capabilities($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCapabilities() => $_has(2);
  @$pb.TagNumber(3)
  void clearCapabilities() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get p2pPort => $_getIZ(3);
  @$pb.TagNumber(4)
  set p2pPort($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasP2pPort() => $_has(3);
  @$pb.TagNumber(4)
  void clearP2pPort() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<CoreEndpoint> get endpoints => $_getList(4);

  @$pb.TagNumber(6)
  HostAgentConnectionStatus get hostAgent => $_getN(5);
  @$pb.TagNumber(6)
  set hostAgent(HostAgentConnectionStatus value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasHostAgent() => $_has(5);
  @$pb.TagNumber(6)
  void clearHostAgent() => $_clearField(6);
  @$pb.TagNumber(6)
  HostAgentConnectionStatus ensureHostAgent() => $_ensure(5);

  @$pb.TagNumber(7)
  $core.bool get networkEnabled => $_getBF(6);
  @$pb.TagNumber(7)
  set networkEnabled($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasNetworkEnabled() => $_has(6);
  @$pb.TagNumber(7)
  void clearNetworkEnabled() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get hostAgentSupported => $_getBF(7);
  @$pb.TagNumber(8)
  set hostAgentSupported($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasHostAgentSupported() => $_has(7);
  @$pb.TagNumber(8)
  void clearHostAgentSupported() => $_clearField(8);

  @$pb.TagNumber(9)
  NetworkRuntimeStatus get network => $_getN(8);
  @$pb.TagNumber(9)
  set network(NetworkRuntimeStatus value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasNetwork() => $_has(8);
  @$pb.TagNumber(9)
  void clearNetwork() => $_clearField(9);
  @$pb.TagNumber(9)
  NetworkRuntimeStatus ensureNetwork() => $_ensure(8);
}

class GetSystemStatusRequest extends $pb.GeneratedMessage {
  factory GetSystemStatusRequest() => create();

  GetSystemStatusRequest._();

  factory GetSystemStatusRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSystemStatusRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSystemStatusRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSystemStatusRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSystemStatusRequest copyWith(
          void Function(GetSystemStatusRequest) updates) =>
      super.copyWith((message) => updates(message as GetSystemStatusRequest))
          as GetSystemStatusRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSystemStatusRequest create() => GetSystemStatusRequest._();
  @$core.override
  GetSystemStatusRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSystemStatusRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSystemStatusRequest>(create);
  static GetSystemStatusRequest? _defaultInstance;
}

class GetSystemStatusResponse extends $pb.GeneratedMessage {
  factory GetSystemStatusResponse({
    SystemStatus? status,
  }) {
    final result = create();
    if (status != null) result.status = status;
    return result;
  }

  GetSystemStatusResponse._();

  factory GetSystemStatusResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSystemStatusResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSystemStatusResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<SystemStatus>(1, _omitFieldNames ? '' : 'status',
        subBuilder: SystemStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSystemStatusResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSystemStatusResponse copyWith(
          void Function(GetSystemStatusResponse) updates) =>
      super.copyWith((message) => updates(message as GetSystemStatusResponse))
          as GetSystemStatusResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSystemStatusResponse create() => GetSystemStatusResponse._();
  @$core.override
  GetSystemStatusResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSystemStatusResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSystemStatusResponse>(create);
  static GetSystemStatusResponse? _defaultInstance;

  @$pb.TagNumber(1)
  SystemStatus get status => $_getN(0);
  @$pb.TagNumber(1)
  set status(SystemStatus value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearStatus() => $_clearField(1);
  @$pb.TagNumber(1)
  SystemStatus ensureStatus() => $_ensure(0);
}

class StartHostAgentRequest extends $pb.GeneratedMessage {
  factory StartHostAgentRequest() => create();

  StartHostAgentRequest._();

  factory StartHostAgentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartHostAgentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartHostAgentRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartHostAgentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartHostAgentRequest copyWith(
          void Function(StartHostAgentRequest) updates) =>
      super.copyWith((message) => updates(message as StartHostAgentRequest))
          as StartHostAgentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartHostAgentRequest create() => StartHostAgentRequest._();
  @$core.override
  StartHostAgentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartHostAgentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartHostAgentRequest>(create);
  static StartHostAgentRequest? _defaultInstance;
}

class StartHostAgentResponse extends $pb.GeneratedMessage {
  factory StartHostAgentResponse({
    HostAgentConnectionStatus? status,
  }) {
    final result = create();
    if (status != null) result.status = status;
    return result;
  }

  StartHostAgentResponse._();

  factory StartHostAgentResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartHostAgentResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartHostAgentResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<HostAgentConnectionStatus>(1, _omitFieldNames ? '' : 'status',
        subBuilder: HostAgentConnectionStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartHostAgentResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartHostAgentResponse copyWith(
          void Function(StartHostAgentResponse) updates) =>
      super.copyWith((message) => updates(message as StartHostAgentResponse))
          as StartHostAgentResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartHostAgentResponse create() => StartHostAgentResponse._();
  @$core.override
  StartHostAgentResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartHostAgentResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartHostAgentResponse>(create);
  static StartHostAgentResponse? _defaultInstance;

  @$pb.TagNumber(1)
  HostAgentConnectionStatus get status => $_getN(0);
  @$pb.TagNumber(1)
  set status(HostAgentConnectionStatus value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearStatus() => $_clearField(1);
  @$pb.TagNumber(1)
  HostAgentConnectionStatus ensureStatus() => $_ensure(0);
}

class StopHostAgentRequest extends $pb.GeneratedMessage {
  factory StopHostAgentRequest() => create();

  StopHostAgentRequest._();

  factory StopHostAgentRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopHostAgentRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopHostAgentRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopHostAgentRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopHostAgentRequest copyWith(void Function(StopHostAgentRequest) updates) =>
      super.copyWith((message) => updates(message as StopHostAgentRequest))
          as StopHostAgentRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopHostAgentRequest create() => StopHostAgentRequest._();
  @$core.override
  StopHostAgentRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopHostAgentRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopHostAgentRequest>(create);
  static StopHostAgentRequest? _defaultInstance;
}

class StopHostAgentResponse extends $pb.GeneratedMessage {
  factory StopHostAgentResponse({
    HostAgentConnectionStatus? status,
  }) {
    final result = create();
    if (status != null) result.status = status;
    return result;
  }

  StopHostAgentResponse._();

  factory StopHostAgentResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopHostAgentResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopHostAgentResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<HostAgentConnectionStatus>(1, _omitFieldNames ? '' : 'status',
        subBuilder: HostAgentConnectionStatus.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopHostAgentResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopHostAgentResponse copyWith(
          void Function(StopHostAgentResponse) updates) =>
      super.copyWith((message) => updates(message as StopHostAgentResponse))
          as StopHostAgentResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopHostAgentResponse create() => StopHostAgentResponse._();
  @$core.override
  StopHostAgentResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopHostAgentResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopHostAgentResponse>(create);
  static StopHostAgentResponse? _defaultInstance;

  @$pb.TagNumber(1)
  HostAgentConnectionStatus get status => $_getN(0);
  @$pb.TagNumber(1)
  set status(HostAgentConnectionStatus value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasStatus() => $_has(0);
  @$pb.TagNumber(1)
  void clearStatus() => $_clearField(1);
  @$pb.TagNumber(1)
  HostAgentConnectionStatus ensureStatus() => $_ensure(0);
}

class Commit extends $pb.GeneratedMessage {
  factory Commit({
    $core.String? hash,
    $core.String? committer,
    $core.String? message,
    $core.Iterable<$core.String>? states,
    $fixnum.Int64? dateUnix,
    $core.Iterable<$core.String>? parentHashes,
    $core.Iterable<$core.String>? refs,
  }) {
    final result = create();
    if (hash != null) result.hash = hash;
    if (committer != null) result.committer = committer;
    if (message != null) result.message = message;
    if (states != null) result.states.addAll(states);
    if (dateUnix != null) result.dateUnix = dateUnix;
    if (parentHashes != null) result.parentHashes.addAll(parentHashes);
    if (refs != null) result.refs.addAll(refs);
    return result;
  }

  Commit._();

  factory Commit.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Commit.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Commit',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'hash')
    ..aOS(2, _omitFieldNames ? '' : 'committer')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..pPS(4, _omitFieldNames ? '' : 'states')
    ..aInt64(5, _omitFieldNames ? '' : 'dateUnix')
    ..pPS(6, _omitFieldNames ? '' : 'parentHashes')
    ..pPS(7, _omitFieldNames ? '' : 'refs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Commit clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Commit copyWith(void Function(Commit) updates) =>
      super.copyWith((message) => updates(message as Commit)) as Commit;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Commit create() => Commit._();
  @$core.override
  Commit createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Commit getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Commit>(create);
  static Commit? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get hash => $_getSZ(0);
  @$pb.TagNumber(1)
  set hash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get committer => $_getSZ(1);
  @$pb.TagNumber(2)
  set committer($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCommitter() => $_has(1);
  @$pb.TagNumber(2)
  void clearCommitter() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get states => $_getList(3);

  @$pb.TagNumber(5)
  $fixnum.Int64 get dateUnix => $_getI64(4);
  @$pb.TagNumber(5)
  set dateUnix($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDateUnix() => $_has(4);
  @$pb.TagNumber(5)
  void clearDateUnix() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<$core.String> get parentHashes => $_getList(5);

  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get refs => $_getList(6);
}

class CommitGraphRelation extends $pb.GeneratedMessage {
  factory CommitGraphRelation({
    $core.String? parentHash,
    $core.int? parentRow,
    $core.int? fromLane,
    $core.int? toLane,
    $core.bool? visible,
  }) {
    final result = create();
    if (parentHash != null) result.parentHash = parentHash;
    if (parentRow != null) result.parentRow = parentRow;
    if (fromLane != null) result.fromLane = fromLane;
    if (toLane != null) result.toLane = toLane;
    if (visible != null) result.visible = visible;
    return result;
  }

  CommitGraphRelation._();

  factory CommitGraphRelation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitGraphRelation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitGraphRelation',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'parentHash')
    ..aI(2, _omitFieldNames ? '' : 'parentRow')
    ..aI(3, _omitFieldNames ? '' : 'fromLane')
    ..aI(4, _omitFieldNames ? '' : 'toLane')
    ..aOB(5, _omitFieldNames ? '' : 'visible')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraphRelation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraphRelation copyWith(void Function(CommitGraphRelation) updates) =>
      super.copyWith((message) => updates(message as CommitGraphRelation))
          as CommitGraphRelation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitGraphRelation create() => CommitGraphRelation._();
  @$core.override
  CommitGraphRelation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitGraphRelation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitGraphRelation>(create);
  static CommitGraphRelation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get parentHash => $_getSZ(0);
  @$pb.TagNumber(1)
  set parentHash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasParentHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearParentHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get parentRow => $_getIZ(1);
  @$pb.TagNumber(2)
  set parentRow($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasParentRow() => $_has(1);
  @$pb.TagNumber(2)
  void clearParentRow() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get fromLane => $_getIZ(2);
  @$pb.TagNumber(3)
  set fromLane($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFromLane() => $_has(2);
  @$pb.TagNumber(3)
  void clearFromLane() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get toLane => $_getIZ(3);
  @$pb.TagNumber(4)
  set toLane($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasToLane() => $_has(3);
  @$pb.TagNumber(4)
  void clearToLane() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get visible => $_getBF(4);
  @$pb.TagNumber(5)
  set visible($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasVisible() => $_has(4);
  @$pb.TagNumber(5)
  void clearVisible() => $_clearField(5);
}

class CommitGraphItem extends $pb.GeneratedMessage {
  factory CommitGraphItem({
    Commit? commit,
    $core.int? row,
    $core.int? lane,
    $core.Iterable<$core.int>? activeLanes,
    $core.Iterable<CommitGraphRelation>? relations,
  }) {
    final result = create();
    if (commit != null) result.commit = commit;
    if (row != null) result.row = row;
    if (lane != null) result.lane = lane;
    if (activeLanes != null) result.activeLanes.addAll(activeLanes);
    if (relations != null) result.relations.addAll(relations);
    return result;
  }

  CommitGraphItem._();

  factory CommitGraphItem.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitGraphItem.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitGraphItem',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<Commit>(1, _omitFieldNames ? '' : 'commit', subBuilder: Commit.create)
    ..aI(2, _omitFieldNames ? '' : 'row')
    ..aI(3, _omitFieldNames ? '' : 'lane')
    ..p<$core.int>(4, _omitFieldNames ? '' : 'activeLanes', $pb.PbFieldType.K3)
    ..pPM<CommitGraphRelation>(5, _omitFieldNames ? '' : 'relations',
        subBuilder: CommitGraphRelation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraphItem clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraphItem copyWith(void Function(CommitGraphItem) updates) =>
      super.copyWith((message) => updates(message as CommitGraphItem))
          as CommitGraphItem;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitGraphItem create() => CommitGraphItem._();
  @$core.override
  CommitGraphItem createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitGraphItem getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitGraphItem>(create);
  static CommitGraphItem? _defaultInstance;

  @$pb.TagNumber(1)
  Commit get commit => $_getN(0);
  @$pb.TagNumber(1)
  set commit(Commit value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCommit() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommit() => $_clearField(1);
  @$pb.TagNumber(1)
  Commit ensureCommit() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.int get row => $_getIZ(1);
  @$pb.TagNumber(2)
  set row($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRow() => $_has(1);
  @$pb.TagNumber(2)
  void clearRow() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get lane => $_getIZ(2);
  @$pb.TagNumber(3)
  set lane($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLane() => $_has(2);
  @$pb.TagNumber(3)
  void clearLane() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.int> get activeLanes => $_getList(3);

  @$pb.TagNumber(5)
  $pb.PbList<CommitGraphRelation> get relations => $_getList(4);
}

class CommitGraph extends $pb.GeneratedMessage {
  factory CommitGraph({
    $core.Iterable<CommitGraphItem>? items,
    $core.int? laneCount,
  }) {
    final result = create();
    if (items != null) result.items.addAll(items);
    if (laneCount != null) result.laneCount = laneCount;
    return result;
  }

  CommitGraph._();

  factory CommitGraph.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitGraph.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitGraph',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<CommitGraphItem>(1, _omitFieldNames ? '' : 'items',
        subBuilder: CommitGraphItem.create)
    ..aI(2, _omitFieldNames ? '' : 'laneCount')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraph clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitGraph copyWith(void Function(CommitGraph) updates) =>
      super.copyWith((message) => updates(message as CommitGraph))
          as CommitGraph;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitGraph create() => CommitGraph._();
  @$core.override
  CommitGraph createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitGraph getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitGraph>(create);
  static CommitGraph? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<CommitGraphItem> get items => $_getList(0);

  @$pb.TagNumber(2)
  $core.int get laneCount => $_getIZ(1);
  @$pb.TagNumber(2)
  set laneCount($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLaneCount() => $_has(1);
  @$pb.TagNumber(2)
  void clearLaneCount() => $_clearField(2);
}

class GetLocalCommitsRequest extends $pb.GeneratedMessage {
  factory GetLocalCommitsRequest() => create();

  GetLocalCommitsRequest._();

  factory GetLocalCommitsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLocalCommitsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLocalCommitsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalCommitsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalCommitsRequest copyWith(
          void Function(GetLocalCommitsRequest) updates) =>
      super.copyWith((message) => updates(message as GetLocalCommitsRequest))
          as GetLocalCommitsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLocalCommitsRequest create() => GetLocalCommitsRequest._();
  @$core.override
  GetLocalCommitsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLocalCommitsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLocalCommitsRequest>(create);
  static GetLocalCommitsRequest? _defaultInstance;
}

class GetLocalCommitsResponse extends $pb.GeneratedMessage {
  factory GetLocalCommitsResponse({
    $core.Iterable<Commit>? commits,
    CommitGraph? graph,
  }) {
    final result = create();
    if (commits != null) result.commits.addAll(commits);
    if (graph != null) result.graph = graph;
    return result;
  }

  GetLocalCommitsResponse._();

  factory GetLocalCommitsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetLocalCommitsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetLocalCommitsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Commit>(1, _omitFieldNames ? '' : 'commits',
        subBuilder: Commit.create)
    ..aOM<CommitGraph>(2, _omitFieldNames ? '' : 'graph',
        subBuilder: CommitGraph.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalCommitsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetLocalCommitsResponse copyWith(
          void Function(GetLocalCommitsResponse) updates) =>
      super.copyWith((message) => updates(message as GetLocalCommitsResponse))
          as GetLocalCommitsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetLocalCommitsResponse create() => GetLocalCommitsResponse._();
  @$core.override
  GetLocalCommitsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetLocalCommitsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetLocalCommitsResponse>(create);
  static GetLocalCommitsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Commit> get commits => $_getList(0);

  @$pb.TagNumber(2)
  CommitGraph get graph => $_getN(1);
  @$pb.TagNumber(2)
  set graph(CommitGraph value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasGraph() => $_has(1);
  @$pb.TagNumber(2)
  void clearGraph() => $_clearField(2);
  @$pb.TagNumber(2)
  CommitGraph ensureGraph() => $_ensure(1);
}

class GetRemoteCommitsRequest extends $pb.GeneratedMessage {
  factory GetRemoteCommitsRequest({
    $core.String? remote,
  }) {
    final result = create();
    if (remote != null) result.remote = remote;
    return result;
  }

  GetRemoteCommitsRequest._();

  factory GetRemoteCommitsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetRemoteCommitsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetRemoteCommitsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'remote')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRemoteCommitsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRemoteCommitsRequest copyWith(
          void Function(GetRemoteCommitsRequest) updates) =>
      super.copyWith((message) => updates(message as GetRemoteCommitsRequest))
          as GetRemoteCommitsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetRemoteCommitsRequest create() => GetRemoteCommitsRequest._();
  @$core.override
  GetRemoteCommitsRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetRemoteCommitsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetRemoteCommitsRequest>(create);
  static GetRemoteCommitsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get remote => $_getSZ(0);
  @$pb.TagNumber(1)
  set remote($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRemote() => $_has(0);
  @$pb.TagNumber(1)
  void clearRemote() => $_clearField(1);
}

class GetRemoteCommitsResponse extends $pb.GeneratedMessage {
  factory GetRemoteCommitsResponse({
    $core.Iterable<Commit>? commits,
    CommitGraph? graph,
  }) {
    final result = create();
    if (commits != null) result.commits.addAll(commits);
    if (graph != null) result.graph = graph;
    return result;
  }

  GetRemoteCommitsResponse._();

  factory GetRemoteCommitsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetRemoteCommitsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetRemoteCommitsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Commit>(1, _omitFieldNames ? '' : 'commits',
        subBuilder: Commit.create)
    ..aOM<CommitGraph>(2, _omitFieldNames ? '' : 'graph',
        subBuilder: CommitGraph.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRemoteCommitsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetRemoteCommitsResponse copyWith(
          void Function(GetRemoteCommitsResponse) updates) =>
      super.copyWith((message) => updates(message as GetRemoteCommitsResponse))
          as GetRemoteCommitsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetRemoteCommitsResponse create() => GetRemoteCommitsResponse._();
  @$core.override
  GetRemoteCommitsResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetRemoteCommitsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetRemoteCommitsResponse>(create);
  static GetRemoteCommitsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Commit> get commits => $_getList(0);

  @$pb.TagNumber(2)
  CommitGraph get graph => $_getN(1);
  @$pb.TagNumber(2)
  set graph(CommitGraph value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasGraph() => $_has(1);
  @$pb.TagNumber(2)
  void clearGraph() => $_clearField(2);
  @$pb.TagNumber(2)
  CommitGraph ensureGraph() => $_ensure(1);
}

class CommitDiffValue extends $pb.GeneratedMessage {
  factory CommitDiffValue({
    $core.String? value,
    $core.bool? isNull,
  }) {
    final result = create();
    if (value != null) result.value = value;
    if (isNull != null) result.isNull = isNull;
    return result;
  }

  CommitDiffValue._();

  factory CommitDiffValue.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiffValue.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiffValue',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'value')
    ..aOB(2, _omitFieldNames ? '' : 'isNull')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffValue clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffValue copyWith(void Function(CommitDiffValue) updates) =>
      super.copyWith((message) => updates(message as CommitDiffValue))
          as CommitDiffValue;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiffValue create() => CommitDiffValue._();
  @$core.override
  CommitDiffValue createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiffValue getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiffValue>(create);
  static CommitDiffValue? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get value => $_getSZ(0);
  @$pb.TagNumber(1)
  set value($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get isNull => $_getBF(1);
  @$pb.TagNumber(2)
  set isNull($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIsNull() => $_has(1);
  @$pb.TagNumber(2)
  void clearIsNull() => $_clearField(2);
}

class CommitDiffField extends $pb.GeneratedMessage {
  factory CommitDiffField({
    $core.String? name,
    CommitDiffValue? before,
    CommitDiffValue? after,
    $core.String? beforeCue,
    $core.String? afterCue,
    $core.bool? changed,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (before != null) result.before = before;
    if (after != null) result.after = after;
    if (beforeCue != null) result.beforeCue = beforeCue;
    if (afterCue != null) result.afterCue = afterCue;
    if (changed != null) result.changed = changed;
    return result;
  }

  CommitDiffField._();

  factory CommitDiffField.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiffField.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiffField',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOM<CommitDiffValue>(2, _omitFieldNames ? '' : 'before',
        subBuilder: CommitDiffValue.create)
    ..aOM<CommitDiffValue>(3, _omitFieldNames ? '' : 'after',
        subBuilder: CommitDiffValue.create)
    ..aOS(4, _omitFieldNames ? '' : 'beforeCue')
    ..aOS(5, _omitFieldNames ? '' : 'afterCue')
    ..aOB(6, _omitFieldNames ? '' : 'changed')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffField clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffField copyWith(void Function(CommitDiffField) updates) =>
      super.copyWith((message) => updates(message as CommitDiffField))
          as CommitDiffField;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiffField create() => CommitDiffField._();
  @$core.override
  CommitDiffField createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiffField getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiffField>(create);
  static CommitDiffField? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  CommitDiffValue get before => $_getN(1);
  @$pb.TagNumber(2)
  set before(CommitDiffValue value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasBefore() => $_has(1);
  @$pb.TagNumber(2)
  void clearBefore() => $_clearField(2);
  @$pb.TagNumber(2)
  CommitDiffValue ensureBefore() => $_ensure(1);

  @$pb.TagNumber(3)
  CommitDiffValue get after => $_getN(2);
  @$pb.TagNumber(3)
  set after(CommitDiffValue value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasAfter() => $_has(2);
  @$pb.TagNumber(3)
  void clearAfter() => $_clearField(3);
  @$pb.TagNumber(3)
  CommitDiffValue ensureAfter() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get beforeCue => $_getSZ(3);
  @$pb.TagNumber(4)
  set beforeCue($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBeforeCue() => $_has(3);
  @$pb.TagNumber(4)
  void clearBeforeCue() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get afterCue => $_getSZ(4);
  @$pb.TagNumber(5)
  set afterCue($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAfterCue() => $_has(4);
  @$pb.TagNumber(5)
  void clearAfterCue() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get changed => $_getBF(5);
  @$pb.TagNumber(6)
  set changed($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasChanged() => $_has(5);
  @$pb.TagNumber(6)
  void clearChanged() => $_clearField(6);
}

class CommitDiffRow extends $pb.GeneratedMessage {
  factory CommitDiffRow({
    $core.String? changeType,
    $core.String? key,
    $core.Iterable<CommitDiffField>? fields,
    $core.String? beforeCue,
    $core.String? afterCue,
    $core.String? cue,
  }) {
    final result = create();
    if (changeType != null) result.changeType = changeType;
    if (key != null) result.key = key;
    if (fields != null) result.fields.addAll(fields);
    if (beforeCue != null) result.beforeCue = beforeCue;
    if (afterCue != null) result.afterCue = afterCue;
    if (cue != null) result.cue = cue;
    return result;
  }

  CommitDiffRow._();

  factory CommitDiffRow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiffRow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiffRow',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'changeType')
    ..aOS(2, _omitFieldNames ? '' : 'key')
    ..pPM<CommitDiffField>(3, _omitFieldNames ? '' : 'fields',
        subBuilder: CommitDiffField.create)
    ..aOS(4, _omitFieldNames ? '' : 'beforeCue')
    ..aOS(5, _omitFieldNames ? '' : 'afterCue')
    ..aOS(6, _omitFieldNames ? '' : 'cue')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffRow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffRow copyWith(void Function(CommitDiffRow) updates) =>
      super.copyWith((message) => updates(message as CommitDiffRow))
          as CommitDiffRow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiffRow create() => CommitDiffRow._();
  @$core.override
  CommitDiffRow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiffRow getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiffRow>(create);
  static CommitDiffRow? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get changeType => $_getSZ(0);
  @$pb.TagNumber(1)
  set changeType($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChangeType() => $_has(0);
  @$pb.TagNumber(1)
  void clearChangeType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get key => $_getSZ(1);
  @$pb.TagNumber(2)
  set key($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearKey() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<CommitDiffField> get fields => $_getList(2);

  @$pb.TagNumber(4)
  $core.String get beforeCue => $_getSZ(3);
  @$pb.TagNumber(4)
  set beforeCue($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBeforeCue() => $_has(3);
  @$pb.TagNumber(4)
  void clearBeforeCue() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get afterCue => $_getSZ(4);
  @$pb.TagNumber(5)
  set afterCue($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAfterCue() => $_has(4);
  @$pb.TagNumber(5)
  void clearAfterCue() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get cue => $_getSZ(5);
  @$pb.TagNumber(6)
  set cue($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasCue() => $_has(5);
  @$pb.TagNumber(6)
  void clearCue() => $_clearField(6);
}

class CommitDiffTable extends $pb.GeneratedMessage {
  factory CommitDiffTable({
    $core.String? name,
    $core.Iterable<CommitDiffRow>? rows,
    $core.String? cue,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (rows != null) result.rows.addAll(rows);
    if (cue != null) result.cue = cue;
    return result;
  }

  CommitDiffTable._();

  factory CommitDiffTable.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiffTable.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiffTable',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..pPM<CommitDiffRow>(2, _omitFieldNames ? '' : 'rows',
        subBuilder: CommitDiffRow.create)
    ..aOS(3, _omitFieldNames ? '' : 'cue')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffTable clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffTable copyWith(void Function(CommitDiffTable) updates) =>
      super.copyWith((message) => updates(message as CommitDiffTable))
          as CommitDiffTable;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiffTable create() => CommitDiffTable._();
  @$core.override
  CommitDiffTable createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiffTable getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiffTable>(create);
  static CommitDiffTable? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<CommitDiffRow> get rows => $_getList(1);

  @$pb.TagNumber(3)
  $core.String get cue => $_getSZ(2);
  @$pb.TagNumber(3)
  set cue($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCue() => $_has(2);
  @$pb.TagNumber(3)
  void clearCue() => $_clearField(3);
}

class CommitDiffTaskContext extends $pb.GeneratedMessage {
  factory CommitDiffTaskContext({
    $core.String? id,
    $core.String? stream,
    $core.String? subjectType,
    $core.String? subjectId,
    $core.String? ownerPeerId,
    $core.String? status,
    $core.String? title,
    $core.String? message,
    $core.int? progress,
    $core.Iterable<$core.String>? changeSources,
    $core.int? eventCount,
    $core.String? summary,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (stream != null) result.stream = stream;
    if (subjectType != null) result.subjectType = subjectType;
    if (subjectId != null) result.subjectId = subjectId;
    if (ownerPeerId != null) result.ownerPeerId = ownerPeerId;
    if (status != null) result.status = status;
    if (title != null) result.title = title;
    if (message != null) result.message = message;
    if (progress != null) result.progress = progress;
    if (changeSources != null) result.changeSources.addAll(changeSources);
    if (eventCount != null) result.eventCount = eventCount;
    if (summary != null) result.summary = summary;
    return result;
  }

  CommitDiffTaskContext._();

  factory CommitDiffTaskContext.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiffTaskContext.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiffTaskContext',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'stream')
    ..aOS(3, _omitFieldNames ? '' : 'subjectType')
    ..aOS(4, _omitFieldNames ? '' : 'subjectId')
    ..aOS(5, _omitFieldNames ? '' : 'ownerPeerId')
    ..aOS(6, _omitFieldNames ? '' : 'status')
    ..aOS(7, _omitFieldNames ? '' : 'title')
    ..aOS(8, _omitFieldNames ? '' : 'message')
    ..aI(9, _omitFieldNames ? '' : 'progress')
    ..pPS(10, _omitFieldNames ? '' : 'changeSources')
    ..aI(11, _omitFieldNames ? '' : 'eventCount')
    ..aOS(12, _omitFieldNames ? '' : 'summary')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffTaskContext clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiffTaskContext copyWith(
          void Function(CommitDiffTaskContext) updates) =>
      super.copyWith((message) => updates(message as CommitDiffTaskContext))
          as CommitDiffTaskContext;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiffTaskContext create() => CommitDiffTaskContext._();
  @$core.override
  CommitDiffTaskContext createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiffTaskContext getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiffTaskContext>(create);
  static CommitDiffTaskContext? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get stream => $_getSZ(1);
  @$pb.TagNumber(2)
  set stream($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStream() => $_has(1);
  @$pb.TagNumber(2)
  void clearStream() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get subjectType => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectType() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get subjectId => $_getSZ(3);
  @$pb.TagNumber(4)
  set subjectId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSubjectId() => $_has(3);
  @$pb.TagNumber(4)
  void clearSubjectId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get ownerPeerId => $_getSZ(4);
  @$pb.TagNumber(5)
  set ownerPeerId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOwnerPeerId() => $_has(4);
  @$pb.TagNumber(5)
  void clearOwnerPeerId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get status => $_getSZ(5);
  @$pb.TagNumber(6)
  set status($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get title => $_getSZ(6);
  @$pb.TagNumber(7)
  set title($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasTitle() => $_has(6);
  @$pb.TagNumber(7)
  void clearTitle() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get message => $_getSZ(7);
  @$pb.TagNumber(8)
  set message($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasMessage() => $_has(7);
  @$pb.TagNumber(8)
  void clearMessage() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.int get progress => $_getIZ(8);
  @$pb.TagNumber(9)
  set progress($core.int value) => $_setSignedInt32(8, value);
  @$pb.TagNumber(9)
  $core.bool hasProgress() => $_has(8);
  @$pb.TagNumber(9)
  void clearProgress() => $_clearField(9);

  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get changeSources => $_getList(9);

  @$pb.TagNumber(11)
  $core.int get eventCount => $_getIZ(10);
  @$pb.TagNumber(11)
  set eventCount($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasEventCount() => $_has(10);
  @$pb.TagNumber(11)
  void clearEventCount() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get summary => $_getSZ(11);
  @$pb.TagNumber(12)
  set summary($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasSummary() => $_has(11);
  @$pb.TagNumber(12)
  void clearSummary() => $_clearField(12);
}

class CommitDiff extends $pb.GeneratedMessage {
  factory CommitDiff({
    $core.String? baseHash,
    $core.String? targetHash,
    $core.Iterable<CommitDiffTable>? tables,
    $core.String? cue,
    $core.bool? truncated,
    $core.String? message,
    $core.String? unifiedDiff,
    $core.Iterable<CommitDiffTaskContext>? relatedTasks,
    $core.String? sql,
  }) {
    final result = create();
    if (baseHash != null) result.baseHash = baseHash;
    if (targetHash != null) result.targetHash = targetHash;
    if (tables != null) result.tables.addAll(tables);
    if (cue != null) result.cue = cue;
    if (truncated != null) result.truncated = truncated;
    if (message != null) result.message = message;
    if (unifiedDiff != null) result.unifiedDiff = unifiedDiff;
    if (relatedTasks != null) result.relatedTasks.addAll(relatedTasks);
    if (sql != null) result.sql = sql;
    return result;
  }

  CommitDiff._();

  factory CommitDiff.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommitDiff.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommitDiff',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'baseHash')
    ..aOS(2, _omitFieldNames ? '' : 'targetHash')
    ..pPM<CommitDiffTable>(3, _omitFieldNames ? '' : 'tables',
        subBuilder: CommitDiffTable.create)
    ..aOS(4, _omitFieldNames ? '' : 'cue')
    ..aOB(5, _omitFieldNames ? '' : 'truncated')
    ..aOS(6, _omitFieldNames ? '' : 'message')
    ..aOS(7, _omitFieldNames ? '' : 'unifiedDiff')
    ..pPM<CommitDiffTaskContext>(8, _omitFieldNames ? '' : 'relatedTasks',
        subBuilder: CommitDiffTaskContext.create)
    ..aOS(9, _omitFieldNames ? '' : 'sql')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiff clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommitDiff copyWith(void Function(CommitDiff) updates) =>
      super.copyWith((message) => updates(message as CommitDiff)) as CommitDiff;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommitDiff create() => CommitDiff._();
  @$core.override
  CommitDiff createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommitDiff getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommitDiff>(create);
  static CommitDiff? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get baseHash => $_getSZ(0);
  @$pb.TagNumber(1)
  set baseHash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBaseHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearBaseHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get targetHash => $_getSZ(1);
  @$pb.TagNumber(2)
  set targetHash($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTargetHash() => $_has(1);
  @$pb.TagNumber(2)
  void clearTargetHash() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<CommitDiffTable> get tables => $_getList(2);

  @$pb.TagNumber(4)
  $core.String get cue => $_getSZ(3);
  @$pb.TagNumber(4)
  set cue($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCue() => $_has(3);
  @$pb.TagNumber(4)
  void clearCue() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get truncated => $_getBF(4);
  @$pb.TagNumber(5)
  set truncated($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTruncated() => $_has(4);
  @$pb.TagNumber(5)
  void clearTruncated() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get message => $_getSZ(5);
  @$pb.TagNumber(6)
  set message($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearMessage() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get unifiedDiff => $_getSZ(6);
  @$pb.TagNumber(7)
  set unifiedDiff($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasUnifiedDiff() => $_has(6);
  @$pb.TagNumber(7)
  void clearUnifiedDiff() => $_clearField(7);

  @$pb.TagNumber(8)
  $pb.PbList<CommitDiffTaskContext> get relatedTasks => $_getList(7);

  @$pb.TagNumber(9)
  $core.String get sql => $_getSZ(8);
  @$pb.TagNumber(9)
  set sql($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasSql() => $_has(8);
  @$pb.TagNumber(9)
  void clearSql() => $_clearField(9);
}

class GetCommitDiffRequest extends $pb.GeneratedMessage {
  factory GetCommitDiffRequest({
    $core.String? commitHash,
    $core.String? baseHash,
    $core.String? remote,
  }) {
    final result = create();
    if (commitHash != null) result.commitHash = commitHash;
    if (baseHash != null) result.baseHash = baseHash;
    if (remote != null) result.remote = remote;
    return result;
  }

  GetCommitDiffRequest._();

  factory GetCommitDiffRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCommitDiffRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCommitDiffRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'commitHash')
    ..aOS(2, _omitFieldNames ? '' : 'baseHash')
    ..aOS(3, _omitFieldNames ? '' : 'remote')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCommitDiffRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCommitDiffRequest copyWith(void Function(GetCommitDiffRequest) updates) =>
      super.copyWith((message) => updates(message as GetCommitDiffRequest))
          as GetCommitDiffRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCommitDiffRequest create() => GetCommitDiffRequest._();
  @$core.override
  GetCommitDiffRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCommitDiffRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCommitDiffRequest>(create);
  static GetCommitDiffRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get commitHash => $_getSZ(0);
  @$pb.TagNumber(1)
  set commitHash($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommitHash() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommitHash() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get baseHash => $_getSZ(1);
  @$pb.TagNumber(2)
  set baseHash($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBaseHash() => $_has(1);
  @$pb.TagNumber(2)
  void clearBaseHash() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get remote => $_getSZ(2);
  @$pb.TagNumber(3)
  set remote($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRemote() => $_has(2);
  @$pb.TagNumber(3)
  void clearRemote() => $_clearField(3);
}

class GetCommitDiffResponse extends $pb.GeneratedMessage {
  factory GetCommitDiffResponse({
    CommitDiff? diff,
  }) {
    final result = create();
    if (diff != null) result.diff = diff;
    return result;
  }

  GetCommitDiffResponse._();

  factory GetCommitDiffResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCommitDiffResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCommitDiffResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<CommitDiff>(1, _omitFieldNames ? '' : 'diff',
        subBuilder: CommitDiff.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCommitDiffResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCommitDiffResponse copyWith(
          void Function(GetCommitDiffResponse) updates) =>
      super.copyWith((message) => updates(message as GetCommitDiffResponse))
          as GetCommitDiffResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCommitDiffResponse create() => GetCommitDiffResponse._();
  @$core.override
  GetCommitDiffResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCommitDiffResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCommitDiffResponse>(create);
  static GetCommitDiffResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CommitDiff get diff => $_getN(0);
  @$pb.TagNumber(1)
  set diff(CommitDiff value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDiff() => $_has(0);
  @$pb.TagNumber(1)
  void clearDiff() => $_clearField(1);
  @$pb.TagNumber(1)
  CommitDiff ensureDiff() => $_ensure(0);
}

class SqlCell extends $pb.GeneratedMessage {
  factory SqlCell({
    $core.String? value,
    $core.bool? isNull,
  }) {
    final result = create();
    if (value != null) result.value = value;
    if (isNull != null) result.isNull = isNull;
    return result;
  }

  SqlCell._();

  factory SqlCell.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SqlCell.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SqlCell',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'value')
    ..aOB(2, _omitFieldNames ? '' : 'isNull')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SqlCell clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SqlCell copyWith(void Function(SqlCell) updates) =>
      super.copyWith((message) => updates(message as SqlCell)) as SqlCell;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SqlCell create() => SqlCell._();
  @$core.override
  SqlCell createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SqlCell getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SqlCell>(create);
  static SqlCell? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get value => $_getSZ(0);
  @$pb.TagNumber(1)
  set value($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get isNull => $_getBF(1);
  @$pb.TagNumber(2)
  set isNull($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIsNull() => $_has(1);
  @$pb.TagNumber(2)
  void clearIsNull() => $_clearField(2);
}

class SqlRow extends $pb.GeneratedMessage {
  factory SqlRow({
    $core.Iterable<SqlCell>? cells,
  }) {
    final result = create();
    if (cells != null) result.cells.addAll(cells);
    return result;
  }

  SqlRow._();

  factory SqlRow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SqlRow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SqlRow',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<SqlCell>(1, _omitFieldNames ? '' : 'cells',
        subBuilder: SqlCell.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SqlRow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SqlRow copyWith(void Function(SqlRow) updates) =>
      super.copyWith((message) => updates(message as SqlRow)) as SqlRow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SqlRow create() => SqlRow._();
  @$core.override
  SqlRow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SqlRow getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SqlRow>(create);
  static SqlRow? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<SqlCell> get cells => $_getList(0);
}

class ExecuteSqlRequest extends $pb.GeneratedMessage {
  factory ExecuteSqlRequest({
    $core.String? sql,
    $core.int? maxRows,
  }) {
    final result = create();
    if (sql != null) result.sql = sql;
    if (maxRows != null) result.maxRows = maxRows;
    return result;
  }

  ExecuteSqlRequest._();

  factory ExecuteSqlRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ExecuteSqlRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ExecuteSqlRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'sql')
    ..aI(2, _omitFieldNames ? '' : 'maxRows')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteSqlRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteSqlRequest copyWith(void Function(ExecuteSqlRequest) updates) =>
      super.copyWith((message) => updates(message as ExecuteSqlRequest))
          as ExecuteSqlRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ExecuteSqlRequest create() => ExecuteSqlRequest._();
  @$core.override
  ExecuteSqlRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ExecuteSqlRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ExecuteSqlRequest>(create);
  static ExecuteSqlRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sql => $_getSZ(0);
  @$pb.TagNumber(1)
  set sql($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSql() => $_has(0);
  @$pb.TagNumber(1)
  void clearSql() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get maxRows => $_getIZ(1);
  @$pb.TagNumber(2)
  set maxRows($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMaxRows() => $_has(1);
  @$pb.TagNumber(2)
  void clearMaxRows() => $_clearField(2);
}

class ExecuteSqlResponse extends $pb.GeneratedMessage {
  factory ExecuteSqlResponse({
    $core.Iterable<$core.String>? columns,
    $core.Iterable<SqlRow>? rows,
    $fixnum.Int64? rowsAffected,
    $core.bool? truncated,
    $core.String? message,
  }) {
    final result = create();
    if (columns != null) result.columns.addAll(columns);
    if (rows != null) result.rows.addAll(rows);
    if (rowsAffected != null) result.rowsAffected = rowsAffected;
    if (truncated != null) result.truncated = truncated;
    if (message != null) result.message = message;
    return result;
  }

  ExecuteSqlResponse._();

  factory ExecuteSqlResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ExecuteSqlResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ExecuteSqlResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'columns')
    ..pPM<SqlRow>(2, _omitFieldNames ? '' : 'rows', subBuilder: SqlRow.create)
    ..aInt64(3, _omitFieldNames ? '' : 'rowsAffected')
    ..aOB(4, _omitFieldNames ? '' : 'truncated')
    ..aOS(5, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteSqlResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExecuteSqlResponse copyWith(void Function(ExecuteSqlResponse) updates) =>
      super.copyWith((message) => updates(message as ExecuteSqlResponse))
          as ExecuteSqlResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ExecuteSqlResponse create() => ExecuteSqlResponse._();
  @$core.override
  ExecuteSqlResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ExecuteSqlResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ExecuteSqlResponse>(create);
  static ExecuteSqlResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get columns => $_getList(0);

  @$pb.TagNumber(2)
  $pb.PbList<SqlRow> get rows => $_getList(1);

  @$pb.TagNumber(3)
  $fixnum.Int64 get rowsAffected => $_getI64(2);
  @$pb.TagNumber(3)
  set rowsAffected($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRowsAffected() => $_has(2);
  @$pb.TagNumber(3)
  void clearRowsAffected() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get truncated => $_getBF(3);
  @$pb.TagNumber(4)
  set truncated($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTruncated() => $_has(3);
  @$pb.TagNumber(4)
  void clearTruncated() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get message => $_getSZ(4);
  @$pb.TagNumber(5)
  set message($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearMessage() => $_clearField(5);
}

class WriteConfirmation extends $pb.GeneratedMessage {
  factory WriteConfirmation({
    $core.String? stage,
    $core.String? eventId,
    $core.String? publishedRootHash,
    $core.int? requiredOtherPeers,
    $core.int? confirmedOtherPeers,
    $core.bool? availabilityPending,
    $core.String? candidateScope,
    $core.Iterable<$core.String>? eligiblePeerIds,
    $core.bool? noCurrentEligiblePeers,
    $core.String? reasonCode,
  }) {
    final result = create();
    if (stage != null) result.stage = stage;
    if (eventId != null) result.eventId = eventId;
    if (publishedRootHash != null) result.publishedRootHash = publishedRootHash;
    if (requiredOtherPeers != null)
      result.requiredOtherPeers = requiredOtherPeers;
    if (confirmedOtherPeers != null)
      result.confirmedOtherPeers = confirmedOtherPeers;
    if (availabilityPending != null)
      result.availabilityPending = availabilityPending;
    if (candidateScope != null) result.candidateScope = candidateScope;
    if (eligiblePeerIds != null) result.eligiblePeerIds.addAll(eligiblePeerIds);
    if (noCurrentEligiblePeers != null)
      result.noCurrentEligiblePeers = noCurrentEligiblePeers;
    if (reasonCode != null) result.reasonCode = reasonCode;
    return result;
  }

  WriteConfirmation._();

  factory WriteConfirmation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WriteConfirmation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WriteConfirmation',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'stage')
    ..aOS(2, _omitFieldNames ? '' : 'eventId')
    ..aOS(3, _omitFieldNames ? '' : 'publishedRootHash')
    ..aI(4, _omitFieldNames ? '' : 'requiredOtherPeers')
    ..aI(5, _omitFieldNames ? '' : 'confirmedOtherPeers')
    ..aOB(6, _omitFieldNames ? '' : 'availabilityPending')
    ..aOS(7, _omitFieldNames ? '' : 'candidateScope')
    ..pPS(8, _omitFieldNames ? '' : 'eligiblePeerIds')
    ..aOB(9, _omitFieldNames ? '' : 'noCurrentEligiblePeers')
    ..aOS(10, _omitFieldNames ? '' : 'reasonCode')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteConfirmation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WriteConfirmation copyWith(void Function(WriteConfirmation) updates) =>
      super.copyWith((message) => updates(message as WriteConfirmation))
          as WriteConfirmation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WriteConfirmation create() => WriteConfirmation._();
  @$core.override
  WriteConfirmation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WriteConfirmation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WriteConfirmation>(create);
  static WriteConfirmation? _defaultInstance;

  /// Strongest observed boundary: no_change, local_accepted, or other_peer_available.
  @$pb.TagNumber(1)
  $core.String get stage => $_getSZ(0);
  @$pb.TagNumber(1)
  set stage($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStage() => $_has(0);
  @$pb.TagNumber(1)
  void clearStage() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get eventId => $_getSZ(1);
  @$pb.TagNumber(2)
  set eventId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEventId() => $_has(1);
  @$pb.TagNumber(2)
  void clearEventId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get publishedRootHash => $_getSZ(2);
  @$pb.TagNumber(3)
  set publishedRootHash($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPublishedRootHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearPublishedRootHash() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get requiredOtherPeers => $_getIZ(3);
  @$pb.TagNumber(4)
  set requiredOtherPeers($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRequiredOtherPeers() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequiredOtherPeers() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get confirmedOtherPeers => $_getIZ(4);
  @$pb.TagNumber(5)
  set confirmedOtherPeers($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasConfirmedOtherPeers() => $_has(4);
  @$pb.TagNumber(5)
  void clearConfirmedOtherPeers() => $_clearField(5);

  /// The mutation was accepted, but exact other-peer retention was not proved before returning; do not replay it.
  @$pb.TagNumber(6)
  $core.bool get availabilityPending => $_getBF(5);
  @$pb.TagNumber(6)
  set availabilityPending($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAvailabilityPending() => $_has(5);
  @$pb.TagNumber(6)
  void clearAvailabilityPending() => $_clearField(6);

  /// How eligible peers were selected for this observation.
  @$pb.TagNumber(7)
  $core.String get candidateScope => $_getSZ(6);
  @$pb.TagNumber(7)
  set candidateScope($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCandidateScope() => $_has(6);
  @$pb.TagNumber(7)
  void clearCandidateScope() => $_clearField(7);

  /// Topology candidates at the observation boundary; this is not receipt evidence.
  @$pb.TagNumber(8)
  $pb.PbList<$core.String> get eligiblePeerIds => $_getList(7);

  /// Weak local-only outcome; this does not prove other-peer availability.
  @$pb.TagNumber(9)
  $core.bool get noCurrentEligiblePeers => $_getBF(8);
  @$pb.TagNumber(9)
  set noCurrentEligiblePeers($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasNoCurrentEligiblePeers() => $_has(8);
  @$pb.TagNumber(9)
  void clearNoCurrentEligiblePeers() => $_clearField(9);

  /// Stable machine-readable status reason.
  @$pb.TagNumber(10)
  $core.String get reasonCode => $_getSZ(9);
  @$pb.TagNumber(10)
  set reasonCode($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasReasonCode() => $_has(9);
  @$pb.TagNumber(10)
  void clearReasonCode() => $_clearField(10);
}

class ProtosClientApiApi {
  final $pb.RpcClient _client;

  ProtosClientApiApi(this._client);

  $async.Future<InitResponse> init(
          $pb.ClientContext? ctx, InitRequest request) =>
      _client.invoke<InitResponse>(
          ctx, 'ProtosClientApi', 'Init', request, InitResponse());
  $async.Future<GetUserDevicesResponse> getUserDevices(
          $pb.ClientContext? ctx, GetUserDevicesRequest request) =>
      _client.invoke<GetUserDevicesResponse>(ctx, 'ProtosClientApi',
          'GetUserDevices', request, GetUserDevicesResponse());
  $async.Future<GetUserInfoResponse> getUserInfo(
          $pb.ClientContext? ctx, GetUserInfoRequest request) =>
      _client.invoke<GetUserInfoResponse>(ctx, 'ProtosClientApi', 'GetUserInfo',
          request, GetUserInfoResponse());
  $async.Future<ListOrganisationsResponse> listOrganisations(
          $pb.ClientContext? ctx, ListOrganisationsRequest request) =>
      _client.invoke<ListOrganisationsResponse>(ctx, 'ProtosClientApi',
          'ListOrganisations', request, ListOrganisationsResponse());
  $async.Future<StartDeviceInviteResponse> startDeviceInvite(
          $pb.ClientContext? ctx, StartDeviceInviteRequest request) =>
      _client.invoke<StartDeviceInviteResponse>(ctx, 'ProtosClientApi',
          'StartDeviceInvite', request, StartDeviceInviteResponse());
  $async.Future<ListNearbyOrganisationsResponse> listNearbyOrganisations(
          $pb.ClientContext? ctx, ListNearbyOrganisationsRequest request) =>
      _client.invoke<ListNearbyOrganisationsResponse>(
          ctx,
          'ProtosClientApi',
          'ListNearbyOrganisations',
          request,
          ListNearbyOrganisationsResponse());
  $async.Future<JoinOrganisationResponse> joinOrganisation(
          $pb.ClientContext? ctx, JoinOrganisationRequest request) =>
      _client.invoke<JoinOrganisationResponse>(ctx, 'ProtosClientApi',
          'JoinOrganisation', request, JoinOrganisationResponse());
  $async.Future<GetLocalSSHKeyResponse> getLocalSSHKey(
          $pb.ClientContext? ctx, GetLocalSSHKeyRequest request) =>
      _client.invoke<GetLocalSSHKeyResponse>(ctx, 'ProtosClientApi',
          'GetLocalSSHKey', request, GetLocalSSHKeyResponse());
  $async.Future<GetAppsResponse> getApps(
          $pb.ClientContext? ctx, GetAppsRequest request) =>
      _client.invoke<GetAppsResponse>(
          ctx, 'ProtosClientApi', 'GetApps', request, GetAppsResponse());
  $async.Future<CreateAppResponse> createApp(
          $pb.ClientContext? ctx, CreateAppRequest request) =>
      _client.invoke<CreateAppResponse>(
          ctx, 'ProtosClientApi', 'CreateApp', request, CreateAppResponse());
  $async.Future<StartAppResponse> startApp(
          $pb.ClientContext? ctx, StartAppRequest request) =>
      _client.invoke<StartAppResponse>(
          ctx, 'ProtosClientApi', 'StartApp', request, StartAppResponse());
  $async.Future<StopAppResponse> stopApp(
          $pb.ClientContext? ctx, StopAppRequest request) =>
      _client.invoke<StopAppResponse>(
          ctx, 'ProtosClientApi', 'StopApp', request, StopAppResponse());
  $async.Future<RemoveAppResponse> removeApp(
          $pb.ClientContext? ctx, RemoveAppRequest request) =>
      _client.invoke<RemoveAppResponse>(
          ctx, 'ProtosClientApi', 'RemoveApp', request, RemoveAppResponse());
  $async.Future<GetAppLogsResponse> getAppLogs(
          $pb.ClientContext? ctx, GetAppLogsRequest request) =>
      _client.invoke<GetAppLogsResponse>(
          ctx, 'ProtosClientApi', 'GetAppLogs', request, GetAppLogsResponse());
  $async.Future<GetSupportedProvisionersResponse> getSupportedProvisioners(
          $pb.ClientContext? ctx, GetSupportedProvisionersRequest request) =>
      _client.invoke<GetSupportedProvisionersResponse>(
          ctx,
          'ProtosClientApi',
          'GetSupportedProvisioners',
          request,
          GetSupportedProvisionersResponse());
  $async.Future<GetProvisionersResponse> getProvisioners(
          $pb.ClientContext? ctx, GetProvisionersRequest request) =>
      _client.invoke<GetProvisionersResponse>(ctx, 'ProtosClientApi',
          'GetProvisioners', request, GetProvisionersResponse());
  $async.Future<GetProvisionerResponse> getProvisioner(
          $pb.ClientContext? ctx, GetProvisionerRequest request) =>
      _client.invoke<GetProvisionerResponse>(ctx, 'ProtosClientApi',
          'GetProvisioner', request, GetProvisionerResponse());
  $async.Future<AddProvisionerResponse> addProvisioner(
          $pb.ClientContext? ctx, AddProvisionerRequest request) =>
      _client.invoke<AddProvisionerResponse>(ctx, 'ProtosClientApi',
          'AddProvisioner', request, AddProvisionerResponse());
  $async.Future<RemoveProvisionerResponse> removeProvisioner(
          $pb.ClientContext? ctx, RemoveProvisionerRequest request) =>
      _client.invoke<RemoveProvisionerResponse>(ctx, 'ProtosClientApi',
          'RemoveProvisioner', request, RemoveProvisionerResponse());
  $async.Future<GetInstancesResponse> getInstances(
          $pb.ClientContext? ctx, GetInstancesRequest request) =>
      _client.invoke<GetInstancesResponse>(ctx, 'ProtosClientApi',
          'GetInstances', request, GetInstancesResponse());
  $async.Future<GetInstanceResponse> getInstance(
          $pb.ClientContext? ctx, GetInstanceRequest request) =>
      _client.invoke<GetInstanceResponse>(ctx, 'ProtosClientApi', 'GetInstance',
          request, GetInstanceResponse());
  $async.Future<GetInstanceDeployOptionsResponse> getInstanceDeployOptions(
          $pb.ClientContext? ctx, GetInstanceDeployOptionsRequest request) =>
      _client.invoke<GetInstanceDeployOptionsResponse>(
          ctx,
          'ProtosClientApi',
          'GetInstanceDeployOptions',
          request,
          GetInstanceDeployOptionsResponse());
  $async.Future<DeployInstanceResponse> deployInstance(
          $pb.ClientContext? ctx, DeployInstanceRequest request) =>
      _client.invoke<DeployInstanceResponse>(ctx, 'ProtosClientApi',
          'DeployInstance', request, DeployInstanceResponse());
  $async.Future<RemoveInstanceResponse> removeInstance(
          $pb.ClientContext? ctx, RemoveInstanceRequest request) =>
      _client.invoke<RemoveInstanceResponse>(ctx, 'ProtosClientApi',
          'RemoveInstance', request, RemoveInstanceResponse());
  $async.Future<StartInstanceResponse> startInstance(
          $pb.ClientContext? ctx, StartInstanceRequest request) =>
      _client.invoke<StartInstanceResponse>(ctx, 'ProtosClientApi',
          'StartInstance', request, StartInstanceResponse());
  $async.Future<StopInstanceResponse> stopInstance(
          $pb.ClientContext? ctx, StopInstanceRequest request) =>
      _client.invoke<StopInstanceResponse>(ctx, 'ProtosClientApi',
          'StopInstance', request, StopInstanceResponse());
  $async.Future<GetInstanceKeyResponse> getInstanceKey(
          $pb.ClientContext? ctx, GetInstanceKeyRequest request) =>
      _client.invoke<GetInstanceKeyResponse>(ctx, 'ProtosClientApi',
          'GetInstanceKey', request, GetInstanceKeyResponse());
  $async.Future<GetInstanceLogsResponse> getInstanceLogs(
          $pb.ClientContext? ctx, GetInstanceLogsRequest request) =>
      _client.invoke<GetInstanceLogsResponse>(ctx, 'ProtosClientApi',
          'GetInstanceLogs', request, GetInstanceLogsResponse());
  $async.Future<InitInstanceResponse> initInstance(
          $pb.ClientContext? ctx, InitInstanceRequest request) =>
      _client.invoke<InitInstanceResponse>(ctx, 'ProtosClientApi',
          'InitInstance', request, InitInstanceResponse());
  $async.Future<UpdateInstanceResponse> updateInstance(
          $pb.ClientContext? ctx, UpdateInstanceRequest request) =>
      _client.invoke<UpdateInstanceResponse>(ctx, 'ProtosClientApi',
          'UpdateInstance', request, UpdateInstanceResponse());
  $async.Future<GetNetworkStateResponse> getNetworkState(
          $pb.ClientContext? ctx, GetNetworkStateRequest request) =>
      _client.invoke<GetNetworkStateResponse>(ctx, 'ProtosClientApi',
          'GetNetworkState', request, GetNetworkStateResponse());
  $async.Future<SetNetworkEnabledResponse> setNetworkEnabled(
          $pb.ClientContext? ctx, SetNetworkEnabledRequest request) =>
      _client.invoke<SetNetworkEnabledResponse>(ctx, 'ProtosClientApi',
          'SetNetworkEnabled', request, SetNetworkEnabledResponse());
  $async.Future<GetExitRoutesResponse> getExitRoutes(
          $pb.ClientContext? ctx, GetExitRoutesRequest request) =>
      _client.invoke<GetExitRoutesResponse>(ctx, 'ProtosClientApi',
          'GetExitRoutes', request, GetExitRoutesResponse());
  $async.Future<GetMobileTunnelConfigResponse> getMobileTunnelConfig(
          $pb.ClientContext? ctx, GetMobileTunnelConfigRequest request) =>
      _client.invoke<GetMobileTunnelConfigResponse>(ctx, 'ProtosClientApi',
          'GetMobileTunnelConfig', request, GetMobileTunnelConfigResponse());
  $async.Future<GetRuntimeStateResponse> getRuntimeState(
          $pb.ClientContext? ctx, GetRuntimeStateRequest request) =>
      _client.invoke<GetRuntimeStateResponse>(ctx, 'ProtosClientApi',
          'GetRuntimeState', request, GetRuntimeStateResponse());
  $async.Future<WatchChangesResponse> watchChanges(
          $pb.ClientContext? ctx, WatchChangesRequest request) =>
      _client.invoke<WatchChangesResponse>(ctx, 'ProtosClientApi',
          'WatchChanges', request, WatchChangesResponse());
  $async.Future<GetTasksResponse> getTasks(
          $pb.ClientContext? ctx, GetTasksRequest request) =>
      _client.invoke<GetTasksResponse>(
          ctx, 'ProtosClientApi', 'GetTasks', request, GetTasksResponse());
  $async.Future<GetTaskResponse> getTask(
          $pb.ClientContext? ctx, GetTaskRequest request) =>
      _client.invoke<GetTaskResponse>(
          ctx, 'ProtosClientApi', 'GetTask', request, GetTaskResponse());
  $async.Future<WatchTaskResponse> watchTask(
          $pb.ClientContext? ctx, WatchTaskRequest request) =>
      _client.invoke<WatchTaskResponse>(
          ctx, 'ProtosClientApi', 'WatchTask', request, WatchTaskResponse());
  $async.Future<SetExitRouteResponse> setExitRoute(
          $pb.ClientContext? ctx, SetExitRouteRequest request) =>
      _client.invoke<SetExitRouteResponse>(ctx, 'ProtosClientApi',
          'SetExitRoute', request, SetExitRouteResponse());
  $async.Future<ClearExitRouteResponse> clearExitRoute(
          $pb.ClientContext? ctx, ClearExitRouteRequest request) =>
      _client.invoke<ClearExitRouteResponse>(ctx, 'ProtosClientApi',
          'ClearExitRoute', request, ClearExitRouteResponse());
  $async.Future<GetProtosdReleasesResponse> getProtosdReleases(
          $pb.ClientContext? ctx, GetProtosdReleasesRequest request) =>
      _client.invoke<GetProtosdReleasesResponse>(ctx, 'ProtosClientApi',
          'GetProtosdReleases', request, GetProtosdReleasesResponse());
  $async.Future<GetProvisionerImagesResponse> getProvisionerImages(
          $pb.ClientContext? ctx, GetProvisionerImagesRequest request) =>
      _client.invoke<GetProvisionerImagesResponse>(ctx, 'ProtosClientApi',
          'GetProvisionerImages', request, GetProvisionerImagesResponse());
  $async.Future<UploadProvisionerImageResponse> uploadProvisionerImage(
          $pb.ClientContext? ctx, UploadProvisionerImageRequest request) =>
      _client.invoke<UploadProvisionerImageResponse>(ctx, 'ProtosClientApi',
          'UploadProvisionerImage', request, UploadProvisionerImageResponse());
  $async.Future<RemoveProvisionerImageResponse> removeProvisionerImage(
          $pb.ClientContext? ctx, RemoveProvisionerImageRequest request) =>
      _client.invoke<RemoveProvisionerImageResponse>(ctx, 'ProtosClientApi',
          'RemoveProvisionerImage', request, RemoveProvisionerImageResponse());
  $async.Future<GetInstanceImageResponse> getInstanceImage(
          $pb.ClientContext? ctx, GetInstanceImageRequest request) =>
      _client.invoke<GetInstanceImageResponse>(ctx, 'ProtosClientApi',
          'GetInstanceImage', request, GetInstanceImageResponse());
  $async.Future<UploadInstanceImageArchiveResponse> uploadInstanceImageArchive(
          $pb.ClientContext? ctx, UploadInstanceImageArchiveRequest request) =>
      _client.invoke<UploadInstanceImageArchiveResponse>(
          ctx,
          'ProtosClientApi',
          'UploadInstanceImageArchive',
          request,
          UploadInstanceImageArchiveResponse());
  $async.Future<GetSystemStatusResponse> getSystemStatus(
          $pb.ClientContext? ctx, GetSystemStatusRequest request) =>
      _client.invoke<GetSystemStatusResponse>(ctx, 'ProtosClientApi',
          'GetSystemStatus', request, GetSystemStatusResponse());
  $async.Future<StartHostAgentResponse> startHostAgent(
          $pb.ClientContext? ctx, StartHostAgentRequest request) =>
      _client.invoke<StartHostAgentResponse>(ctx, 'ProtosClientApi',
          'StartHostAgent', request, StartHostAgentResponse());
  $async.Future<StopHostAgentResponse> stopHostAgent(
          $pb.ClientContext? ctx, StopHostAgentRequest request) =>
      _client.invoke<StopHostAgentResponse>(ctx, 'ProtosClientApi',
          'StopHostAgent', request, StopHostAgentResponse());
  $async.Future<GetLocalCommitsResponse> getLocalCommits(
          $pb.ClientContext? ctx, GetLocalCommitsRequest request) =>
      _client.invoke<GetLocalCommitsResponse>(ctx, 'ProtosClientApi',
          'GetLocalCommits', request, GetLocalCommitsResponse());
  $async.Future<GetRemoteCommitsResponse> getRemoteCommits(
          $pb.ClientContext? ctx, GetRemoteCommitsRequest request) =>
      _client.invoke<GetRemoteCommitsResponse>(ctx, 'ProtosClientApi',
          'GetRemoteCommits', request, GetRemoteCommitsResponse());
  $async.Future<GetCommitDiffResponse> getCommitDiff(
          $pb.ClientContext? ctx, GetCommitDiffRequest request) =>
      _client.invoke<GetCommitDiffResponse>(ctx, 'ProtosClientApi',
          'GetCommitDiff', request, GetCommitDiffResponse());
  $async.Future<ExecuteSqlResponse> executeSql(
          $pb.ClientContext? ctx, ExecuteSqlRequest request) =>
      _client.invoke<ExecuteSqlResponse>(
          ctx, 'ProtosClientApi', 'ExecuteSql', request, ExecuteSqlResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
