package hid

import "testing"

func TestCollections(t *testing.T) {
	descriptor := []byte{
		0x05, 0x01, // Generic Desktop
		0x09, 0x02, // Mouse
		0xa1, 0x01, // Application collection
		0x85, 0x01, // Report ID 1
		0x75, 0x01, // One bit per field
		0x95, 0x03, 0x81, 0x02, // Three data bits
		0x95, 0x05, 0x81, 0x03, // Five padding bits
		0xc0,
		0x06, 0xc2, 0xff, // Vendor page 0xffc2
		0x09, 0x01,
		0xa1, 0x01,
		0x85, 0x04,
		0x75, 0x08, 0x95, 0x3f,
		0x81, 0x02, 0x91, 0x02, // Input and output, 63 data bytes each
		0xa4, // Push globals, then describe a short output report
		0x85, 0x05, 0x95, 0x03, 0x91, 0x02,
		0xb4,       // Pop: ID 4, count 63
		0xb1, 0x02, // Feature
		0xc0,
	}
	infos := collections(Info{Interface: 2}, descriptor)
	if len(infos) != 2 {
		t.Fatalf("got %d collections; want 2", len(infos))
	}
	mouse, vendor := infos[0], infos[1]
	if mouse.UsagePage != 1 || mouse.Usage != 2 || mouse.InputLen != 2 || mouse.OutputLen != 0 {
		t.Fatalf("unexpected mouse capabilities: %+v", mouse)
	}
	if vendor.Interface != 2 || vendor.UsagePage != 0xffc2 || vendor.InputLen != 64 ||
		vendor.OutputLen != 64 || vendor.FeatureLen != 64 {
		t.Fatalf("unexpected vendor capabilities: %+v", vendor)
	}
	dev := Device{Info: vendor}
	buf, err := dev.outgoing(outputReport, []byte{5, 0x99})
	if err != nil || len(buf) != 4 || buf[0] != 5 || buf[1] != 0x99 || buf[2] != 0 {
		t.Fatalf("short report must be padded to its own length: %x, %v", buf, err)
	}
	if _, err := dev.outgoing(outputReport, []byte{5, 1, 2, 3, 4}); err == nil {
		t.Fatal("accepted oversized output report")
	}
	if len(collections(Info{}, descriptor[:len(descriptor)-1])) != 0 {
		t.Fatal("accepted truncated descriptor")
	}
}

func TestUnnumberedFeatureLength(t *testing.T) {
	desc := []byte{0x06, 0, 0xff, 0x09, 1, 0xa1, 1, 0x75, 8, 0x95, 90, 0xb1, 2, 0xc0}
	infos := collections(Info{}, desc)
	if len(infos) != 1 || infos[0].FeatureLen != 91 || infos[0].reports[featureReport][0] != 91 {
		t.Fatalf("expected a 90-byte report with an extra zero ID slot: %+v", infos)
	}
}
