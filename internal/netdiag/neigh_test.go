package netdiag

import (
	"context"
	"testing"
)

const darwinARP = `? (192.168.1.1) at a4:2b:b0:1:2:3 on en0 ifscope [ethernet]
? (192.168.1.50) at (incomplete) on en0 ifscope [ethernet]
? (192.168.1.255) at ff:ff:ff:ff:ff:ff on en0 ifscope [ethernet]
? (224.0.0.251) at 1:0:5e:0:0:fb on en0 ifscope permanent [ethernet]
`

const procARP = `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1      0x1         0x2         a4:2b:b0:01:02:03     *        eth0
192.168.1.77     0x1         0x0         00:00:00:00:00:00     *        eth0
192.168.1.20     0x1         0x6         de:ad:be:ef:00:01     *        eth0
`

const winARP = `
Interface: 192.168.1.10 --- 0x5
  Internet Address      Physical Address      Type
  192.168.1.1           a4-2b-b0-01-02-03     dynamic
  192.168.1.255         ff-ff-ff-ff-ff-ff     static
  224.0.0.22            01-00-5e-00-00-16     static
`

func withRaw(t *testing.T, out string) {
	t.Helper()
	old := neighborsRaw
	neighborsRaw = func(context.Context) (string, error) { return out, nil }
	t.Cleanup(func() { neighborsRaw = old })
}

func TestNeighbors_DropsIncompleteAndFailed(t *testing.T) {
	for name, fx := range map[string]string{"darwin": darwinARP, "proc": procARP} {
		withRaw(t, fx)
		got, err := Neighbors(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range got {
			if n.State != "reachable" && n.State != "stale" {
				t.Errorf("%s: state %q leaked", name, n.State)
			}
			if n.IP == "192.168.1.50" || n.IP == "192.168.1.77" {
				t.Errorf("%s: incomplete/failed %s not dropped", name, n.IP)
			}
		}
		if len(got) == 0 {
			t.Errorf("%s: no good rows", name)
		}
	}
}

func TestNeighbors_ParsesIPAndMAC(t *testing.T) {
	withRaw(t, darwinARP)
	got, _ := Neighbors(context.Background())
	if len(got) != 1 || got[0].IP != "192.168.1.1" || got[0].MAC != "a4:2b:b0:01:02:03" || got[0].State != "reachable" {
		t.Errorf("darwin: %+v", got)
	}
	withRaw(t, procARP)
	got, _ = Neighbors(context.Background())
	if len(got) != 2 || got[1].MAC != "de:ad:be:ef:00:01" {
		t.Errorf("proc: %+v", got)
	}
	withRaw(t, winARP)
	got, _ = Neighbors(context.Background())
	if len(got) != 1 || got[0].IP != "192.168.1.1" || got[0].MAC != "a4:2b:b0:01:02:03" {
		t.Errorf("windows: %+v", got)
	}
}

func TestNeighbors_LinuxCarriesRealState(t *testing.T) {
	withRaw(t, procARP)
	got, _ := Neighbors(context.Background())
	if len(got) != 2 || got[0].State != "reachable" || got[1].State != "stale" {
		t.Errorf("proc states: %+v", got)
	}
}
