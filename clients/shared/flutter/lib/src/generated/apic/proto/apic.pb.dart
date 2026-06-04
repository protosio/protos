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
    $core.String? organization,
  }) {
    final result = create();
    if (username != null) result.username = username;
    if (name != null) result.name = name;
    if (organization != null) result.organization = organization;
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
    ..aOS(3, _omitFieldNames ? '' : 'organization')
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
  $core.String get organization => $_getSZ(2);
  @$pb.TagNumber(3)
  set organization($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOrganization() => $_has(2);
  @$pb.TagNumber(3)
  void clearOrganization() => $_clearField(3);
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
  }) {
    final result = create();
    if (username != null) result.username = username;
    if (name != null) result.name = name;
    if (isAdmin != null) result.isAdmin = isAdmin;
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
  }) {
    final result = create();
    if (id != null) result.id = id;
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
  factory StartAppResponse() => create();

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
  factory StopAppResponse() => create();

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
  factory RemoveAppResponse() => create();

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

class Installer extends $pb.GeneratedMessage {
  factory Installer({
    $core.String? id,
    $core.String? name,
    $core.String? version,
    $core.String? description,
    $core.Iterable<$core.String>? requiresResources,
    $core.Iterable<$core.String>? providesResources,
    $core.Iterable<$core.String>? capabilities,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (version != null) result.version = version;
    if (description != null) result.description = description;
    if (requiresResources != null)
      result.requiresResources.addAll(requiresResources);
    if (providesResources != null)
      result.providesResources.addAll(providesResources);
    if (capabilities != null) result.capabilities.addAll(capabilities);
    return result;
  }

  Installer._();

  factory Installer.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Installer.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Installer',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'version')
    ..aOS(4, _omitFieldNames ? '' : 'description')
    ..pPS(5, _omitFieldNames ? '' : 'requiresResources')
    ..pPS(6, _omitFieldNames ? '' : 'providesResources')
    ..pPS(7, _omitFieldNames ? '' : 'capabilities')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Installer clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Installer copyWith(void Function(Installer) updates) =>
      super.copyWith((message) => updates(message as Installer)) as Installer;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Installer create() => Installer._();
  @$core.override
  Installer createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Installer getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Installer>(create);
  static Installer? _defaultInstance;

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
  $core.String get description => $_getSZ(3);
  @$pb.TagNumber(4)
  set description($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDescription() => $_has(3);
  @$pb.TagNumber(4)
  void clearDescription() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<$core.String> get requiresResources => $_getList(4);

  @$pb.TagNumber(6)
  $pb.PbList<$core.String> get providesResources => $_getList(5);

  @$pb.TagNumber(7)
  $pb.PbList<$core.String> get capabilities => $_getList(6);
}

class GetInstallersRequest extends $pb.GeneratedMessage {
  factory GetInstallersRequest() => create();

  GetInstallersRequest._();

  factory GetInstallersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstallersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstallersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallersRequest copyWith(void Function(GetInstallersRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstallersRequest))
          as GetInstallersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstallersRequest create() => GetInstallersRequest._();
  @$core.override
  GetInstallersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstallersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstallersRequest>(create);
  static GetInstallersRequest? _defaultInstance;
}

class GetInstallersResponse extends $pb.GeneratedMessage {
  factory GetInstallersResponse({
    $core.Iterable<Installer>? installers,
  }) {
    final result = create();
    if (installers != null) result.installers.addAll(installers);
    return result;
  }

  GetInstallersResponse._();

  factory GetInstallersResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstallersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstallersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<Installer>(1, _omitFieldNames ? '' : 'installers',
        subBuilder: Installer.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallersResponse copyWith(
          void Function(GetInstallersResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstallersResponse))
          as GetInstallersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstallersResponse create() => GetInstallersResponse._();
  @$core.override
  GetInstallersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstallersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstallersResponse>(create);
  static GetInstallersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<Installer> get installers => $_getList(0);
}

class GetInstallerRequest extends $pb.GeneratedMessage {
  factory GetInstallerRequest({
    $core.String? id,
  }) {
    final result = create();
    if (id != null) result.id = id;
    return result;
  }

  GetInstallerRequest._();

  factory GetInstallerRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstallerRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstallerRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallerRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallerRequest copyWith(void Function(GetInstallerRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstallerRequest))
          as GetInstallerRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstallerRequest create() => GetInstallerRequest._();
  @$core.override
  GetInstallerRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstallerRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstallerRequest>(create);
  static GetInstallerRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);
}

class GetInstallerResponse extends $pb.GeneratedMessage {
  factory GetInstallerResponse({
    Installer? installer,
  }) {
    final result = create();
    if (installer != null) result.installer = installer;
    return result;
  }

  GetInstallerResponse._();

  factory GetInstallerResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetInstallerResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstallerResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<Installer>(1, _omitFieldNames ? '' : 'installer',
        subBuilder: Installer.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallerResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstallerResponse copyWith(void Function(GetInstallerResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstallerResponse))
          as GetInstallerResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstallerResponse create() => GetInstallerResponse._();
  @$core.override
  GetInstallerResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetInstallerResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstallerResponse>(create);
  static GetInstallerResponse? _defaultInstance;

  @$pb.TagNumber(1)
  Installer get installer => $_getN(0);
  @$pb.TagNumber(1)
  set installer(Installer value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasInstaller() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstaller() => $_clearField(1);
  @$pb.TagNumber(1)
  Installer ensureInstaller() => $_ensure(0);
}

class CloudMachineSpec extends $pb.GeneratedMessage {
  factory CloudMachineSpec({
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

  CloudMachineSpec._();

  factory CloudMachineSpec.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudMachineSpec.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudMachineSpec',
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
  CloudMachineSpec clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudMachineSpec copyWith(void Function(CloudMachineSpec) updates) =>
      super.copyWith((message) => updates(message as CloudMachineSpec))
          as CloudMachineSpec;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudMachineSpec create() => CloudMachineSpec._();
  @$core.override
  CloudMachineSpec createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudMachineSpec getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudMachineSpec>(create);
  static CloudMachineSpec? _defaultInstance;

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

class CloudType extends $pb.GeneratedMessage {
  factory CloudType({
    $core.String? name,
    $core.Iterable<$core.String>? authenticationFields,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (authenticationFields != null)
      result.authenticationFields.addAll(authenticationFields);
    return result;
  }

  CloudType._();

  factory CloudType.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudType.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudType',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..pPS(2, _omitFieldNames ? '' : 'authenticationFields')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudType clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudType copyWith(void Function(CloudType) updates) =>
      super.copyWith((message) => updates(message as CloudType)) as CloudType;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudType create() => CloudType._();
  @$core.override
  CloudType createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudType getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CloudType>(create);
  static CloudType? _defaultInstance;

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

class CloudProvider extends $pb.GeneratedMessage {
  factory CloudProvider({
    $core.String? name,
    CloudType? type,
    $core.Iterable<$core.String>? supportedLocations,
    $core.Iterable<$core.MapEntry<$core.String, CloudMachineSpec>>?
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

  CloudProvider._();

  factory CloudProvider.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudProvider.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudProvider',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOM<CloudType>(2, _omitFieldNames ? '' : 'type',
        subBuilder: CloudType.create)
    ..pPS(3, _omitFieldNames ? '' : 'supportedLocations')
    ..m<$core.String, CloudMachineSpec>(
        4, _omitFieldNames ? '' : 'supportedMachines',
        entryClassName: 'CloudProvider.SupportedMachinesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: CloudMachineSpec.create,
        valueDefaultOrMaker: CloudMachineSpec.getDefault,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProvider clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudProvider copyWith(void Function(CloudProvider) updates) =>
      super.copyWith((message) => updates(message as CloudProvider))
          as CloudProvider;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudProvider create() => CloudProvider._();
  @$core.override
  CloudProvider createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudProvider getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudProvider>(create);
  static CloudProvider? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  CloudType get type => $_getN(1);
  @$pb.TagNumber(2)
  set type(CloudType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => $_clearField(2);
  @$pb.TagNumber(2)
  CloudType ensureType() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get supportedLocations => $_getList(2);

  @$pb.TagNumber(4)
  $pb.PbMap<$core.String, CloudMachineSpec> get supportedMachines =>
      $_getMap(3);
}

class GetSupportedCloudProvidersRequest extends $pb.GeneratedMessage {
  factory GetSupportedCloudProvidersRequest() => create();

  GetSupportedCloudProvidersRequest._();

  factory GetSupportedCloudProvidersRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSupportedCloudProvidersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSupportedCloudProvidersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedCloudProvidersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedCloudProvidersRequest copyWith(
          void Function(GetSupportedCloudProvidersRequest) updates) =>
      super.copyWith((message) =>
              updates(message as GetSupportedCloudProvidersRequest))
          as GetSupportedCloudProvidersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSupportedCloudProvidersRequest create() =>
      GetSupportedCloudProvidersRequest._();
  @$core.override
  GetSupportedCloudProvidersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSupportedCloudProvidersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSupportedCloudProvidersRequest>(
          create);
  static GetSupportedCloudProvidersRequest? _defaultInstance;
}

class GetSupportedCloudProvidersResponse extends $pb.GeneratedMessage {
  factory GetSupportedCloudProvidersResponse({
    $core.Iterable<CloudType>? cloudTypes,
  }) {
    final result = create();
    if (cloudTypes != null) result.cloudTypes.addAll(cloudTypes);
    return result;
  }

  GetSupportedCloudProvidersResponse._();

  factory GetSupportedCloudProvidersResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetSupportedCloudProvidersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetSupportedCloudProvidersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<CloudType>(1, _omitFieldNames ? '' : 'cloudTypes',
        subBuilder: CloudType.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedCloudProvidersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetSupportedCloudProvidersResponse copyWith(
          void Function(GetSupportedCloudProvidersResponse) updates) =>
      super.copyWith((message) =>
              updates(message as GetSupportedCloudProvidersResponse))
          as GetSupportedCloudProvidersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetSupportedCloudProvidersResponse create() =>
      GetSupportedCloudProvidersResponse._();
  @$core.override
  GetSupportedCloudProvidersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetSupportedCloudProvidersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetSupportedCloudProvidersResponse>(
          create);
  static GetSupportedCloudProvidersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<CloudType> get cloudTypes => $_getList(0);
}

class GetCloudProvidersRequest extends $pb.GeneratedMessage {
  factory GetCloudProvidersRequest() => create();

  GetCloudProvidersRequest._();

  factory GetCloudProvidersRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudProvidersRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudProvidersRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProvidersRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProvidersRequest copyWith(
          void Function(GetCloudProvidersRequest) updates) =>
      super.copyWith((message) => updates(message as GetCloudProvidersRequest))
          as GetCloudProvidersRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudProvidersRequest create() => GetCloudProvidersRequest._();
  @$core.override
  GetCloudProvidersRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudProvidersRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudProvidersRequest>(create);
  static GetCloudProvidersRequest? _defaultInstance;
}

class GetCloudProvidersResponse extends $pb.GeneratedMessage {
  factory GetCloudProvidersResponse({
    $core.Iterable<CloudProvider>? cloudProviders,
  }) {
    final result = create();
    if (cloudProviders != null) result.cloudProviders.addAll(cloudProviders);
    return result;
  }

  GetCloudProvidersResponse._();

  factory GetCloudProvidersResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudProvidersResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudProvidersResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..pPM<CloudProvider>(1, _omitFieldNames ? '' : 'cloudProviders',
        subBuilder: CloudProvider.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProvidersResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProvidersResponse copyWith(
          void Function(GetCloudProvidersResponse) updates) =>
      super.copyWith((message) => updates(message as GetCloudProvidersResponse))
          as GetCloudProvidersResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudProvidersResponse create() => GetCloudProvidersResponse._();
  @$core.override
  GetCloudProvidersResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudProvidersResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudProvidersResponse>(create);
  static GetCloudProvidersResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<CloudProvider> get cloudProviders => $_getList(0);
}

class GetCloudProviderRequest extends $pb.GeneratedMessage {
  factory GetCloudProviderRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetCloudProviderRequest._();

  factory GetCloudProviderRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudProviderRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudProviderRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProviderRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProviderRequest copyWith(
          void Function(GetCloudProviderRequest) updates) =>
      super.copyWith((message) => updates(message as GetCloudProviderRequest))
          as GetCloudProviderRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudProviderRequest create() => GetCloudProviderRequest._();
  @$core.override
  GetCloudProviderRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudProviderRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudProviderRequest>(create);
  static GetCloudProviderRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetCloudProviderResponse extends $pb.GeneratedMessage {
  factory GetCloudProviderResponse({
    CloudProvider? cloudProvider,
  }) {
    final result = create();
    if (cloudProvider != null) result.cloudProvider = cloudProvider;
    return result;
  }

  GetCloudProviderResponse._();

  factory GetCloudProviderResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudProviderResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudProviderResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOM<CloudProvider>(1, _omitFieldNames ? '' : 'cloudProvider',
        subBuilder: CloudProvider.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProviderResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudProviderResponse copyWith(
          void Function(GetCloudProviderResponse) updates) =>
      super.copyWith((message) => updates(message as GetCloudProviderResponse))
          as GetCloudProviderResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudProviderResponse create() => GetCloudProviderResponse._();
  @$core.override
  GetCloudProviderResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudProviderResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudProviderResponse>(create);
  static GetCloudProviderResponse? _defaultInstance;

  @$pb.TagNumber(1)
  CloudProvider get cloudProvider => $_getN(0);
  @$pb.TagNumber(1)
  set cloudProvider(CloudProvider value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCloudProvider() => $_has(0);
  @$pb.TagNumber(1)
  void clearCloudProvider() => $_clearField(1);
  @$pb.TagNumber(1)
  CloudProvider ensureCloudProvider() => $_ensure(0);
}

class AddCloudProviderRequest extends $pb.GeneratedMessage {
  factory AddCloudProviderRequest({
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

  AddCloudProviderRequest._();

  factory AddCloudProviderRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AddCloudProviderRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AddCloudProviderRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'type')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'credentials',
        entryClassName: 'AddCloudProviderRequest.CredentialsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddCloudProviderRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddCloudProviderRequest copyWith(
          void Function(AddCloudProviderRequest) updates) =>
      super.copyWith((message) => updates(message as AddCloudProviderRequest))
          as AddCloudProviderRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AddCloudProviderRequest create() => AddCloudProviderRequest._();
  @$core.override
  AddCloudProviderRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AddCloudProviderRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AddCloudProviderRequest>(create);
  static AddCloudProviderRequest? _defaultInstance;

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

class AddCloudProviderResponse extends $pb.GeneratedMessage {
  factory AddCloudProviderResponse() => create();

  AddCloudProviderResponse._();

  factory AddCloudProviderResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AddCloudProviderResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AddCloudProviderResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddCloudProviderResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddCloudProviderResponse copyWith(
          void Function(AddCloudProviderResponse) updates) =>
      super.copyWith((message) => updates(message as AddCloudProviderResponse))
          as AddCloudProviderResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AddCloudProviderResponse create() => AddCloudProviderResponse._();
  @$core.override
  AddCloudProviderResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AddCloudProviderResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AddCloudProviderResponse>(create);
  static AddCloudProviderResponse? _defaultInstance;
}

class RemoveCloudProviderRequest extends $pb.GeneratedMessage {
  factory RemoveCloudProviderRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  RemoveCloudProviderRequest._();

  factory RemoveCloudProviderRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveCloudProviderRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveCloudProviderRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudProviderRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudProviderRequest copyWith(
          void Function(RemoveCloudProviderRequest) updates) =>
      super.copyWith(
              (message) => updates(message as RemoveCloudProviderRequest))
          as RemoveCloudProviderRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveCloudProviderRequest create() => RemoveCloudProviderRequest._();
  @$core.override
  RemoveCloudProviderRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveCloudProviderRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveCloudProviderRequest>(create);
  static RemoveCloudProviderRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class RemoveCloudProviderResponse extends $pb.GeneratedMessage {
  factory RemoveCloudProviderResponse() => create();

  RemoveCloudProviderResponse._();

  factory RemoveCloudProviderResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveCloudProviderResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveCloudProviderResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudProviderResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudProviderResponse copyWith(
          void Function(RemoveCloudProviderResponse) updates) =>
      super.copyWith(
              (message) => updates(message as RemoveCloudProviderResponse))
          as RemoveCloudProviderResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveCloudProviderResponse create() =>
      RemoveCloudProviderResponse._();
  @$core.override
  RemoveCloudProviderResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveCloudProviderResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveCloudProviderResponse>(create);
  static RemoveCloudProviderResponse? _defaultInstance;
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
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
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
  factory RemoveInstanceResponse() => create();

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
  factory StartInstanceResponse() => create();

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
  factory StopInstanceResponse() => create();

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
  }) {
    final result = create();
    if (instance != null) result.instance = instance;
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
  }) {
    final result = create();
    if (status != null) result.status = status;
    if (stream != null) result.stream = stream;
    if (subjectType != null) result.subjectType = subjectType;
    if (subjectId != null) result.subjectId = subjectId;
    if (maxResults != null) result.maxResults = maxResults;
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
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (includeEvents != null) result.includeEvents = includeEvents;
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
  }) {
    final result = create();
    if (route != null) result.route = route;
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
  factory ClearExitRouteResponse() => create();

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
}

class RuntimeState extends $pb.GeneratedMessage {
  factory RuntimeState({
    $core.String? peerId,
    $core.String? manifestDigest,
    $core.String? finalizedRootHash,
    $core.String? tentativeRootHash,
    $core.String? protocolFinalizedRootHash,
    $core.String? durableMainRootHash,
    $core.String? activeEpochId,
    $core.Iterable<$core.String>? activeWitnessIds,
    $core.Iterable<$core.String>? eligibleWitnessIds,
    $core.Iterable<$core.String>? stateProviders,
    $core.Iterable<$core.String>? connectedPeers,
    $core.String? fatalState,
    $core.bool? runtimeRefreshPending,
    $core.String? runtimeRefreshLastError,
    $core.bool? runtimeFinalizedPending,
    $core.String? runtimeFinalizedLastError,
    $core.String? runtimeMaterializationPolicy,
    $core.Iterable<RuntimePeerStatus>? peerStatuses,
    $core.Iterable<RuntimeCompatibility>? compatibility,
    $core.Iterable<$core.String>? contentSyncTrace,
    $core.Iterable<$core.String>? knownEpochIds,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>?
        epochDescriptorDigestById,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>?
        epochFinalizedDigestById,
    $core.String? protocolFinalizedDigest,
  }) {
    final result = create();
    if (peerId != null) result.peerId = peerId;
    if (manifestDigest != null) result.manifestDigest = manifestDigest;
    if (finalizedRootHash != null) result.finalizedRootHash = finalizedRootHash;
    if (tentativeRootHash != null) result.tentativeRootHash = tentativeRootHash;
    if (protocolFinalizedRootHash != null)
      result.protocolFinalizedRootHash = protocolFinalizedRootHash;
    if (durableMainRootHash != null)
      result.durableMainRootHash = durableMainRootHash;
    if (activeEpochId != null) result.activeEpochId = activeEpochId;
    if (activeWitnessIds != null)
      result.activeWitnessIds.addAll(activeWitnessIds);
    if (eligibleWitnessIds != null)
      result.eligibleWitnessIds.addAll(eligibleWitnessIds);
    if (stateProviders != null) result.stateProviders.addAll(stateProviders);
    if (connectedPeers != null) result.connectedPeers.addAll(connectedPeers);
    if (fatalState != null) result.fatalState = fatalState;
    if (runtimeRefreshPending != null)
      result.runtimeRefreshPending = runtimeRefreshPending;
    if (runtimeRefreshLastError != null)
      result.runtimeRefreshLastError = runtimeRefreshLastError;
    if (runtimeFinalizedPending != null)
      result.runtimeFinalizedPending = runtimeFinalizedPending;
    if (runtimeFinalizedLastError != null)
      result.runtimeFinalizedLastError = runtimeFinalizedLastError;
    if (runtimeMaterializationPolicy != null)
      result.runtimeMaterializationPolicy = runtimeMaterializationPolicy;
    if (peerStatuses != null) result.peerStatuses.addAll(peerStatuses);
    if (compatibility != null) result.compatibility.addAll(compatibility);
    if (contentSyncTrace != null)
      result.contentSyncTrace.addAll(contentSyncTrace);
    if (knownEpochIds != null) result.knownEpochIds.addAll(knownEpochIds);
    if (epochDescriptorDigestById != null)
      result.epochDescriptorDigestById.addEntries(epochDescriptorDigestById);
    if (epochFinalizedDigestById != null)
      result.epochFinalizedDigestById.addEntries(epochFinalizedDigestById);
    if (protocolFinalizedDigest != null)
      result.protocolFinalizedDigest = protocolFinalizedDigest;
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
    ..aOS(3, _omitFieldNames ? '' : 'finalizedRootHash')
    ..aOS(4, _omitFieldNames ? '' : 'tentativeRootHash')
    ..aOS(5, _omitFieldNames ? '' : 'protocolFinalizedRootHash')
    ..aOS(6, _omitFieldNames ? '' : 'durableMainRootHash')
    ..aOS(7, _omitFieldNames ? '' : 'activeEpochId')
    ..pPS(8, _omitFieldNames ? '' : 'activeWitnessIds')
    ..pPS(9, _omitFieldNames ? '' : 'eligibleWitnessIds')
    ..pPS(10, _omitFieldNames ? '' : 'stateProviders')
    ..pPS(11, _omitFieldNames ? '' : 'connectedPeers')
    ..aOS(12, _omitFieldNames ? '' : 'fatalState')
    ..aOB(13, _omitFieldNames ? '' : 'runtimeRefreshPending')
    ..aOS(14, _omitFieldNames ? '' : 'runtimeRefreshLastError')
    ..aOB(15, _omitFieldNames ? '' : 'runtimeFinalizedPending')
    ..aOS(16, _omitFieldNames ? '' : 'runtimeFinalizedLastError')
    ..aOS(17, _omitFieldNames ? '' : 'runtimeMaterializationPolicy')
    ..pPM<RuntimePeerStatus>(18, _omitFieldNames ? '' : 'peerStatuses',
        subBuilder: RuntimePeerStatus.create)
    ..pPM<RuntimeCompatibility>(19, _omitFieldNames ? '' : 'compatibility',
        subBuilder: RuntimeCompatibility.create)
    ..pPS(20, _omitFieldNames ? '' : 'contentSyncTrace')
    ..pPS(21, _omitFieldNames ? '' : 'knownEpochIds')
    ..m<$core.String, $core.String>(
        22, _omitFieldNames ? '' : 'epochDescriptorDigestById',
        entryClassName: 'RuntimeState.EpochDescriptorDigestByIdEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..m<$core.String, $core.String>(
        23, _omitFieldNames ? '' : 'epochFinalizedDigestById',
        entryClassName: 'RuntimeState.EpochFinalizedDigestByIdEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('apic'))
    ..aOS(24, _omitFieldNames ? '' : 'protocolFinalizedDigest')
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
  $core.String get finalizedRootHash => $_getSZ(2);
  @$pb.TagNumber(3)
  set finalizedRootHash($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFinalizedRootHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearFinalizedRootHash() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get tentativeRootHash => $_getSZ(3);
  @$pb.TagNumber(4)
  set tentativeRootHash($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTentativeRootHash() => $_has(3);
  @$pb.TagNumber(4)
  void clearTentativeRootHash() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get protocolFinalizedRootHash => $_getSZ(4);
  @$pb.TagNumber(5)
  set protocolFinalizedRootHash($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProtocolFinalizedRootHash() => $_has(4);
  @$pb.TagNumber(5)
  void clearProtocolFinalizedRootHash() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get durableMainRootHash => $_getSZ(5);
  @$pb.TagNumber(6)
  set durableMainRootHash($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDurableMainRootHash() => $_has(5);
  @$pb.TagNumber(6)
  void clearDurableMainRootHash() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get activeEpochId => $_getSZ(6);
  @$pb.TagNumber(7)
  set activeEpochId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasActiveEpochId() => $_has(6);
  @$pb.TagNumber(7)
  void clearActiveEpochId() => $_clearField(7);

  @$pb.TagNumber(8)
  $pb.PbList<$core.String> get activeWitnessIds => $_getList(7);

  @$pb.TagNumber(9)
  $pb.PbList<$core.String> get eligibleWitnessIds => $_getList(8);

  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get stateProviders => $_getList(9);

  @$pb.TagNumber(11)
  $pb.PbList<$core.String> get connectedPeers => $_getList(10);

  @$pb.TagNumber(12)
  $core.String get fatalState => $_getSZ(11);
  @$pb.TagNumber(12)
  set fatalState($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasFatalState() => $_has(11);
  @$pb.TagNumber(12)
  void clearFatalState() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.bool get runtimeRefreshPending => $_getBF(12);
  @$pb.TagNumber(13)
  set runtimeRefreshPending($core.bool value) => $_setBool(12, value);
  @$pb.TagNumber(13)
  $core.bool hasRuntimeRefreshPending() => $_has(12);
  @$pb.TagNumber(13)
  void clearRuntimeRefreshPending() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get runtimeRefreshLastError => $_getSZ(13);
  @$pb.TagNumber(14)
  set runtimeRefreshLastError($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasRuntimeRefreshLastError() => $_has(13);
  @$pb.TagNumber(14)
  void clearRuntimeRefreshLastError() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get runtimeFinalizedPending => $_getBF(14);
  @$pb.TagNumber(15)
  set runtimeFinalizedPending($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(15)
  $core.bool hasRuntimeFinalizedPending() => $_has(14);
  @$pb.TagNumber(15)
  void clearRuntimeFinalizedPending() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get runtimeFinalizedLastError => $_getSZ(15);
  @$pb.TagNumber(16)
  set runtimeFinalizedLastError($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasRuntimeFinalizedLastError() => $_has(15);
  @$pb.TagNumber(16)
  void clearRuntimeFinalizedLastError() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.String get runtimeMaterializationPolicy => $_getSZ(16);
  @$pb.TagNumber(17)
  set runtimeMaterializationPolicy($core.String value) =>
      $_setString(16, value);
  @$pb.TagNumber(17)
  $core.bool hasRuntimeMaterializationPolicy() => $_has(16);
  @$pb.TagNumber(17)
  void clearRuntimeMaterializationPolicy() => $_clearField(17);

  @$pb.TagNumber(18)
  $pb.PbList<RuntimePeerStatus> get peerStatuses => $_getList(17);

  @$pb.TagNumber(19)
  $pb.PbList<RuntimeCompatibility> get compatibility => $_getList(18);

  @$pb.TagNumber(20)
  $pb.PbList<$core.String> get contentSyncTrace => $_getList(19);

  @$pb.TagNumber(21)
  $pb.PbList<$core.String> get knownEpochIds => $_getList(20);

  @$pb.TagNumber(22)
  $pb.PbMap<$core.String, $core.String> get epochDescriptorDigestById =>
      $_getMap(21);

  @$pb.TagNumber(23)
  $pb.PbMap<$core.String, $core.String> get epochFinalizedDigestById =>
      $_getMap(22);

  @$pb.TagNumber(24)
  $core.String get protocolFinalizedDigest => $_getSZ(23);
  @$pb.TagNumber(24)
  set protocolFinalizedDigest($core.String value) => $_setString(23, value);
  @$pb.TagNumber(24)
  $core.bool hasProtocolFinalizedDigest() => $_has(23);
  @$pb.TagNumber(24)
  void clearProtocolFinalizedDigest() => $_clearField(24);
}

class RuntimePeerStatus extends $pb.GeneratedMessage {
  factory RuntimePeerStatus({
    $core.String? peerId,
    $core.bool? connected,
    $core.bool? dialable,
    $core.bool? stateProvider,
    $core.bool? witness,
    $core.bool? eligibleWitness,
    $core.bool? compatible,
    $core.bool? incompatible,
    $core.bool? ignored,
    $core.bool? relayOnly,
    $core.Iterable<$core.String>? addresses,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? lastDialErrors,
    $core.String? reason,
  }) {
    final result = create();
    if (peerId != null) result.peerId = peerId;
    if (connected != null) result.connected = connected;
    if (dialable != null) result.dialable = dialable;
    if (stateProvider != null) result.stateProvider = stateProvider;
    if (witness != null) result.witness = witness;
    if (eligibleWitness != null) result.eligibleWitness = eligibleWitness;
    if (compatible != null) result.compatible = compatible;
    if (incompatible != null) result.incompatible = incompatible;
    if (ignored != null) result.ignored = ignored;
    if (relayOnly != null) result.relayOnly = relayOnly;
    if (addresses != null) result.addresses.addAll(addresses);
    if (lastDialErrors != null)
      result.lastDialErrors.addEntries(lastDialErrors);
    if (reason != null) result.reason = reason;
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
    ..aOB(2, _omitFieldNames ? '' : 'connected')
    ..aOB(3, _omitFieldNames ? '' : 'dialable')
    ..aOB(4, _omitFieldNames ? '' : 'stateProvider')
    ..aOB(5, _omitFieldNames ? '' : 'witness')
    ..aOB(6, _omitFieldNames ? '' : 'eligibleWitness')
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

  @$pb.TagNumber(2)
  $core.bool get connected => $_getBF(1);
  @$pb.TagNumber(2)
  set connected($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConnected() => $_has(1);
  @$pb.TagNumber(2)
  void clearConnected() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get dialable => $_getBF(2);
  @$pb.TagNumber(3)
  set dialable($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDialable() => $_has(2);
  @$pb.TagNumber(3)
  void clearDialable() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get stateProvider => $_getBF(3);
  @$pb.TagNumber(4)
  set stateProvider($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStateProvider() => $_has(3);
  @$pb.TagNumber(4)
  void clearStateProvider() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get witness => $_getBF(4);
  @$pb.TagNumber(5)
  set witness($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasWitness() => $_has(4);
  @$pb.TagNumber(5)
  void clearWitness() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get eligibleWitness => $_getBF(5);
  @$pb.TagNumber(6)
  set eligibleWitness($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasEligibleWitness() => $_has(5);
  @$pb.TagNumber(6)
  void clearEligibleWitness() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get compatible => $_getBF(6);
  @$pb.TagNumber(7)
  set compatible($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCompatible() => $_has(6);
  @$pb.TagNumber(7)
  void clearCompatible() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get incompatible => $_getBF(7);
  @$pb.TagNumber(8)
  set incompatible($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasIncompatible() => $_has(7);
  @$pb.TagNumber(8)
  void clearIncompatible() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get ignored => $_getBF(8);
  @$pb.TagNumber(9)
  set ignored($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIgnored() => $_has(8);
  @$pb.TagNumber(9)
  void clearIgnored() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.bool get relayOnly => $_getBF(9);
  @$pb.TagNumber(10)
  set relayOnly($core.bool value) => $_setBool(9, value);
  @$pb.TagNumber(10)
  $core.bool hasRelayOnly() => $_has(9);
  @$pb.TagNumber(10)
  void clearRelayOnly() => $_clearField(10);

  @$pb.TagNumber(11)
  $pb.PbList<$core.String> get addresses => $_getList(10);

  @$pb.TagNumber(12)
  $pb.PbMap<$core.String, $core.String> get lastDialErrors => $_getMap(11);

  @$pb.TagNumber(13)
  $core.String get reason => $_getSZ(12);
  @$pb.TagNumber(13)
  set reason($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasReason() => $_has(12);
  @$pb.TagNumber(13)
  void clearReason() => $_clearField(13);
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

class CloudSpecificImage extends $pb.GeneratedMessage {
  factory CloudSpecificImage({
    $core.String? id,
    $core.String? name,
    $core.String? location,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (location != null) result.location = location;
    return result;
  }

  CloudSpecificImage._();

  factory CloudSpecificImage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CloudSpecificImage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CloudSpecificImage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'location')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudSpecificImage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CloudSpecificImage copyWith(void Function(CloudSpecificImage) updates) =>
      super.copyWith((message) => updates(message as CloudSpecificImage))
          as CloudSpecificImage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CloudSpecificImage create() => CloudSpecificImage._();
  @$core.override
  CloudSpecificImage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CloudSpecificImage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CloudSpecificImage>(create);
  static CloudSpecificImage? _defaultInstance;

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

class GetCloudImagesRequest extends $pb.GeneratedMessage {
  factory GetCloudImagesRequest({
    $core.String? name,
  }) {
    final result = create();
    if (name != null) result.name = name;
    return result;
  }

  GetCloudImagesRequest._();

  factory GetCloudImagesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudImagesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudImagesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudImagesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudImagesRequest copyWith(
          void Function(GetCloudImagesRequest) updates) =>
      super.copyWith((message) => updates(message as GetCloudImagesRequest))
          as GetCloudImagesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudImagesRequest create() => GetCloudImagesRequest._();
  @$core.override
  GetCloudImagesRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudImagesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudImagesRequest>(create);
  static GetCloudImagesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);
}

class GetCloudImagesResponse extends $pb.GeneratedMessage {
  factory GetCloudImagesResponse({
    $core.Iterable<$core.MapEntry<$core.String, CloudSpecificImage>>?
        cloudImages,
  }) {
    final result = create();
    if (cloudImages != null) result.cloudImages.addEntries(cloudImages);
    return result;
  }

  GetCloudImagesResponse._();

  factory GetCloudImagesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GetCloudImagesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetCloudImagesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..m<$core.String, CloudSpecificImage>(
        1, _omitFieldNames ? '' : 'cloudImages',
        entryClassName: 'GetCloudImagesResponse.CloudImagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: CloudSpecificImage.create,
        valueDefaultOrMaker: CloudSpecificImage.getDefault,
        packageName: const $pb.PackageName('apic'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudImagesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetCloudImagesResponse copyWith(
          void Function(GetCloudImagesResponse) updates) =>
      super.copyWith((message) => updates(message as GetCloudImagesResponse))
          as GetCloudImagesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetCloudImagesResponse create() => GetCloudImagesResponse._();
  @$core.override
  GetCloudImagesResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GetCloudImagesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetCloudImagesResponse>(create);
  static GetCloudImagesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbMap<$core.String, CloudSpecificImage> get cloudImages => $_getMap(0);
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
    $core.Iterable<$core.MapEntry<$core.String, CloudSpecificImage>>? images,
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
    ..m<$core.String, CloudSpecificImage>(1, _omitFieldNames ? '' : 'images',
        entryClassName: 'GetProvisionerImagesResponse.ImagesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OM,
        valueCreator: CloudSpecificImage.create,
        valueDefaultOrMaker: CloudSpecificImage.getDefault,
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
  $pb.PbMap<$core.String, CloudSpecificImage> get images => $_getMap(0);
}

class UploadCloudImageRequest extends $pb.GeneratedMessage {
  factory UploadCloudImageRequest({
    $core.String? imagePath,
    $core.String? imageName,
    $core.String? cloudName,
    $core.String? cloudLocation,
    $core.int? timeout,
  }) {
    final result = create();
    if (imagePath != null) result.imagePath = imagePath;
    if (imageName != null) result.imageName = imageName;
    if (cloudName != null) result.cloudName = cloudName;
    if (cloudLocation != null) result.cloudLocation = cloudLocation;
    if (timeout != null) result.timeout = timeout;
    return result;
  }

  UploadCloudImageRequest._();

  factory UploadCloudImageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadCloudImageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadCloudImageRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'imagePath')
    ..aOS(2, _omitFieldNames ? '' : 'imageName')
    ..aOS(3, _omitFieldNames ? '' : 'cloudName')
    ..aOS(4, _omitFieldNames ? '' : 'cloudLocation')
    ..aI(5, _omitFieldNames ? '' : 'timeout')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadCloudImageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadCloudImageRequest copyWith(
          void Function(UploadCloudImageRequest) updates) =>
      super.copyWith((message) => updates(message as UploadCloudImageRequest))
          as UploadCloudImageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadCloudImageRequest create() => UploadCloudImageRequest._();
  @$core.override
  UploadCloudImageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadCloudImageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadCloudImageRequest>(create);
  static UploadCloudImageRequest? _defaultInstance;

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
  $core.String get cloudName => $_getSZ(2);
  @$pb.TagNumber(3)
  set cloudName($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCloudName() => $_has(2);
  @$pb.TagNumber(3)
  void clearCloudName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get cloudLocation => $_getSZ(3);
  @$pb.TagNumber(4)
  set cloudLocation($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCloudLocation() => $_has(3);
  @$pb.TagNumber(4)
  void clearCloudLocation() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get timeout => $_getIZ(4);
  @$pb.TagNumber(5)
  set timeout($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTimeout() => $_has(4);
  @$pb.TagNumber(5)
  void clearTimeout() => $_clearField(5);
}

class UploadCloudImageResponse extends $pb.GeneratedMessage {
  factory UploadCloudImageResponse() => create();

  UploadCloudImageResponse._();

  factory UploadCloudImageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UploadCloudImageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UploadCloudImageResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadCloudImageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UploadCloudImageResponse copyWith(
          void Function(UploadCloudImageResponse) updates) =>
      super.copyWith((message) => updates(message as UploadCloudImageResponse))
          as UploadCloudImageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UploadCloudImageResponse create() => UploadCloudImageResponse._();
  @$core.override
  UploadCloudImageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UploadCloudImageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UploadCloudImageResponse>(create);
  static UploadCloudImageResponse? _defaultInstance;
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
  factory UploadProvisionerImageResponse() => create();

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
}

class RemoveCloudImageRequest extends $pb.GeneratedMessage {
  factory RemoveCloudImageRequest({
    $core.String? imageName,
    $core.String? cloudName,
    $core.String? cloudLocation,
  }) {
    final result = create();
    if (imageName != null) result.imageName = imageName;
    if (cloudName != null) result.cloudName = cloudName;
    if (cloudLocation != null) result.cloudLocation = cloudLocation;
    return result;
  }

  RemoveCloudImageRequest._();

  factory RemoveCloudImageRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveCloudImageRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveCloudImageRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..aOS(2, _omitFieldNames ? '' : 'imageName')
    ..aOS(3, _omitFieldNames ? '' : 'cloudName')
    ..aOS(4, _omitFieldNames ? '' : 'cloudLocation')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudImageRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudImageRequest copyWith(
          void Function(RemoveCloudImageRequest) updates) =>
      super.copyWith((message) => updates(message as RemoveCloudImageRequest))
          as RemoveCloudImageRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveCloudImageRequest create() => RemoveCloudImageRequest._();
  @$core.override
  RemoveCloudImageRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveCloudImageRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveCloudImageRequest>(create);
  static RemoveCloudImageRequest? _defaultInstance;

  @$pb.TagNumber(2)
  $core.String get imageName => $_getSZ(0);
  @$pb.TagNumber(2)
  set imageName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(2)
  $core.bool hasImageName() => $_has(0);
  @$pb.TagNumber(2)
  void clearImageName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get cloudName => $_getSZ(1);
  @$pb.TagNumber(3)
  set cloudName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(3)
  $core.bool hasCloudName() => $_has(1);
  @$pb.TagNumber(3)
  void clearCloudName() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get cloudLocation => $_getSZ(2);
  @$pb.TagNumber(4)
  set cloudLocation($core.String value) => $_setString(2, value);
  @$pb.TagNumber(4)
  $core.bool hasCloudLocation() => $_has(2);
  @$pb.TagNumber(4)
  void clearCloudLocation() => $_clearField(4);
}

class RemoveCloudImageResponse extends $pb.GeneratedMessage {
  factory RemoveCloudImageResponse() => create();

  RemoveCloudImageResponse._();

  factory RemoveCloudImageResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveCloudImageResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveCloudImageResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'apic'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudImageResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveCloudImageResponse copyWith(
          void Function(RemoveCloudImageResponse) updates) =>
      super.copyWith((message) => updates(message as RemoveCloudImageResponse))
          as RemoveCloudImageResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveCloudImageResponse create() => RemoveCloudImageResponse._();
  @$core.override
  RemoveCloudImageResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveCloudImageResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveCloudImageResponse>(create);
  static RemoveCloudImageResponse? _defaultInstance;
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
  }) {
    final result = create();
    if (coreStatus != null) result.coreStatus = coreStatus;
    if (workDir != null) result.workDir = workDir;
    if (capabilities != null) result.capabilities = capabilities;
    if (p2pPort != null) result.p2pPort = p2pPort;
    if (endpoints != null) result.endpoints.addAll(endpoints);
    if (hostAgent != null) result.hostAgent = hostAgent;
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

class Commit extends $pb.GeneratedMessage {
  factory Commit({
    $core.String? hash,
    $core.String? committer,
    $core.String? message,
    $core.Iterable<$core.String>? states,
    $fixnum.Int64? dateUnix,
  }) {
    final result = create();
    if (hash != null) result.hash = hash;
    if (committer != null) result.committer = committer;
    if (message != null) result.message = message;
    if (states != null) result.states.addAll(states);
    if (dateUnix != null) result.dateUnix = dateUnix;
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
  }) {
    final result = create();
    if (commits != null) result.commits.addAll(commits);
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
  }) {
    final result = create();
    if (commits != null) result.commits.addAll(commits);
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
  $async.Future<GetSupportedCloudProvidersResponse> getSupportedCloudProviders(
          $pb.ClientContext? ctx, GetSupportedCloudProvidersRequest request) =>
      _client.invoke<GetSupportedCloudProvidersResponse>(
          ctx,
          'ProtosClientApi',
          'GetSupportedCloudProviders',
          request,
          GetSupportedCloudProvidersResponse());
  $async.Future<GetCloudProvidersResponse> getCloudProviders(
          $pb.ClientContext? ctx, GetCloudProvidersRequest request) =>
      _client.invoke<GetCloudProvidersResponse>(ctx, 'ProtosClientApi',
          'GetCloudProviders', request, GetCloudProvidersResponse());
  $async.Future<GetCloudProviderResponse> getCloudProvider(
          $pb.ClientContext? ctx, GetCloudProviderRequest request) =>
      _client.invoke<GetCloudProviderResponse>(ctx, 'ProtosClientApi',
          'GetCloudProvider', request, GetCloudProviderResponse());
  $async.Future<AddCloudProviderResponse> addCloudProvider(
          $pb.ClientContext? ctx, AddCloudProviderRequest request) =>
      _client.invoke<AddCloudProviderResponse>(ctx, 'ProtosClientApi',
          'AddCloudProvider', request, AddCloudProviderResponse());
  $async.Future<RemoveCloudProviderResponse> removeCloudProvider(
          $pb.ClientContext? ctx, RemoveCloudProviderRequest request) =>
      _client.invoke<RemoveCloudProviderResponse>(ctx, 'ProtosClientApi',
          'RemoveCloudProvider', request, RemoveCloudProviderResponse());
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
  $async.Future<GetCloudImagesResponse> getCloudImages(
          $pb.ClientContext? ctx, GetCloudImagesRequest request) =>
      _client.invoke<GetCloudImagesResponse>(ctx, 'ProtosClientApi',
          'GetCloudImages', request, GetCloudImagesResponse());
  $async.Future<UploadCloudImageResponse> uploadCloudImage(
          $pb.ClientContext? ctx, UploadCloudImageRequest request) =>
      _client.invoke<UploadCloudImageResponse>(ctx, 'ProtosClientApi',
          'UploadCloudImage', request, UploadCloudImageResponse());
  $async.Future<RemoveCloudImageResponse> removeCloudImage(
          $pb.ClientContext? ctx, RemoveCloudImageRequest request) =>
      _client.invoke<RemoveCloudImageResponse>(ctx, 'ProtosClientApi',
          'RemoveCloudImage', request, RemoveCloudImageResponse());
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
  $async.Future<GetSystemStatusResponse> getSystemStatus(
          $pb.ClientContext? ctx, GetSystemStatusRequest request) =>
      _client.invoke<GetSystemStatusResponse>(ctx, 'ProtosClientApi',
          'GetSystemStatus', request, GetSystemStatusResponse());
  $async.Future<GetLocalCommitsResponse> getLocalCommits(
          $pb.ClientContext? ctx, GetLocalCommitsRequest request) =>
      _client.invoke<GetLocalCommitsResponse>(ctx, 'ProtosClientApi',
          'GetLocalCommits', request, GetLocalCommitsResponse());
  $async.Future<GetRemoteCommitsResponse> getRemoteCommits(
          $pb.ClientContext? ctx, GetRemoteCommitsRequest request) =>
      _client.invoke<GetRemoteCommitsResponse>(ctx, 'ProtosClientApi',
          'GetRemoteCommits', request, GetRemoteCommitsResponse());
  $async.Future<ExecuteSqlResponse> executeSql(
          $pb.ClientContext? ctx, ExecuteSqlRequest request) =>
      _client.invoke<ExecuteSqlResponse>(
          ctx, 'ProtosClientApi', 'ExecuteSql', request, ExecuteSqlResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
