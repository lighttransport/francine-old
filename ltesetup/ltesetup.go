package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
		`--image=https://www.googleapis.com/compute/v1/projects/gcp-samples/global/images/coreos-v282-0-0`,
		`--metadata_from_file=user-data:`+config.cloudConfig,
		`--boot_disk_size_gb=15`,
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
	reqs := []string{"cloud-config-master.yaml"}
	var err error
	if err = CheckFiles(reqs); err != nil {
		return err
	}

	prev, err := ioutil.ReadFile("cloud-config-master.yaml")
	if err != nil {
		return err
	}

	token, err := GetToken()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile("cloud-config-master.yaml",
		[]byte(strings.Replace(string(prev), "<token_url>", token, -1)), 0644); err != nil {
		return err
	}
	defer ioutil.WriteFile("cloud-config-master.yaml", prev, 0644)

	if err = AddInstance(&InstConfig{
		name:        instance,
		zone:        "us-central1-a",
		machineType: "n1-standard-1",
		network:     "lte-cluster",
		ipType:      "ephemeral",
		cloudConfig: "cloud-config-master.yaml"}); err != nil {
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

func UpdateImages(masterInstance string, imageName string) error {
	images := []string{"lte_master", "lte_worker", "lte_demo"}
	if imageName != "" {
		images = []string{imageName}
	}
	ssh := exec.Command("gcutil", "ssh", "--ssh_arg", "-L 5000:localhost:5000", "--ssh_arg", "-n", "--ssh_arg", "-t", "--ssh_arg", "-t", masterInstance)
	ssh.Stdout = os.Stdout
	ssh.Stderr = os.Stderr
	if err := ssh.Start(); err != nil {
		return err
	}

	fmt.Println("Wait 15 seconds for ssh tunneling to connect...")
	time.Sleep(15 * time.Second)

	for _, image := range images {
		cmd := exec.Command("sudo", "docker", "tag", "lighttransport/"+image, "localhost:5000/"+image)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			return err
		}
		cmd = exec.Command("sudo", "docker", "push", "localhost:5000/"+image)
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
	create_master
	update_images
	delete_master
	create_worker
	auth <client id> <client secret>

How to Setup:
	./ltesetup create_master
	# build demo, master and worker containers first
	./ltesetup update_images
	./ltesetup auth <client id> <client secret>
	./ltesetup create_worker
`)
		// gcutil ssh --ssh_arg "-L 5000:localhost:5000" lte-master
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
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown command %s\n", os.Args[0], commandName)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err.Error())
	}

}
