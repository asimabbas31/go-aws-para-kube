package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/sts"
)

// Create a Session using profile values and get the Session Token
func awssess() *session.Session {

	//Enable CONFIG to pick the region from profile
	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	var mfaCode string
	var profilename string
	profilename = os.Args[1]
	fmt.Println("--", profilename, "loaded --")
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile: profilename,
	})

	_iam := iam.New(sess)
	devices, err := _iam.ListMFADevices(&iam.ListMFADevicesInput{})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Println("Error:", awsErr.Code(), awsErr.Message())
		}
	}
	sn := devices.MFADevices[0].SerialNumber

	fmt.Println(awsutil.StringValue(_iam.Config.Credentials))
	svc := sts.New(sess)
	fmt.Println("##ENTER 6 Digits MFA CODE")
	fmt.Scanln(&mfaCode)

	params := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(900),
		SerialNumber:    aws.String(*sn),
		TokenCode:       aws.String(mfaCode),
	}
	resp, err := svc.GetSessionToken(params)

	os.Setenv("AWS_SESSION_TOKEN", awsutil.StringValue(resp.Credentials.SessionToken))
	fmt.Println(awsutil.Prettify(devices.MFADevices[0].UserName), "Logged in Successfully")
	return sess
}

// To Get the Paramaters and Values store in SSM Parameter Store

// To Get the Paramaters and Values store in SSM Parameter Store

func ssid(sess *session.Session) {
	var envvar string
	fmt.Println("Enter Required App Variables Name eg: /dev/appname")
	fmt.Scanln(&envvar)
	ssmsvc := ssm.New(sess)

	param, err := ssmsvc.GetParametersByPath(&ssm.GetParametersByPathInput{
		Path:           aws.String(envvar),
		WithDecryption: aws.Bool(false),
		Recursive:      aws.Bool(true),
		MaxResults:     aws.Int64(10),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Println("Error:", awsErr.Code(), awsErr.Message())
		}
	}

	if param.NextToken != nil {
		fmt.Println("In if nexttoken is not empty")
		fmt.Println("Print NextToken: ", *param.NextToken)
		parameter := &param.Parameters
		fmt.Println(awsutil.Prettify(parameter))
	}

	data, err := ssmsvc.GetParametersByPath(&ssm.GetParametersByPathInput{
		Path:           aws.String(envvar),
		WithDecryption: aws.Bool(true),
		Recursive:      aws.Bool(true),
		MaxResults:     aws.Int64(10),
		NextToken:      aws.String(*param.NextToken),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Println("Error:", awsErr.Code(), awsErr.Message())
		}
	}

	for _, p := range data.Parameters {
		split := strings.Split(*p.Name, "/")
		name := strings.ToUpper(split[len(split)-1])
		fmt.Println(name, ":", *p.Value)
	}

}

// To put the paramerter in parameter store.
func putpara(sess *session.Session) {
	var envname, envvalue, envtype string
	fmt.Println("Supply the Name of the parameter eg: /dev/")
	fmt.Scanln(&envname)
	fmt.Println("Supply the Value of the parameter")
	fmt.Scanln(&envvalue)
	fmt.Println("Supply one of the listed value for Type (String,StringList,SecureString)")
	fmt.Scanln(&envtype)

	ssmsvc := ssm.New(sess)
	input, err := ssmsvc.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(envname),
		Value:     aws.String(envvalue),
		Type:      aws.String(envtype),
		Overwrite: aws.Bool(true),
	})
	if err != nil {
		panic(err)

	}
	fmt.Println("Version:", *input.Version, "added")
}

type clustername struct {
	dev  string
	prod string
}

func k8s(sess *session.Session) {

	svc := eks.New(sess)
	input := &eks.DescribeClusterInput{
		Name: aws.String(os.Args[3]),
	}
	result, err := svc.DescribeCluster(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Println("Error:", awsErr.Code(), awsErr.Message())
		}
	}

	fmt.Println(result.Cluster.Arn)
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	path := (homeDirectory + "/.kube/config")
	filename, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}

	replacer := strings.NewReplacer("DATA", awsutil.Prettify(result.Cluster.CertificateAuthority.Data), "SERVER", awsutil.Prettify(result.Cluster.Endpoint), "CLUSTERNAME", awsutil.Prettify(result.Cluster.Name), "ARN", awsutil.Prettify(result.Cluster.Arn), "MYPROFILE", os.Args[1])

	output := replacer.Replace(string(filename))

	fmt.Println("Your Connection Successful with Cluster", awsutil.Prettify(result.Cluster.Name))
	fmt.Println("\n", "Some Useful K8s command:", "\n", "\n", "To list the PODs:", "\n", "   kubectl get pods --all-namespaces", "\n", "To list the Services:", "\n", "   kubectl get svc --all-namespaces", "\n", "To connect with Pods:", "\n", "kubectl exec -it <PODNAME> --/bin/bash -n <dev or prod>")

	err = ioutil.WriteFile(path, []byte(output), 0)

	if err != nil {
		panic(err)
	}

	//      fmt.Println(lines, endpoint, name, arn)
}

//func s3(sess *session.Session) {
//
//      downloader := s3manager.NewDownloader(sess)
//
//      numBytes, err := downloader.Download(file,
//              &s3.GetObjectInput{
//                      Bucket: aws.String(bucket),
//                      Key:    aws.String(item),
//              })
//      if err != nil {
//              exitErrorf("Unable to download item %q, %v", item, err)
//      }
//
//      fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
//}

func main() {

	// Getting Session
	var sess *session.Session
	sess = awssess()

	// Calling put parameter function
	if os.Args[2] == "add-var" {
		putpara(sess)
	}
	// calling get parameter function
	if os.Args[2] == "get-var" {
		ssid(sess)
	}
	// calling EKS
	if os.Args[2] == "eks" {
		k8s(sess)
	}
}
