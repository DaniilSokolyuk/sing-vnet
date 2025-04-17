package internal

import (
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/gopacket/gopacket/pcap"
)

func NewPCAP(cfg InterfaceConfig) (*PCAP, error) {
	iface, dev := findDevInterface(cfg.Name)
	slog.Info("Using interface",
		"name", iface.Name,
		"device", dev.Name,
		"mac", iface.HardwareAddr.String())

	_, network, err := net.ParseCIDR(cfg.Network)
	if err != nil {
		return nil, fmt.Errorf("parse cidr error: %w", err)
	}

	localIP := net.ParseIP(cfg.LocalIP)
	if localIP == nil {
		return nil, fmt.Errorf("invalid local IP: %s", cfg.LocalIP)
	}

	localIP = localIP.To4()
	if !network.Contains(localIP) {
		return nil, fmt.Errorf("local ip (%s) not in network (%s)", localIP, network)
	}

	inactive, err := createPcapHandle(dev)
	if err != nil {
		return nil, fmt.Errorf("create pcap handle error: %w", err)
	}

	handle, err := inactive.Activate()
	if err != nil {
		return nil, fmt.Errorf("activate handle error: %w", err)
	}

	// Set BPF filter to capture ARP and IP traffic for our network
	filter := fmt.Sprintf("arp or (src net %s or dst net %s)", network, network)
	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		return nil, fmt.Errorf("set BPF filter error: %w", err)
	}

	return &PCAP{
		name:      cfg.Name,
		Interface: iface,
		network:   network,
		localIP:   localIP,
		localMAC:  iface.HardwareAddr,
		handle:    handle,
	}, nil
}

type PCAP struct {
	name      string
	Interface net.Interface
	network   *net.IPNet
	localIP   net.IP
	localMAC  net.HardwareAddr
	handle    *pcap.Handle
	readMux   sync.Mutex
}

func createPcapHandle(dev pcap.Interface) (*pcap.InactiveHandle, error) {
	handle, err := pcap.NewInactiveHandle(dev.Name)
	if err != nil {
		return nil, fmt.Errorf("new inactive handle error: %w", err)
	}

	err = handle.SetPromisc(true)
	if err != nil {
		return nil, fmt.Errorf("set promisc error: %w", err)
	}

	err = handle.SetSnapLen(1600)
	if err != nil {
		return nil, fmt.Errorf("set snap len error: %w", err)
	}

	err = handle.SetTimeout(pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("set timeout error: %w", err)
	}

	err = handle.SetImmediateMode(true)
	if err != nil {
		return nil, fmt.Errorf("set immediate mode error: %w", err)
	}

	err = handle.SetBufferSize(512 * 1024)
	if err != nil {
		return nil, fmt.Errorf("set buffer size error: %w", err)
	}

	return handle, nil
}

func (t *PCAP) Read() []byte {
	t.readMux.Lock()
	defer t.readMux.Unlock()
	data, _, err := t.handle.ZeroCopyReadPacketData()
	if err != nil {
		if err != pcap.NextErrorTimeoutExpired {
			slog.Error("read packet error", "err", err)
		}
		return nil
	}
	return data
}

func (t *PCAP) Write(p []byte) error {
	err := t.handle.WritePacketData(p)
	if err != nil {
		return fmt.Errorf("write packet error: %w", err)
	}
	return nil
}

func (t *PCAP) Close() {
	if t.handle != nil {
		t.handle.Close()
	}
}

// findDevInterface returns both net.Interface and pcap.Interface for a given interface name
func findDevInterface(deviceName string) (net.Interface, pcap.Interface) {
	// Get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		slog.Error("Failed to get network interfaces", "error", err)
		panic(err)
	}

	// Get all pcap devices
	devices, err := pcap.FindAllDevs()
	if err != nil {
		slog.Error("Failed to get pcap devices", "error", err)
		panic(err)
	}

	// Debug logging of available interfaces
	slog.Debug("Available network interfaces:")
	for _, iface := range ifaces {
		slog.Debug("net.Interface",
			"name", iface.Name,
			"mac", iface.HardwareAddr,
			"flags", iface.Flags)
	}

	slog.Debug("Available pcap devices:")
	for _, dev := range devices {
		slog.Debug("pcap.Interface",
			"name", dev.Name,
			"description", dev.Description,
			"addresses", dev.Addresses)
	}

	// Find net.Interface
	var foundIface net.Interface
	for _, iface := range ifaces {
		if iface.Name == deviceName {
			foundIface = iface
			slog.Debug("Found network interface",
				"name", iface.Name,
				"mac", iface.HardwareAddr)
			break
		}
	}

	if foundIface.Name == "" {
		slog.Error("Network interface not found", "device", deviceName)
		panic(fmt.Errorf("interface %s not found", deviceName))
	}

	// Find pcap.Interface
	var foundDev pcap.Interface
	for _, dev := range devices {
		if dev.Name == deviceName {
			foundDev = dev
			slog.Debug("Found pcap device",
				"name", dev.Name,
				"description", dev.Description)
			break
		}
	}

	if foundDev.Name == "" {
		slog.Error("PCAP device not found", "device", deviceName)
		panic(fmt.Errorf("pcap device %s not found", deviceName))
	}

	return foundIface, foundDev
}
