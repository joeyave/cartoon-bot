package controller

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type BotController struct {
}

func (c *BotController) Photo(bot *gotgbot.Bot, ctx *ext.Context) error {

	ctxWithCancel, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctxWithCancel context.Context) {
		select {
		case <-ctxWithCancel.Done():
			return
		default:
			ctx.EffectiveChat.SendAction(bot, "upload_photo", nil)
		}
		time.Sleep(4 * time.Second)
	}(ctxWithCancel)

	f, err := bot.GetFile(ctx.Message.Photo[len(ctx.Message.Photo)-1].FileId, nil)
	if err != nil {
		return err
	}

	reader, err := File(bot, f)
	if err != nil {
		return err
	}

	imgBytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	base64Img := base64.StdEncoding.EncodeToString(imgBytes)

	imgs, err := ImgToCartoon(base64Img)
	if err != nil {
		var qqErr Error
		if errors.As(err, &qqErr) {
			_, err := ctx.EffectiveChat.SendMessage(bot, qqErr.Message, nil)
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}

	_, err = bot.SendPhoto(ctx.EffectiveChat.Id, imgs[0], nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *BotController) Start(bot *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveChat.SendMessage(bot, "Send me a photo!", nil)
	if err != nil {
		return err
	}
	return nil
}

func (c *BotController) Error(bot *gotgbot.Bot, ctx *ext.Context, botErr error) ext.DispatcherAction {

	log.Error().Msgf("Error handling update: %v", botErr)

	if ctx.CallbackQuery != nil {
		_, err := ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
			Text: "Server error.",
		})
		if err != nil {
			log.Error().Err(err).Msg("Error!")
			return ext.DispatcherActionEndGroups
		}
	} else if ctx.EffectiveChat != nil {
		_, err := ctx.EffectiveChat.SendMessage(bot, "Server error.", nil)
		if err != nil {
			log.Error().Err(err).Msg("Error!")
			return ext.DispatcherActionEndGroups
		}
	}

	// todo: send message to the logs channel
	logsChannelID, err := strconv.ParseInt(os.Getenv("LOG_CHANNEL"), 10, 64)
	if err == nil {
		_, err = bot.SendMessage(logsChannelID, fmt.Sprintf("Error handling update!\n<pre>error=%v</pre>", botErr), &gotgbot.SendMessageOpts{
			DisableWebPagePreview: true,
			ParseMode:             "HTML",
		})
		if err != nil {
			log.Error().Err(err).Msg("Error!")
			return ext.DispatcherActionEndGroups
		}
	}

	return ext.DispatcherActionEndGroups
}

type ReqBody struct {
	BusiID string   `json:"busiId"`
	Images []string `json:"images"`
}

type RespBody struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Extra string `json:"extra"`
}

type Extra struct {
	ImgURLs []string `json:"img_urls"`
}

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("api error: code=%d, message=%s", e.Code, e.Message)
}

func ImgToCartoon(base64Img string) ([]string, error) {

	reqURL := "https://ai.tu.qq.com/trpc.shadow_cv.ai_processor_cgi.AIProcessorCgi/Process"
	reqBody := &ReqBody{
		BusiID: "different_dimension_me_img_entry",
		Images: []string{base64Img},
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "https://h5.tu.qq.com")
	req.Header.Add("x-sign-version", "v1")
	req.Header.Add("x-sign-value", GetMD5Hash(fmt.Sprintf("https://h5.tu.qq.com%dHQ31X02e", len(jsonBytes))))

	log.Info().RawJSON("body", jsonBytes).Msg("API request:")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Info().RawJSON("body", respBytes).Msg("API response:")

	var respBody *RespBody
	err = json.Unmarshal(respBytes, &respBody)
	if err != nil {
		return nil, err
	}

	if respBody.Code != 0 {
		return nil, Error{Code: respBody.Code, Message: respBody.Msg}
	}

	var extra *Extra
	err = json.Unmarshal([]byte(respBody.Extra), &extra)
	if err != nil {
		return nil, err
	}

	return extra.ImgURLs, err
}

func GetMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func File(bot *gotgbot.Bot, file *gotgbot.File) (io.ReadCloser, error) {

	url := bot.GetAPIURL() + "/file/bot" + bot.GetToken() + "/" + file.FilePath

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("telebot: expected status 200 but got %s", resp.Status)
	}

	return resp.Body, nil
}
