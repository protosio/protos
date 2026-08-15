package provisioners

import (
	"strings"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
)

const provisionerOperationTestNamespace = "/protos/test/provisioners"

func acceptedOperationResolutionForTest(
	operation db.PublishedWriteOperation,
	eventID string,
	publishedRootHash string,
) swarmionapp.OperationResolution {
	receipt := swarmionapp.EventReceipt{
		EventID:           strings.TrimSpace(eventID),
		PublishedRootHash: strings.TrimSpace(publishedRootHash),
	}
	return swarmionapp.OperationResolution{
		Identity: swarmionapp.OperationIdentity{
			Key:          strings.TrimSpace(operation.Key),
			IntentDigest: strings.TrimSpace(operation.IntentDigest),
		},
		Scope:          swarmionapp.DatabasePublicationScope(provisionerOperationTestNamespace),
		AuthorPeerID:   strings.TrimSpace(operation.AuthorPeerID),
		State:          swarmionapp.OperationResolvedAccepted,
		Receipt:        &receipt,
		EvidenceSource: "local_outbox",
	}
}

func absentOperationResolutionForTest(operation db.PublishedWriteOperation) swarmionapp.OperationResolution {
	return swarmionapp.OperationResolution{
		Identity: swarmionapp.OperationIdentity{
			Key:          strings.TrimSpace(operation.Key),
			IntentDigest: strings.TrimSpace(operation.IntentDigest),
		},
		Scope:         swarmionapp.DatabasePublicationScope(provisionerOperationTestNamespace),
		AuthorPeerID:  strings.TrimSpace(operation.AuthorPeerID),
		State:         swarmionapp.OperationResolvedAbsent,
		SafeToExecute: true,
	}
}
