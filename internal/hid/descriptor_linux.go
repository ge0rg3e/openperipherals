package hid

const (
	inputReport = iota
	outputReport
	featureReport
)

type reportLengths [3]map[byte]int

// collections extracts the capabilities used for interface selection. Report
// sizes are counted per ID across the entire descriptor: a raw report can
// contain fields from several top-level collections.
func collections(base Info, descriptor []byte) []Info {
	type globals struct {
		page, size, count uint32
		id                byte
	}
	type collection struct {
		page, usage uint16
		reports     [3]map[byte]bool
	}
	var g globals
	var stack []globals
	var found []collection
	var bits [3][256]uint64
	var usage uint32
	var hasUsage bool
	depth := 0
	for len(descriptor) != 0 {
		tag := descriptor[0]
		descriptor = descriptor[1:]
		if tag == 0xfe {
			if len(descriptor) < 2 || len(descriptor) < int(descriptor[0])+2 {
				return nil
			}
			descriptor = descriptor[int(descriptor[0])+2:]
			continue
		}
		n := int(tag & 3)
		if n == 3 {
			n = 4
		}
		if n > len(descriptor) {
			return nil
		}
		var v uint32
		for i, b := range descriptor[:n] {
			v |= uint32(b) << (8 * i)
		}
		descriptor = descriptor[n:]
		switch tag & 0xfc {
		case 0x04:
			g.page = v
		case 0x74:
			g.size = v
		case 0x94:
			g.count = v
		case 0x84:
			if v == 0 || v > 255 {
				return nil
			}
			g.id = byte(v)
		case 0xa4:
			stack = append(stack, g)
		case 0xb4:
			if len(stack) == 0 {
				return nil
			}
			g = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		case 0x08:
			if !hasUsage {
				usage = v
				if n < 4 {
					usage |= g.page << 16
				}
				hasUsage = true
			}
		case 0xa0:
			if depth == 0 {
				found = append(found, collection{page: uint16(usage >> 16), usage: uint16(usage)})
			}
			depth++
		case 0xc0:
			if depth == 0 {
				return nil
			}
			depth--
		case 0x80, 0x90, 0xb0:
			kind := inputReport
			if tag&0xfc == 0x90 {
				kind = outputReport
			}
			if tag&0xfc == 0xb0 {
				kind = featureReport
			}
			size := uint64(g.size) * uint64(g.count)
			// Leave room for the report ID within the hidraw transfer limit.
			const maxBits = (4096 - 1) * 8
			if size > maxBits || bits[kind][g.id]+size > maxBits {
				return nil
			}
			bits[kind][g.id] += size
			if depth > 0 {
				c := &found[len(found)-1]
				if c.reports[kind] == nil {
					c.reports[kind] = make(map[byte]bool)
				}
				c.reports[kind][g.id] = true
			}
		}
		if tag&0x0c == 0 {
			usage, hasUsage = 0, false
		}
	}
	if depth != 0 {
		return nil
	}

	var lengths reportLengths
	for kind, byID := range bits {
		lengths[kind] = make(map[byte]int)
		for id, bitCount := range byID {
			if bitCount > 0 {
				lengths[kind][byte(id)] = int((bitCount+7)/8) + 1
			}
		}
	}
	out := make([]Info, 0, len(found))
	for _, c := range found {
		info := base
		info.UsagePage, info.Usage = c.page, c.usage
		info.reports = &lengths
		for kind, dest := range []*int{&info.InputLen, &info.OutputLen, &info.FeatureLen} {
			for id := range c.reports[kind] {
				*dest = max(*dest, lengths[kind][id])
			}
		}
		out = append(out, info)
	}
	return out
}
