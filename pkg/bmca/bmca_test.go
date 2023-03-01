package bmca

import (
	"testing"
)

func TestSortAndFilterGmClockClass(t *testing.T) {
	var ports []BmcaPort

	var port = BmcaPort{
		GmClockClass: 165,
	}
	ports = append(ports, port)
	port = BmcaPort{
		GmClockClass: 254,
	}
	ports = append(ports, port)
	port = BmcaPort{
		GmClockClass: 254,
	}
	ports = append(ports, port)
	port = BmcaPort{
		GmClockClass: 165,
	}
	ports = append(ports, port)

	pts := SortAndFilter(ports, GmClockClass)
	for _, port := range pts {
		t.Log(port.GmClockClass)
	}
}
