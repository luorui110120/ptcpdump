package consumer

import (
	"context"
	"github.com/mozillazg/ptcpdump/bpf"
	"github.com/mozillazg/ptcpdump/internal/event"
	"github.com/mozillazg/ptcpdump/internal/log"
	"github.com/mozillazg/ptcpdump/internal/metadata"
	"github.com/mozillazg/ptcpdump/internal/writer"
	"time"
)

type PacketEventConsumer struct {
	writers        []writer.PacketWriter
	deviceCache    *metadata.DeviceCache
	processedCount int

	delay time.Duration
}

func NewPacketEventConsumer(writers []writer.PacketWriter, deviceCache *metadata.DeviceCache) *PacketEventConsumer {
	return &PacketEventConsumer{
		writers:     writers,
		deviceCache: deviceCache,
	}
}

func (c *PacketEventConsumer) WithDelay(delay time.Duration) *PacketEventConsumer {
	c.delay = delay
	return c
}

func (c *PacketEventConsumer) Start(ctx context.Context, ch <-chan bpf.BpfPacketEventWithPayloadT, maxPacketCount uint) {
	if c.delay > 0 {
		time.Sleep(c.delay)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case pt := <-ch:
			c.handlePacketEvent(pt)
			c.processedCount++
			if maxPacketCount > 0 && c.processedCount >= int(maxPacketCount) {
				return
			}
		}
	}
}

func (c *PacketEventConsumer) handlePacketEvent(pt bpf.BpfPacketEventWithPayloadT) {
	pevent, err := event.ParsePacketEvent(c.deviceCache, pt)
	if err != nil {
		log.Errorf("[PacketEventConsumer] parse event failed: %s", err)
		return
	}

	for _, w := range c.writers {
		if err := w.Write(pevent); err != nil {
			log.Errorf("[PacketEventConsumer] write packet failed: %s, device: %#v", err, pevent.Device)
		}
		w.Flush()
	}
}

func (c *PacketEventConsumer) Stop() {

}

func (c *PacketEventConsumer) ProcessedCount() int {
	return c.processedCount
}
