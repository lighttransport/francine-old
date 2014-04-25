package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func AddInstance(instance string) error {
	const (
		zone        = "us-central1-a"
		machineType = "f1-micro"
		image       = "https://www.googleapis.com/compute/v1/projects/gcp-samples/global/images/coreos-v282-0-0"
	)

	cmd := exec.Command("gcutil",
		`--service_version=v1`,
		`--project=gcp-samples`,
		"addinstance",
		instance,
		`--zone=`+zone,
		`--machine_type=`+machineType,
		`--network=default`,
		`--external_ip_address=ephemeral`,
		`--service_account_scopes=https://www.googleapis.com/auth/userinfo.email,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control`,
		`--image=`+image,
		`--persistent_boot_disk=true`,
		`--auto_delete_boot_disk=false`)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func DeleteInstance(instance string) error {
	cmd := exec.Command("gcutil", "deleteinstance", instance)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SendCommand(instance string, command string) error {
	cmd := exec.Command("gcutil", "ssh", instance, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SendFile(instance string, from string, to string) error {
	cmd := exec.Command("gcutil", "push", instance, from, to)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func GenerateInstanceName() string {
	return "lte-instance-" + time.Now().Format("20060102150405")
}

func SetupLte(instance string, containerDir string) error {
	err := SendFile(instance, containerDir + "/lighttransport-lte_bin.tar.gz", "lighttransport-lte_bin.tar.gz")
	if err != nil {
		return err
	}

	err = SendFile(instance, containerDir + "/setup_coreos.sh", "setup_coreos.sh")
	if err != nil {
		return err
	}

	err = SendCommand(instance, "chmod +x ./setup_coreos.sh")
	if err != nil {
		return err
	}

	err = SendCommand(instance, "./setup_coreos.sh")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	containerDir := "../builder/"

	name := GenerateInstanceName()

	{
		path := containerDir + "/lighttransport-lte_bin.tar.gz";
		if  _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("%s does not exist; build LTE container first\n", path)
			os.Exit(1)
		}
	}

	if err := AddInstance(name); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := SetupLte(name, containerDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
