package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/rancher/go-rancher/client"
)

// Rancher holds the configuration parameters
type Rancher struct {
	URL         string
	AccessKey   string
	SecretKey   string
	RegistryIds []string
	client      *client.RancherClient
}

func main() {
	r := Rancher{
		URL:         os.Getenv("CATTLE_URL"),
		AccessKey:   os.Getenv("CATTLE_ACCESS_KEY"),
		SecretKey:   os.Getenv("CATTLE_SECRET_KEY"),
		RegistryIds: []string{},
	}
	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       r.URL,
		AccessKey: r.AccessKey,
		SecretKey: r.SecretKey,
	})
	if err != nil {
		log.Fatalf("[main] Unable to create Rancher API client: %s\n", err)
	}
	r.client = rancher

	if ids, ok := os.LookupEnv("AWS_ECR_REGISTRY_IDS"); ok && ids != "" {
		r.RegistryIds = strings.Split(ids, ",")
	}

	go healthcheck()

	r.updateEcr(ecr.New(session.New()), r.client.Registry, r.client.RegistryCredential)
	ticker := time.NewTicker(6 * time.Hour)
	for {
		<-ticker.C
		r.updateEcr(ecr.New(session.New()), r.client.Registry, r.client.RegistryCredential)
	}
}

func (r *Rancher) updateEcr(
	svc ecriface.ECRAPI,
	registryClient client.RegistryOperations,
	registryCredentialClient client.RegistryCredentialOperations) {
	log.Println("[main] Updating ECR Credentials")

	request := &ecr.GetAuthorizationTokenInput{}
	if len(r.RegistryIds) > 0 {
		request = &ecr.GetAuthorizationTokenInput{RegistryIds: aws.StringSlice(r.RegistryIds)}
	}
	resp, err := svc.GetAuthorizationToken(request)
	if err != nil {
		log.Printf("[updateEcr] Error calling AWS API: %s\n", err)
		return
	}
	log.Println("[updateEcr] Returned from AWS GetAuthorizationToken call successfully")

	if len(resp.AuthorizationData) < 1 {
		log.Println("[updateEcr] Request did not return authorization data")
		return
	}

	for _, data := range resp.AuthorizationData {
		r.processToken(data, registryClient, registryCredentialClient)
	}
}

func (r *Rancher) processToken(
	data *ecr.AuthorizationData,
	registryClient client.RegistryOperations,
	registryCredentialClient client.RegistryCredentialOperations) {

	bytes, err := base64.StdEncoding.DecodeString(*data.AuthorizationToken)
	if err != nil {
		log.Printf("[processToken %s] Error decoding authorization token: %s\n", *data.ProxyEndpoint, err)
		return
	}
	token := string(bytes[:len(bytes)])

	authTokens := strings.Split(token, ":")
	if len(authTokens) != 2 {
		log.Printf("[processToken %s] Authorization token does not contain data in <user>:<password> format: %s\n", *data.ProxyEndpoint, token)
		return
	}

	registryURL, err := url.Parse(*data.ProxyEndpoint)
	if err != nil {
		log.Printf("[processToken %s] Error parsing registry URL: %s\n", *data.ProxyEndpoint, err)
		return
	}

	ecrUsername := authTokens[0]
	ecrPassword := authTokens[1]
	ecrHost := registryURL.Host

	registries, err := registryClient.List(&client.ListOpts{})
	if err != nil {
		log.Printf("[processToken %s] Failed to retrieve registries: %s\n", *data.ProxyEndpoint, err)
		return
	}
	log.Printf("[processToken %s] Looking for configured registry for host: %s\n", *data.ProxyEndpoint, ecrHost)
	for _, registry := range registries.Data {
		serverAddress, err := url.Parse(registry.ServerAddress)
		if err != nil {
			log.Printf("[processToken %s] Failed to parse configured registry URL: %s\n", *data.ProxyEndpoint, registry.ServerAddress)
			break
		}
		registryHost := serverAddress.Host
		if registryHost == "" {
			registryHost = serverAddress.Path
		}
		if registryHost == ecrHost {
			credentials, err := registryCredentialClient.List(&client.ListOpts{
				Filters: map[string]interface{}{
					"registryId": registry.Id,
				},
			})
			if err != nil {
				log.Printf("[processToken %s] Failed to retrieved registry credentials for id: %s, %s\n", *data.ProxyEndpoint, registry.Id, err)
				break
			}
			if len(credentials.Data) != 1 {
				log.Printf("[processToken %s] No credentials retrieved for registry: %s\n", *data.ProxyEndpoint, registry.Id)
				break
			}
			credential := credentials.Data[0]
			_, err = registryCredentialClient.Update(&credential, &client.RegistryCredential{
				PublicValue: ecrUsername,
				SecretValue: ecrPassword,
			})
			if err != nil {
				log.Printf("[processToken %s] Failed to update registry credential %s, %s\n", *data.ProxyEndpoint, credential.Id, err)
			} else {
				log.Printf("[processToken %s] Successfully updated credentials %s for registry %s; registry address: %s\n", *data.ProxyEndpoint, credential.Id, registry.Id, registryHost)
			}
			return
		}
	}
	log.Printf("[processToken %s] Failed to find Rancher registry to update for ECR Host: %s\n", *data.ProxyEndpoint, ecrHost)
	return
}

func healthcheck() {
	listenPort := "8080"
	p, ok := os.LookupEnv("LISTEN_PORT")
	if ok {
		listenPort = p
	}
	http.HandleFunc("/ping", ping)
	log.Printf("[healthcheck] Starting Healthcheck listener at :%s/ping\n", listenPort)
	err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil)
	if err != nil {
		log.Fatal("[healthcheck] Error creating health check listener: ", err)
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong!")
}
