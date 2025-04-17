package internal

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/DaniilSokolyuk/sing-vnet/arpr"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"gvisor.dev/gvisor/pkg/tcpip/header"
)

//cfg := Config{
//	FromInterface: InterfaceConfig{
//		Name:    "en0",
//		Network: "172.24.0.0/16",
//		LocalIP: "172.24.0.1",
//	},
//	ToInterface: InterfaceConfig{
//		Name:    "utun128",
//		Network: "172.24.0.0/16",
//		LocalIP: "172.24.0.1",
//	},
//}

type Config struct {
	FromInterface InterfaceConfig
	ToInterface   InterfaceConfig
}

type InterfaceConfig struct {
	Name    string
	Network string
	LocalIP string
}

type Bridge struct {
	from       *PCAP // en0 - L2 interface
	to         *PCAP // utun128 - L3 interface
	ipMacTable map[string]net.HardwareAddr
	mapMux     sync.RWMutex
	stop       context.CancelFunc
}

var (
	bridgeMx sync.Mutex
)

func Start(ctx context.Context, cfg Config) (*Bridge, error) {
	ctx, cancel := context.WithCancel(ctx)

	bridgeMx.Lock()
	defer bridgeMx.Unlock()

	from, err := NewPCAP(cfg.FromInterface)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create from pcap error: %w", err)
	}

	to, err := NewPCAP(cfg.ToInterface)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create to pcap error: %w", err)
	}

	bridge := &Bridge{
		from:       from,
		to:         to,
		ipMacTable: make(map[string]net.HardwareAddr),
		stop:       cancel,
	}

	// Send initial gratuitous ARP only for the L2 interface (en0)
	if err := bridge.announcePresence(); err != nil {
		cancel()
		return nil, fmt.Errorf("announce presence error: %w", err)
	}

	go bridge.handleTraffic(ctx)

	return bridge, nil
}

const (
	tunHeaderSize  = 4
	tunHeader      = "\x02\x00\x00\x00"
	ethernetHeight = 14
)

func (b *Bridge) handleTraffic(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)

	// Handle traffic from L2 (en0) to L3 (utun128)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				packet := b.from.Read()
				if packet == nil {
					continue
				}

				ethPacket := header.Ethernet(packet)
				switch ethPacket.Type() {
				case header.ARPProtocolNumber:
					b.handleARP(packet)
				case header.IPv4ProtocolNumber:
					// Store the source MAC for future responses
					ipHeader := header.IPv4(packet[14:])
					srcIP := ipHeader.SourceAddress()

					if !b.from.network.Contains(srcIP.AsSlice()) {
						continue
					}

					//store
					b.StoreMAC(srcIP.String(), []byte(ethPacket.SourceAddress()))

					//gPckt := gopacket.NewPacket(ipHeader, layers.LayerTypeIPv4, gopacket.Default)
					//fmt.Println("FROM L2>L3", gPckt.String(), packet, "\n")

					// Forward to L3 interface
					copy(packet[ethernetHeight-tunHeaderSize:ethernetHeight], tunHeader)
					b.to.Write(packet[ethernetHeight-tunHeaderSize:])
				}
			}
		}
	}()

	// Handle traffic from L3 (utun128) to L2 (en0)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				packet := b.to.Read()
				if packet == nil {
					continue
				}
				// Add L2 header for en0
				ipHeader := header.IPv4(packet[4:])
				//srcIP := ipHeader.SourceAddress().String()
				dstIP := ipHeader.DestinationAddress().String()

				// Look up destination MAC
				dstMAC, ok := b.GetMAC(dstIP)
				if !ok {
					continue
				}

				//gPckt1 := gopacket.NewPacket(packet[4:], layers.LayerTypeIPv4, gopacket.Default)
				//fmt.Println("TO L3>L2", gPckt1.String(), packet, "\n")

				// Create ethernet frame
				eth := &layers.Ethernet{
					SrcMAC:       b.from.localMAC,
					DstMAC:       dstMAC,
					EthernetType: layers.EthernetTypeIPv4,
				}

				// Serialize packet with ethernet header
				buffer := gopacket.NewSerializeBuffer()
				opts := gopacket.SerializeOptions{
					FixLengths:       true,
					ComputeChecksums: true,
				}

				err := gopacket.SerializeLayers(buffer, opts,
					eth,
					gopacket.Payload(ipHeader), // Only use the IP packet part
				)
				if err != nil {
					slog.Error("failed to serialize packet", "err", err)
					continue
				}

				b.from.Write(buffer.Bytes())
			}
		}
	}()

	wg.Wait()
}

func (b *Bridge) handleARP(packet []byte) {
	gPckt := gopacket.NewPacket(packet, layers.LayerTypeEthernet, gopacket.Default)
	arpLayer, ok := gPckt.Layer(layers.LayerTypeARP).(*layers.ARP)
	if !ok {
		return
	}

	if arpLayer.Operation == layers.ARPRequest {
		srcIP := net.IP(arpLayer.SourceProtAddress)
		if b.from.network.Contains(srcIP) {
			reply, err := arpr.SendReply(arpLayer, b.from.localIP, b.from.localMAC)
			if err != nil {
				slog.Error("send arp reply error", "err", err)
				return
			}
			b.from.Write(reply)

			srcMAC := net.HardwareAddr(arpLayer.SourceHwAddress)
			b.StoreMAC(srcIP.To4().String(), srcMAC)
			slog.Info("stored arp mapping", "ip", srcIP, "mac", srcMAC.String())
		}
	}
}

func (b *Bridge) announcePresence() error {
	// Only send gratuitous ARP on the L2 interface
	arpPacket, err := arpr.SendGratuitousArp(b.from.localIP, b.from.localMAC)
	if err != nil {
		return err
	}
	return b.from.Write(arpPacket)
}

func (b *Bridge) StoreMAC(ip string, mac net.HardwareAddr) {
	b.mapMux.Lock()
	defer b.mapMux.Unlock()
	b.ipMacTable[ip] = mac
}

func (b *Bridge) GetMAC(ip string) (net.HardwareAddr, bool) {
	b.mapMux.RLock()
	defer b.mapMux.RUnlock()
	mac, ok := b.ipMacTable[ip]
	return mac, ok
}

func (b *Bridge) Close() {
	b.stop()
	b.from.Close()
	b.to.Close()
}
