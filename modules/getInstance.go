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
type SSHKeyInfo struct {
	SSHUser           string
	SSHPass           string
	SSHPort           string
	PrivateKeyContent string
	PrivateKeyName    string
	Project           string
}
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

type SecurityGroupInfo struct {
	SGName string
	SGID   string
	SGDesc string
	// SGCidrs []types.IpPermission
	SGCidrs []string
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
	var rssc database.RSSHConfig

	f, err := os.Open(database.DBConFile)
	if err != nil {
		return nil, err
	}
	d := json.NewDecoder(f)
	if err := d.Decode(&rssc); err != nil {
		return nil, fmt.Errorf("failed to decode %s with error %s", f.Name(), err)
	}
	if rssc.VPSKey == "" {
		return nil, fmt.Errorf("no VPSKey settings found in %s", f.Name())
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
func ImportVPSInstancesToDB(db *sql.DB) error {
	// defer db.Close()

	var sql string
	res, err := GetVPSInstances()
	if err != nil {
		return err
	}

	for _, instance := range res {
		if !strings.Contains(strings.ToLower(instance.Label), "test") {

			if strings.Contains(strings.ToLower(instance.Label), "ssh") {
				sql = fmt.Sprintf(`INSERT INTO %s (INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT,PLATFORM) 
				values ('%s','%s','%s','%s','%s','%s')`,
					database.InstanceTableName,
					instance.Label,
					instance.PublicIP,
					instance.PrivateIP,
					instance.Region, "ssh", "vps")
			} else if strings.Contains(strings.ToLower(instance.Label), "turn") {
				sql = fmt.Sprintf(`INSERT INTO %s (INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT,PLATFORM) 
				values ('%s','%s','%s','%s','%s','%s')`,
					database.InstanceTableName,
					instance.Label,
					instance.PublicIP,
					instance.PrivateIP,
					instance.Region, "turn", "vps")
			} else {
				sql = fmt.Sprintf(`INSERT INTO %s (INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP, REGION, PROJECT,PLATFORM) 
				values ('%s','%s','%s','%s','%s','%s')`,
					database.InstanceTableName,
					instance.Label,
					instance.PublicIP,
					instance.PrivateIP,
					instance.Region, "other", "vps")
			}

			if database.IsRecordExist(db, instance.PublicIP) {
				log.Printf("%s is Exist in db, update its instance name", instance.PublicIP)
				sql = fmt.Sprintf("UPDATE  %s SET INSTANCE_NAME = '%s' where PUBLIC_IP ='%s';", database.InstanceTableName, instance.Label, instance.PublicIP)
			}

			if err := database.DBExecute(db, sql); err != nil {
				return fmt.Errorf("failed to import instances from VPS with error %s", err)
			}
		}
	}
	log.Println("import VPS instances to DB successfully.")
	return nil
}

func GetAWSInstances(project, region string) ([]map[string]string, error) {
	client, err := newEC2Client(project, region)
	if err != nil {
		return nil, err
	}
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
				tagIP["InstanceType"] = string(instance.InstanceType)
				tagIP["InstanceID"] = aws.ToString(instance.InstanceId)
				tagIP["Region"] = awsRegions[region]
				// tagIP["SecurityGroupID"] = aws.ToString(instance.SecurityGroups[0].GroupId)

			}
			result = append(result, tagIP)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("there's no instance found in %s for %s", awsRegions[region], project)
	}
	return result, nil
}

func newEC2Client(project, region string) (*ec2.Client, error) {
	// about aws credentials refer to https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/
	var ec2Client *ec2.Client
	awsConfig := path.Join(os.Getenv("HOME"), ".aws/credentials")
	if _, err := os.Stat(awsConfig); !os.IsNotExist(err) {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(project), config.WithRegion(region))
		// cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(ac))
		if err != nil {
			return nil, err
		}
		ec2Client = ec2.NewFromConfig(cfg)
	} else {
		return nil, fmt.Errorf("%s not readable or not exist", awsConfig)
	}

	return ec2Client, nil
}

// import instances hosted in Amazon to db
func ImportAWSInstancesToDB(db *sql.DB, project, region string) error {
	res, err := GetAWSInstances(project, region)
	if err != nil {
		return err
	}
	for _, instance := range res {
		sql := fmt.Sprintf(`INSERT INTO %s (INSTANCE_NAME, PUBLIC_IP, PRIVATE_IP,INSTANCE_TYPE,INSTANCE_ID, REGION, PROJECT,PLATFORM) 
		values ('%s','%s','%s','%s','%s','%s','%s','%s')`,
			database.InstanceTableName,
			instance["Name"],
			instance["PublicIP"],
			instance["PrivateIP"],
			instance["InstanceType"],
			instance["InstanceID"],
			instance["Region"],
			project, "aws")

		if err := database.DBExecute(db, sql); err != nil {
			return fmt.Errorf("failed to import instances from AWS with error %s", err)
		}
	}
	log.Println("import aws instance to DB successfully.")
	return nil
}

// update instance list stored in db
func UpdateInstanceListsInDB(db *sql.DB, project, region string) {

	// update ssh instances placed in VPS if update gdms's instances
	switch project {
	case "ssh", "turn":
		sql := fmt.Sprintf("DELETE FROM %s  WHERE PROJECT = '%s';", database.InstanceTableName, project)
		if err := database.DBExecute(db, sql); err != nil {
			log.Fatal(err)
		}
		if err := ImportVPSInstancesToDB(db); err != nil {
			log.Fatal(err)
		}
	default:
		sql := fmt.Sprintf("DELETE FROM %s  WHERE PROJECT = '%s' and REGION = '%s';", database.InstanceTableName, project, awsRegions[region])
		if err := database.DBExecute(db, sql); err != nil {
			log.Fatal(err)
		}
		if err := ImportAWSInstancesToDB(db, project, region); err != nil {
			log.Fatal(err)
		}
	}

}

// import ssh key to specified db and use `passphrase` as key to encrypt ssh key content
// encrypt program:
// [string --> encrypted --> base64 encode --> db]
func ImportSSHAuthentication(db *sql.DB, keyFile, project, ssh_user, ssh_port, ssh_password, passphrase string) {
	// var ePass, eKey, project, privateKeyName string
	var ePass, eKey, privateKeyName string
	var err error
	// encrypted ssh password
	if ssh_password != "" {
		ePass, err = cipherText.EncryptData([]byte(ssh_password), passphrase)
		if err != nil {
			log.Fatal(err)
		}
	}

	// encrypted key
	if keyFile != "" {
		buf, err := os.ReadFile(keyFile)
		if err != nil {
			log.Fatal(err)
		}
		eKey, err = cipherText.EncryptData(buf, passphrase)
		if err != nil {
			log.Fatal(err)
		}
		if project == "" {
			project = strings.TrimSuffix(path.Base(keyFile), ".pem")
		}

		privateKeyName = path.Base(keyFile)
	}

	// c := base64.StdEncoding.EncodeToString(econtent)
	sql := fmt.Sprintf(`INSERT INTO %s (project, privateKey_name, privateKey_content, ssh_user, ssh_port, ssh_password) 
	values 
	('%s','%s', '%s', '%s', '%s', '%s')`, database.SSHKeyTableName, project, privateKeyName, eKey, ssh_user, ssh_port, ePass)

	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
}

/* func ImportSSHPassword(db *sql.DB, project, ssh_password, ssh_user, ssh_port, passcode string) {
	econtent, err := cipherText.EncryptData([]byte(ssh_password), passcode)
	if err != nil {
		log.Fatal(err)
	}

	sql := fmt.Sprintf(`INSERT INTO sshkeys (project, privateKey_name, privateKey_content, ssh_user, ssh_port)
	values
	('%s','%s', '%s', '%s', '%s')`, project, strings.Join([]string{project, "pass"}, "."), econtent, ssh_user, ssh_port)

	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
} */

// return sshkey map
// decrypted program:
// [encyptedString --> base64 decode --> decrypted --> return (ssh_user,private_key)]
func GetSSHKey(db *sql.DB, project, passphrase string) SSHKeyInfo {
	sql := fmt.Sprintf("SELECT privateKey_name,ssh_user, privateKey_content, ssh_password FROM %s WHERE project='%s'", database.SSHKeyTableName, project)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal("query sql failed with error: ", err)
	}
	defer rows.Close()

	// sshKey := map[string]string{}
	sshkey := SSHKeyInfo{}
	// var privateKey_name, sshUser, privateKeyContent string
	for rows.Next() {
		var privateKeyContent, sshPasswd string
		// rows.Scan(&privateKey_name, &sshUser, &privateKeyContent)
		rows.Scan(&sshkey.PrivateKeyName, &sshkey.SSHUser, &privateKeyContent, &sshPasswd)
		eKey, err := cipherText.DecryptData(privateKeyContent, passphrase)
		if err != nil {
			log.Fatal(err)
		}
		ePass, err := cipherText.DecryptData(privateKeyContent, passphrase)
		if err != nil {
			log.Fatal(err)
		}
		sshkey.PrivateKeyContent = string(eKey)
		sshkey.SSHPass = string(ePass)
	}
	// fmt.Println(sshKey)
	// return privateKey_name, sshUser, privateKeyContent
	return sshkey
}

// import jumper host info to db
func ImportJumperHosts(db *sql.DB, jumpHost, jumpUser, jumpPass, jumpKeyFile, jumpPort, passphrase string) {
	var (
		jPass, jKey string
		err         error
	)
	if jumpPass != "" {
		jPass, err = cipherText.EncryptData([]byte(jumpPass), passphrase)
		if err != nil {
			log.Fatal(err)
		}
	}
	// encrypted key
	if jumpKeyFile != "" {

		buf, err := os.ReadFile(jumpKeyFile)
		if err != nil {
			log.Fatal(err)
		}
		jKey, err = cipherText.EncryptData(buf, passphrase)
		if err != nil {
			log.Fatal(err)
		}
	}

	sql := fmt.Sprintf(`INSERT INTO %s (jmphost, jmpuser, jmppass,jmpkey, jmpport)
	values
	('%s','%s','%s','%s','%s')`, database.JumpHostsTableName, jumpHost, jumpUser, jPass, jKey, jumpPort)

	if err := database.DBExecute(db, sql); err != nil {
		log.Fatal(err)
	}
}

// add spcified port to aws security-group so you can access aws resources
// revoke the rule after finishing the ssh connection
func AddSpcifiedPortToAWSSGRP(client *ec2.Client, CidrIpv4, grpid string, port int32) {
	defer func(client *ec2.Client) {
		rks := ec2.RevokeSecurityGroupIngressInput{
			CidrIp:   aws.String(CidrIpv4),
			GroupId:  aws.String(grpid),
			FromPort: aws.Int32(port),
			ToPort:   aws.Int32(port)}
		output, err := client.RevokeSecurityGroupIngress(context.TODO(), &rks)
		if err != nil {
			log.Fatal(err)
		}
		if aws.ToBool(output.Return) {
			log.Printf("succeed to revoke SecurityGroup rules for %s on port %d\n", CidrIpv4, port)
		}
	}(client)

	sgrr := types.SecurityGroupRuleRequest{
		CidrIpv4: aws.String(CidrIpv4),
		// CidrIpv6: ,
		Description: aws.String("remoteSsh auto add"),
		FromPort:    aws.Int32(port),
		IpProtocol:  aws.String("tcp"),
		ToPort:      aws.Int32(port),
	}
	ms := ec2.ModifySecurityGroupRulesInput{
		GroupId: aws.String(grpid),
		SecurityGroupRules: []types.SecurityGroupRuleUpdate{
			{
				SecurityGroupRule: &sgrr,
				// SecurityGroupRuleId: aws.String(sgid),
			},
		},
	}
	output, err := client.ModifySecurityGroupRules(context.TODO(), &ms)
	if err != nil {
		log.Fatal(err)
	}
	if aws.ToBool(output.Return) {
		log.Printf("succeed to add SecurityGroup rule for %s on port %d\n", CidrIpv4, port)
	}
}

// get AWS SecurityGroup info by SecurityGroup ID
func GetAWSSecuirtyGroupInfo(client *ec2.Client, grpid string) SecurityGroupInfo {
	sg := SecurityGroupInfo{}

	sgi := ec2.DescribeSecurityGroupsInput{Filters: []types.Filter{{Name: aws.String("group-id"), Values: []string{grpid}}}}
	output, err := client.DescribeSecurityGroups(context.TODO(), &sgi)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(JsonOutput(output.SecurityGroups))
	for _, k := range output.SecurityGroups {
		sg.SGID = aws.ToString(k.GroupId)
		sg.SGName = aws.ToString(k.GroupName)
		sg.SGDesc = aws.ToString(k.Description)
		// fmt.Println(aws.ToString(k.GroupName), aws.ToString(k.GroupId), aws.ToString(k.Description))
		for _, t := range k.IpPermissions {
			// fmt.Printf("\t%s\t%d\n", aws.ToString(t.IpProtocol), aws.ToInt32(t.FromPort))
			for _, ip := range t.IpRanges {
				// fmt.Printf("\t\t%s\n", aws.ToString(ip.CidrIp))
				sg.SGCidrs = append(sg.SGCidrs, aws.ToString(ip.CidrIp))
			}
		}
		// sg.SGCidrs = k.IpPermissions

	}

	/* sgr := ec2.DescribeSecurityGroupRulesInput{}
	output, err := client.DescribeSecurityGroupRules(context.TODO(), &sgr)
	if err != nil {
		log.Fatal(err)
	}
	for _, k := range output.SecurityGroupRules {
		fmt.Println(aws.ToString(k.CidrIpv4), aws.ToString(k.Description), aws.ToString(k.GroupId), aws.ToString(k.IpProtocol), aws.ToString(k.SecurityGroupRuleId),
			aws.ToInt32(k.FromPort))
	} */
	return sg
}
