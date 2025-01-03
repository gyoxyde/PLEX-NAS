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
	"sort"
)

func GetDownloadStatus(sid string) string {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")

	// URL pour récupérer le statut des tâches
	statusURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":       {"SYNO.DownloadStation.Task"},
		"version":   {"1"},
		"method":    {"list"},
		"_sid":      {sid},
		"additional": {"detail,transfer"}, // Ajout des détails supplémentaires
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

		// Catégories de tâches
		statusMap := map[string][]map[string]interface{}{
			"waiting":           {},
			"downloading":       {},
			"paused":            {},
			"finishing":         {},
			"finished":          {},
			"hash_checking":     {},
			"seeding":           {},
			"filehosting_waiting": {},
			"extracting":        {},
			"error":             {},
		}

		// Parcourir les tâches et les classer par statut
		for _, task := range tasks {
			taskData, ok := task.(map[string]interface{})
			if !ok {
				continue
			}

			status := taskData["status"].(string)
			statusMap[status] = append(statusMap[status], taskData)
		}

		// Construire le message final
		statusMessage := "📊 *Statut des téléchargements :*\n\n"
		for status, taskList := range statusMap {
			if len(taskList) > 0 {
				// Trier les tâches par date (descendant)
				sort.Slice(taskList, func(i, j int) bool {
					timeI := taskList[i]["additional"].(map[string]interface{})["detail"].(map[string]interface{})["create_time"].(float64)
					timeJ := taskList[j]["additional"].(map[string]interface{})["detail"].(map[string]interface{})["create_time"].(float64)
					return timeI > timeJ
				})

				// Limiter à 2 tâches
				if len(taskList) > 2 {
					taskList = taskList[:2]
				}

				// Construire la section pour ce statut
				statusMessage += fmt.Sprintf("*%s*\n", escapeMarkdown(getStatusTitle(status)))
				for _, task := range taskList {
					title := task["title"].(string)
					size := task["size"].(float64)

					if status == "downloading" || status == "finishing" {
						additional := task["additional"].(map[string]interface{})
						transfer := additional["transfer"].(map[string]interface{})
						downloaded := transfer["size_downloaded"].(float64)
						progress := int((downloaded / size) * 10)
						bar := fmt.Sprintf("[%s%s]", strings.Repeat("⬜️", progress)+strings.Repeat("⬛️", 10-progress))
						statusMessage += fmt.Sprintf("⬇️ %s : %s (%.2f MB / %.2f MB) %s\n", escapeMarkdown(title), status, downloaded/(1024*1024), size/(1024*1024), bar)
					} else {
						statusMessage += fmt.Sprintf("%s %s (%.2f MB)\n", getStatusIcon(status), escapeMarkdown(title), size/(1024*1024))
					}
				}
				statusMessage += "\n"
			}
		}

		return statusMessage
	}

	log.Printf("Erreur lors de la récupération des tâches : %v", result)
	return "❌ Impossible de récupérer le statut des téléchargements."
}

// Helper : Échappe les caractères Markdown V2
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// Helper : Renvoie un titre pour chaque statut
func getStatusTitle(status string) string {
	switch status {
	case "waiting":
		return "⌛ En attente"
	case "downloading":
		return "🚀 En cours de téléchargement"
	case "paused":
		return "⏸️ Téléchargements en pause"
	case "finishing":
		return "✅ Finalisation"
	case "finished":
		return "🎉 Téléchargements terminés"
	case "hash_checking":
		return "🔍 Vérification d'intégrité"
	case "seeding":
		return "🌱 Partage en cours"
	case "filehosting_waiting":
		return "⌛ En attente de filehosting"
	case "extracting":
		return "📦 Extraction en cours"
	case "error":
		return "❌ Erreurs"
	default:
		return "📂 Autres"
	}
}

// Helper : Renvoie un icône pour chaque statut
func getStatusIcon(status string) string {
	switch status {
	case "waiting":
		return "⌛"
	case "downloading":
		return "⬇️"
	case "paused":
		return "⏸️"
	case "finishing":
		return "✅"
	case "finished":
		return "🎉"
	case "hash_checking":
		return "🔍"
	case "seeding":
		return "🌱"
	case "filehosting_waiting":
		return "⌛"
	case "extracting":
		return "📦"
	case "error":
		return "❌"
	default:
		return "📂"
	}
}
