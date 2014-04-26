package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"log"
	"io/ioutil"
	"net/http"
	"strings"
)

type InstConfig struct {
	name        string
	zone        string
	machineType string
	network     string
	ipType      string
	cloudConfig string
}

func AddInstance(config *InstConfig) error {
	cmd := exec.Command("gcutil",
		`--service_version=v1`,
		`--project=gcp-samples`,
		"addinstance",
		config.name,
		`--zone=`+config.zone,
		`--machine_type=`+config.machineType,
		`--network=`+config.network,
		`--external_ip_address=`+config.ipType,
		`--service_account_scopes=https://www.googleapis.com/auth/userinfo.email,https://www.googleapis.com/auth/compute,https://www.googleapis.com/auth/devstorage.full_control`,
		`--image=https://www.googleapis.com/compute/v1/projects/gcp-samples/global/images/coreos-v282-0-0`,
		`--metadata_from_file=user-data:`+config.cloudConfig,
		`--persistent_boot_disk=true`,
		`--auto_delete_boot_disk=true`)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func DeleteInstance(instance string) error {
	cmd := exec.Command("gcutil", "deleteinstance", instance)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func SendCommand(instance, command string) error {
	cmd := exec.Command("gcutil", "ssh", instance, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SendFile(instance, from, to string) error {
	cmd := exec.Command("gcutil", "push", instance, from, to)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func CheckFiles(files []string) error {
	for _, path := range files {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("file %s, required for this operation, not found", path))
		}
	}
	return nil
}

func UpdateImages(instance string) error {
	reqs := []string{
		"../master/lighttransport-lte_master.tar.gz",
		"../builder/lighttransport-lte_bin.tar.gz",
		"setup_master.sh"}
	if err := CheckFiles(reqs); err != nil {
		return err
	}

	if err := SendFile(instance,
		"../master/lighttransport-lte_master.tar.gz",
		"lighttransport-lte_master.tar.gz"); err != nil {
		return err
	}
	if err := SendCommand(instance, "mkdir -p lte_bin"); err != nil {
		return err
	}
	if err := SendFile(instance,
		"../builder/lighttransport-lte_bin.tar.gz",
		"lte_bin/lighttransport-lte_bin.tar.gz"); err != nil {
		return err
	}
	if err := SendFile(instance, "setup_master.sh", "setup_master.sh"); err != nil {
		return err
	}
	if err := SendCommand(instance, "chmod +x setup_master.sh"); err != nil {
		return err
	}
	if err := SendCommand(instance, "./setup_master.sh"); err != nil {
		return err
	}

	return nil
}

func GetToken() (string, error) {
	resp, err := http.Get("https://discovery.etcd.io/new")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func CreateMaster(instance string) error {
	reqs := []string{
		"../master/lighttransport-lte_master.tar.gz",
		"../builder/lighttransport-lte_bin.tar.gz",
		"cloud-config.yaml",
		"setup_master.sh"}
	var err error
	if err = CheckFiles(reqs); err != nil {
		return err
	}

	prev, err := ioutil.ReadFile("cloud-config.yaml")
	if err != nil {
		return err
	}

	token, err := GetToken()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile("cloud-config.yaml",
		[]byte(strings.Replace(string(prev), "<token_url>", token, -1)), 0644); err != nil {
		return err
	}
	defer ioutil.WriteFile("cloud-config.yaml", prev, 0644)


	if err = AddInstance(&InstConfig{
		name:        instance,
		zone:        "us-central1-a",
		machineType: "n1-standard-1",
		network:     "lte-cluster",
		ipType:      "ephemeral",
		cloudConfig: "cloud-config.yaml"}); err != nil {
		return err
	}

	for i := 0; i < 5; i++ {
		if err = SendCommand(instance, "etcdctl set /token-url " + token); err != nil {
			fmt.Println("failed, try again")
		}
	}

	if err != nil {
		return err
	}

	if err = UpdateImages(instance); err != nil {
		return err
	}

	return nil
}

func SendCreateWorker(masterInstance string) error { 
	if err := SendCommand(masterInstance,
		"sudo docker run relateiq/redis-cli -h `sudo printenv COREOS_PRIVATE_IPV4` rpush worker-q create"); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [<options>] <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, `commands:
	create_master
	delete_master
	create_worker
	update_images
`)
	// create_network
	// delete_worker

		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	commandName := flag.Args()[0]

	var err error
	switch commandName {
	/*case "create_network":
		err = CreateNetwork("lte-cluster")*/
	case "create_master":
		err = CreateMaster("lte-master")
	case "delete_master":
		err = DeleteInstance("lte-master")
	case "update_images":
		err = UpdateImages("lte-master")
	case "create_worker":
		err = SendCreateWorker("lte-master")
	default:
		log.Fatalf("%s: unknown command %s\n", os.Args[0], commandName)
	}

	if err != nil {
		log.Fatal(err)
	}

}
