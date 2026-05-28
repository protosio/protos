import Foundation
import Observation
import SwiftUI

enum SidebarSection: String, CaseIterable, Identifiable {
    case overview = "Overview"
    case apps = "Apps"
    case provisioners = "Provisioners"
    case instances = "Instances"
    case network = "Network"
    case releases = "Releases"
    case dvc = "P2P DVC"

    var id: String { rawValue }

    var icon: String {
        switch self {
        case .overview: "gauge.with.dots.needle.67percent"
        case .apps: "shippingbox"
        case .provisioners: "server.rack"
        case .instances: "desktopcomputer"
        case .network: "network"
        case .releases: "arrow.up.circle"
        case .dvc: "point.3.connected.trianglepath.dotted"
        }
    }
}

enum DaemonState: Equatable {
    case starting
    case running
    case failed(String)

    var title: String {
        switch self {
        case .starting:
            return "Starting"
        case .running:
            return "Running"
        case .failed:
            return "Failed"
        }
    }
}

@MainActor
@Observable
final class AppModel {
    var daemonState: DaemonState = .starting
    var selectedSection: SidebarSection? = .overview
    var message: String?
    var isBusy = false

    var userInfo: Apic_GetUserInfoResponse?
    var devices: [Apic_UserDevice] = []
    var sshKey: Apic_GetLocalSSHKeyResponse?
    var apps: [Apic_App] = []
    var supportedProvisioners: [Apic_ProvisionerType] = []
    var provisioners: [Apic_Provisioner] = []
    var instances: [Apic_CloudInstance] = []
    var networkState: Apic_NetworkState?
    var exitRoutes: [Apic_ExitRoute] = []
    var releases: [Apic_Release] = []
    var provisionerImages: [String: Apic_CloudSpecificImage] = [:] {
        didSet {
            sortedProvisionerImages = provisionerImages.sorted { $0.key < $1.key }
        }
    }
    private(set) var sortedProvisionerImages: [(key: String, value: Apic_CloudSpecificImage)] = []
    var runtimeState: Apic_RuntimeState? {
        didSet {
            runtimePeers = sortedRuntimePeers(runtimeState?.peerStatuses ?? [])
        }
    }
    private(set) var runtimePeers: [Apic_RuntimePeerStatus] = []
    var localCommits: [Apic_Commit] = []
    var outputText = ""

    @ObservationIgnored private let bridge: NativeProtosBridge
    @ObservationIgnored let api: ProtosAPI
    @ObservationIgnored private var hasStarted = false
    @ObservationIgnored private var changeWatchTask: Task<Void, Never>?
    @ObservationIgnored private var liveRefreshTask: Task<Void, Never>?
    @ObservationIgnored private var pendingLiveTables = Set<String>()
    @ObservationIgnored private var pendingRuntimeChange = false
    @ObservationIgnored private var pendingHeartbeat = false

    init(bridge: NativeProtosBridge = NativeProtosBridge()) {
        self.bridge = bridge
        self.api = ProtosAPI(bridge: bridge)
    }

    deinit {
        changeWatchTask?.cancel()
        liveRefreshTask?.cancel()
    }

    func startIfNeeded() async {
        guard !hasStarted else { return }
        hasStarted = true
        await run { [self] in
            try await self.api.start()
            self.daemonState = .running
            await self.refreshAll()
            self.startChangeWatcher()
        } onError: { [self] error in
            self.daemonState = .failed(error.localizedDescription)
        }
    }

    func refreshAll() async {
        await run { [self] in
            async let user = try? self.api.userInfo()
            async let deviceList = try? self.api.userDevices()
            async let key = try? self.api.localSSHKey()
            async let appList = try? self.api.apps()
            async let provisionerTypeList = try? self.api.supportedProvisioners()
            async let provisionerList = try? self.api.provisioners()
            async let instanceList = try? self.api.instances()
            async let network = try? self.api.networkState()
            async let routeList = try? self.api.exitRoutes()
            async let runtime = try? self.api.runtimeState()
            async let commitList = try? self.api.localCommits()

            self.userInfo = await user
            self.devices = await deviceList?.devices ?? []
            self.sshKey = await key
            self.apps = await appList?.apps ?? []
            self.supportedProvisioners = await provisionerTypeList?.provisionerTypes ?? []
            self.provisioners = await provisionerList?.provisioners ?? []
            self.instances = await instanceList?.instances ?? []
            if let networkResponse = await network, networkResponse.hasState {
                self.networkState = networkResponse.state
            } else {
                self.networkState = nil
            }
            self.exitRoutes = await routeList?.routes ?? []
            if let runtimeResponse = await runtime, runtimeResponse.hasState {
                self.runtimeState = runtimeResponse.state
            } else {
                self.runtimeState = nil
            }
            self.localCommits = await commitList?.commits ?? []
        }
    }

    func refreshNetwork(instance: String = "") async throws {
        let networkResponse = try await api.networkState(instance: instance)
        networkState = networkResponse.hasState ? networkResponse.state : nil
        let routeResponse = try await api.exitRoutes(instance: instance)
        exitRoutes = routeResponse.routes
    }

    func refreshDVC(instance: String = "") async throws {
        let runtimeResponse = try await api.runtimeState(instance: instance)
        runtimeState = runtimeResponse.hasState ? runtimeResponse.state : nil
        let commitResponse = try await api.localCommits()
        localCommits = commitResponse.commits
    }

    private func startChangeWatcher() {
        guard changeWatchTask == nil else { return }
        let stream = api.watchChanges(includeSnapshot: false, heartbeatIntervalMs: 3_000)
        changeWatchTask = Task { [weak self] in
            do {
                for try await change in stream {
                    self?.scheduleLiveRefresh(for: change)
                }
            } catch is CancellationError {
                return
            } catch {
                await MainActor.run {
                    self?.message = "Live refresh stopped: \(error.localizedDescription)"
                    self?.changeWatchTask = nil
                }
            }
        }
    }

    private func scheduleLiveRefresh(for change: Apic_WatchChangesResponse) {
        if change.reason == "heartbeat", selectedSection != .dvc {
            return
        }
        pendingLiveTables.formUnion(change.tableNames)
        pendingRuntimeChange = pendingRuntimeChange || change.runtimeChanged
        pendingHeartbeat = pendingHeartbeat || change.reason == "heartbeat"
        guard liveRefreshTask == nil else { return }
        liveRefreshTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 250_000_000)
            await self?.performPendingLiveRefresh()
        }
    }

    private func performPendingLiveRefresh() async {
        let tables = pendingLiveTables
        let runtimeChanged = pendingRuntimeChange
        let heartbeat = pendingHeartbeat
        pendingLiveTables.removeAll()
        pendingRuntimeChange = false
        pendingHeartbeat = false
        liveRefreshTask = nil

        do {
            try await refreshVisibleSection(tables: tables, runtimeChanged: runtimeChanged, heartbeat: heartbeat)
        } catch {
            message = error.localizedDescription
        }
    }

    private func refreshVisibleSection(tables: Set<String>, runtimeChanged: Bool, heartbeat: Bool) async throws {
        switch selectedSection ?? .overview {
        case .overview:
            guard !heartbeat, !runtimeChanged || !tables.isEmpty else { return }
            async let user = try? api.userInfo()
            async let deviceList = try? api.userDevices()
            async let key = try? api.localSSHKey()
            userInfo = await user
            devices = await deviceList?.devices ?? []
            sshKey = await key
        case .apps:
            guard !heartbeat, !runtimeChanged || !tables.isEmpty else { return }
            apps = try await api.apps().apps
        case .provisioners:
            guard !heartbeat, !runtimeChanged || !tables.isEmpty else { return }
            async let supported = try? api.supportedProvisioners()
            async let existing = try? api.provisioners()
            supportedProvisioners = await supported?.provisionerTypes ?? []
            provisioners = await existing?.provisioners ?? []
        case .instances:
            guard !heartbeat, !runtimeChanged || !tables.isEmpty else { return }
            instances = try await api.instances().instances
        case .network:
            guard !heartbeat else { return }
            try await refreshNetwork()
        case .releases:
            return
        case .dvc:
            if heartbeat || runtimeChanged || !tables.isEmpty {
                try await refreshDVC()
            }
        }
    }

    func run(_ operation: @escaping () async throws -> Void) async {
        await run(operation, onError: nil)
    }

    func run(
        _ operation: @escaping () async throws -> Void,
        onError: ((Error) -> Void)?
    ) async {
        isBusy = true
        message = nil
        do {
            try await operation()
        } catch {
            message = error.localizedDescription
            onError?(error)
        }
        isBusy = false
    }
}

private func sortedRuntimePeers(_ peers: [Apic_RuntimePeerStatus]) -> [Apic_RuntimePeerStatus] {
    peers.sorted {
        if $0.connected != $1.connected {
            return $0.connected && !$1.connected
        }
        return $0.peerID.localizedStandardCompare($1.peerID) == .orderedAscending
    }
}

extension String {
    var nonEmpty: String? {
        let trimmed = trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    var credentialPairs: [String: String] {
        split(whereSeparator: { $0 == "\n" || $0 == "," })
            .reduce(into: [String: String]()) { result, rawPair in
                let pair = rawPair.split(separator: "=", maxSplits: 1).map {
                    String($0).trimmingCharacters(in: .whitespacesAndNewlines)
                }
                guard pair.count == 2, !pair[0].isEmpty else { return }
                result[pair[0]] = pair[1]
            }
    }

    var routeCIDRs: [String] {
        split(whereSeparator: { $0 == "\n" || $0 == "," })
            .map { String($0).trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }
}
