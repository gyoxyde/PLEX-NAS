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
		"api":     {"SYNO.DownloadStation.Task"},
		"version": {"1"},
		"method":  {"create"},
		"_sid":    {sid},
		"uri":     {link},
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

	print(resp.Body)

	// Lire et analyser la réponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Erreur lors de la lecture de la réponse : %v", err)
		return
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Erreur lors du décodage de la réponse JSON : %v", err)
		return
	}

	log.Printf("Réponse ajout de téléchargement : %v", result)
}