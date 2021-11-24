package modules

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sshtunnel/cipherText"
	"sshtunnel/database"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

/*
{
  "regions": [
    {
      "id": "ams",
      "city": "Amsterdam",
      "country": "NL",
      "continent": "Europe",
      "options": [
        "ddos_protection"
      ]
    }
  ],
  "meta": {
    "total": 1,
    "links": {
      "next": "",
      "prev": ""
    }
  }
}
*/
type instanceInfo struct {
	DateCreate string `json:"date_created"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	PublicIP   string `json:"main_ip"`
	V4NetMask  string `json:"netmask_v4,omitempty"`
	Region     string `json:"region"`
	PrivateIP  string `json:"internal_ip,omitempty"`
	PublicV6IP string `json:"v6_main_ip,omitempty"`
	V6NetMask  int    `json:"v6_network_size,omitempty"`
}
type result struct {
	Error     string         `json:"error"`
	Status    int            `json:"status"`
	Instances []instanceInfo `json:"instances"`
	Meta      interface{}    `json:"meta"`
}

const (
	vpsInstanceURL       = "https://api.vultr.com/v2/instances"
	vpsInstanceRegionURL = "https://api.vultr.com/v2/regions"
)

var awsRegions = map[string]string{
	"us-east-2":      "US East (Ohio)",
	"us-east-1":      "US East (N. Virginia)",
	"us-west-1":      "US West (N. California)",
	"us-west-2":      "US West (Oregon)",
	"af-south-1":     "Africa (Cape Town)",
	"ap-east-1":      "Asia Pacific (Hong Kong)",
	"ap-south-1":     "Asia Pacific (Mumbai)",
	"ap-northeast-3": "Asia Pacific (Osaka)",
	"ap-northeast-2": "Asia Pacific (Seoul)",
	"ap-southeast-1": "Asia Pacific (Singapore)",
	"ap-southeast-2": "Asia Pacific (Sydney)",
	"ap-northeast-1": "Asia Pacific (Tokyo)",
	"ca-central-1":   "Canada (Central)",
	"cn-north-1":     "China (Beijing)",
	"cn-northwest-1": "China (Ningxia)",
	"eu-central-1":   "Europe (Frankfurt)",

	"eu-west-1":  "Europe (Ireland)",
	"eu-west-2":  "Europe (London)",
	"eu-south-1": "Europe (Milan)",
	"eu-west-3":  "Europe (Paris)",
	"eu-north-1": "Europe (Stockholm)",
	"me-south-1": "Middle East (Bahrain)",
	"sa-east-1":  "South America (São Paulo)",
}

// get instances hosted in https://my.vultr.com/
func GetVPSInstances() ([]instanceInfo, error) {
	// api url https://www.vultr.com/api/#operation/list-instances
	// VPSKey = "BSH32BR3NGLCHSUGZI3LS6YLEFDRM4222T4A"
	// os.Setenv("VPSKey", "BSH32BR3NGLCHSUGZI3LS6YLEFDRM4222T4A")
	f, err := os.Open(database.DBConFile)
	if err != nil {
		log.Fatal(err)
	}
	d := json.NewDecoder(f)
	var rssc database.RSSHConfig
	if err := d.Decode(&rssc); err != nil {
		log.Fatal("decode failed with error ", err)
	}
	if rssc.VPSKey == "" {
		log.Fatalf("no VPSKey settings found in %s", f.Name())
	}
	os.Setenv("VPSKey", rssc.VPSKey)

	apiKey, ok := os.LookupEnv("VPSKey")
	if !ok {
		return nil, fmt.Errorf("VPSKey not exist in system env, set it first")
	}

	req, err := http.NewRequest("GET", vpsInstanceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res result
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	if res.Status == 401 {
		return nil, errors.New(res.Error)
	}

	return res.Instances, nil
}

// import instances hosted in https://my.vultr.com/
func ImportVPSInstancesToDB(db *sql.DB) {
	// defer db.Close()

	var sql string
	res, err := GetVPSInstances()
	if err != nil {
		log.Fatal("get vps instances info failed with error ", err)
	}

	for _, instance := range res {
		if !strings.Contains(strings.ToLower(instance.Label), "test") {

			if strings.Contains(strings.ToLower(instance.Label), "ssh") {
				sql = fmt.Sprintf(`INSERT INTO instances
				(INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT) 
				values 
				('%s','%s','%s','%s','%s')`, instance.Label, instance.PublicIP, instance.PrivateIP, instance.Region, "ssh")
			} else if strings.Contains(strings.ToLower(instance.Label), "turn") {
				sql = fmt.Sprintf(`INSERT INTO instances 
				(INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT) 
				values 
				('%s','%s','%s','%s','%s')`, instance.Label, instance.PublicIP, instance.PrivateIP, instance.Region, "turn")
			} else {
				continue
			}

			if database.IsRecordExist(db, instance.PublicIP) {
				log.Printf("%s is Exist in db, update its instance name", instance.PublicIP)
				sql = fmt.Sprintf("UPDATE  instances SET INSTANCE_NAME = '%s' where PUBLIC_IP ='%s';", instance.Label, instance.PublicIP)
			}

			if err := database.DBExecute(db, sql); err != nil {
				log.Fatal(err)
			}
		}
	}
	log.Println("import VPS instances to DB successfully.")
}

func GetAWSInstances(project, region string) ([]map[string]string, error) {
	client := newEC2Client(project, region)
	input := ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "pending"},
			},
		},
	}

	out, err := client.DescribeInstances(context.TODO(), &input)
	if err != nil {
		return nil, err
	}

	result := []map[string]string{}
	for _, rs := range out.Reservations {
		for _, instance := range rs.Instances {
			tagIP := make(map[string]string)
			for _, t := range instance.Tags {
				tagIP["Name"] = aws.ToString(t.Value)
				tagIP["PublicIP"] = aws.ToString(instance.PublicIpAddress)
				tagIP["PrivateIP"] = aws.ToString(instance.PrivateIpAddress)
				tagIP["Region"] = awsRegions[region]

			}
			result = append(result, tagIP)
		}
	}
	// fmt.Println(result)
	return result, nil
}

func newEC2Client(project, region string) *ec2.Client {
	// about aws credentials refer to https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
	var ec2Client *ec2.Client
	awsConfig := path.Join(os.Getenv("HOME"), ".aws/config")
	if _, err := os.Stat(awsConfig); !os.IsNotExist(err) {
		os.Setenv("AWS_PROFILE", strings.Join([]string{project, "account"}, "-"))
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
		// ac := strings.Join([]string{project, ""}, "-")
		// cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(ac))
		if err != nil {
			panic(err)
		}
		ec2Client = ec2.NewFromConfig(cfg)
	} else {
		log.Printf("%s not readable or not exist", awsConfig)
		return nil
	}

	return ec2Client
}

// import instances hosted in Amazon to db
func ImportAWSInstancesToDB(db *sql.DB, project, region string) {
	// defer db.Close()

	res, err := GetAWSInstances(project, region)
	if err != nil {
		log.Fatal(err)
	}
	for _, instance := range res {
		sql := fmt.Sprintf(`INSERT INTO instances 
		(INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT) 
		values 
		('%s','%s','%s','%s','%s')`, instance["Name"], instance["PublicIP"], instance["PrivateIP"], instance["Region"], project)

		if database.IsRecordExist(db, instance["PublicIP"]) {
			log.Printf("%s is Exist in db, update its instance name", instance["PublicIP"])
			sql = fmt.Sprintf("UPDATE instances SET INSTANCE_NAME = '%s' where PUBLIC_IP ='%s';", instance["Name"], instance["PublicIP"])
		}
		if err := database.DBExecute(db, sql); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("import aws instance to DB successfully.")
}

// import ssh key to specified db and use `passphrase` as key to encrypt ssh key content
// encrypt program:
// [string --> encrypted --> base64 encode --> db]
func ImportSSHKey(db *sql.DB, keyFile, ssh_user, passphrase string) {
	buf, err := os.ReadFile(keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// encrypted key
	econtent, err := cipherText.EncryptData(buf, passphrase)
	if err != nil {
		log.Fatal(err)
	}
	// c := base64.StdEncoding.EncodeToString(econtent)
	// sql := fmt.Sprintf("INSERT INTO sshkeys (project, privateKey_name, privateKey_content, ssh_user) values ('%s','%s', '%s', '%s')", strings.TrimSuffix(keyFile, ".pem"), path.Base(keyFile), econtent, ssh_user)
	sql := fmt.Sprintf(`INSERT INTO sshkeys (project, privateKey_name, privateKey_content, ssh_user) 
	values 
	('%s','%s', '%s', '%s')`, strings.TrimSuffix(keyFile, ".pem"), path.Base(keyFile), econtent, ssh_user)

	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
}

// return sshkey map
// decrypted program:
// [encyptedString --> base64 decode --> decrypted --> return (ssh_user,private_key)]
func GetSSHKey(db *sql.DB, project, passphrase string) (string, string) {
	sql := fmt.Sprintf("SELECT ssh_user, privateKey_content FROM sshkeys WHERE project='%s'", project)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal("query sql failed with error: ", err)
	}
	defer rows.Close()

	// sshKey := map[string]string{}
	var sshUser, privateKeyContent string
	for rows.Next() {
		// var sshUser, privateKeyContent string
		rows.Scan(&sshUser, &privateKeyContent)
		key, err := cipherText.DecryptData(privateKeyContent, passphrase)
		if err != nil {
			log.Fatal(err)
		}
		// sshKey[project] = string(key)
		privateKeyContent = string(key)
	}
	// fmt.Println(sshKey)
	return sshUser, privateKeyContent
}

// import jumper host info to db
func ImportJumperHosts(db *sql.DB, jumpHost, jumpUser, jumpPass, jumpPort, passphrase string) {
	pass, err := cipherText.EncryptData([]byte(jumpPass), passphrase)
	if err != nil {
		log.Fatal(err)
	}
	sql := fmt.Sprintf(`INSERT INTO jumperHosts (jmphost, jmpuser, jmppass, jmpport)
	values
	('%s','%s','%s','%s')`, jumpHost, jumpUser, pass, jumpPort)

	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
}
