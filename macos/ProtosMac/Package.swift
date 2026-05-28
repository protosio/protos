// swift-tools-version: 6.2

import PackageDescription

let package = Package(
    name: "ProtosMac",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(name: "ProtosMacApp", targets: ["ProtosMacApp"])
    ],
    dependencies: [
        .package(url: "https://github.com/apple/swift-protobuf.git", exact: "1.38.0")
    ],
    targets: [
        .target(
            name: "CProtosBridge",
            publicHeadersPath: "include"
        ),
        .executableTarget(
            name: "ProtosMacApp",
            dependencies: [
                "CProtosBridge",
                .product(name: "SwiftProtobuf", package: "swift-protobuf")
            ],
            linkerSettings: [
                .unsafeFlags(["-L", ".build/go", "-lprotos", "-lresolv"])
            ]
        )
    ]
)
