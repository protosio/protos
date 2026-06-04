import Foundation
import NetworkExtension
import Security

final class PacketTunnelProvider: NEPacketTunnelProvider {
  private let keychainService = "io.protos.mobile-tunnel"
  private var lastConfig: [String: Any] = [:]
  private var lastStatus = "idle"
  private var lastDetail = ""

  override func startTunnel(
    options: [String: NSObject]?,
    completionHandler: @escaping (Error?) -> Void
  ) {
    do {
      guard let tunnelProtocol = protocolConfiguration as? NETunnelProviderProtocol,
            let providerConfig = tunnelProtocol.providerConfiguration else {
        throw PacketTunnelError.missingProviderConfig
      }
      lastConfig = providerConfig

      let keychainAccount = stringValue(providerConfig["keychainAccount"])
        ?? "protos.mobile-tunnel.wireguard-private-key"
      let accessGroup = keychainAccessGroup(config: providerConfig)
      let privateKey = try ProtosTunnelKeychain.load(
        service: keychainService,
        account: keychainAccount,
        accessGroup: accessGroup
      )
      guard !privateKey.isEmpty else {
        throw PacketTunnelError.missingPrivateKey
      }

      let settings = try makeNetworkSettings(from: providerConfig)
      setTunnelNetworkSettings(settings) { [weak self] error in
        if let error {
          self?.recordStatus("failed", detail: error.localizedDescription)
          completionHandler(error)
          return
        }

        self?.recordStatus(
          "engine-unavailable",
          detail: "WireGuardKit is not linked into the packet tunnel target yet."
        )
        completionHandler(PacketTunnelError.engineUnavailable)
      }
    } catch {
      recordStatus("failed", detail: error.localizedDescription)
      completionHandler(error)
    }
  }

  override func stopTunnel(
    with reason: NEProviderStopReason,
    completionHandler: @escaping () -> Void
  ) {
    recordStatus("stopped", detail: "\(reason.rawValue)")
    completionHandler()
  }

  override func handleAppMessage(
    _ messageData: Data,
    completionHandler: ((Data?) -> Void)?
  ) {
    let response: [String: Any] = [
      "status": lastStatus,
      "detail": lastDetail,
      "configId": stringValue(lastConfig["configId"]) ?? "",
      "includedRoutes": stringArray(lastConfig["includedRoutes"]),
      "excludedRoutes": stringArray(lastConfig["excludedRoutes"]),
    ]
    completionHandler?(try? JSONSerialization.data(withJSONObject: response))
  }

  private func makeNetworkSettings(from config: [String: Any]) throws -> NEPacketTunnelNetworkSettings {
    let remoteAddress = endpointHost(stringValue(config["peerEndpoint"])) ?? "protos"
    let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: remoteAddress)
    if let mtu = config["mtu"] as? Int, mtu > 0 {
      settings.mtu = NSNumber(value: mtu)
    }

    var ipv4Addresses: [String] = []
    var ipv4Masks: [String] = []
    var ipv6Addresses: [String] = []
    var ipv6Prefixes: [NSNumber] = []

    for value in stringArray(config["interfaceAddresses"]) {
      let prefix = try IPPrefix(value)
      if prefix.isIPv4 {
        ipv4Addresses.append(prefix.address)
        ipv4Masks.append(ipv4Mask(prefix.bits))
      } else {
        ipv6Addresses.append(prefix.address)
        ipv6Prefixes.append(NSNumber(value: prefix.bits))
      }
    }

    let includedRoutes = try stringArray(config["includedRoutes"]).map(IPPrefix.init)
    let excludedRoutes = try stringArray(config["excludedRoutes"]).map(IPPrefix.init)

    if !ipv4Addresses.isEmpty {
      let ipv4Settings = NEIPv4Settings(addresses: ipv4Addresses, subnetMasks: ipv4Masks)
      let included = includedRoutes.filter(\.isIPv4).map(ipv4Route)
      let excluded = excludedRoutes.filter(\.isIPv4).map(ipv4Route)
      if !included.isEmpty {
        ipv4Settings.includedRoutes = included
      }
      if !excluded.isEmpty {
        ipv4Settings.excludedRoutes = excluded
      }
      settings.ipv4Settings = ipv4Settings
    }

    if !ipv6Addresses.isEmpty {
      let ipv6Settings = NEIPv6Settings(
        addresses: ipv6Addresses,
        networkPrefixLengths: ipv6Prefixes
      )
      let included = includedRoutes.filter { !$0.isIPv4 }.map(ipv6Route)
      let excluded = excludedRoutes.filter { !$0.isIPv4 }.map(ipv6Route)
      if !included.isEmpty {
        ipv6Settings.includedRoutes = included
      }
      if !excluded.isEmpty {
        ipv6Settings.excludedRoutes = excluded
      }
      settings.ipv6Settings = ipv6Settings
    }

    let dnsServers = stringArray(config["dnsServers"])
    if !dnsServers.isEmpty {
      settings.dnsSettings = NEDNSSettings(servers: dnsServers)
    }

    return settings
  }

  private func ipv4Route(_ prefix: IPPrefix) -> NEIPv4Route {
    if prefix.bits == 0 {
      return NEIPv4Route.default()
    }
    return NEIPv4Route(destinationAddress: prefix.address, subnetMask: ipv4Mask(prefix.bits))
  }

  private func ipv6Route(_ prefix: IPPrefix) -> NEIPv6Route {
    NEIPv6Route(
      destinationAddress: prefix.address,
      networkPrefixLength: NSNumber(value: prefix.bits)
    )
  }

  private func endpointHost(_ endpoint: String?) -> String? {
    guard let endpoint, !endpoint.isEmpty else {
      return nil
    }
    if let parsed = URLComponents(string: "udp://\(endpoint)")?.host {
      return parsed
    }
    return endpoint.components(separatedBy: ":").first
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

  private func recordStatus(_ status: String, detail: String) {
    lastStatus = status
    lastDetail = detail
    guard let group = Bundle.main.object(forInfoDictionaryKey: "ProtosAppGroup") as? String,
          let defaults = UserDefaults(suiteName: group) else {
      return
    }
    defaults.set(status, forKey: "mobileTunnelStatus")
    defaults.set(detail, forKey: "mobileTunnelStatusDetail")
    defaults.set(stringValue(lastConfig["configId"]) ?? "", forKey: "mobileTunnelConfigId")
  }
}

private struct IPPrefix {
  let address: String
  let bits: Int
  let isIPv4: Bool

  init(_ value: String) throws {
    let parts = value.split(separator: "/", maxSplits: 1).map(String.init)
    guard let address = parts.first, !address.isEmpty else {
      throw PacketTunnelError.invalidCIDR(value)
    }
    let isIPv4 = address.contains(".")
    let defaultBits = isIPv4 ? 32 : 128
    let bits = parts.count == 2 ? Int(parts[1]) ?? -1 : defaultBits
    guard bits >= 0 && bits <= defaultBits else {
      throw PacketTunnelError.invalidCIDR(value)
    }
    self.address = address
    self.bits = bits
    self.isIPv4 = isIPv4
  }
}

private enum ProtosTunnelKeychain {
  static func load(service: String, account: String, accessGroup: String?) throws -> String {
    var query: [String: Any] = [
      kSecClass as String: kSecClassGenericPassword,
      kSecAttrService as String: service,
      kSecAttrAccount as String: account,
      kSecReturnData as String: true,
      kSecMatchLimit as String: kSecMatchLimitOne,
    ]
    if let accessGroup, !accessGroup.isEmpty {
      query[kSecAttrAccessGroup as String] = accessGroup
    }

    var result: CFTypeRef?
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    guard status == errSecSuccess else {
      throw PacketTunnelError.keychainStatus(status)
    }
    guard let data = result as? Data, let value = String(data: data, encoding: .utf8) else {
      throw PacketTunnelError.missingPrivateKey
    }
    return value
  }
}

private enum PacketTunnelError: LocalizedError {
  case missingProviderConfig
  case missingPrivateKey
  case invalidCIDR(String)
  case keychainStatus(OSStatus)
  case engineUnavailable

  var errorDescription: String? {
    switch self {
    case .missingProviderConfig:
      return "The Protos tunnel provider configuration is missing."
    case .missingPrivateKey:
      return "The Protos tunnel private key is missing from the shared keychain."
    case .invalidCIDR(let value):
      return "Invalid route or address prefix: \(value)."
    case .keychainStatus(let status):
      return "Keychain operation failed with status \(status)."
    case .engineUnavailable:
      return "WireGuardKit is not linked into the packet tunnel target yet."
    }
  }
}

private func stringValue(_ value: Any?) -> String? {
  if let value = value as? String {
    return value.trimmingCharacters(in: .whitespacesAndNewlines)
  }
  return nil
}

private func stringArray(_ value: Any?) -> [String] {
  if let values = value as? [String] {
    return values
  }
  if let values = value as? [Any] {
    return values.compactMap { $0 as? String }
  }
  return []
}

private func ipv4Mask(_ bits: Int) -> String {
  guard bits > 0 else {
    return "0.0.0.0"
  }
  let mask = UInt32.max << UInt32(32 - bits)
  return [
    (mask >> 24) & 0xff,
    (mask >> 16) & 0xff,
    (mask >> 8) & 0xff,
    mask & 0xff,
  ].map(String.init).joined(separator: ".")
}
