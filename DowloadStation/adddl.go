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


func AddDownload(sid, link string) {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")
	destination := "/volume1/MOVIES/Downloads"

	// URL pour ajouter une tâche
	taskURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":         {"SYNO.DownloadStation.Task"},
		"version":     {"1"},
		"method":      {"create"},
		"_sid":        {sid},
		"uri":         {link},
		"destination": {destination},
	}

	// Configurer un client HTTPS qui ignore les erreurs de certificat
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// Envoyer la requête
	resp, err := client.Get(taskURL + "?" + params.Encode())
	if err != nil {
		log.Printf("Erreur lors de l'ajout du téléchargement : %v", err)
		return
	}
	defer resp.Body.Close()

	// Lire la réponse brute
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Erreur lors de la lecture de la réponse : %v", err)
		return
	}

	log.Printf("Réponse brute de l'API : %s", string(body))
	log.Printf("Paramètres de la requête : %s", params.Encode())

	// Décoder la réponse JSON
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Erreur lors du décodage de la réponse JSON : %v", err)
		return
	}

	// Vérifier le statut de la réponse
	if success, ok := result["success"].(bool); ok && success {
		log.Printf("Téléchargement ajouté avec succès : %v", result)
	} else {
		log.Printf("Erreur API : %v", result)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		if errorData, ok := result["error"].(map[string]interface{}); ok {
			errorCode := errorData["code"]
			log.Printf("Erreur API : Code %v, Détails : %v", errorCode, errorData)
		}
	}
}
