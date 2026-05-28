import CProtosBridge
import Foundation
import SwiftProtobuf

struct ProtosBridgeConfig: Codable {
    var configFile = "protos.yaml"
    var dataDir = ""
    var capabilities = "default,no-api"
    var logLevel = "info"

    enum CodingKeys: String, CodingKey {
        case configFile = "config_file"
        case dataDir = "data_dir"
        case capabilities
        case logLevel = "log_level"
    }
}

enum ProtosBridgeError: LocalizedError {
    case native(String)
    case invalidResponse

    var errorDescription: String? {
        switch self {
        case .native(let message):
            return message
        case .invalidResponse:
            return "The embedded daemon returned an invalid response."
        }
    }
}

private final class ProtosWatchBox: @unchecked Sendable {
    private let lock = NSLock()
    private var watchID: Int64 = 0
    private var cancelled = false
    private let continuation: AsyncThrowingStream<Apic_WatchChangesResponse, Error>.Continuation

    init(_ continuation: AsyncThrowingStream<Apic_WatchChangesResponse, Error>.Continuation) {
        self.continuation = continuation
    }

    func setWatchID(_ id: Int64) {
        lock.lock()
        let shouldCancel = cancelled
        if shouldCancel {
            watchID = 0
        } else {
            watchID = id
        }
        lock.unlock()
        if shouldCancel, id > 0 {
            ProtosCancelWatch(id)
        }
    }

    func cancel() {
        lock.lock()
        cancelled = true
        let id = watchID
        watchID = 0
        lock.unlock()
        if id > 0 {
            ProtosCancelWatch(id)
        }
    }

    func yield(_ result: ProtosResult) {
        guard result.len > 0, let dataPointer = result.data else {
            return
        }
        defer { ProtosFree(dataPointer) }
        let data = Data(bytes: dataPointer, count: Int(result.len))
        do {
            var response = Apic_WatchChangesResponse()
            try response.merge(serializedBytes: data)
            continuation.yield(response)
        } catch {
            continuation.finish(throwing: error)
            cancel()
        }
    }

    func finish(_ result: ProtosResult) {
        if let errorPointer = result.err {
            defer { ProtosFree(errorPointer) }
            continuation.finish(throwing: ProtosBridgeError.native(String(cString: errorPointer)))
        } else {
            continuation.finish()
        }
    }
}

private let protosWatchCallback: ProtosWatchCallback = { context, result in
    guard let context else {
        if let error = result.err {
            ProtosFree(error)
        }
        if let data = result.data {
            ProtosFree(data)
        }
        return
    }

    if result.err != nil || (result.len == 0 && result.data == nil) {
        let box = Unmanaged<ProtosWatchBox>.fromOpaque(context).takeRetainedValue()
        box.finish(result)
        return
    }

    let box = Unmanaged<ProtosWatchBox>.fromOpaque(context).takeUnretainedValue()
    box.yield(result)
}

final class NativeProtosBridge: @unchecked Sendable {
    private let lock = NSLock()
    private var started = false

    func start(config: ProtosBridgeConfig = ProtosBridgeConfig()) async throws {
        try await Task.detached {
            try self.startSync(config: config)
        }.value
    }

    func stop() {
        lock.lock()
        let shouldStop = started
        started = false
        lock.unlock()
        guard shouldStop else { return }

        if let error = ProtosStop() {
            ProtosFree(error)
        }
    }

    func call<Request: SwiftProtobuf.Message, Response: SwiftProtobuf.Message>(
        _ method: String,
        request: Request
    ) async throws -> Response {
        try await Task.detached {
            let requestData = try request.serializedData()
            let responseData = try self.callSync(method, requestData: requestData)
            var response = Response()
            try response.merge(serializedBytes: responseData)
            return response
        }.value
    }

    func watchChanges(
        request: Apic_WatchChangesRequest
    ) -> AsyncThrowingStream<Apic_WatchChangesResponse, Error> {
        AsyncThrowingStream(bufferingPolicy: .bufferingNewest(8)) { continuation in
            lock.lock()
            let isStarted = started
            lock.unlock()
            guard isStarted else {
                continuation.finish(throwing: ProtosBridgeError.native("The embedded daemon is not running."))
                return
            }

            let box = ProtosWatchBox(continuation)
            let retainedBox = Unmanaged.passRetained(box)
            let context = retainedBox.toOpaque()
            continuation.onTermination = { @Sendable [weak box] _ in
                box?.cancel()
            }

            do {
                let requestData = try request.serializedData()
                let result = requestData.withUnsafeBytes { rawBuffer in
                    ProtosWatchChanges(
                        UnsafeMutableRawPointer(mutating: rawBuffer.baseAddress),
                        Int64(requestData.count),
                        context,
                        protosWatchCallback
                    )
                }
                if let error = result.err {
                    defer { ProtosFree(error) }
                    retainedBox.release()
                    continuation.finish(throwing: ProtosBridgeError.native(String(cString: error)))
                    return
                }
                box.setWatchID(result.watch_id)
            } catch {
                retainedBox.release()
                continuation.finish(throwing: error)
            }
        }
    }

    deinit {
        stop()
    }

    private func startSync(config: ProtosBridgeConfig) throws {
        lock.lock()
        if started {
            lock.unlock()
            return
        }
        lock.unlock()

        let encoded = try JSONEncoder().encode(config)
        let json = String(decoding: encoded, as: UTF8.self)
        let error = json.withCString { pointer in
            ProtosStart(UnsafeMutablePointer(mutating: pointer))
        }
        if let error {
            defer { ProtosFree(error) }
            throw ProtosBridgeError.native(String(cString: error))
        }

        lock.lock()
        started = true
        lock.unlock()
    }

    private func callSync(_ method: String, requestData: Data) throws -> Data {
        lock.lock()
        let isStarted = started
        lock.unlock()
        guard isStarted else {
            throw ProtosBridgeError.native("The embedded daemon is not running.")
        }

        let result = method.withCString { methodPointer in
            requestData.withUnsafeBytes { rawBuffer in
                ProtosCall(
                    UnsafeMutablePointer(mutating: methodPointer),
                    UnsafeMutableRawPointer(mutating: rawBuffer.baseAddress),
                    Int64(requestData.count)
                )
            }
        }

        if let error = result.err {
            defer { ProtosFree(error) }
            throw ProtosBridgeError.native(String(cString: error))
        }
        guard result.len == 0 || result.data != nil else {
            throw ProtosBridgeError.invalidResponse
        }
        defer {
            if let data = result.data {
                ProtosFree(data)
            }
        }
        if result.len == 0 {
            return Data()
        }
        return Data(bytes: result.data!, count: Int(result.len))
    }
}

struct ProtosAPI: Sendable {
    let bridge: NativeProtosBridge

    func start() async throws {
        try await bridge.start()
    }

    func initUser(username: String, name: String, organization: String) async throws {
        var request = Apic_InitRequest()
        request.username = username
        request.name = name
        request.organization = organization
        let _: Apic_InitResponse = try await bridge.call("Init", request: request)
    }

    func userInfo() async throws -> Apic_GetUserInfoResponse {
        try await bridge.call("GetUserInfo", request: Apic_GetUserInfoRequest())
    }

    func userDevices() async throws -> Apic_GetUserDevicesResponse {
        try await bridge.call("GetUserDevices", request: Apic_GetUserDevicesRequest())
    }

    func localSSHKey() async throws -> Apic_GetLocalSSHKeyResponse {
        try await bridge.call("GetLocalSSHKey", request: Apic_GetLocalSSHKeyRequest())
    }

    func apps() async throws -> Apic_GetAppsResponse {
        try await bridge.call("GetApps", request: Apic_GetAppsRequest())
    }

    func createApp(name: String, installerID: String, instanceID: String, persistence: Bool) async throws {
        var request = Apic_CreateAppRequest()
        request.name = name
        request.installerID = installerID
        request.instanceID = instanceID
        request.persistence = persistence
        let _: Apic_CreateAppResponse = try await bridge.call("CreateApp", request: request)
    }

    func startApp(_ name: String) async throws {
        var request = Apic_StartAppRequest()
        request.name = name
        let _: Apic_StartAppResponse = try await bridge.call("StartApp", request: request)
    }

    func stopApp(_ name: String) async throws {
        var request = Apic_StopAppRequest()
        request.name = name
        let _: Apic_StopAppResponse = try await bridge.call("StopApp", request: request)
    }

    func removeApp(_ name: String) async throws {
        var request = Apic_RemoveAppRequest()
        request.name = name
        let _: Apic_RemoveAppResponse = try await bridge.call("RemoveApp", request: request)
    }

    func appLogs(_ name: String) async throws -> String {
        var request = Apic_GetAppLogsRequest()
        request.name = name
        let response: Apic_GetAppLogsResponse = try await bridge.call("GetAppLogs", request: request)
        return String(decoding: response.logs, as: UTF8.self)
    }

    func supportedProvisioners() async throws -> Apic_GetSupportedProvisionersResponse {
        try await bridge.call("GetSupportedProvisioners", request: Apic_GetSupportedProvisionersRequest())
    }

    func provisioners() async throws -> Apic_GetProvisionersResponse {
        try await bridge.call("GetProvisioners", request: Apic_GetProvisionersRequest())
    }

    func addProvisioner(name: String, type: String, credentials: [String: String]) async throws {
        var request = Apic_AddProvisionerRequest()
        request.name = name
        request.type = type
        request.credentials = credentials
        let _: Apic_AddProvisionerResponse = try await bridge.call("AddProvisioner", request: request)
    }

    func removeProvisioner(_ name: String) async throws {
        var request = Apic_RemoveProvisionerRequest()
        request.name = name
        let _: Apic_RemoveProvisionerResponse = try await bridge.call("RemoveProvisioner", request: request)
    }

    func instances() async throws -> Apic_GetInstancesResponse {
        try await bridge.call("GetInstances", request: Apic_GetInstancesRequest())
    }

    func networkState(instance: String = "") async throws -> Apic_GetNetworkStateResponse {
        var request = Apic_GetNetworkStateRequest()
        request.instance = instance
        return try await bridge.call("GetNetworkState", request: request)
    }

    func exitRoutes(instance: String = "") async throws -> Apic_GetExitRoutesResponse {
        var request = Apic_GetExitRoutesRequest()
        request.instance = instance
        return try await bridge.call("GetExitRoutes", request: request)
    }

    func setExitRoute(
        instance: String,
        deviceID: String = "",
        dnsServer: String = "",
        cidrs: [String]
    ) async throws -> Apic_SetExitRouteResponse {
        var request = Apic_SetExitRouteRequest()
        request.instance = instance
        request.deviceID = deviceID
        request.dnsServer = dnsServer
        request.cidrs = cidrs
        return try await bridge.call("SetExitRoute", request: request)
    }

    func clearExitRoute(deviceID: String = "") async throws {
        var request = Apic_ClearExitRouteRequest()
        request.deviceID = deviceID
        let _: Apic_ClearExitRouteResponse = try await bridge.call("ClearExitRoute", request: request)
    }

    func deployInstance(
        name: String,
        provisioner: String,
        location: String,
        machineType: String,
        version: String,
        devImage: String
    ) async throws {
        var request = Apic_DeployInstanceRequest()
        request.name = name
        request.cloudName = provisioner
        request.cloudLocation = location
        request.machineType = machineType
        request.protosVersion = version
        request.devImg = devImage
        let _: Apic_DeployInstanceResponse = try await bridge.call("DeployInstance", request: request)
    }

    func removeInstance(_ name: String, localOnly: Bool) async throws {
        var request = Apic_RemoveInstanceRequest()
        request.name = name
        request.localOnly = localOnly
        let _: Apic_RemoveInstanceResponse = try await bridge.call("RemoveInstance", request: request)
    }

    func startInstance(_ name: String) async throws {
        var request = Apic_StartInstanceRequest()
        request.name = name
        let _: Apic_StartInstanceResponse = try await bridge.call("StartInstance", request: request)
    }

    func stopInstance(_ name: String) async throws {
        var request = Apic_StopInstanceRequest()
        request.name = name
        let _: Apic_StopInstanceResponse = try await bridge.call("StopInstance", request: request)
    }

    func instanceKey(_ name: String) async throws -> String {
        var request = Apic_GetInstanceKeyRequest()
        request.name = name
        let response: Apic_GetInstanceKeyResponse = try await bridge.call("GetInstanceKey", request: request)
        return response.key
    }

    func instanceLogs(_ name: String) async throws -> String {
        var request = Apic_GetInstanceLogsRequest()
        request.name = name
        let response: Apic_GetInstanceLogsResponse = try await bridge.call("GetInstanceLogs", request: request)
        return response.logs
    }

    func initInstance(_ name: String, ip: String) async throws {
        var request = Apic_InitInstanceRequest()
        request.name = name
        request.ip = ip
        let _: Apic_InitInstanceResponse = try await bridge.call("InitInstance", request: request)
    }

    func updateInstance(id: String, ip: String) async throws {
        var request = Apic_UpdateInstanceRequest()
        request.id = id
        request.ip = ip
        let _: Apic_UpdateInstanceResponse = try await bridge.call("UpdateInstance", request: request)
    }

    func releases() async throws -> Apic_GetProtosdReleasesResponse {
        try await bridge.call("GetProtosdReleases", request: Apic_GetProtosdReleasesRequest())
    }

    func provisionerImages(_ provisioner: String) async throws -> Apic_GetProvisionerImagesResponse {
        var request = Apic_GetProvisionerImagesRequest()
        request.name = provisioner
        return try await bridge.call("GetProvisionerImages", request: request)
    }

    func uploadProvisionerImage(
        imagePath: String,
        imageName: String,
        provisioner: String,
        location: String,
        timeout: Int32
    ) async throws {
        var request = Apic_UploadProvisionerImageRequest()
        request.imagePath = imagePath
        request.imageName = imageName
        request.provisionerName = provisioner
        request.location = location
        request.timeout = timeout
        let _: Apic_UploadProvisionerImageResponse = try await bridge.call("UploadProvisionerImage", request: request)
    }

    func removeProvisionerImage(imageName: String, provisioner: String, location: String) async throws {
        var request = Apic_RemoveProvisionerImageRequest()
        request.imageName = imageName
        request.provisionerName = provisioner
        request.location = location
        let _: Apic_RemoveProvisionerImageResponse = try await bridge.call("RemoveProvisionerImage", request: request)
    }

    func localCommits() async throws -> Apic_GetLocalCommitsResponse {
        try await bridge.call("GetLocalCommits", request: Apic_GetLocalCommitsRequest())
    }

    func runtimeState(instance: String = "") async throws -> Apic_GetRuntimeStateResponse {
        var request = Apic_GetRuntimeStateRequest()
        request.instance = instance
        return try await bridge.call("GetRuntimeState", request: request)
    }

    func watchChanges(includeSnapshot: Bool = false, heartbeatIntervalMs: UInt32 = 0) -> AsyncThrowingStream<Apic_WatchChangesResponse, Error> {
        var request = Apic_WatchChangesRequest()
        request.includeSnapshot = includeSnapshot
        request.heartbeatIntervalMs = heartbeatIntervalMs
        return bridge.watchChanges(request: request)
    }
}
