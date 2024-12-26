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
)

// APIResponse represents the standard Synology API response structure
type APIResponse struct {
	Success bool `json:"success"`
	Error   struct {
		Code int `json:"code"`
	} `json:"error,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

// ErrorCode maps Synology error codes to human-readable messages
var ErrorCode = map[int]string{
	100: "Erreur inconnue",
	101: "Paramètre invalide",
	102: "L'API demandée n'existe pas",
	103: "La méthode demandée n'existe pas",
	104: "La version demandée ne supporte pas cette fonctionnalité",
	105: "La session n'a pas les permissions nécessaires",
	106: "Session expirée",
	107: "Session interrompue par une connexion multiple",
	108: "Fichier inexistant",
	109: "Destination invalide",
	403: "Accès refusé - Authentification requise",
}

func AddDownload(sid, link string) string {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")
	destination := "/volume1/MOVIES/Downloads"

	// Validate input
	if link == "" {
		return "❌ Le lien de téléchargement ne peut pas être vide"
	}

	// Validate URL format
	parsedURL, err := url.ParseRequestURI(link)
	if err != nil {
		return fmt.Sprintf("❌ Lien invalide : %s", link)
	}

	// Special handling for 1fichier.com
	if strings.Contains(parsedURL.Host, "1fichier.com") {
		oneFichierUser := "zarconecesar@gmail.com"
		oneFichierPass := "C2.&B_$9H@i52Hc"
		
		if oneFichierUser == "" || oneFichierPass == "" {
			return "❌ Configuration manquante pour 1fichier.com - Veuillez configurer les identifiants"
		}
		
		// Add authentication parameters to the URL
		q := parsedURL.Query()
		q.Set("auth_user", oneFichierUser)
		q.Set("auth_pass", oneFichierPass)
		parsedURL.RawQuery = q.Encode()
		link = parsedURL.String()
	}

	// Build API URL
	taskURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":         {"SYNO.DownloadStation.Task"},
		"version":     {"1"},
		"method":      {"create"},
		"_sid":        {sid},
		"uri":         {link},
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
	log.Printf("Réponse API : %s", string(body))
	log.Printf("Paramètres : %s", params.Encode())

	// Parse response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("Erreur de parsing JSON : %v", err)
		return "❌ Erreur lors de l'analyse de la réponse"
	}

	// Handle API response
	if !apiResp.Success {
		errorMessage := ErrorCode[apiResp.Error.Code]
		if errorMessage == "" {
			errorMessage = fmt.Sprintf("Erreur inconnue (Code: %d)", apiResp.Error.Code)
		}
		log.Printf("Erreur API : %s (Code: %d)", errorMessage, apiResp.Error.Code)
		return fmt.Sprintf("❌ %s", errorMessage)
	}

	return "✅ Téléchargement ajouté avec succès"
}