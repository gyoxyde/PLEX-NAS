package main

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

	"github.com/joho/godotenv"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Chargement des variables d'environnement
func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Erreur lors du chargement du fichier .env : %v", err)
	}
}

func authenticate() string {
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
		"version": {"6"}, // Utilisation de la version correcte
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

	log.Printf("Réponse brute : %s", string(body))

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

func getDownloadStatus(sid string) string {
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

	log.Printf("Réponse brute : %s", string(body))

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
				progress := int((downloaded / size) * 10) // 10 blocs pour la barre
				bar := fmt.Sprintf("[%s%s]", string([]rune("⬜️")[:progress])+string([]rune("⬛️")[:10-progress]), "⬛️")
				ongoingDownloads = append(ongoingDownloads, fmt.Sprintf("⬇️ %s : %s (%.2f MB / %.2f MB) %s", title, status, downloaded/(1024*1024), size/(1024*1024), bar))
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

		if len(completedDownloads) > 0 {
			statusMessage += "🎉 **Derniers téléchargements terminés :**\n"
			// Afficher uniquement les 5 derniers
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


func addDownload(sid, link string) {
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

func main() {
	loadEnv()

	// Authentification et récupération du SID
	sid := authenticate()

	// Création du bot Telegram
	botToken := os.Getenv("TELEGRAM_TOKEN")
	authorizedUserID := os.Getenv("TELEGRAM_USER_ID") // Récupération de l'ID autorisé

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Erreur lors de la création du bot Telegram : %v", err)
	}

	bot.Debug = true
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // Ignorer les messages non-textes
			continue
		}

		// Vérifier l'ID utilisateur
		if fmt.Sprintf("%d", update.Message.From.ID) != authorizedUserID {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Vous n'êtes pas autorisé à utiliser ce bot.")
			bot.Send(msg)
			continue
		}

		// Gérer les commandes autorisées
		switch update.Message.Command() {
		case "download":
			link := update.Message.CommandArguments()
			if link == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Envoie un lien valide.")
				bot.Send(msg)
				continue
			}
			addDownload(sid, link)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Téléchargement ajouté pour : "+link)
			bot.Send(msg)
		case "status":
			status := getDownloadStatus(sid)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, status)
			bot.Send(msg)
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Commande inconnue.")
			bot.Send(msg)
		}
	}
}
