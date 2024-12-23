package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/gyoxyde/PLEX-NAS/DowloadStation"
)

// Chargement des variables d'environnement
func loadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Erreur lors du chargement du fichier .env : %v", err)
	}
}

func main() {
	loadEnv()

	// Authentification et récupération du SID
	sid := DownloadStation.Authenticate()

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
		case "dl":
			link := update.Message.CommandArguments()
			if link == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Envoie un lien valide.")
				bot.Send(msg)
				continue
			}
			DownloadStation.AddDownload(sid, link)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Téléchargement ajouté pour : "+link)
			bot.Send(msg)
		case "status":
			status := DownloadStation.GetDownloadStatus(sid)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, status)
			bot.Send(msg)
		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Commande inconnue.")
			bot.Send(msg)
		}
	}
}
