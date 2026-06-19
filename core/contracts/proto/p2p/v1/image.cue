package p2pv1

#DescribeImageRequest: {
	image_ref?: string
}

#DescribeImageResponse: {
	found?:         bool
	image_ref?:     string
	target_digest?: string
	platform?:      string
	labels?:        [string]: string
}

#ImageContentDescriptor: {
	media_type?:   string
	digest?:       string
	size_bytes?:   uint
	platform?:     string
	annotations?:  [string]: string
}

#GetImageContentRequest: {
	image_ref?: string
}

#GetImageContentResponse: {
	found?:       bool
	image_ref?:   string
	target?:      #ImageContentDescriptor
	platform?:    string
	labels?:      [string]: string
	descriptors?: [...#ImageContentDescriptor]
}

#GetImageBlobRequest: {
	digest?:     string
	chunk_size?: uint
	offset?:     uint
	length?:     uint
}

#GetImageBlobResponse: {
	digest?: string
	offset?: uint
	data?:   bytes
	eof?:    bool
}

#LoadImageArchiveRequest: {
	archive_path?: string
	image_ref?:     string
}

#LoadImageArchiveResponse: {
	image_ref?:      string
	target_digest?:  string
	platform?:       string
}

#UploadImageArchiveChunkRequest: {
	upload_id?: string
	image_ref?:  string
	offset?:     uint
	data?:       bytes
	eof?:        bool
}

#UploadImageArchiveChunkResponse: {
	received_bytes?: uint
	loaded?:         bool
	image_ref?:      string
	target_digest?:  string
	platform?:       string
}

contract: {
	surface: "p2p-images-grpc"
	migration: {
		id:                  "protos-p2p-images-v0.0"
		lineage_id:          "protos.p2p.images"
		from_version:        ""
		to_version:          "0.0"
		compatibility:       "full"
		backward_compatible: true
		forward_compatible:  true
	}
	proto: {
		syntax:     "proto3"
		package:    "proto"
		go_package: "./proto"
		services: [{
			name: "Images"
			rpcs: [
				{name: "DescribeImage", request: "DescribeImageRequest", response: "DescribeImageResponse"},
				{name: "GetImageContent", request: "GetImageContentRequest", response: "GetImageContentResponse"},
				{name: "GetImageBlob", request: "GetImageBlobRequest", response: "GetImageBlobResponse", response_stream: true},
				{name: "LoadImageArchive", request: "LoadImageArchiveRequest", response: "LoadImageArchiveResponse"},
				{name: "UploadImageArchiveChunk", request: "UploadImageArchiveChunkRequest", response: "UploadImageArchiveChunkResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "DescribeImageRequest", fields: [
				{type: "string", name: "image_ref", number: 1},
			]},
			{kind: "message", name: "DescribeImageResponse", fields: [
				{type: "bool", name: "found", number: 1},
				{type: "string", name: "image_ref", number: 2},
				{type: "string", name: "target_digest", number: 3},
				{type: "string", name: "platform", number: 4},
				{type: "map<string, string>", name: "labels", number: 5},
			]},
			{kind: "message", name: "ImageContentDescriptor", fields: [
				{type: "string", name: "media_type", number: 1},
				{type: "string", name: "digest", number: 2},
				{type: "uint64", name: "size_bytes", number: 3},
				{type: "string", name: "platform", number: 4},
				{type: "map<string, string>", name: "annotations", number: 5},
			]},
			{kind: "message", name: "GetImageContentRequest", fields: [
				{type: "string", name: "image_ref", number: 1},
			]},
			{kind: "message", name: "GetImageContentResponse", fields: [
				{type: "bool", name: "found", number: 1},
				{type: "string", name: "image_ref", number: 2},
				{type: "ImageContentDescriptor", name: "target", number: 3},
				{type: "string", name: "platform", number: 4},
				{type: "map<string, string>", name: "labels", number: 5},
				{rule: "repeated", type: "ImageContentDescriptor", name: "descriptors", number: 6},
			]},
			{kind: "message", name: "GetImageBlobRequest", fields: [
				{type: "string", name: "digest", number: 1},
				{type: "uint64", name: "chunk_size", number: 2},
				{type: "uint64", name: "offset", number: 3},
				{type: "uint64", name: "length", number: 4},
			]},
			{kind: "message", name: "GetImageBlobResponse", fields: [
				{type: "string", name: "digest", number: 1},
				{type: "uint64", name: "offset", number: 2},
				{type: "bytes", name: "data", number: 3},
				{type: "bool", name: "eof", number: 4},
			]},
			{kind: "message", name: "LoadImageArchiveRequest", fields: [
				{type: "string", name: "archive_path", number: 1},
				{type: "string", name: "image_ref", number: 2},
			]},
			{kind: "message", name: "LoadImageArchiveResponse", fields: [
				{type: "string", name: "image_ref", number: 1},
				{type: "string", name: "target_digest", number: 2},
				{type: "string", name: "platform", number: 3},
			]},
			{kind: "message", name: "UploadImageArchiveChunkRequest", fields: [
				{type: "string", name: "upload_id", number: 1},
				{type: "string", name: "image_ref", number: 2},
				{type: "uint64", name: "offset", number: 3},
				{type: "bytes", name: "data", number: 4},
				{type: "bool", name: "eof", number: 5},
			]},
			{kind: "message", name: "UploadImageArchiveChunkResponse", fields: [
				{type: "uint64", name: "received_bytes", number: 1},
				{type: "bool", name: "loaded", number: 2},
				{type: "string", name: "image_ref", number: 3},
				{type: "string", name: "target_digest", number: 4},
				{type: "string", name: "platform", number: 5},
			]},
		]
	}
}

lineage: {
	name: "protos.p2p.images"
	schemas: [{
		version: [0, 0]
		schema: {
			DescribeImageRequest?:        #DescribeImageRequest
			DescribeImageResponse?:       #DescribeImageResponse
			ImageContentDescriptor?:      #ImageContentDescriptor
			GetImageContentRequest?:      #GetImageContentRequest
			GetImageContentResponse?:     #GetImageContentResponse
			GetImageBlobRequest?:         #GetImageBlobRequest
			GetImageBlobResponse?:        #GetImageBlobResponse
			LoadImageArchiveRequest?:     #LoadImageArchiveRequest
			LoadImageArchiveResponse?:    #LoadImageArchiveResponse
			UploadImageArchiveChunkRequest?:  #UploadImageArchiveChunkRequest
			UploadImageArchiveChunkResponse?: #UploadImageArchiveChunkResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
