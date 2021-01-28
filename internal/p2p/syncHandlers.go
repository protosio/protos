package p2p

import "fmt"

const getRootHandler = "getRoot"

type getRootReq struct {
	last    string
	current string
}

type getRootResp struct {
}

func getRoot(data interface{}) (interface{}, error) {
	_, ok := data.(*getRootReq)
	if !ok {
		return getRootResp{}, fmt.Errorf("Unknown data struct for init request")
	}

	return getRootResp{}, nil
}
