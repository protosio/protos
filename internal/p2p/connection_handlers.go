package p2p

import "github.com/libp2p/go-libp2p/core/network"

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	go func() {
		if conn.Stat().Transient {
			return
		}

		var machineID string
		var found bool
		if !p2p.externalDB.Initialized() {
			machineID = "init"
		} else {
			machineID, found = p2p.machinePeerMapping.Get(conn.RemotePeer().String())
			if !found {
				log.Debugf("new connection with unknown peer '%s'. Closing connection", conn.RemotePeer().String())
				conn.Close()
				return
			}

			_, found = p2p.machines.Get(machineID)
			if !found {
				log.Debugf("new connection with peer '%s' with unknown machine. Closing connection", conn.RemotePeer().String())
				conn.Close()
				return
			}
		}

		log.Debugf("new connection with peer '%s'(%s). Creating client", machineID, conn.RemotePeer().String())
		client, err := p2p.createClientForPeer(conn.RemotePeer().String())
		if err != nil {
			log.Errorf("failed to create client for new peer '%s'(%s): %s", machineID, conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}
		p2p.clients.Set(machineID, client)
		if err := p2p.externalDB.AddPeer(machineID, client.grpcConnection); err != nil {
			log.Errorf("failed to add peer %s(%s) to external DB: %s", machineID, conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}
	}()
}

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {

	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("error while disconnecting from peer '%s': %v", conn.RemotePeer().String(), err)
		}
	}()

	machineID, found := p2p.machinePeerMapping.Get(conn.RemotePeer().String())
	if !found {
		log.Debugf("disconnected from unknown peer '%s'", conn.RemotePeer().String())
		return
	}

	log.Infof("disconnected from  %s(%s)", machineID, conn.RemotePeer().String())

	p2p.clients.Delete(machineID)
	if p2p.externalDB != nil {
		if err := p2p.externalDB.RemovePeer(machineID); err != nil {
			log.Errorf("failed to remove DB peer for %s(%s): %v", machineID, conn.RemotePeer().String(), err)
		}
	}
}
