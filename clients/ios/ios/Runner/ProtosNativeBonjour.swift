import Darwin
import Foundation

@_cdecl("ProtosNativeBonjourStart")
public func ProtosNativeBonjourStart(
  _ instancePointer: UnsafePointer<CChar>?,
  _ servicePointer: UnsafePointer<CChar>?,
  _ domainPointer: UnsafePointer<CChar>?,
  _ port: Int32,
  _ txtJSONPointer: UnsafePointer<CChar>?
) -> UnsafeMutablePointer<CChar>? {
  do {
    let instance = try stringArgument(instancePointer, "instance")
    let service = try stringArgument(servicePointer, "service")
    let domain = try stringArgument(domainPointer, "domain")
    let txtJSON = try stringArgument(txtJSONPointer, "txt_json")
    let txtRecords = try decodeTXTRecords(txtJSON)

    var startError: Error?
    let start = {
      do {
        try ProtosNativeBonjour.shared.startAdvertisement(
          instance: instance,
          service: service,
          domain: domain,
          port: Int(port),
          txtRecords: txtRecords
        )
      } catch {
        startError = error
      }
    }
    if Thread.isMainThread {
      start()
    } else {
      DispatchQueue.main.sync(execute: start)
    }
    if let startError {
      return makeCString(startError.localizedDescription)
    }
    return nil
  } catch {
    return makeCString(error.localizedDescription)
  }
}

@_cdecl("ProtosNativeBonjourBrowse")
public func ProtosNativeBonjourBrowse(
  _ servicePointer: UnsafePointer<CChar>?,
  _ domainPointer: UnsafePointer<CChar>?,
  _ timeoutMs: Int32,
  _ outJSON: UnsafeMutablePointer<UnsafeMutablePointer<CChar>?>?
) -> UnsafeMutablePointer<CChar>? {
  do {
    guard !Thread.isMainThread else {
      return makeCString("native Bonjour browse cannot run on the main thread")
    }
    let service = try stringArgument(servicePointer, "service")
    let domain = try stringArgument(domainPointer, "domain")
    let timeout = max(Int(timeoutMs), 250)

    let semaphore = DispatchSemaphore(value: 0)
    var browseResult: Result<[[String: Any]], Error>?
    DispatchQueue.main.async {
      ProtosNativeBonjour.shared.browse(
        service: service,
        domain: domain,
        timeout: TimeInterval(timeout) / 1000.0
      ) { result in
        browseResult = result
        semaphore.signal()
      }
    }

    let waitResult = semaphore.wait(timeout: .now() + .milliseconds(timeout + 1000))
    guard waitResult == .success, let browseResult else {
      return makeCString("native Bonjour browse timed out")
    }
    switch browseResult {
    case .success(let entries):
      let data = try JSONSerialization.data(withJSONObject: entries, options: [])
      let json = String(data: data, encoding: .utf8) ?? "[]"
      outJSON?.pointee = makeCString(json)
      return nil
    case .failure(let error):
      return makeCString(error.localizedDescription)
    }
  } catch {
    return makeCString(error.localizedDescription)
  }
}

@_cdecl("ProtosNativeBonjourStop")
public func ProtosNativeBonjourStop() {
  let stop = {
    ProtosNativeBonjour.shared.stopAdvertisement()
  }
  if Thread.isMainThread {
    stop()
  } else {
    DispatchQueue.main.sync(execute: stop)
  }
}

private final class ProtosNativeBonjour {
  static let shared = ProtosNativeBonjour()

  private var advertisement: NetService?
  private var browserSessions: [ObjectIdentifier: ProtosBonjourBrowserSession] = [:]

  func startAdvertisement(
    instance: String,
    service: String,
    domain: String,
    port: Int,
    txtRecords: [String]
  ) throws {
    stopAdvertisement()
    guard port > 0 && port <= 65535 else {
      throw bonjourError("Bonjour advertisement port is invalid.")
    }
    let txtData = NetService.data(fromTXTRecord: Self.txtDictionary(from: txtRecords))

    let netService = NetService(
      domain: Self.normalizedDomain(domain),
      type: Self.normalizedServiceType(service),
      name: instance,
      port: Int32(port)
    )
    netService.includesPeerToPeer = true
    netService.setTXTRecord(txtData)
    advertisement = netService
    netService.publish()
  }

  func stopAdvertisement() {
    advertisement?.stop()
    advertisement = nil
  }

  func browse(
    service: String,
    domain: String,
    timeout: TimeInterval,
    completion: @escaping (Result<[[String: Any]], Error>) -> Void
  ) {
    let session = ProtosBonjourBrowserSession(
      service: Self.normalizedServiceType(service),
      domain: Self.normalizedDomain(domain),
      timeout: timeout
    ) { [weak self] session, result in
      self?.browserSessions.removeValue(forKey: ObjectIdentifier(session))
      completion(result)
    }
    browserSessions[ObjectIdentifier(session)] = session
    session.start()
  }

  static func txtDictionary(from records: [String]) -> [String: Data] {
    var out: [String: Data] = [:]
    for record in records {
      let parts = record.split(separator: "=", maxSplits: 1, omittingEmptySubsequences: false)
      guard parts.count == 2, !parts[0].isEmpty else {
        continue
      }
      out[String(parts[0])] = Data(String(parts[1]).utf8)
    }
    return out
  }

  static func txtRecords(from data: Data?) -> [String] {
    guard let data else {
      return []
    }
    let dictionary = NetService.dictionary(fromTXTRecord: data)
    return dictionary.keys.sorted().compactMap { key in
      guard let valueData = dictionary[key],
            let value = String(data: valueData, encoding: .utf8),
            !key.isEmpty else {
        return nil
      }
      return "\(key)=\(value)"
    }
  }

  static func ipAddresses(from service: NetService) -> [String] {
    var out: [String] = []
    for addressData in service.addresses ?? [] {
      addressData.withUnsafeBytes { rawBuffer in
        guard let baseAddress = rawBuffer.baseAddress else {
          return
        }
        let family = baseAddress.assumingMemoryBound(to: sockaddr.self).pointee.sa_family
        if Int32(family) == AF_INET {
          var address = baseAddress.assumingMemoryBound(to: sockaddr_in.self).pointee.sin_addr
          var buffer = [CChar](repeating: 0, count: Int(INET_ADDRSTRLEN))
          if inet_ntop(AF_INET, &address, &buffer, socklen_t(INET_ADDRSTRLEN)) != nil {
            out.append(String(cString: buffer))
          }
        } else if Int32(family) == AF_INET6 {
          var address = baseAddress.assumingMemoryBound(to: sockaddr_in6.self).pointee.sin6_addr
          var buffer = [CChar](repeating: 0, count: Int(INET6_ADDRSTRLEN))
          if inet_ntop(AF_INET6, &address, &buffer, socklen_t(INET6_ADDRSTRLEN)) != nil {
            out.append(String(cString: buffer))
          }
        }
      }
    }
    return Array(Set(out)).sorted()
  }

  static func normalizedServiceType(_ value: String) -> String {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    return trimmed.hasSuffix(".") ? trimmed : "\(trimmed)."
  }

  static func normalizedDomain(_ value: String) -> String {
    let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
    return trimmed.hasSuffix(".") ? trimmed : "\(trimmed)."
  }
}

private final class ProtosBonjourBrowserSession: NSObject, NetServiceBrowserDelegate, NetServiceDelegate {
  private let browser = NetServiceBrowser()
  private let service: String
  private let domain: String
  private let timeout: TimeInterval
  private let completion: (ProtosBonjourBrowserSession, Result<[[String: Any]], Error>) -> Void
  private var services: [NetService] = []
  private var entries: [String: [String: Any]] = [:]
  private var finished = false

  init(
    service: String,
    domain: String,
    timeout: TimeInterval,
    completion: @escaping (ProtosBonjourBrowserSession, Result<[[String: Any]], Error>) -> Void
  ) {
    self.service = service
    self.domain = domain
    self.timeout = timeout
    self.completion = completion
  }

  func start() {
    browser.delegate = self
    browser.includesPeerToPeer = true
    browser.searchForServices(ofType: service, inDomain: domain)
    DispatchQueue.main.asyncAfter(deadline: .now() + timeout) { [weak self] in
      guard let self else {
        return
      }
      self.finish(.success(Array(self.entries.values)))
    }
  }

  func netServiceBrowserWillSearch(_ browser: NetServiceBrowser) {}

  func netServiceBrowserDidStopSearch(_ browser: NetServiceBrowser) {
    finish(.success(Array(entries.values)))
  }

  func netServiceBrowser(
    _ browser: NetServiceBrowser,
    didNotSearch errorDict: [String: NSNumber]
  ) {
    finish(.failure(bonjourError("Bonjour browse failed: \(errorDict).")))
  }

  func netServiceBrowser(
    _ browser: NetServiceBrowser,
    didFind service: NetService,
    moreComing: Bool
  ) {
    services.append(service)
    service.delegate = self
    service.includesPeerToPeer = true
    service.resolve(withTimeout: max(0.25, min(timeout, 3.0)))
  }

  func netServiceDidResolveAddress(_ sender: NetService) {
    record(sender)
  }

  func netService(
    _ sender: NetService,
    didNotResolve errorDict: [String: NSNumber]
  ) {}

  private func record(_ service: NetService) {
    let text = ProtosNativeBonjour.txtRecords(from: service.txtRecordData())
    guard !text.isEmpty else {
      return
    }
    let hostName = service.hostName ?? ""
    let key = "\(service.name)|\(service.type)|\(service.domain)|\(hostName)|\(service.port)"
    entries[key] = [
      "name": service.name,
      "hostName": hostName,
      "port": service.port,
      "ips": ProtosNativeBonjour.ipAddresses(from: service),
      "text": text,
    ]
  }

  private func finish(_ result: Result<[[String: Any]], Error>) {
    guard !finished else {
      return
    }
    finished = true
    browser.stop()
    browser.delegate = nil
    for service in services {
      service.stop()
      service.delegate = nil
    }
    completion(self, result)
  }
}

private func stringArgument(_ pointer: UnsafePointer<CChar>?, _ name: String) throws -> String {
  guard let pointer else {
    throw bonjourError("Bonjour \(name) is required.")
  }
  return String(cString: pointer)
}

private func decodeTXTRecords(_ json: String) throws -> [String] {
  guard let data = json.data(using: .utf8),
        let records = try JSONSerialization.jsonObject(with: data) as? [String] else {
    throw bonjourError("Bonjour TXT JSON is invalid.")
  }
  return records
}

private func makeCString(_ value: String) -> UnsafeMutablePointer<CChar>? {
  strdup(value)
}

private func bonjourError(_ message: String) -> NSError {
  NSError(domain: "io.protos.bonjour", code: 1, userInfo: [NSLocalizedDescriptionKey: message])
}
