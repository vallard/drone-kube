package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	utilyaml "k8s.io/kubernetes/pkg/util/yaml"
)

type (
	Repo struct {
		Owner string
		Name  string
	}

	Build struct {
		Tag     string
		Event   string
		Number  int
		Commit  string
		Ref     string
		Branch  string
		Author  string
		Status  string
		Link    string
		Started int64
		Created int64
	}

	Job struct {
		Started int64
	}

	Config struct {
		Ca        string
		Server    string
		Token     string
		Namespace string
		Template  string
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Config Config
		Job    Job
	}
)

func (p Plugin) Exec() error {

	if p.Config.Server == "" {
		log.Fatal("KUBE_SERVER is not defined")
	}
	if p.Config.Token == "" {
		log.Fatal("KUBE_TOKEN is not defined")
	}
	if p.Config.Ca == "" {
		log.Fatal("KUBE_CA is not defined")
	}
	if p.Config.Namespace == "" {
		p.Config.Namespace = "default"
	}
	if p.Config.Template == "" {
		log.Fatal("KUBE_TEMPLATE, or template must be defined")
	}

	// connect to Kubernetes
	clientset, err := p.createKubeClient()
	if err != nil {
		log.Fatal(err.Error())
	}

	// parse the template file and do substitutions
	txt, err := openAndSub(p.Config.Template, p)
	if err != nil {
		return err
	}
	// convert txt back to []byte and convert to json
	json, err := utilyaml.ToJSON([]byte(txt))
	if err != nil {
		return err
	}

	var dep v1beta1.Deployment

	e := runtime.DecodeInto(api.Codecs.UniversalDecoder(), json, &dep)
	if e != nil {
		log.Fatal("Error decoding yaml file to json", e)
	}
	// check and see if there is a deployment already.  If there is, update it.
	oldDep, err := findDeployment(dep.ObjectMeta.Name, dep.ObjectMeta.Namespace, clientset)
	if err != nil {
		return err
	}
	if oldDep.ObjectMeta.Name == dep.ObjectMeta.Name {
		// update the existing deployment, ignore the deployment that it comes back with
		_, err = clientset.ExtensionsV1beta1().Deployments(p.Config.Namespace).Update(&dep)
		return err
	}
	// create the new deployment since this never existed.
	_, err = clientset.ExtensionsV1beta1().Deployments(p.Config.Namespace).Create(&dep)

	return err
}

func findDeployment(depName string, namespace string, c *kubernetes.Clientset) (v1beta1.Deployment, error) {
	if namespace == "" {
		namespace = "default"
	}
	var d v1beta1.Deployment
	deployments, err := listDeployments(c, namespace)
	if err != nil {
		return d, err
	}
	for _, thisDep := range deployments {
		if thisDep.ObjectMeta.Name == depName {
			return thisDep, err
		}
	}
	return d, err
}

// List the deployments
func listDeployments(clientset *kubernetes.Clientset, namespace string) ([]v1beta1.Deployment, error) {
	// docs on this:
	// https://github.com/kubernetes/client-go/blob/master/pkg/apis/extensions/types.go
	deployments, err := clientset.ExtensionsV1beta1().Deployments(namespace).List(v1.ListOptions{})
	if err != nil {
		log.Fatal(err.Error())
	}
	return deployments.Items, err
}

// open up the template and then sub variables in. Handlebar stuff.
func openAndSub(templateFile string, p Plugin) (string, error) {
	t, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return "", err
	}
	//potty humor!  Render trim toilet paper!  Ha ha, so funny.
	return RenderTrim(string(t), p)
}

// create the connection to kubernetes based on parameters passed in.
// the kubernetes/client-go project is really hard to understand.
func (p Plugin) createKubeClient() (*kubernetes.Clientset, error) {

	ca, err := base64.StdEncoding.DecodeString(p.Config.Ca)
	config := clientcmdapi.NewConfig()
	config.Clusters["drone"] = &clientcmdapi.Cluster{
		Server: p.Config.Server,
		CertificateAuthorityData: ca,
	}
	config.AuthInfos["drone"] = &clientcmdapi.AuthInfo{
		Token: p.Config.Token,
	}

	config.Contexts["drone"] = &clientcmdapi.Context{
		Cluster:  "drone",
		AuthInfo: "drone",
	}
	//config.Clusters["drone"].CertificateAuthorityData = ca
	config.CurrentContext = "drone"

	clientBuilder := clientcmd.NewNonInteractiveClientConfig(*config, "drone", &clientcmd.ConfigOverrides{}, nil)
	actualCfg, err := clientBuilder.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	return kubernetes.NewForConfig(actualCfg)
}

// Just an example from the client specification.  Code not really used.
func watchPodCounts(clientset *kubernetes.Clientset) {
	for {
		pods, err := clientset.Core().Pods("").List(v1.ListOptions{})
		if err != nil {
			log.Fatal(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		time.Sleep(10 * time.Second)
	}
}
