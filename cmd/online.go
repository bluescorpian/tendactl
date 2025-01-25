/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

type OnlineClient struct {
	DeviceID      string `json:"deviceId"`
	IP            string `json:"ip"`
	DevName       string `json:"devName"`
	UploadSpeed   string `json:"uploadSpeed"`
	DownloadSpeed string `json:"downloadSpeed"`
	IsGuestClient string `json:"isGuestClient"`
	LinkType      string `json:"linkType"`
	Line          string `json:"line"`
}

type NetworkStatus struct {
	BlackNum      int    `json:"blackNum"`
	MacFilterType string `json:"macFilterType"`
	LocalhostIP   string `json:"localhostIP"`
	LocalhostName string `json:"localhostName"`
	LocalhostMac  string `json:"localhostMac"`
}

// onlineCmd represents the online command
var onlineCmd = &cobra.Command{
	Use:   "online",
	Short: "List all connected devices and network security status",
	Long: `Retrieve and display real-time network connection information including:
- Currently connected devices (name, IP, and connection speeds)
- Guest network client identification
- MAC filtering configuration status
- Blacklisted devices count
- Local device identification (name, IP, MAC)

Displays upload/download speeds in KB/s and identifies guest network clients.
Shows active MAC filtering policy and number of blocked devices.

Example:
  tendactl online
  → Lists all connected devices with network usage and security status`,
	Run: func(cmd *cobra.Command, args []string) {
		client := tenda.CreateHTTPClient()

		request, err := tenda.TendaRequest("GET", "/goform/getOnlineList", nil)
		if err != nil {
			panic(err)
		}

		resp, err := tenda.TendaDoAuthRequest(client, request)
		if err != nil {
			panic(err)
		}

		// Read and print the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Error reading the response body:", err)
			return
		}
		defer resp.Body.Close()

		var raw []json.RawMessage
		err = json.Unmarshal([]byte(body), &raw)
		if err != nil {
			panic(err)
		}

		// First element is network status
		var networkStatus NetworkStatus
		err = json.Unmarshal(raw[0], &networkStatus)
		if err != nil {
			panic(err)
		}

		// Remaining elements are clients
		var clients []OnlineClient
		for _, clientData := range raw[1:] {
			var client OnlineClient
			err = json.Unmarshal(clientData, &client)
			if err != nil {
				panic(err)
			}
			clients = append(clients, client)
		}

		fmt.Println(formatNetworkStatus(networkStatus, clients))
	},
}

func init() {
	rootCmd.AddCommand(onlineCmd)

}

func formatNetworkStatus(status NetworkStatus, clients []OnlineClient) string {
	var sb strings.Builder

	// Network status header
	sb.WriteString(fmt.Sprintf("%s @ %s\n", status.LocalhostName, status.LocalhostIP))
	sb.WriteString(fmt.Sprintf("MAC: %s | Blacklisted: %d | Filter: %s\n",
		status.LocalhostMac,
		status.BlackNum,
		status.MacFilterType))

	// Connected clients header
	sb.WriteString("\nConnected Devices:\n")

	// Client list
	for _, client := range clients {
		sb.WriteString(formatClientLine(client))
	}

	return sb.String()
}

func formatClientLine(client OnlineClient) string {
	guestMarker := ""
	if client.IsGuestClient == "true" {
		guestMarker = "[Guest]"
	}

	return fmt.Sprintf("  %-20s %-15s ↑%4s ↓%4s KB/s %s\n",
		truncateName(client.DevName, 18),
		client.IP,
		client.UploadSpeed,
		client.DownloadSpeed,
		guestMarker)
}

// Existing helper functions
func truncateName(name string, maxLen int) string {
	if len(name) > maxLen {
		return name[:maxLen-3] + "..."
	}
	return name
}
