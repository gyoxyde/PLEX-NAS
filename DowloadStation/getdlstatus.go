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

func GetDownloadStatus(sid string) string {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")

	// URL pour récupérer le statut des tâches
	statusURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":     {"SYNO.DownloadStation.Task"},
		"version": {"1"},
		"method":  {"list"},
		"_sid":    {sid},
	}

	// Configurer un client HTTPS qui ignore les erreurs de certificat
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(statusURL + "?" + params.Encode())
	if err != nil {
		log.Printf("Erreur lors de la récupération du statut des téléchargements : %v", err)
		return "❌ Erreur lors de la récupération des données."
	}
	defer resp.Body.Close()

	// Lire et analyser la réponse brute
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Erreur lors de la lecture de la réponse : %v", err)
		return "❌ Erreur lors de l'analyse des données."
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Erreur lors du décodage JSON : %v", err)
		return "❌ Erreur lors de l'analyse des données."
	}

	if success, ok := result["success"].(bool); ok && success {
		data, ok := result["data"].(map[string]interface{})
		if !ok || data == nil {
			return "📂 Aucune donnée disponible."
		}

		tasks, ok := data["tasks"].([]interface{})
		if !ok || len(tasks) == 0 {
			return "📂 Aucune tâche trouvée."
		}

		// Listes pour les tâches
		ongoingDownloads := []string{}
		pausedDownloads := []string{}
		waitingDownloads := []string{}
		completedDownloads := []string{}

		// Construire les listes
		for _, task := range tasks {
			taskData, ok := task.(map[string]interface{})
			if !ok {
				continue
			}

			title := taskData["title"].(string)
			status := taskData["status"].(string)
			size := taskData["size"].(float64)

			if status == "downloading" {
				// Barre de progression
				downloaded := taskData["additional"].(map[string]interface{})["transfer"].(map[string]interface{})["size_downloaded"].(float64)
				progress := int((downloaded / size) * 10)
				bar := fmt.Sprintf("[%s%s]", strings.Repeat("⬜️", progress)+strings.Repeat("⬛️", 10-progress))
				ongoingDownloads = append(ongoingDownloads, fmt.Sprintf("⬇️ %s : %s (%.2f MB / %.2f MB) %s", title, status, downloaded/(1024*1024), size/(1024*1024), bar))
			} else if status == "paused" {
				pausedDownloads = append(pausedDownloads, fmt.Sprintf("⏸️ %s (%.2f MB)", title, size/(1024*1024)))
			} else if status == "waiting" {
				waitingDownloads = append(waitingDownloads, fmt.Sprintf("⌛ %s (%.2f MB)", title, size/(1024*1024)))
			} else if status == "finished" {
				completedDownloads = append(completedDownloads, fmt.Sprintf("✅ %s (%.2f MB)", title, size/(1024*1024)))
			}
		}

		// Construire le message final
		statusMessage := "📊 **Statut des téléchargements :**\n\n"

		if len(ongoingDownloads) > 0 {
			statusMessage += "🚀 **Téléchargements en cours :**\n"
			statusMessage += strings.Join(ongoingDownloads, "\n")
			statusMessage += "\n\n"
		} else {
			statusMessage += "🚀 Aucun téléchargement en cours.\n\n"
		}

		if len(pausedDownloads) > 0 {
			statusMessage += "⏸️ **Téléchargements en pause :**\n"
			statusMessage += strings.Join(pausedDownloads, "\n")
			statusMessage += "\n\n"
		}

		if len(waitingDownloads) > 0 {
			statusMessage += "⌛ **Téléchargements en attente :**\n"
			statusMessage += strings.Join(waitingDownloads, "\n")
			statusMessage += "\n\n"
		}

		if len(completedDownloads) > 0 {
			statusMessage += "🎉 **Derniers téléchargements terminés :**\n"
			count := 5
			if len(completedDownloads) < 5 {
				count = len(completedDownloads)
			}
			statusMessage += strings.Join(completedDownloads[:count], "\n")
		} else {
			statusMessage += "🎉 Aucun téléchargement terminé."
		}

		return statusMessage
	}

	log.Printf("Erreur lors de la récupération des tâches : %v", result)
	return "❌ Impossible de récupérer le statut des téléchargements."
}
