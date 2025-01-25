/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/bluescorpian/tendactl/tenda"

	"github.com/spf13/cobra"
)

type WanInfo struct {
	WanStatus        string `json:"wanStatus"`
	WanIp            string `json:"wanIp"`
	WanUploadSpeed   string `json:"wanUploadSpeed"`
	WanDownloadSpeed string `json:"wanDownloadSpeed"`
}

type OnlineUpgradeInfo struct {
	NewVersionExist string `json:"newVersionExist"`
	NewVersion      string `json:"newVersion"`
	CurVersion      string `json:"curVersion"`
}

type RouterStatus struct {
	Wl5gEn            string            `json:"wl5gEn"`
	Wl5gName          string            `json:"wl5gName"`
	Wl24gEn           string            `json:"wl24gEn"`
	Wl24gName         string            `json:"wl24gName"`
	Lineup            string            `json:"lineup"`
	ClientNum         int               `json:"clientNum"`
	BlackNum          int               `json:"blackNum"`
	ListNum           int               `json:"listNum"`
	DeviceName        string            `json:"deviceName"`
	LanIP             string            `json:"lanIP"`
	LanMAC            string            `json:"lanMAC"`
	WorkMode          string            `json:"workMode"`
	ApStatus          string            `json:"apStatus"`
	WanInfo           []WanInfo         `json:"wanInfo"`
	OnlineUpgradeInfo OnlineUpgradeInfo `json:"onlineUpgradeInfo"`
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display current status and configuration of the Tenda router",
	Long: `Retrieve and display comprehensive status information from the Tenda router including:
- Network status (WAN IP and connection speeds)
- Wireless configuration (2.4G/5G WiFi status and SSIDs)
- Connected clients count
- Firmware version and updates
- Device information (MAC address, LAN IP, operation mode)

Example:
  tendactl status
  → Shows current router status with network metrics and device information`,
	Run: func(cmd *cobra.Command, args []string) {

		client := tenda.CreateHTTPClient()

		request, err := tenda.TendaRequest("GET", "/goform/GetRouterStatus", nil)
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

		var data RouterStatus
		err = json.Unmarshal([]byte(body), &data)
		if err != nil {
			log.Fatalf("Error parsing JSON: %v", err)
		}

		// Call formatMinimalist to use boolToStatus function
		fmt.Println(formatMinimalist(data))
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func formatMinimalist(d RouterStatus) string {
	var sb strings.Builder

	// Basic device info
	sb.WriteString(fmt.Sprintf("%s (%s) @ %s\n",
		d.DeviceName,
		d.WorkMode,
		d.LanIP))

	// Wireless status
	wifiStatus := "[WIFI]"
	if d.Wl5gEn == "1" || d.Wl24gEn == "1" {
		wifiStatus = fmt.Sprintf("5G:%s 2.4G:%s",
			boolToStatus(d.Wl5gEn, d.Wl5gName),
			boolToStatus(d.Wl24gEn, d.Wl24gName))
	}

	// Network status
	wanStatus := ""
	if len(d.WanInfo) > 0 {
		wan := d.WanInfo[0]
		wanStatus = fmt.Sprintf("WAN: %s (Down:%s KB/s Up:%s KB/s)",
			wan.WanIp,
			wan.WanDownloadSpeed,
			wan.WanUploadSpeed)
	}

	// Combined status line
	sb.WriteString(fmt.Sprintf("%-40s %s\n", wifiStatus, wanStatus))

	// Clients and firmware
	sb.WriteString(fmt.Sprintf("Clients: %-4d  MAC: %s\n",
		d.ClientNum,
		d.LanMAC))
	sb.WriteString(fmt.Sprintf("Firmware: %s", d.OnlineUpgradeInfo.CurVersion))

	if d.OnlineUpgradeInfo.NewVersionExist == "1" {
		sb.WriteString(fmt.Sprintf(" → Update Available: %s",
			d.OnlineUpgradeInfo.NewVersion))
	}

	return sb.String()
}

// Helper function remains the same
func boolToStatus(flag string, name string) string {
	if flag == "1" {
		return fmt.Sprintf("[ON]%s", name)
	}
	return fmt.Sprintf("[OFF]%s", name)
}
