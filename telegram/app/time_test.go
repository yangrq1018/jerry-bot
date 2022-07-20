package app

import (
	"bytes"
	"testing"
)

func TestSnapshotCPU(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	SnapshotHardware(buf)
	t.Log(buf.String())
}
