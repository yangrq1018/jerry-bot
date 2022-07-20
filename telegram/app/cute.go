package app

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/yangrq1018/jerry-bot/telegram"
	tgbotapi "github.com/yangrq1018/telegram-bot-api/v5"
)

// these files has already been uploaded to server
var jerryPicIDs = []string{
	"AgACAgUAAxkDAAPDYOv5-NmxRyH6PqehSlYUuflUdn4AAtusMRtLMWFXfNTrewF7SD0BAAMCAANzAAMgBA",
	"AgACAgUAAxkDAAPEYOv5-1oMDJMyzpQceQVhu5Vb9fQAAt6sMRtLMWFXx8IwTfXGO9kBAAMCAANzAAMgBA",
	"AgACAgUAAxkDAAPFYOv5_rZUyuH1Cnf5UekKRtxtSSsAAt-sMRtLMWFXjCkSHnFVeiABAAMCAANzAAMgBA",
	"AgACAgUAAxkDAAPGYOv6AAG6W65IYVbd5vmuEcq2NvmAAALgrDEbSzFhV2d9R6A6OSJnAQADAgADcwADIAQ",
	"AgACAgUAAxkDAAPHYOv6AsUBsnHcsQxCnRH4tu-YXKEAAuGsMRtLMWFXQZjDa7yVgg0BAAMCAANzAAMgBA",
	"AgACAgUAAxkDAAPIYOv6BcZM5iDSbkncT5hUy7Nm11IAAuKsMRtLMWFXm5FvmX0M77EBAAMCAANzAAMgBA",
}

func drawFrom(sl []string) string {
	if len(sl) == 0 {
		return ""
	}
	return sl[rand.Int()%len(sl)]
}

// UpdatePicFromDisk
// randomly load a file from local os
// upload to telegram server and send it
func UpdatePicFromDisk(dir string) telegram.HandleFunc[tgbotapi.Update] {
	return func(b *telegram.Bot, u tgbotapi.Update) error {
		files, _ := ioutil.ReadDir(dir)
		imageFiles := make([]string, 0)

		// select JPG only
		for i := range files {
			if !files[i].IsDir() && strings.HasSuffix(files[i].Name(), "jpg") {
				imageFiles = append(imageFiles, files[i].Name())
			}
		}
		name := drawFrom(imageFiles)
		if name == "" {
			return fmt.Errorf("no photo under directory")
		}
		file := tgbotapi.NewPhotoUpload(
			u.Message.Chat.ID,
			filepath.Join(dir, name),
		)
		_, err := b.Bot().Send(file)
		//fmt.Printf("%s\n", (*res.Photo)[0].FileID)
		return err
	}
}

func RandomSendPicFromFileIDs(fileIDs []string) telegram.HandleFunc[tgbotapi.Update] {
	return func(b *telegram.Bot, u tgbotapi.Update) error {
		file := tgbotapi.NewPhotoShare(u.Message.Chat.ID, fileIDs[rand.Int()%len(fileIDs)])
		res, err := b.Bot().Send(file)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", (*res.Photo)[0].FileID)
		return nil
	}
}

func CutePhotoFromDisk(picDir string) telegram.Command {
	return SimpleCommand{
		name:        "pic",
		description: "展示Jerry照片",
		handle:      UpdatePicFromDisk(picDir),
	}
}

func CutePhotoFromServer() telegram.Command {
	return SimpleCommand{
		name:        "picserver",
		description: "randomly show a cute jerry image",
		handle:      RandomSendPicFromFileIDs(jerryPicIDs),
	}
}
