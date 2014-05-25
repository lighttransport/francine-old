package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
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
		//`--image=https://www.googleapis.com/compute/v1/projects/gcp-samples/global/images/coreos-v282-0-0`,
		`--image=projects/coreos-cloud/global/images/coreos-alpha-324-1-0-v20140522`,
		`--metadata_from_file=user-data:`+config.cloudConfig,
		`--boot_disk_size_gb=15`,
		`--persistent_boot_disk=true`,
		`--auto_delete_boot_disk=true`)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func DeleteInstance(instance string) error {
	cmd := exec.Command("gcutil", "--project", "gcp-samples", "deleteinstance", instance)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func SendCommand(instance, command string) error {
	cmd := exec.Command("gcutil", "--project", "gcp-samples", "ssh", instance, command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SendFile(instance, from, to string) error {
	cmd := exec.Command("gcutil", "--project", "gcp-samples", "push", instance, from, to)
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

func CreateMaster(instance string) error {
	reqs := []string{"cloud-config-master.yaml"}
	var err error
	if err = CheckFiles(reqs); err != nil {
		return err
	}

	if err = AddInstance(&InstConfig{
		name:        instance,
		//zone:        "us-central1-a",
		zone:        "asia-east1-a",
		machineType: "n1-standard-1",
		network:     "lte-cluster",
		ipType:      "ephemeral",
		cloudConfig: "cloud-config-master.yaml"}); err != nil {
		return err
	}

	return nil
}

func SendCreateWorker(masterInstance string) error {
	return SendCommand(masterInstance,
		"sudo docker run relateiq/redis-cli -h `sudo printenv COREOS_PRIVATE_IPV4` rpush cmd:lte-master create")
}

func RestartWorkers(masterInstance string) error {
	return SendCommand(masterInstance,
		"sudo docker run relateiq/redis-cli -h `sudo printenv COREOS_PRIVATE_IPV4` rpush cmd:lte-master restart_workers")
}

func RestartMaster(masterInstance string) error {
	return SendCommand(masterInstance, "sudo systemctl restart ltemaster.service")
}

func RestartDemo(masterInstance string) error {
	return SendCommand(masterInstance, "sudo systemctl restart ltedemo.service")
}

func UpdateImages(masterInstance string, imageName string) error {
	images := []string{"lte_master", "lte_worker", "lte_demo"}
	if imageName != "" {
		images = []string{imageName}
	}
	ssh := exec.Command("gcutil", "--project", "gcp-samples", "ssh", "--ssh_arg", "-L 5001:localhost:5000", "--ssh_arg", "-n", "--ssh_arg", "-t", "--ssh_arg", "-t", masterInstance)
	ssh.Stdout = os.Stdout
	ssh.Stderr = os.Stderr
	if err := ssh.Start(); err != nil {
		return err
	}

	fmt.Println("Wait 15 seconds for ssh tunneling to connect...")
	time.Sleep(15 * time.Second)

	for _, image := range images {
		cmd := exec.Command("sudo", "docker", "tag", "lighttransport/"+image, "localhost:5001/"+image)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
		cmd = exec.Command("sudo", "docker", "push", "localhost:5001/"+image)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	if err := ssh.Process.Kill(); err != nil {
		return err
	}

	return nil
}

func SendOAuthToken(masterInstance, clientId, clientSecret string) error {
	config := &oauth.Config{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Scope:        "https://www.googleapis.com/auth/compute",
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token"}

	fmt.Printf("Visit the following URL, then type the code: %s\n", config.AuthCodeURL(""))
	fmt.Printf("code: ")

	var code string
	fmt.Scanf("%s", &code)

	transport := &oauth.Transport{Config: config}
	token, err := transport.Exchange(code)
	if err != nil {
		return err
	}

	transport.Token = token

	jsoned, err := json.Marshal(transport)
	if err != nil {
		return err
	}

	encoded := base64.StdEncoding.EncodeToString(jsoned)
	if err = SendCommand(masterInstance, "etcdctl set /gce-oauth-token "+encoded); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [<options>] <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, `commands:
	create_master : Create the master instance in GCE
	update_images : Update docker images
	delete_master : Delete the master instance in GCE
	create_worker : Create a worker instance in GCE
	auth <client id> <client secret> : Register OAuth token for worker instance creation
	restart_workers : Restart worker containers
	restart_master : Restart the master container
	restart_demo : Restart the demo container

How to Setup:
	./ltesetup create_master
	# build demo, master and worker containers first
	./ltesetup update_images
	./ltesetup auth <client id> <client secret>
	./ltesetup create_worker

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
		imageName := ""
		if len(flag.Args()) >= 2 {
			imageName = flag.Args()[1]
		}
		err = UpdateImages("lte-master", imageName)
	case "create_worker":
		err = SendCreateWorker("lte-master")
	case "auth":
		if flag.NArg() < 3 {
			fmt.Fprintf(os.Stderr, "%s: too few arguments\n", os.Args[0])
			flag.Usage()
			os.Exit(1)
		}
		err = SendOAuthToken("lte-master", flag.Args()[1], flag.Args()[2])
	case "restart_workers":
		err = RestartWorkers("lte-master")
	case "restart_master":
		err = RestartMaster("lte-master")
	case "restart_demo":
		err = RestartDemo("lte-master")
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command %s\n", os.Args[0], commandName)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err.Error())
	}

}
