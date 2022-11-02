package main

import (
	"fmt"
	"math"
	"strings"
)

const EC2Region = "ca-central-1"

func main() {
	// Set up AWS
	cfg := GetAWSConfig(EC2Region)

	// Get a list of all instances with some selected properties: name, instance type, CPU cores, gp2/gp3 size
	// Grouped by networks (cosmoshub, regen, etc)
	instancesByGroup := NewInstancesByGroup(cfg)

	fmt.Println("Group,Instance_Num,Instance_type,Instance_cost,EBS_Cost,Traffic_Cost,TotalCost,Traffic_GB")

	totalInstanceCost := 0.0
	totalEBSCost := 0.0
	totalTrafficCost := 0.0
	totalTrafficGB := 0.0
	for groupName, instances := range instancesByGroup {
		totalInstanceCostForGroup := 0.0
		totalEBSCostForGroup := 0.0
		totalTrafficCostForGroup := 0.0
		totalTrafficGBForGroup := 0.0

		var instanceTypes []string

		for _, instance := range instances {
			instanceTypes = append(instanceTypes, instance.Type)
			totalInstanceCostForGroup += instance.GetInstanceCostFor30Days()
			totalEBSCostForGroup += instance.GetEBSCostForMonth()
			trafficCost, trafficGB := instance.GetTrafficCostFor30Days(cfg)
			if math.IsNaN(trafficGB) {
				trafficCost = 0
				trafficGB = 0
			}
			totalTrafficCostForGroup += trafficCost
			totalTrafficGBForGroup += trafficGB
		}

		fmt.Printf("%s,%d,%s,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			groupName,
			len(instances),
			strings.Join(instanceTypes, ";"),
			totalInstanceCostForGroup,
			totalEBSCostForGroup,
			totalTrafficCostForGroup,
			totalInstanceCostForGroup+totalEBSCostForGroup+totalTrafficCostForGroup,
			totalTrafficGBForGroup,
		)
		totalInstanceCost += totalInstanceCostForGroup
		totalEBSCost += totalEBSCostForGroup
		totalTrafficCost += totalTrafficCostForGroup
		totalTrafficGB += totalTrafficGBForGroup
	}
	fmt.Printf("Nodes total,,,%.2f,%.2f,%.2f,%.2f,%.2f\n", totalInstanceCost, totalEBSCost, totalTrafficCost, totalInstanceCost+totalEBSCost+totalTrafficCost, totalTrafficGB)
	fmt.Println("S3,Backup/Config,,,,,?")
	fmt.Println("VPC_cross-traffic,VPN,,,,,?")
	fmt.Println("Route_53,Resolver,,,,,?")
	fmt.Println("RDS,Zabbix,,,,,?")
	fmt.Println("ELB,Nautilus,,,,,?")
	fmt.Println("Tax,,,,,,?")
	fmt.Println("Total,,,,,,?")
}
