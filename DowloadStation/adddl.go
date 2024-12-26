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
	"strings"
	"bytes"
)

// OneFichierAPIResponse represents the 1fichier API response
type OneFichierAPIResponse struct {
	Status  string `json:"status"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

// Get direct download link from 1fichier using API key
func Get1FichierDirectLink(fileURL string) (string, error) {
	apiKey := "G_baFm-uemr8mYMPwzx=ZfKgPK5DQ4Ae"
	if apiKey == "" {
		return "", fmt.Errorf("clé API 1fichier manquante")
	}

	// Prepare API request
	apiURL := "https://api.1fichier.com/v1/download/get_token.cgi"
	requestData := map[string]string{
		"link": fileURL,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("erreur de préparation de la requête API: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("erreur de création de la requête: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	
	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erreur d'appel API 1fichier: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erreur de lecture de la réponse: %v", err)
	}

	log.Printf("Réponse API 1fichier: %s", string(body))

	// Parse response
	var apiResp OneFichierAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("erreur de parsing JSON: %v", err)
	}

	if apiResp.Status != "OK" {
		return "", fmt.Errorf("erreur 1fichier: %s", apiResp.Message)
	}

	return apiResp.URL, nil
}

func AddDownload(sid, link string) string {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")
	destination := "/volume1/MOVIES/Downloads"

	// Validate input
	if link == "" {
		return "❌ Le lien de téléchargement ne peut pas être vide"
	}

	downloadLink := link
	// Special handling for 1fichier.com
	if strings.Contains(link, "1fichier.com") {
		directLink, err := Get1FichierDirectLink(link)
		if err != nil {
			log.Printf("Erreur 1fichier: %v", err)
			return fmt.Sprintf("❌ Erreur 1fichier: %v", err)
		}
		downloadLink = directLink
		log.Printf("Lien direct 1fichier obtenu: %s", downloadLink)
	}

	// Build API URL
	taskURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":         {"SYNO.DownloadStation.Task"},
		"version":     {"1"},
		"method":      {"create"},
		"_sid":        {sid},
		"uri":         {downloadLink},
		"destination": {destination},
	}

	// Configure HTTPS client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Send request
	resp, err := client.Get(taskURL + "?" + params.Encode())
	if err != nil {
		log.Printf("Erreur de requête HTTP : %v", err)
		return "❌ Erreur de connexion au NAS"
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Erreur de lecture de la réponse : %v", err)
		return "❌ Erreur lors de la lecture de la réponse"
	}

	// Log raw response for debugging
	log.Printf("Réponse API Download Station: %s", string(body))

	// Parse response
	var dsResp APIResponse
	if err := json.Unmarshal(body, &dsResp); err != nil {
		log.Printf("Erreur de parsing JSON : %v", err)
		return "❌ Erreur lors de l'analyse de la réponse"
	}

	if !dsResp.Success {
		errorMessage := ErrorCode[dsResp.Error.Code]
		if errorMessage == "" {
			errorMessage = fmt.Sprintf("Erreur inconnue (Code: %d)", dsResp.Error.Code)
		}
		log.Printf("Erreur API : %s (Code: %d)", errorMessage, dsResp.Error.Code)
		return fmt.Sprintf("❌ %s", errorMessage)
	}

	return "✅ Téléchargement ajouté avec succès"
}