package p2p

// import (
// 	"context"
// 	"encoding/json"
// 	"io/ioutil"

// 	"github.com/libp2p/go-libp2p-core/helpers"
// 	"github.com/libp2p/go-libp2p-core/host"
// 	"github.com/libp2p/go-libp2p-core/network"
// 	"github.com/libp2p/go-libp2p-core/peer"
// 	"github.com/libp2p/go-libp2p-core/protocol"
// )

// var syncProtocolID = protocol.ID("/protos/sync/0.0.1")

// type Msg struct {
// 	Type string `json:"type"`
// 	Msg  string `json:"msg"`
// }

// func handleSync(s network.Stream) {

// 	buf, err := ioutil.ReadAll(s)
// 	if err != nil {
// 		s.Reset()
// 		log.Errorf("Failed to read data from stream: %s", err.Error())
// 		return
// 	}
// 	s.Close()

// 	var msg Msg
// 	err = json.Unmarshal(buf, &msg)
// 	if err != nil {
// 		log.Errorf("Failed to decode JSON payload from stream: %s", err.Error())
// 		return
// 	}
// 	log.Println(msg)
// }

// func sendSync(msg Msg, hst host.Host, id peer.ID) error {

// 	// Start a stream with the destination.
// 	// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
// 	s, err := hst.NewStream(context.Background(), id, syncProtocolID)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// // Create a buffered stream so that read and writes are non blocking.
// 	// rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

// 	// jsonPayload, _ := json.Marshal(Msg{Type: "test", Msg: "yolo"})

// 	// jsonPayload = append(jsonPayload, '\n')

// 	// // Create a thread to read and write data.
// 	// writeStream(rw, jsonPayload)
// 	// // go readStream(rw)

// 	err = helpers.FullClose(s)
// 	if err != nil {
// 		log.Println(err)
// 		s.Reset()
// 		return err
// 	}
// 	return nil
// }
