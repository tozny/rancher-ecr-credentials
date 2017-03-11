package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/rancher/go-rancher/client"
)

// Rancher holds the configuration parameters
type Rancher struct {
	URL       string
	AccessKey string
	SecretKey string
}

func main() {
	vargs := Rancher{
		URL:       os.Getenv("CATTLE_URL"),
		AccessKey: os.Getenv("CATTLE_ACCESS_KEY"),
		SecretKey: os.Getenv("CATTLE_SECRET_KEY"),
	}

	go healthcheck()

	err := updateEcr(vargs)
	if err != nil {
		log.Printf("Error updating ECR, %s\n", err)
	}
	ticker := time.NewTicker(6 * time.Hour)
	for {
		<-ticker.C
		err := updateEcr(vargs)
		if err != nil {
			log.Printf("Error updating ECR, %s\n", err)
		}
	}
}

func updateEcr(vargs Rancher) error {
	log.Println("Updating ECR Credentials")
	ecrClient := ecr.New(session.New())

	resp, err := ecrClient.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return err
	}
	log.Println("Returned from AWS GetAuthorizationToken call successfully")

	if len(resp.AuthorizationData) < 1 {
		return errors.New("Request did not return authorization data")
	}

	bytes, err := base64.StdEncoding.DecodeString(*resp.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		log.Printf("Error decoding authorization token: %s\n", err)
		return err
	}
	token := string(bytes[:len(bytes)])

	authTokens := strings.Split(token, ":")
	if len(authTokens) != 2 {
		return fmt.Errorf("Authorization token does not contain data in <user>:<password> format: %s", token)
	}

	registryURL, err := url.Parse(*resp.AuthorizationData[0].ProxyEndpoint)
	if err != nil {
		log.Printf("Error parsing registry URL: %s\n", err)
		return err
	}

	ecrUsername := authTokens[0]
	ecrPassword := authTokens[1]
	ecrURL := registryURL.Host
	rancher, err := client.NewRancherClient(&client.ClientOpts{
		Url:       vargs.URL,
		AccessKey: vargs.AccessKey,
		SecretKey: vargs.SecretKey,
	})

	if err != nil {
		log.Printf("Failed to create rancher client: %s\n", err)
		return err
	}
	registries, err := rancher.Registry.List(&client.ListOpts{})
	if err != nil {
		log.Printf("Failed to retrieve registries: %s\n", err)
		return err
	}
	log.Printf("Looking for configured registry for host %s\n", ecrURL)
	for _, registry := range registries.Data {
		serverAddress, err := url.Parse(registry.ServerAddress)
		if err != nil {
			log.Printf("Failed to parse configured registry URL %s\n", registry.ServerAddress)
			break
		}
		registryHost := serverAddress.Host
		if registryHost == "" {
			registryHost = serverAddress.Path
		}
		if registryHost == ecrURL {
			credentials, err := rancher.RegistryCredential.List(&client.ListOpts{
				Filters: map[string]interface{}{
					"registryId": registry.Id,
				},
			})
			if err != nil {
				log.Printf("Failed to retrieved registry credentials for id: %s, %s\n", registry.Id, err)
				break
			}
			if len(credentials.Data) != 1 {
				log.Printf("No credentials retrieved for registry: %s\n", registry.Id)
				break
			}
			credential := credentials.Data[0]
			_, err = rancher.RegistryCredential.Update(&credential, &client.RegistryCredential{
				PublicValue: ecrUsername,
				SecretValue: ecrPassword,
			})
			if err != nil {
				log.Printf("Failed to update registry credential %s, %s\n", credential.Id, err)
			} else {
				log.Printf("Successfully updated credentials %s for registry %s; registry address: %s\n", credential.Id, registry.Id, registryHost)
			}
			return nil
		}
	}
	log.Printf("Failed to find configured registry to update for URL %s\n", ecrURL)
	return nil
}

func healthcheck() {
	listenPort := "8080"
	p, ok := os.LookupEnv("LISTEN_PORT")
	if ok {
		listenPort = p
	}
	http.HandleFunc("/ping", ping)
	log.Printf("Starting Healthcheck listener at :%s/ping\n", listenPort)
	err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil)
	if err != nil {
		log.Fatal("Error creating health check listener: ", err)
	}
}

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong!")
}
