const protosJoinModeAny = 'any';
const protosJoinModeNewUser = 'new_user';
const protosJoinModeNewDevice = 'new_device';

String normalizeProtosJoinMode(String value) {
  value = value.trim().toLowerCase().replaceAll('-', '_').replaceAll(' ', '_');
  switch (value) {
    case '':
    case protosJoinModeAny:
      return value;
    case 'user':
    case 'newuser':
    case protosJoinModeNewUser:
      return protosJoinModeNewUser;
    case 'device':
    case 'newdevice':
    case protosJoinModeNewDevice:
      return protosJoinModeNewDevice;
    default:
      return value;
  }
}

bool protosJoinModeMatches({
  required String inviteMode,
  required String requestedMode,
}) {
  final invite = normalizeProtosJoinMode(inviteMode);
  final requested = normalizeProtosJoinMode(requestedMode);
  return invite.isEmpty ||
      invite == protosJoinModeAny ||
      requested.isEmpty ||
      requested == protosJoinModeAny ||
      invite == requested;
}

String protosJoinModeLabel(String value) {
  switch (normalizeProtosJoinMode(value)) {
    case protosJoinModeNewUser:
      return 'New user';
    case protosJoinModeNewDevice:
      return 'New device';
    case protosJoinModeAny:
    case '':
      return 'Any';
    default:
      return value;
  }
}
