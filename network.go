package main

import (
	"context"
	"log"
	"net"
	"sort"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func serverPort() string {
	parts := strings.Split(defaultBindAddress, ":")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return "8899"
	}
	return parts[len(parts)-1]
}

func detectLANAddresses() []NetworkAddress {
	port := serverPort()
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("检测局域网地址失败: %v", err)
		return nil
	}

	seen := make(map[string]bool)
	addresses := make([]NetworkAddress, 0)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range ifaceAddrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ipv4 := ip.To4()
			if ipv4 == nil || !isReachableIPv4(ipv4) {
				continue
			}
			ipText := ipv4.String()
			if seen[ipText] {
				continue
			}
			seen[ipText] = true
			addresses = append(addresses, NetworkAddress{
				Name:           iface.Name,
				InterfaceIndex: iface.Index,
				IP:             ipText,
				URL:            "http://" + ipText + ":" + port,
			})
		}
	}
	sort.SliceStable(addresses, func(i, j int) bool {
		if addresses[i].InterfaceIndex == addresses[j].InterfaceIndex {
			return addresses[i].IP < addresses[j].IP
		}
		return addresses[i].InterfaceIndex < addresses[j].InterfaceIndex
	})
	return addresses
}

func NetworkInfoHandler(ctx context.Context, c *app.RequestContext) {
	lanURLs := detectLANAddresses()
	c.JSON(consts.StatusOK, utils.H{
		"port":          serverPort(),
		"local_url":     "http://localhost:" + serverPort(),
		"lan_urls":      lanURLs,
		"lan_url_count": len(lanURLs),
		"hint":          "已列出检测到的所有可用 IPv4 网卡地址。手机和电脑连接同一个 Wi-Fi、热点或 VPN 后，在手机浏览器输入对应地址即可访问；若打不开，请检查系统防火墙是否允许本程序访问专用网络。",
	})
}

func isReachableIPv4(ip net.IP) bool {
	return ip != nil &&
		!ip.IsUnspecified() &&
		!ip.IsLoopback() &&
		!ip.IsMulticast() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast()
}
