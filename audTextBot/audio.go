package audTextBot

import (
	"errors"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	unpremiumMaxAudioDuration = 300  // maximum allowed audio duration for non-premium users (in seconds)
	premiumMaxAudioDuration   = 3600 // maximum allowed audio duration for premium users (in seconds)
	maxMessageLength          = 4000
)

func handleAudio(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.MessageConfig{}
	msg.ChatID = update.Message.Chat.ID

	// Check if user is premium
	isPremium := isPremium(bot, update.Message.From.ID, update.Message.Chat.ID)

	// Check if audio duration is allowed
	err := isAudioDurationAllowed(update.Message.Audio.Duration, update.Message.From.ID, update.Message.From.FirstName, isPremium)
	if err != nil {
		sendMessage(bot, update.Message.Chat.ID, err.Error())
		return
	}

	// Send initial message
	sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Decoding will take from 15%% to 30%% of file duration if it is not too short"))

	// Decode audio file
	text, err := decodeAudioFile(bot, update.Message.Audio.FileID)
	if err != nil {
		log.Println(err)
		sendMessage(bot, update.Message.Chat.ID, "There was an error decoding the file")
		return
	}

	// Paginate text into messages
	chunks := paginateText(text, maxMessageLength)

	sendMessage(bot, update.Message.Chat.ID, "Here is the script of the audio:")
	// Send each message chunk
	for _, chunk := range chunks {
		sendMessage(bot, update.Message.Chat.ID, chunk)
	}

	if !isPremium {
		// Add time spent to redis
		timeSpent, err := redisService.IncrementUnpremiumTime(update.Message.From.ID, update.Message.Audio.Duration)
		if err != nil {
			log.Println(err)
		}
		// Send decoded text to user
		remainingTime := unpremiumMaxAudioDuration - timeSpent
		minutes := remainingTime / 60
		seconds := remainingTime % 60
		sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Remaining free time: %02d minutes %02d seconds", minutes, seconds))
		return
	}
	timeSpent, err := redisService.IcrementPremiumTime(update.Message.From.ID, update.Message.Audio.Duration)
	if err != nil {
		log.Println(err)
	}
	remainingTime := premiumMaxAudioDuration - timeSpent
	minutes := remainingTime / 60
	seconds := remainingTime % 60
	sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Remaining free time: %02d minutes %02d seconds", minutes, seconds))
}

func paginateText(text string, chunkSize int) []string {
	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[i:end])
		chunks = append(chunks, chunk)
	}

	return chunks
}

// isAudioDurationAllowed checks if audio duration is allowed for user
func isAudioDurationAllowed(audioDuration int, userId int, userName string, isPremium bool) error {
	if isPremium {
		return nil
	}
	if audioDuration > unpremiumMaxAudioDuration {
		return errors.New(fmt.Sprintf("The audio file is too long. Only %v minutes allowed for users without premium", unpremiumMaxAudioDuration/60))
	}
	notPremiumTime, err := redisService.GetUnpremiumTimeSpent(userId)
	if err != nil {
		log.Println(err)
		if err.Error() == "Max time exceeded" {
			return errors.New(fmt.Sprintf("Dear %v, You have exceeded maximum numbers of free decoding of audio,to get premium - type /premium", userName))
		}
		if err.Error() == "User doesn't exist" {
			err := redisService.SaveUnpremiumUser(userId)
			if err != nil {
				log.Println(err)
				return errors.New("There is an error occurred, please try again later")
			}
		}
	}
	// Check if user has enough time
	if audioDuration+notPremiumTime > unpremiumMaxAudioDuration {
		remainingTime := unpremiumMaxAudioDuration - notPremiumTime
		minutes := remainingTime / 60
		seconds := remainingTime % 60
		return errors.New(fmt.Sprintf("Too long audio, you dont have enough free time, remaining time: %02d minutes %02d seconds", minutes, seconds))
	}
	return nil
}

// decodeAudioFile decodes audio file to text
func decodeAudioFile(bot *tgbotapi.BotAPI, fileID string) (string, error) {
	// Get audio file
	fileURL, err := uploadUserFileData(bot, fileID)
	if err != nil {
		log.Println(err)
		return "", errors.New("There is an error occurred while decoding the file")
	}
	text, err := decodeFile(fileURL)
	if err != nil {
		log.Println(err)
		return "", errors.New("There is an error occurred while decoding the file")
	}
	return text, nil
}

func handleVoice(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	msg := tgbotapi.MessageConfig{}
	msg.ChatID = update.Message.Chat.ID
	// Check if user is premium
	isPremium := isPremium(bot, update.Message.From.ID, update.Message.Chat.ID)

	// Check if voice duration is allowed
	err := isAudioDurationAllowed(update.Message.Voice.Duration, update.Message.From.ID, update.Message.From.FirstName, isPremium)
	if err != nil {
		sendMessage(bot, update.Message.Chat.ID, err.Error())
		return
	}

	// Send initial message
	sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Decoding will take from 15%% to 30%% of file duration if it is not too short"))

	// Decode voice file
	text, err := decodeAudioFile(bot, update.Message.Voice.FileID)
	if err != nil {
		log.Println(err)
		sendMessage(bot, update.Message.Chat.ID, "There was an error decoding the file")
		return
	}
	// Paginate text into messages
	chunks := paginateText(text, maxMessageLength)

	sendMessage(bot, update.Message.Chat.ID, "Here is the script of the voice message:")
	// Send each message chunk
	for _, chunk := range chunks {
		sendMessage(bot, update.Message.Chat.ID, chunk)
	}

	if !isPremium {
		// Add time spent to redis
		timeSpent, err := redisService.IncrementUnpremiumTime(update.Message.From.ID, update.Message.Voice.Duration)
		if err != nil {
			log.Println(err)
		}
		// Send decoded text to user
		remainingTime := unpremiumMaxAudioDuration - timeSpent
		minutes := remainingTime / 60
		seconds := remainingTime % 60
		sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Remaining free time: %02d minutes %02d seconds", minutes, seconds))
		return
	}

	timeSpent, err := redisService.IcrementPremiumTime(update.Message.From.ID, update.Message.Voice.Duration)
	if err != nil {
		log.Println(err)
	}
	remainingTime := premiumMaxAudioDuration - timeSpent
	minutes := remainingTime / 60
	seconds := remainingTime % 60
	sendMessage(bot, update.Message.Chat.ID, fmt.Sprintf("Remaining free time: %02d minutes %02d seconds", minutes, seconds))
}
