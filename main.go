package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

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
	authURL := fmt.Sprintf("http://%s:%s/webapi/auth.cgi", nasIP, nasPort)
	params := url.Values{
		"api":     {"SYNO.API.Auth"},
		"version": {"6"}, // Utilisation de la version correcte
		"method":  {"login"},
		"account": {username},
		"passwd":  {password},
		"session": {"DownloadStation"},
		"format":  {"sid"},
	}

	resp, err := http.Get(authURL + "?" + params.Encode())
	if err != nil {
		log.Fatalf("Erreur lors de l'authentification : %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
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

func addDownload(sid, link string) {
	nasIP := os.Getenv("NAS_LOCAL_IP")
	nasPort := os.Getenv("NAS_LOCAL_PORT")

	taskURL := fmt.Sprintf("https://%s:%s/webapi/DownloadStation/task.cgi", nasIP, nasPort)
	params := url.Values{
		"api":     {"SYNO.DownloadStation.Task"},
		"version": {"1"},
		"method":  {"create"},
		"_sid":    {sid},
		"uri":     {link},
	}

	resp, err := http.Get(taskURL + "?" + params.Encode())
	if err != nil {
		log.Printf("Erreur lors de l'ajout du téléchargement : %v", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
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
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Statut des téléchargements : Fonctionnalité à venir.")
			bot.Send(msg)
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Commande inconnue.")
			bot.Send(msg)
		}
	}
}
