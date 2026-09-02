package packetmanager

import (
	"fmt"
	"io"
	"time"
)

type progressReader struct {
	r     io.Reader
	total int64
	read  int64
	last  time.Time
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)

	if time.Since(p.last) > 100*time.Millisecond || err == io.EOF {
		p.last = time.Now()
		if p.total > 0 {
			pct := float64(p.read) / float64(p.total) * 100
			fmt.Printf("\r  %.1f / %.1f MB (%.0f%%)",
				float64(p.read)/1e6, float64(p.total)/1e6, pct)
		} else {
			fmt.Printf("\r  %.1f MB", float64(p.read)/1e6)
		}
	}
	return n, err
}
