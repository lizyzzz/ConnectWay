package ConnectWay

import (
	"testing"

	log "github.com/cihub/seelog"
)

func TestCreateClient(t *testing.T) {
	for i := 0; i < 100; i++ {
		log.Info("seelog test")
	}
}
