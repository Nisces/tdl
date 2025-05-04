package tmessage

import (
	"github.com/gotd/td/tg"
)

type Dialog struct {
	Peer     tg.InputPeerClass
	Messages []int
	URLs     []string
}

type ParseSource func() ([]*Dialog, error)

func Parse(src ParseSource) ([]*Dialog, error) {
	return src()
}
