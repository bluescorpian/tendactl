/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/bluescorpian/tendactl/tenda"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete <ip> <inPort> <outPort> <protocol>",
	Short: "Delete an existing port forwarding rule (NAT)",
	Long: `Delete an existing port forwarding rule by specifying the target IP address, 
the internal (inPort) and external (outPort) ports, and the protocol (0=TCP&UDP, 1=TCP, 2=UDP). 
For example:

  tendactl vs delete 192.168.0.100 80 8080 0
  → Removes an existing forwarding rule for TCP&UDP`,
	Args: cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		ip := args[0]
		inPort := args[1]
		outPort := args[2]
		protocol := args[3]

		client := tenda.CreateHTTPClient()
		virtualServerCfg, err := GetVirtualServerCfg(client)
		if err != nil {
			fmt.Printf("Error getting virtual server configuration: %v\n", err)
			return
		}

		newVirtualList := []VirtualEntry{}
		for _, entry := range virtualServerCfg.VirtualList {
			if entry.Ip != ip || entry.InPort != inPort || entry.OutPort != outPort || entry.Protocol != protocol {
				newVirtualList = append(newVirtualList, entry)
			}
		}
		virtualServerCfg.VirtualList = newVirtualList
		err = SetVirtualServerCfg(client, virtualServerCfg)
		if err != nil {
			fmt.Printf("Error setting virtual server configuration: %v\n", err)
			return
		}
		fmt.Println("Deleted port forwarding rule")
	},
}

func init() {
	virtualServerCmd.AddCommand(deleteCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
