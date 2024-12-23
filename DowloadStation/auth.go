package DownloadStation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"io"
	"crypto/tls"
)

func Authenticate() string {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")
	username := os.Getenv("NAS_USER")
	password := os.Getenv("NAS_PASSWORD")

	// Vérification des variables d'environnement
	if nasIP == "" || nasPort == "" || username == "" || password == "" {
		log.Fatalf("Erreur : Une ou plusieurs variables d'environnement sont manquantes. NAS_IP=%s, NAS_PORT=%s", nasIP, nasPort)
	}

	// Utiliser la version 6 de l'API
	authURL := fmt.Sprintf("https://%s:%s/webapi/auth.cgi", nasIP, nasPort)
	params := url.Values{
		"api":     {"SYNO.API.Auth"},
		"version": {"6"},
		"method":  {"login"},
		"account": {username},
		"passwd":  {password},
		"session": {"DownloadStation"},
		"format":  {"sid"},
	}

	// Configurer un client HTTPS qui ignore les erreurs de certificat
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(authURL + "?" + params.Encode())
	if err != nil {
		log.Fatalf("Erreur lors de l'authentification : %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Erreur lors de la lecture de la réponse : %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Fatalf("Erreur lors du décodage de la réponse JSON : %v", err)
	}

	if success, ok := result["success"].(bool); ok && success {
		data := result["data"].(map[string]interface{})
		return data["sid"].(string)
	}

	log.Fatalf("Authentification échouée : %v", result)
	return ""
}