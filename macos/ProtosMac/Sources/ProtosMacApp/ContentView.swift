import SwiftUI

struct ContentView: View {
    @Environment(AppModel.self) private var model: AppModel

    var body: some View {
        NavigationSplitView {
            List {
                ForEach(SidebarSection.allCases) { section in
                    Button {
                        model.selectedSection = section
                    } label: {
                        Label(section.rawValue, systemImage: section.icon)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .listRowBackground(
                        model.selectedSection == section ? Color.accentColor.opacity(0.14) : Color.clear
                    )
                }
            }
            .navigationTitle("Protos")
        } detail: {
            VStack(spacing: 0) {
                HeaderBar()
                Divider()
                detailView
                if let message = model.message {
                    Divider()
                    HStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle")
                            .foregroundStyle(.orange)
                        Text(message)
                            .lineLimit(2)
                        Spacer()
                        Button {
                            model.message = nil
                        } label: {
                            Image(systemName: "xmark")
                        }
                        .buttonStyle(.borderless)
                        .help("Dismiss")
                    }
                    .padding(10)
                    .background(.bar)
                }
            }
            .toolbar {
                ToolbarItemGroup {
                    if model.isBusy {
                        ProgressView()
                            .controlSize(.small)
                    }
                    Button {
                        Task { await model.refreshAll() }
                    } label: {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                    .help("Refresh")
                }
            }
        }
    }

    @ViewBuilder
    private var detailView: some View {
        switch model.selectedSection ?? .overview {
        case .overview:
            OverviewView()
        case .apps:
            AppsView()
        case .provisioners:
            ProvisionersView()
        case .instances:
            InstancesView()
        case .network:
            NetworkView()
        case .releases:
            ReleasesView()
        case .dvc:
            DVCView()
        }
    }
}

struct HeaderBar: View {
    @Environment(AppModel.self) private var model: AppModel

    var body: some View {
        HStack(spacing: 12) {
            VStack(alignment: .leading, spacing: 2) {
                Text(model.selectedSection?.rawValue ?? "Overview")
                    .font(.title2.weight(.semibold))
                Text("Embedded daemon: \(model.daemonState.title)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            StatusPill(state: model.daemonState)
        }
        .padding(.horizontal, 20)
        .padding(.vertical, 14)
    }
}

struct StatusPill: View {
    let state: DaemonState

    var body: some View {
        Label(state.title, systemImage: icon)
            .font(.caption.weight(.medium))
            .padding(.horizontal, 9)
            .padding(.vertical, 5)
            .background(color.opacity(0.14), in: Capsule())
            .foregroundStyle(color)
    }

    private var icon: String {
        switch state {
        case .starting: "clock"
        case .running: "checkmark.circle"
        case .failed: "xmark.octagon"
        }
    }

    private var color: Color {
        switch state {
        case .starting: .orange
        case .running: .green
        case .failed: .red
        }
    }
}

struct OverviewView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var username = ""
    @State private var name = ""
    @State private var organization = ""

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                SectionTitle("User")
                Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 8) {
                    KeyValue("Username", model.userInfo?.username)
                    KeyValue("Name", model.userInfo?.name)
                    KeyValue("Role", model.userInfo?.isAdmin == true ? "Admin" : "User")
                }

                Divider()
                SectionTitle("Initialize")
                HStack(alignment: .bottom, spacing: 10) {
                    TextField("Username", text: $username)
                    TextField("Name", text: $name)
                    TextField("Organization", text: $organization)
                    Button {
                        Task {
                            await model.run {
                                try await model.api.initUser(
                                    username: username,
                                    name: name,
                                    organization: organization
                                )
                                await model.refreshAll()
                            }
                        }
                    } label: {
                        Label("Init", systemImage: "person.badge.plus")
                    }
                    .disabled(username.nonEmpty == nil || name.nonEmpty == nil)
                    .help("Initialize local Protos data")
                }

                Divider()
                SectionTitle("Devices")
                SimpleRows(
                    rows: model.devices,
                    emptyTitle: "No devices",
                    id: { $0.id },
                    columns: [
                        ("Name", { $0.name }),
                        ("ID", { $0.id }),
                        ("WireGuard", { $0.publicKeyWireguard })
                    ]
                )

                Divider()
                SectionTitle("Local SSH Key")
                Text(model.sshKey?.public ?? "")
                    .font(.system(.body, design: .monospaced))
                    .textSelection(.enabled)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .padding(22)
        }
    }
}

struct AppsView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var selectedName = ""
    @State private var name = ""
    @State private var installer = ""
    @State private var instance = ""
    @State private var persistence = false

    var body: some View {
        @Bindable var model = model
        VStack(alignment: .leading, spacing: 16) {
            SimpleRows(
                rows: model.apps,
                emptyTitle: "No apps",
                selection: $selectedName,
                id: { $0.name },
                columns: [
                    ("Name", { $0.name }),
                    ("Instance", { $0.instanceName }),
                    ("Status", { $0.status }),
                    ("IP", { $0.ip }),
                    ("Installer", { $0.installer })
                ]
            )

            Divider()
            HStack(alignment: .bottom, spacing: 10) {
                TextField("Name", text: $name)
                TextField("Installer", text: $installer)
                TextField("Instance", text: $instance)
                Toggle("Persistent", isOn: $persistence)
                    .toggleStyle(.checkbox)
                Button {
                    Task {
                        await model.run {
                            try await model.api.createApp(
                                name: name,
                                installerID: installer,
                                instanceID: instance,
                                persistence: persistence
                            )
                            await model.refreshAll()
                        }
                    }
                } label: {
                    Label("Create", systemImage: "plus")
                }
                .disabled(name.nonEmpty == nil || installer.nonEmpty == nil || instance.nonEmpty == nil)
            }

            HStack(spacing: 8) {
                CommandButton("Start", icon: "play.fill", enabled: selectedName.nonEmpty != nil) {
                    try await model.api.startApp(selectedName)
                }
                CommandButton("Stop", icon: "stop.fill", enabled: selectedName.nonEmpty != nil) {
                    try await model.api.stopApp(selectedName)
                }
                CommandButton("Remove", icon: "trash", role: .destructive, enabled: selectedName.nonEmpty != nil) {
                    try await model.api.removeApp(selectedName)
                }
                CommandButton("Logs", icon: "doc.text", enabled: selectedName.nonEmpty != nil, refresh: false) {
                    model.outputText = try await model.api.appLogs(selectedName)
                }
            }

            OutputPane(text: $model.outputText)
        }
        .padding(22)
    }
}

struct ProvisionersView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var selectedName = ""
    @State private var name = ""
    @State private var type = "local_macos"
    @State private var credentials = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack(alignment: .top, spacing: 18) {
                VStack(alignment: .leading, spacing: 10) {
                    SectionTitle("Provisioners")
                    SimpleRows(
                        rows: model.provisioners,
                        emptyTitle: "No provisioners",
                        selection: $selectedName,
                        id: { $0.name },
                        columns: [
                            ("Name", { $0.name }),
                            ("Type", { $0.type.name }),
                            ("Locations", { $0.supportedLocations.joined(separator: ", ") })
                        ]
                    )
                }
                VStack(alignment: .leading, spacing: 10) {
                    SectionTitle("Supported")
                    SimpleRows(
                        rows: model.supportedProvisioners,
                        emptyTitle: "No types",
                        id: { $0.name },
                        columns: [
                            ("Type", { $0.name }),
                            ("Fields", { $0.authenticationFields.joined(separator: ", ") })
                        ]
                    )
                    .frame(width: 300)
                }
            }

            Divider()
            HStack(alignment: .bottom, spacing: 10) {
                TextField("Name", text: $name)
                TextField("Type", text: $type)
                TextField("Credentials: KEY=value, one per line", text: $credentials, axis: .vertical)
                    .lineLimit(1...4)
                Button {
                    Task {
                        await model.run {
                            try await model.api.addProvisioner(
                                name: name,
                                type: type,
                                credentials: credentials.credentialPairs
                            )
                            await model.refreshAll()
                        }
                    }
                } label: {
                    Label("Add", systemImage: "plus")
                }
                .disabled(name.nonEmpty == nil || type.nonEmpty == nil)
            }

            HStack {
                CommandButton("Remove", icon: "trash", role: .destructive, enabled: selectedName.nonEmpty != nil) {
                    try await model.api.removeProvisioner(selectedName)
                }
            }
        }
        .padding(22)
    }
}

struct InstancesView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var selectedName = ""
    @State private var name = ""
    @State private var provisioner = ""
    @State private var location = "local"
    @State private var machine = "vz-2c-2g"
    @State private var version = ""
    @State private var image = ""
    @State private var ip = ""
    @State private var localOnly = false

    var body: some View {
        @Bindable var model = model
        VStack(alignment: .leading, spacing: 16) {
            SimpleRows(
                rows: model.instances,
                emptyTitle: "No instances",
                selection: $selectedName,
                id: { $0.name },
                columns: [
                    ("Name", { $0.name }),
                    ("Provisioner", { $0.cloudName }),
                    ("Location", { $0.location }),
                    ("Status", { $0.status }),
                    ("Public IP", { $0.publicIp }),
                    ("Internal IP", { $0.internalIp })
                ]
            )

            Divider()
            Grid(alignment: .leading, horizontalSpacing: 10, verticalSpacing: 10) {
                GridRow {
                    TextField("Name", text: $name)
                    TextField("Provisioner", text: $provisioner)
                    TextField("Location", text: $location)
                }
                GridRow {
                    TextField("Machine", text: $machine)
                    TextField("Version", text: $version)
                    TextField("Image path/name", text: $image)
                }
            }
            HStack(spacing: 8) {
                Button {
                    Task {
                        await model.run {
                            try await model.api.deployInstance(
                                name: name,
                                provisioner: provisioner,
                                location: location,
                                machineType: machine,
                                version: version,
                                devImage: image
                            )
                            await model.refreshAll()
                        }
                    }
                } label: {
                    Label("Deploy", systemImage: "plus")
                }
                .disabled(name.nonEmpty == nil || provisioner.nonEmpty == nil)

                Toggle("Local only", isOn: $localOnly)
                    .toggleStyle(.checkbox)
                CommandButton("Start", icon: "play.fill", enabled: selectedName.nonEmpty != nil) {
                    try await model.api.startInstance(selectedName)
                }
                CommandButton("Stop", icon: "stop.fill", enabled: selectedName.nonEmpty != nil) {
                    try await model.api.stopInstance(selectedName)
                }
                CommandButton("Remove", icon: "trash", role: .destructive, enabled: selectedName.nonEmpty != nil) {
                    try await model.api.removeInstance(selectedName, localOnly: localOnly)
                }
                CommandButton("Key", icon: "key", enabled: selectedName.nonEmpty != nil, refresh: false) {
                    model.outputText = try await model.api.instanceKey(selectedName)
                }
                CommandButton("Logs", icon: "doc.text", enabled: selectedName.nonEmpty != nil, refresh: false) {
                    model.outputText = try await model.api.instanceLogs(selectedName)
                }
            }

            HStack(spacing: 10) {
                TextField("IP", text: $ip)
                CommandButton("Init", icon: "network", enabled: selectedName.nonEmpty != nil && ip.nonEmpty != nil) {
                    try await model.api.initInstance(selectedName, ip: ip)
                }
                CommandButton("Update IP", icon: "arrow.triangle.2.circlepath", enabled: selectedName.nonEmpty != nil && ip.nonEmpty != nil) {
                    try await model.api.updateInstance(id: selectedName, ip: ip)
                }
            }

            OutputPane(text: $model.outputText)
        }
        .padding(22)
    }
}

struct NetworkView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var inspectInstance = ""
    @State private var routeInstance = ""
    @State private var dnsServer = ""
    @State private var cidrs = ""

    var body: some View {
        @Bindable var model = model
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                HStack(alignment: .bottom, spacing: 10) {
                    TextField("Target instance", text: $inspectInstance)
                    Button {
                        Task {
                            await model.run {
                                try await model.refreshNetwork(instance: inspectInstance.nonEmpty ?? "")
                            }
                        }
                    } label: {
                        Label("Load", systemImage: "arrow.down.circle")
                    }
                }

                NetworkSummary(state: model.networkState)

                Divider()
                SectionTitle("Exit Routes")
                SimpleRows(
                    rows: model.exitRoutes,
                    emptyTitle: "No exit routes",
                    id: { route in
                        route.id.nonEmpty ?? "\(route.deviceID):\(route.instanceID)"
                    },
                    columns: [
                        ("Device", { $0.deviceID }),
                        ("Instance", { $0.instanceID }),
                        ("Name", { $0.instanceName }),
                        ("Public IP", { $0.publicIp }),
                        ("Location", { $0.location }),
                        ("CIDRs", { $0.cidrs.joined(separator: ", ") }),
                        ("DNS", { $0.dnsServer.nonEmpty ?? "default" }),
                        ("Status", { $0.status })
                    ]
                )

                HStack(alignment: .bottom, spacing: 10) {
                    TextField("Instance", text: $routeInstance)
                    TextField("DNS server", text: $dnsServer)
                        .frame(width: 150)
                    TextField("CIDRs", text: $cidrs, axis: .vertical)
                        .lineLimit(1...4)
                    Button {
                        Task {
                            await model.run {
                                let response = try await model.api.setExitRoute(
                                    instance: routeInstance,
                                    dnsServer: dnsServer.nonEmpty ?? "",
                                    cidrs: cidrs.routeCIDRs
                                )
                                model.outputText = routeOutput(response.route)
                                try await model.refreshNetwork(instance: inspectInstance.nonEmpty ?? "")
                            }
                        }
                    } label: {
                        Label("Set", systemImage: "point.topleft.down.curvedto.point.bottomright.up")
                    }
                    .disabled(routeInstance.nonEmpty == nil)
                    .help("Set exit route")

                    Button(role: .destructive) {
                        Task {
                            await model.run {
                                try await model.api.clearExitRoute()
                                model.outputText = "Exit route disabled"
                                try await model.refreshNetwork(instance: inspectInstance.nonEmpty ?? "")
                            }
                        }
                    } label: {
                        Label("Clear", systemImage: "xmark.circle")
                    }
                    .help("Clear exit route")
                }

                Divider()
                SectionTitle("Observed Routes")
                SimpleRows(
                    rows: model.networkState?.routes ?? [],
                    emptyTitle: "No observed routes",
                    id: { route in
                        "\(route.destination)|\(route.gateway)|\(route.interfaceName)|\(route.source)|\(route.priority)"
                    },
                    columns: [
                        ("Destination", { $0.destination.nonEmpty ?? "default" }),
                        ("Gateway", { $0.gateway }),
                        ("Interface", { $0.interfaceName }),
                        ("Source", { $0.source }),
                        ("Family", { $0.family }),
                        ("Kind", { $0.kind }),
                        ("Priority", { $0.priority })
                    ]
                )

                HStack(alignment: .top, spacing: 18) {
                    VStack(alignment: .leading, spacing: 10) {
                        SectionTitle("Addresses")
                        SimpleRows(
                            rows: model.networkState?.addresses ?? [],
                            emptyTitle: "No addresses",
                            id: { "\($0.interfaceName)|\($0.cidr)" },
                            columns: [
                                ("Interface", { $0.interfaceName }),
                                ("CIDR", { $0.cidr }),
                                ("Scope", { $0.scope })
                            ]
                        )
                    }
                    VStack(alignment: .leading, spacing: 10) {
                        SectionTitle("DNS")
                        SimpleRows(
                            rows: model.networkState?.dns ?? [],
                            emptyTitle: "No DNS entries",
                            id: { "\($0.scope)|\($0.domain)|\($0.servers.joined(separator: ","))|\($0.port)" },
                            columns: [
                                ("Scope", { $0.scope }),
                                ("Domain", { $0.domain }),
                                ("Servers", { $0.servers.joined(separator: ", ") }),
                                ("Port", { "\($0.port)" }),
                                ("Active", { $0.active ? "yes" : "no" })
                            ]
                        )
                    }
                }

                SectionTitle("WireGuard Peers")
                SimpleRows(
                    rows: model.networkState?.wireguardPeers ?? [],
                    emptyTitle: "No peers",
                    id: { $0.publicKey },
                    columns: [
                        ("Public Key", { String($0.publicKey.prefix(18)) }),
                        ("Endpoint", { $0.endpoint }),
                        ("Allowed IPs", { $0.allowedIps.joined(separator: ", ") }),
                        ("Handshake", { $0.latestHandshake }),
                        ("RX", { "\($0.rxBytes)" }),
                        ("TX", { "\($0.txBytes)" })
                    ]
                )

                OutputPane(text: $model.outputText)
                    .frame(minHeight: 90)
            }
            .padding(22)
        }
    }

    private func routeOutput(_ route: Apic_ExitRoute) -> String {
        var lines = ["Routing local traffic through \(route.instanceName) (\(route.publicIp))"]
        if !route.cidrs.isEmpty {
            lines.append("Routed CIDRs: \(route.cidrs.joined(separator: ", "))")
        }
        if let dns = route.dnsServer.nonEmpty {
            lines.append("Forwarding external DNS through \(dns)")
        }
        return lines.joined(separator: "\n")
    }
}

struct NetworkSummary: View {
    let state: Apic_NetworkState?

    var body: some View {
        Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 8) {
            KeyValue("Module", state?.module)
            KeyValue("Status", state == nil ? nil : (state?.up == true ? "up" : "down"))
            KeyValue("Interface", state?.interfaceName)
            KeyValue("Messages", state?.messages.joined(separator: ", "))
        }
    }
}

struct ReleasesView: View {
    @Environment(AppModel.self) private var model: AppModel
    @State private var provisioner = ""
    @State private var imagePath = ""
    @State private var imageName = ""
    @State private var location = "local"
    @State private var timeout = 600
    @State private var selectedImage = ""

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            HStack {
                SectionTitle("Available Releases")
                Button {
                    Task {
                        await model.run {
                            let response = try await model.api.releases()
                            model.releases = response.releases
                        }
                    }
                } label: {
                    Label("Load", systemImage: "arrow.down.circle")
                }
            }
            SimpleRows(
                rows: model.releases,
                emptyTitle: "No releases",
                id: { $0.version },
                columns: [
                    ("Version", { $0.version }),
                    ("Description", { $0.description_p }),
                    ("Images", { "\($0.cloudImages.count)" })
                ]
            )

            Divider()
            HStack(alignment: .bottom, spacing: 10) {
                TextField("Provisioner", text: $provisioner)
                Button {
                    Task {
                        await model.run {
                            let response = try await model.api.provisionerImages(provisioner)
                            model.provisionerImages = response.images
                        }
                    }
                } label: {
                    Label("Load Images", systemImage: "photo.stack")
                }
                .disabled(provisioner.nonEmpty == nil)
            }

            SimpleRows(
                rows: model.sortedProvisionerImages,
                emptyTitle: "No provisioner images",
                selection: $selectedImage,
                id: { $0.key },
                columns: [
                    ("Key", { $0.key }),
                    ("Name", { $0.value.name }),
                    ("Location", { $0.value.location }),
                    ("ID", { $0.value.id })
                ]
            )

            HStack(alignment: .bottom, spacing: 10) {
                TextField("Image path", text: $imagePath)
                TextField("Image name", text: $imageName)
                TextField("Location", text: $location)
                TextField("Timeout", value: $timeout, format: .number)
                    .frame(width: 90)
                Button {
                    Task {
                        await model.run {
                            try await model.api.uploadProvisionerImage(
                                imagePath: imagePath,
                                imageName: imageName,
                                provisioner: provisioner,
                                location: location,
                                timeout: Int32(timeout)
                            )
                            let response = try await model.api.provisionerImages(provisioner)
                            model.provisionerImages = response.images
                        }
                    }
                } label: {
                    Label("Upload", systemImage: "square.and.arrow.up")
                }
                .disabled(provisioner.nonEmpty == nil || imagePath.nonEmpty == nil || imageName.nonEmpty == nil)
                CommandButton("Remove", icon: "trash", role: .destructive, enabled: selectedImage.nonEmpty != nil) {
                    try await model.api.removeProvisionerImage(
                        imageName: selectedImage,
                        provisioner: provisioner,
                        location: location
                    )
                    let response = try await model.api.provisionerImages(provisioner)
                    model.provisionerImages = response.images
                }
            }
        }
        .padding(22)
    }
}

struct DVCView: View {
    @Environment(AppModel.self) private var model: AppModel

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                HStack(alignment: .center) {
                    SectionTitle("Peers")
                    Spacer()
                    Button {
                        Task {
                            await model.run {
                                try await model.refreshDVC()
                            }
                        }
                    } label: {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                    .help("Refresh P2P DVC")
                }
                DVCRuntimeSummary(state: model.runtimeState)
                PeerRows(peers: model.runtimePeers)
                SectionTitle("Commits")
                CommitRows(commits: model.localCommits)
            }
            .padding(22)
        }
        .task {
            await model.run {
                try await model.refreshDVC()
            }
        }
    }
}

struct DVCRuntimeSummary: View {
    let state: Apic_RuntimeState?

    var body: some View {
        Grid(alignment: .leading, horizontalSpacing: 18, verticalSpacing: 8) {
            KeyValue("Peer ID", state?.peerID)
            KeyValue("Finalized root", shortHash(state?.finalizedRootHash))
            KeyValue("Tentative root", shortHash(state?.tentativeRootHash))
            KeyValue("Connected peers", state.map { "\($0.connectedPeers.count)" })
            KeyValue("State providers", state?.stateProviders.joined(separator: ", "))
            KeyValue("Materialization", state?.runtimeMaterializationPolicy)
        }
        .textSelection(.enabled)
    }
}

struct PeerRows: View {
    let peers: [Apic_RuntimePeerStatus]

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 12) {
                header("Peer")
                header("Connection")
                header("Roles")
                header("Compatibility")
                header("Reason")
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            Divider()
            if peers.isEmpty {
                Text("No peers")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, minHeight: 72)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(peers, id: \.peerID) { peer in
                            HStack(spacing: 12) {
                                Text(shortHash(peer.peerID) ?? "n/a")
                                    .font(.system(.body, design: .monospaced))
                                    .textSelection(.enabled)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                PeerConnectionLabel(peer: peer)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                Text(peerRoles(peer))
                                    .lineLimit(1)
                                    .truncationMode(.tail)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                Text(peerCompatibility(peer))
                                    .lineLimit(1)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                Text(peerReason(peer))
                                    .lineLimit(1)
                                    .truncationMode(.middle)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 7)
                            Divider()
                        }
                    }
                }
                .frame(minHeight: 110, maxHeight: 260)
            }
        }
        .overlay {
            RoundedRectangle(cornerRadius: 6)
                .stroke(Color(nsColor: .separatorColor), lineWidth: 1)
        }
    }

    private func header(_ title: String) -> some View {
        Text(title)
            .font(.caption.weight(.semibold))
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct PeerConnectionLabel: View {
    let peer: Apic_RuntimePeerStatus

    var body: some View {
        Label(title, systemImage: icon)
            .font(.caption.weight(.medium))
            .padding(.horizontal, 8)
            .padding(.vertical, 4)
            .background(color.opacity(0.14), in: Capsule())
            .foregroundStyle(color)
            .lineLimit(1)
    }

    private var title: String {
        if peer.connected {
            return "Connected"
        }
        if peer.dialable {
            return "Dialable"
        }
        return "Not connected"
    }

    private var icon: String {
        peer.connected ? "checkmark.circle" : "circle"
    }

    private var color: Color {
        if peer.connected {
            return .green
        }
        if peer.dialable {
            return .blue
        }
        return .secondary
    }
}

struct CommitRows: View {
    let commits: [Apic_Commit]

    var body: some View {
        SimpleRows(
            rows: commits,
            emptyTitle: "No commits",
            id: { $0.hash },
            columns: [
                ("State", { commitStateLabel($0) }),
                ("Hash", { String($0.hash.prefix(12)) }),
                ("Committer", { $0.committer }),
                ("Message", { $0.message })
            ]
        )
    }
}

private func shortHash(_ value: String?) -> String? {
    guard let value = value?.nonEmpty else {
        return nil
    }
    if value.count <= 12 {
        return value
    }
    return String(value.prefix(12))
}

private func peerRoles(_ peer: Apic_RuntimePeerStatus) -> String {
    var roles: [String] = []
    if peer.stateProvider {
        roles.append("provider")
    }
    if peer.witness {
        roles.append("witness")
    }
    if peer.eligibleWitness {
        roles.append("eligible")
    }
    if peer.relayOnly {
        roles.append("relay-only")
    }
    if peer.ignored {
        roles.append("ignored")
    }
    return roles.isEmpty ? "peer" : roles.joined(separator: ", ")
}

private func peerCompatibility(_ peer: Apic_RuntimePeerStatus) -> String {
    if peer.incompatible {
        return "Incompatible"
    }
    if peer.compatible {
        return "Compatible"
    }
    return "Unknown"
}

private func peerReason(_ peer: Apic_RuntimePeerStatus) -> String {
    if let reason = peer.reason.nonEmpty {
        return reason
    }
    if let error = peer.lastDialErrors.values.sorted().first?.nonEmpty {
        return error
    }
    return "n/a"
}

private func commitStateLabel(_ commit: Apic_Commit) -> String {
    if commit.states.isEmpty {
        return "unknown"
    }
    return commit.states.map { $0.capitalized }.joined(separator: ", ")
}

struct CommandButton: View {
    @Environment(AppModel.self) private var model: AppModel
    let title: String
    let icon: String
    let role: ButtonRole?
    let enabled: Bool
    let refresh: Bool
    let action: () async throws -> Void

    init(
        _ title: String,
        icon: String,
        role: ButtonRole? = nil,
        enabled: Bool = true,
        refresh: Bool = true,
        action: @escaping () async throws -> Void
    ) {
        self.title = title
        self.icon = icon
        self.role = role
        self.enabled = enabled
        self.refresh = refresh
        self.action = action
    }

    var body: some View {
        Button(role: role) {
            Task {
                await model.run {
                    try await action()
                    if refresh {
                        await model.refreshAll()
                    }
                }
            }
        } label: {
            Label(title, systemImage: icon)
        }
        .disabled(!enabled)
        .help(title)
    }
}

struct SectionTitle: View {
    let text: String

    init(_ text: String) {
        self.text = text
    }

    var body: some View {
        Text(text)
            .font(.headline)
            .frame(maxWidth: .infinity, alignment: .leading)
    }
}

struct KeyValue: View {
    let key: String
    let value: String?

    init(_ key: String, _ value: String?) {
        self.key = key
        self.value = value
    }

    var body: some View {
        GridRow {
            Text(key)
                .foregroundStyle(.secondary)
            Text(value?.nonEmpty ?? "n/a")
                .textSelection(.enabled)
        }
    }
}

struct OutputPane: View {
    @Binding var text: String

    var body: some View {
        TextEditor(text: $text)
            .font(.system(.body, design: .monospaced))
            .scrollContentBackground(.hidden)
            .background(Color(nsColor: .textBackgroundColor))
            .clipShape(RoundedRectangle(cornerRadius: 6))
            .frame(minHeight: 120)
    }
}

struct SimpleRows<Row>: View {
    private struct IdentifiedRow: Identifiable {
        let id: String
        let row: Row
    }

    let rows: [Row]
    let emptyTitle: String
    var selection: Binding<String>?
    let id: (Row) -> String
    let columns: [(String, (Row) -> String)]

    init(
        rows: [Row],
        emptyTitle: String,
        selection: Binding<String>? = nil,
        id: @escaping (Row) -> String,
        columns: [(String, (Row) -> String)]
    ) {
        self.rows = rows
        self.emptyTitle = emptyTitle
        self.selection = selection
        self.id = id
        self.columns = columns
    }

    private var identifiedRows: [IdentifiedRow] {
        var seen: [String: Int] = [:]
        return rows.enumerated().map { index, row in
            let baseID = id(row).nonEmpty ?? "row-\(index)"
            let count = seen[baseID, default: 0]
            seen[baseID] = count + 1
            return IdentifiedRow(id: count == 0 ? baseID : "\(baseID)#\(count)", row: row)
        }
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 12) {
                ForEach(columns.indices, id: \.self) { index in
                    Text(columns[index].0)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            Divider()
            if rows.isEmpty {
                Text(emptyTitle)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, minHeight: 72)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(identifiedRows) { row in
                            rowView(row)
                            Divider()
                        }
                    }
                }
                .frame(minHeight: 110, maxHeight: 280)
            }
        }
        .overlay {
            RoundedRectangle(cornerRadius: 6)
                .stroke(Color(nsColor: .separatorColor), lineWidth: 1)
        }
    }

    private func rowView(_ item: IdentifiedRow) -> some View {
        let rowID = item.id
        return Button {
            selection?.wrappedValue = rowID
        } label: {
            HStack(spacing: 12) {
                ForEach(columns.indices, id: \.self) { index in
                    Text(columns[index].1(item.row).nonEmpty ?? "n/a")
                        .lineLimit(1)
                        .truncationMode(.middle)
                        .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .contentShape(Rectangle())
            .background(selection?.wrappedValue == rowID ? Color.accentColor.opacity(0.16) : Color.clear)
        }
        .buttonStyle(.plain)
    }
}
