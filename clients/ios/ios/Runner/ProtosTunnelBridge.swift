import Flutter
import Foundation
import NetworkExtension
import Security

final class ProtosTunnelBridge: NSObject {
  private static let shared = ProtosTunnelBridge()
  private static let channelName = "io.protos/tunnel"

  private var channel: FlutterMethodChannel?
  private let tunnelDescription = "Protos Tunnel"
  private let keychainService = "io.protos.mobile-tunnel"

  static func register(with messenger: FlutterBinaryMessenger) {
    let channel = FlutterMethodChannel(name: channelName, binaryMessenger: messenger)
    shared.channel = channel
    channel.setMethodCallHandler(shared.handle)
  }

  private func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "installOrUpdateTunnel":
      guard let config = call.arguments as? [String: Any] else {
        result(error("bad-arguments", "installOrUpdateTunnel expects a config map."))
        return
      }
      installOrUpdateTunnel(config: config, result: result)
    case "startTunnel":
      startTunnel(result: result)
    case "stopTunnel":
      stopTunnel(result: result)
    case "tunnelStatus":
      tunnelStatus(result: result)
    case "sendProviderMessage":
      guard let message = call.arguments as? [String: Any] else {
        result(error("bad-arguments", "sendProviderMessage expects a message map."))
        return
      }
      sendProviderMessage(message: message, result: result)
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private var providerBundleIdentifier: String {
    let appBundle = Bundle.main.bundleIdentifier ?? "io.protos.protosFlutter"
    return "\(appBundle).PacketTunnel"
  }

  private func installOrUpdateTunnel(config: [String: Any], result: @escaping FlutterResult) {
    var providerConfig = config
    let keychainAccount = stringValue(providerConfig["keychainAccount"])
      ?? "protos.mobile-tunnel.wireguard-private-key"
    let accessGroup = keychainAccessGroup(config: providerConfig)

    if let privateKey = stringValue(providerConfig["wireguardPrivateKey"]), !privateKey.isEmpty {
      do {
        try ProtosKeychain.store(
          privateKey,
          service: keychainService,
          account: keychainAccount,
          accessGroup: accessGroup
        )
      } catch {
        result(self.error("keychain-write-failed", error.localizedDescription))
        return
      }
    }

    providerConfig["wireguardPrivateKey"] = nil
    providerConfig["keychainAccount"] = keychainAccount
    providerConfig["keychainAccessGroup"] = accessGroup ?? ""
    providerConfig["installedAtUnix"] = Int(Date().timeIntervalSince1970)

    loadOrCreateManager { manager, loadError in
      if let loadError {
        result(self.error("load-vpn-profile-failed", loadError.localizedDescription))
        return
      }

      let proto = NETunnelProviderProtocol()
      proto.providerBundleIdentifier = self.providerBundleIdentifier
      proto.providerConfiguration = providerConfig
      proto.serverAddress = self.stringValue(providerConfig["instanceName"]) ?? "Protos"

      manager.localizedDescription = self.tunnelDescription
      manager.protocolConfiguration = proto
      manager.isEnabled = true
      manager.saveToPreferences { saveError in
        DispatchQueue.main.async {
          if let saveError {
            result(self.error("save-vpn-profile-failed", saveError.localizedDescription))
          } else {
            result(nil)
          }
        }
      }
    }
  }

  private func startTunnel(result: @escaping FlutterResult) {
    loadExistingManager { manager, loadError in
      if let loadError {
        result(self.error("load-vpn-profile-failed", loadError.localizedDescription))
        return
      }
      guard let manager else {
        result(self.error("vpn-profile-missing", "The Protos tunnel profile is not installed."))
        return
      }

      manager.loadFromPreferences { reloadError in
        DispatchQueue.main.async {
          if let reloadError {
            result(self.error("load-vpn-profile-failed", reloadError.localizedDescription))
            return
          }
          do {
            try manager.connection.startVPNTunnel()
            result(nil)
          } catch {
            result(self.error("start-vpn-failed", error.localizedDescription))
          }
        }
      }
    }
  }

  private func stopTunnel(result: @escaping FlutterResult) {
    loadExistingManager { manager, loadError in
      if let loadError {
        result(self.error("load-vpn-profile-failed", loadError.localizedDescription))
        return
      }
      guard let manager else {
        result(nil)
        return
      }
      manager.connection.stopVPNTunnel()
      result(nil)
    }
  }

  private func tunnelStatus(result: @escaping FlutterResult) {
    loadExistingManager { manager, loadError in
      if let loadError {
        result([
          "supported": true,
          "installed": false,
          "status": "error",
          "detail": loadError.localizedDescription,
          "configId": "",
        ])
        return
      }
      guard let manager else {
        result([
          "supported": true,
          "installed": false,
          "status": "not-installed",
          "detail": "",
          "configId": "",
        ])
        return
      }
      let proto = manager.protocolConfiguration as? NETunnelProviderProtocol
      let config = proto?.providerConfiguration
      result([
        "supported": true,
        "installed": true,
        "status": self.statusName(manager.connection.status),
        "detail": "",
        "configId": self.stringValue(config?["configId"]) ?? "",
      ])
    }
  }

  private func sendProviderMessage(message: [String: Any], result: @escaping FlutterResult) {
    loadExistingManager { manager, loadError in
      if let loadError {
        result(self.error("load-vpn-profile-failed", loadError.localizedDescription))
        return
      }
      guard let session = manager?.connection as? NETunnelProviderSession else {
        result(self.error("vpn-profile-missing", "The Protos tunnel profile is not installed."))
        return
      }
      guard JSONSerialization.isValidJSONObject(message),
            let data = try? JSONSerialization.data(withJSONObject: message) else {
        result(self.error("bad-arguments", "Provider message must be JSON-compatible."))
        return
      }

      do {
        try session.sendProviderMessage(data) { responseData in
          DispatchQueue.main.async {
            guard let responseData, !responseData.isEmpty else {
              result([:])
              return
            }
            let response = (try? JSONSerialization.jsonObject(with: responseData)) as? [String: Any]
            result(response ?? [:])
          }
        }
      } catch {
        result(self.error("provider-message-failed", error.localizedDescription))
      }
    }
  }

  private func loadOrCreateManager(
    completion: @escaping (NETunnelProviderManager, Error?) -> Void
  ) {
    NETunnelProviderManager.loadAllFromPreferences { managers, error in
      DispatchQueue.main.async {
        if let error {
          completion(NETunnelProviderManager(), error)
          return
        }
        let existing = managers?.first { manager in
          let proto = manager.protocolConfiguration as? NETunnelProviderProtocol
          return proto?.providerBundleIdentifier == self.providerBundleIdentifier
        }
        completion(existing ?? NETunnelProviderManager(), nil)
      }
    }
  }

  private func loadExistingManager(
    completion: @escaping (NETunnelProviderManager?, Error?) -> Void
  ) {
    NETunnelProviderManager.loadAllFromPreferences { managers, error in
      DispatchQueue.main.async {
        if let error {
          completion(nil, error)
          return
        }
        let existing = managers?.first { manager in
          let proto = manager.protocolConfiguration as? NETunnelProviderProtocol
          return proto?.providerBundleIdentifier == self.providerBundleIdentifier
        }
        completion(existing, nil)
      }
    }
  }

  private func keychainAccessGroup(config: [String: Any]) -> String? {
    if let value = stringValue(config["keychainAccessGroup"]), !value.isEmpty {
      return value
    }
    if let value = Bundle.main.object(forInfoDictionaryKey: "ProtosKeychainAccessGroup") as? String,
       !value.isEmpty,
       !value.contains("$(") {
      return value
    }
    return nil
  }

  private func statusName(_ status: NEVPNStatus) -> String {
    switch status {
    case .invalid:
      return "invalid"
    case .disconnected:
      return "disconnected"
    case .connecting:
      return "connecting"
    case .connected:
      return "connected"
    case .reasserting:
      return "reasserting"
    case .disconnecting:
      return "disconnecting"
    @unknown default:
      return "unknown"
    }
  }

  private func stringValue(_ value: Any?) -> String? {
    if let value = value as? String {
      return value.trimmingCharacters(in: .whitespacesAndNewlines)
    }
    return nil
  }

  private func error(_ code: String, _ message: String) -> FlutterError {
    FlutterError(code: code, message: message, details: nil)
  }
}

private enum ProtosKeychain {
  static func store(
    _ value: String,
    service: String,
    account: String,
    accessGroup: String?
  ) throws {
    guard let data = value.data(using: .utf8) else {
      throw KeychainError.invalidValue
    }

    var query: [String: Any] = [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: service,
      kSecAttrAccount as String: account,
    ]
    if let accessGroup, !accessGroup.isEmpty {
      query[kSecAttrAccessGroup as String] = accessGroup
    }

    SecItemDelete(query as CFDictionary)

    query[kSecValueData as String] = data
    query[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly

    let status = SecItemAdd(query as CFDictionary, nil)
    guard status == errSecSuccess else {
      throw KeychainError.securityStatus(status)
    }
  }
}

private enum KeychainError: LocalizedError {
  case invalidValue
  case securityStatus(OSStatus)

  var errorDescription: String? {
    switch self {
    case .invalidValue:
      return "The keychain value could not be encoded."
    case .securityStatus(let status):
      return "Keychain operation failed with status \(status)."
    }
  }
}
