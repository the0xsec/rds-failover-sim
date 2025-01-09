package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/fatih/color"
)

type SimulationConfig struct {
	DBIdentifier string `json:"dbIdentifier"`
	Endpoint     string `json:"endpoint"`
	Port         int    `json:"port"`
}

var (
	success = color.New(color.FgGreen, color.Bold)
	warning = color.New(color.FgYellow, color.Bold)
	failure = color.New(color.FgRed, color.Bold)
	info    = color.New(color.FgCyan)
	header  = color.New(color.FgMagenta, color.Bold)
)

func main() {
	header.Println("RDS Failover Simulation Tool")

	configFile := flag.String("config", "config.json", "Path to configuration file")
	mode := flag.String("mode", "monitor", "Simulation mode: monitor or failover")
	flag.Parse()

	info.Printf("Mode: %s\n", *mode)

	simConfig, err := loadConfig(*configFile)
	if err != nil {
		failure.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
	}

	switch *mode {
	case "monitor":
		err = monitorDatabase(simConfig)
	case "failover":
		err = triggerFailover(simConfig)
	default:
		failure.Fprintf(os.Stderr, "Invalid mode specified")
	}

	if err != nil {
		failure.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func loadConfig(path string) (*SimulationConfig, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var simConfig SimulationConfig
	if err := json.Unmarshal(file, &simConfig); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &simConfig, nil
}

func monitorDatabase(simConfig *SimulationConfig) error {
	success.Println("Starting RDS monitoring...")
	info.Printf("Instance: %s\n", simConfig.DBIdentifier)
	info.Printf("Endpoint: %s\n", simConfig.Endpoint)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %v", err)
	}

	rdsClient := rds.NewFromConfig(cfg)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		status, az, err := getRDSStatus(rdsClient, simConfig.DBIdentifier)
		if err != nil {
			failure.Printf("Error checking RDS: %v\n", err)
		} else {
			printStatus(status, az)
		}
	}

	return nil
}

func printStatus(status, az string) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] ", timestamp)

	switch status {
	case "available":
		success.Printf("%-12s", status)
	case "rebooting", "modifying":
		warning.Printf("%-12s", status)
	default:
		failure.Printf("%-12s", status)
	}

	info.Printf(" AZ: %s\n", az)
}

func getRDSStatus(client *rds.Client, dbIdentifier string) (string, string, error) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &dbIdentifier,
	}

	result, err := client.DescribeDBInstances(context.TODO(), input)
	if err != nil {
		return "", "", err
	}

	if len(result.DBInstances) == 0 {
		return "", "", fmt.Errorf("no DB instance found")
	}

	instance := result.DBInstances[0]
	return *instance.DBInstanceStatus, *instance.AvailabilityZone, nil
}

func triggerFailover(simConfig *SimulationConfig) error {
	warning.Println("Initiating failover simulation...")
	info.Printf("Target instance: %s\n", simConfig.DBIdentifier)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %v", err)
	}

	rdsClient := rds.NewFromConfig(cfg)

	input := &rds.RebootDBInstanceInput{
		DBInstanceIdentifier: &simConfig.DBIdentifier,
		ForceFailover:        aws.Bool(true),
	}

	_, err = rdsClient.RebootDBInstance(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to trigger failover: %v", err)
	}

	success.Println("Failover initiated successfully")
	return monitorFailoverStatus(rdsClient, simConfig.DBIdentifier)
}

func monitorFailoverStatus(client *rds.Client, dbIdentifier string) error {
	header.Println("Monitoring failover status...")

	for {
		input := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: &dbIdentifier,
		}

		result, err := client.DescribeDBInstances(context.TODO(), input)
		if err != nil {
			return fmt.Errorf("error describing DB instance: %v", err)
		}

		if len(result.DBInstances) == 0 {
			return fmt.Errorf("no DB instance found")
		}

		status := *result.DBInstances[0].DBInstanceStatus
		printFailoverStatus(status)

		if status == "available" {
			success.Println("Failover completed successfully")
			return nil
		}

		time.Sleep(10 * time.Second)
	}
}

func printFailoverStatus(status string) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] Status: ", timestamp)

	switch status {
	case "available":
		success.Println(status)
	case "rebooting", "modifying":
		warning.Println(status)
	default:
		failure.Println(status)
	}
}
