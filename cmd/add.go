/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <ip> <inPort> [outPort] [protocol]",
	Short: "Add a new port forwarding rule (NAT)",
	Long: `Add a new port forwarding rule for a target IP address, specifying the internal (inPort) 
and external (outPort) ports. If not specified, outPort defaults to inPort. The last argument 
determines the network protocol (0=TCP&UDP, 1=TCP, 2=UDP). For example:

  tendactl vs add 192.168.0.100 80 8080 0
  → Creates a new forwarding rule for TCP&UDP on port 80 (internal) mapped to 8080 (external).`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ip := args[0]
		inPort := args[1]
		outPort := args[1]
		protocol := "0"

		if len(args) >= 3 {
			outPort = args[2]
		}
		if len(args) >= 4 {
			protocol = args[3]
		}
		client := tenda.CreateHTTPClient()
		virtualServerCfg, err := GetVirtualServerCfg(client)
		if err != nil {
			fmt.Printf("Error getting virtual server configuration: %v\n", err)
			return
		}

		virtualServerCfg.VirtualList = append(virtualServerCfg.VirtualList, VirtualEntry{
			Ip:       ip,
			InPort:   inPort,
			OutPort:  outPort,
			Protocol: protocol,
		})

		err = SetVirtualServerCfg(client, virtualServerCfg)
		if err != nil {
			fmt.Printf("Error setting virtual server configuration: %v\n", err)
			return
		}
		fmt.Println("Added new port forwarding rule")

	},
}

func init() {
	virtualServerCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
