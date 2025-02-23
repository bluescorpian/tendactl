/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

type VirtualServerCfg struct {
	LanIp       string         `json:"lanIp"`
	LanMask     string         `json:"lanMask"`
	VirtualList []VirtualEntry `json:"virtualList"`
}

type VirtualEntry struct {
	Ip       string `json:"ip"`
	InPort   string `json:"inPort"`
	OutPort  string `json:"outPort"`
	Protocol string `json:"protocol"`
}

func (v *VirtualServerCfg) VirtualListString() string {
	var parts []string
	for _, entry := range v.VirtualList {
		// TODO: make sure ~ does not exist in string
		parts = append(parts, fmt.Sprintf("%s,%s,%s,%s", entry.Ip, entry.InPort, entry.OutPort, entry.Protocol))
	}
	return strings.Join(parts, "~")
}

// virtualServerCmd represents the virtualServer command
var virtualServerCmd = &cobra.Command{
	Use:   "vs",
	Short: "Manage port forwarding rules (NAT)",
	Long: `Configure and display port forwarding rules for network address translation (NAT).
Displays current rules including:
- Target device IP address
- Internal/external port mappings
- Network protocol (TCP/UDP/Both)

Examples:
  tendactl vs
  → Lists all active port forwarding rules
  tendactl vs add 192.168.0.100 80 8080 0
  → Creates new forwarding rule (TCP&UDP)
  tendactl vs delete 192.168.0.100 80 8080 0
  → Removes existing forwarding rule`,
	Run: func(cmd *cobra.Command, args []string) {
		client := tenda.CreateHTTPClient()
		virtualServerCfg, err := GetVirtualServerCfg(client)
		if err != nil {
			fmt.Printf("Error getting virtual server configuration: %v\n", err)
			return
		}

		fmt.Println(formatVirtualServer(virtualServerCfg))
	},
}

func init() {
	rootCmd.AddCommand(virtualServerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// virtualServerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// virtualServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func formatVirtualServer(cfg *VirtualServerCfg) string {
	var sb strings.Builder

	if len(cfg.VirtualList) == 0 {
		sb.WriteString("\nNo port forwarding rules configured\n")
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("%-15s %-14s %-14s %-10s\n",
		"IP",
		"INTERNAL PORT",
		"EXTERNAL PORT",
		"PROTOCOL"))
	sb.WriteString(fmt.Sprintf("%-15s %-14s %-14s %-10s\n",
		strings.Repeat("─", 15),
		strings.Repeat("─", 14),
		strings.Repeat("─", 14),
		strings.Repeat("─", 10)))

	// Protocol mapping
	protocolMap := map[string]string{
		"0": "TCP&UDP",
		"1": "TCP",
		"2": "UDP",
	}

	// Table rows
	for _, entry := range cfg.VirtualList {
		protocol := protocolMap[entry.Protocol]
		if protocol == "" {
			protocol = entry.Protocol
		}

		sb.WriteString(fmt.Sprintf("%-15s %-14s %-14s %-10s\n",
			entry.Ip,
			entry.InPort,
			entry.OutPort,
			protocol))
	}

	return sb.String()
}

func GetVirtualServerCfg(client *http.Client) (*VirtualServerCfg, error) {
	request, err := tenda.TendaRequest("GET", "/goform/GetVirtualServerCfg", nil)
	if err != nil {
		return &VirtualServerCfg{}, fmt.Errorf("error creating request: %v", err)
	}

	resp, err := tenda.TendaDoAuthRequest(client, request)
	if err != nil {
		return &VirtualServerCfg{}, fmt.Errorf("error sending request: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &VirtualServerCfg{}, fmt.Errorf("error reading response: %v", err)
	}
	defer resp.Body.Close()
	var virtualServerCfg VirtualServerCfg
	err = json.Unmarshal(body, &virtualServerCfg)
	if err != nil {
		return &VirtualServerCfg{}, fmt.Errorf("error unmarshalling response: %v", err)
	}
	return &virtualServerCfg, nil
}

func SetVirtualServerCfg(client *http.Client, cfg *VirtualServerCfg) error {
	body := "list=" + cfg.VirtualListString()
	request, err := tenda.TendaRequest("POST", "/goform/SetVirtualServerCfg", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	resp, err := tenda.TendaDoAuthRequest(client, request)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	// TODO: handle error
	defer resp.Body.Close()
	return nil
}
